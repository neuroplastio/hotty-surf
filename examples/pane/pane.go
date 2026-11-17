package main

import (
	"bytes"
	"context"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/reload"
)

// listW is the list's width in cells; the surface has the rest.
const listW = 24

// program is the raw-mode program: it owns the terminal, the loop and the
// layout, and is its surface's driver (every call to s is on run's loop).
type program struct {
	t      *hottyterm.Term
	a      *App
	s      *surf.Surface
	native bool
	at     rect // the surface's cells
	// inPane says whether the surface has the keyboard. The surface's own
	// focus and blur say so: the host's (or the Screen's) when the user
	// clicks or tabs, and the program's when it gives it (focus).
	inPane bool
	// refocused is set once focus picked again after the host refused the
	// element it named, until the program's next Tab.
	refocused bool

	// On a terminal that is no host, the surface is in cells: scr, drawn
	// by the program at at.
	scr   *cells.Screen
	stale bool  // the Screen refused a delta: it gets the document again
	err   error // the first error an event the Screen sent led to

	calls chan func() // work for the loop from other goroutines
}

type rect struct{ X, Y, W, H int }

func newProgram(t *hottyterm.Term, a *App) *program {
	return &program{t: t, a: a, s: newSurface(t.Surface("details"), Details(a)), calls: make(chan func())}
}

// run shows the program until ctx is done or the user quits.
func (p *program) run(ctx context.Context) error {
	p.native = p.t.Detect(ctx)
	p.layout()
	name := p.s.Name()
	if p.native {
		// Each command in one write of its own; Press and Hover on, as
		// Outside and Hover handlers need (surf.Surface.Attach).
		p.s.Attach(commands{p.t}, p.t.Caps())
		if err := p.show(); err != nil {
			return err
		}
		defer func() {
			p.t.Send(hotty.Del(name))
			p.t.Fence(context.Background())
		}()
	} else {
		if err := p.t.Send(ansi.SetModeMouseButtonEvent, ansi.SetModeMouseExtSgr, ansi.SetModeBracketedPaste); err != nil {
			return err
		}
		defer p.t.Send(ansi.ResetModeBracketedPaste, ansi.ResetModeMouseExtSgr, ansi.ResetModeMouseButtonEvent)
		p.scr = cells.New(p.at.W, p.at.H, p.event)
		if err := p.start(); err != nil {
			return err
		}
	}
	events := p.t.Events(ctx)
	sizes := p.t.Sizes()
	for {
		if err := p.draw(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case f := <-p.calls:
			f()
		case <-p.s.Pending():
			if err := p.s.Update(); err != nil {
				return err
			}
		case _, ok := <-sizes:
			if !ok {
				return nil
			}
			p.layout()
			if p.native {
				if err := p.t.Send(p.place()); err != nil {
					return err
				}
			} else {
				p.scr.Resize(p.at.W, p.at.H)
			}
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			quit, err := p.input(ev)
			if err != nil || quit {
				return err
			}
		}
		if p.err != nil {
			return p.err
		}
		if p.stale {
			if err := p.start(); err != nil {
				return err
			}
		}
	}
}

// layout is the program's: the list on the left, the surface on the rest.
// The program places the surface itself, so it tells live reload the size
// (nothing outside a dev build).
func (p *program) layout() {
	z := p.t.Size()
	p.at = rect{X: listW, Y: 0, W: max(z.Cols-listW, 1), H: max(z.Rows, 1)}
	reload.Placed(p.s, p.at.W, p.at.H)
}

// show sends a host the surface's document and places it.
func (p *program) show() error {
	doc, err := p.s.Document()
	if err != nil {
		return err
	}
	return p.t.Send(hotty.Doc(p.s.Name(), doc), p.place())
}

func (p *program) place() string {
	return hotty.PlaceAt(p.s.Name(), p.at.X, p.at.Y, hotty.Placement{
		Cols: p.at.W, Rows: p.at.H, Press: hotty.On, Hover: hotty.On})
}

// commands writes each command the surface writes in a write of its own.
type commands struct{ t *hottyterm.Term }

func (c commands) Write(b []byte) (int, error) {
	if err := c.t.Send(string(b)); err != nil {
		return 0, err
	}
	return len(b), nil
}

// start puts the surface on its Screen.
func (p *program) start() error {
	p.s.Attach(cellsOut{p}, hotty.Caps{})
	p.stale = false
	doc, err := p.s.Document()
	if err != nil {
		return err
	}
	_, err = p.scr.Write([]byte(hotty.Doc(p.s.Name(), doc)))
	return err
}

// cellsOut is the surface's writer in cells. The Screen's refusal is a
// host's error reply, not the program's: a refused delta sends the
// document again, and the rest change nothing.
type cellsOut struct{ p *program }

func (c cellsOut) Write(b []byte) (int, error) {
	if _, err := c.p.scr.Write(b); err != nil && bytes.HasPrefix(b, []byte("\x1b]7279;a=delta")) {
		c.p.stale = true
	}
	return len(b), nil
}

// event is an event the Screen sends the surface, as a host would.
func (p *program) event(e hotty.Event) {
	p.track(e)
	if err := p.s.Event(e); err != nil && p.err == nil {
		p.err = err
	}
}

// track follows the keyboard: the surface's focus and blur.
func (p *program) track(e hotty.Event) {
	switch e.Kind {
	case hotty.EventFocus:
		p.inPane = true
	case hotty.EventBlur:
		p.inPane = false
	}
}

// input takes what the terminal sent; quit says the user quit.
func (p *program) input(ev hottyterm.Event) (quit bool, err error) {
	name := p.s.Name()
	switch ev := ev.(type) {
	case hottyterm.Message: // a host's events and replies
		if e, ok := hotty.EventOf(ev.Message); ok && e.Surface == name {
			p.track(e)
			return false, p.s.Event(e)
		}
		if r, ok := hotty.ReplyOf(ev.Message); ok && !r.OK && r.Surface == name {
			switch {
			case r.Re == "delta":
				return false, p.show() // what the host has is no longer the surface's
			case r.Re == "focus" && p.inPane:
				// The host refused the element focus named. The surface
				// took the move already: it hears that it didn't happen.
				// Where the element is gone (ENOTARGET) it picks again from
				// its tree as it is now, once; a second refusal, or a
				// surface that can't have the keyboard, leaves the keyboard
				// with the list.
				p.inPane = false
				if err := p.s.Event(hotty.Event{Kind: hotty.EventBlur, Surface: name}); err != nil {
					return false, err
				}
				if r.Code == hotty.ENOTARGET && !p.refocused {
					p.refocused = true
					return false, p.giveFocus()
				}
			}
		}
	case uv.KeyPressEvent:
		k := hottyterm.KeyName(uv.Key(ev))
		if k == "" {
			return false, nil
		}
		if p.inPane {
			// The Screen's first (a field, Tab inside the surface and
			// past its last element), then the surface's Key handlers.
			if p.scr != nil && p.scr.Key(k) {
				return false, nil
			}
			taken, err := p.s.Key(k)
			if err != nil || taken {
				return false, err
			}
		}
		return p.own(k)
	case uv.PasteEvent:
		if p.scr != nil && p.inPane {
			p.scr.Paste(ev.Content)
		}
	case uv.MouseClickEvent: // in cells only: a host sends the program none
		if p.scr == nil || ev.Button != uv.MouseLeft {
			return false, nil
		}
		if x, y, in := p.in(ev.X, ev.Y); in {
			p.scr.Click(x, y)
			return false, nil
		}
		if p.inPane {
			// A click elsewhere gives the keyboard back, as on a host.
			p.scr.Focus(false)
		}
		if i := ev.Y - 1; ev.X < listW && i >= 0 && i < len(p.a.Files) {
			p.pick(i)
		}
	case uv.MouseWheelEvent:
		if p.scr == nil {
			return false, nil
		}
		if x, y, in := p.in(ev.X, ev.Y); in {
			switch ev.Button {
			case uv.MouseWheelDown:
				p.scr.Wheel(x, y, 3)
			case uv.MouseWheelUp:
				p.scr.Wheel(x, y, -3)
			}
		}
	}
	return false, nil
}

// in is a screen cell in the surface's own cells, and whether it is in it.
func (p *program) in(x, y int) (int, int, bool) {
	x, y = x-p.at.X, y-p.at.Y
	return x, y, x >= 0 && y >= 0 && x < p.at.W && y < p.at.H
}

// own is a key the surface didn't take: the program's.
func (p *program) own(k string) (quit bool, err error) {
	switch k {
	case "q", "ctrl+c":
		return true, nil
	case "j", "arrow_down":
		p.pick(min(p.a.Sel+1, len(p.a.Files)-1))
	case "k", "arrow_up":
		p.pick(max(p.a.Sel-1, 0))
	case "tab":
		p.refocused = false
		return false, p.giveFocus()
	}
	return false, nil
}

// pick selects file i: the surface renders it, through Do.
func (p *program) pick(i int) {
	p.s.Do(func() { p.a.Sel = i })
}

// giveFocus gives the surface the keyboard (the program's Tab). In cells the
// Screen gets it first, and gives the element it kept back, which the
// surface hears as the user's focus; otherwise (Tab went past the last
// element, which leaves the Screen active with nothing focused) the
// surface's Focus picks one and sends its a=focus to the Screen. On a host
// Focus sends a=focus to the host, which echoes nothing (SPEC §10.1): the
// surface takes the move as its focus event itself. Where nothing in the
// surface takes focus, the list keeps the keyboard.
func (p *program) giveFocus() error {
	if p.scr != nil {
		p.scr.Focus(true)
	}
	got, err := p.s.Focus()
	if err != nil {
		return err
	}
	if !got && p.scr != nil {
		p.scr.Focus(false)
	}
	p.inPane = got
	return nil
}

// draw writes the list, and in cells the surface beside it, in one write.
func (p *program) draw() error {
	var b strings.Builder
	b.WriteString(ansi.HideCursor)
	var lines []string
	if p.scr != nil {
		lines = p.scr.Lines()
	}
	for row := range p.at.H {
		b.WriteString(ansi.CursorPosition(1, row+1))
		b.WriteString(p.listLine(row))
		b.WriteString(ansi.ResetStyle)
		if row < len(lines) {
			b.WriteString(lines[row])
			b.WriteString(ansi.ResetStyle)
		}
	}
	if p.scr != nil && p.inPane {
		if col, row, ok := p.scr.Cursor(); ok {
			b.WriteString(ansi.CursorPosition(p.at.X+col+1, p.at.Y+row+1))
			b.WriteString(ansi.ShowCursor)
		}
	}
	return p.t.Send(b.String())
}

// listLine is the list's row, listW cells wide: the heading, bold while the
// list has the keyboard, then a file a row.
func (p *program) listLine(row int) string {
	var s string
	switch i := row - 1; {
	case row == 0:
		s = "Files"
		if !p.inPane {
			return ansi.SGR(ansi.AttrBold) + pad(s)
		}
	case i < len(p.a.Files):
		s = "  " + p.a.Files[i].Name
		if i == p.a.Sel {
			s = "> " + p.a.Files[i].Name
		}
	}
	return pad(s)
}

func pad(s string) string {
	r := []rune(s)
	if len(r) > listW-1 {
		r = r[:listW-1]
	}
	return string(r) + strings.Repeat(" ", listW-len(r))
}
