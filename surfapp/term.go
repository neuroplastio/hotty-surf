//go:build !js

package surfapp

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfui"
)

// app runs s in the process's terminal: on the alternate screen, as a
// surface over all of it when the terminal is a hotty host, in cells when
// it isn't.
func app(ctx context.Context, s *surf.Surface) error {
	driverDone := driverStarted(s)
	defer driverDone()
	t, err := hottyterm.Open(s.Name())
	if err != nil {
		return err
	}
	defer t.Close()
	restore, err := t.Raw()
	if err != nil {
		return err
	}
	defer restore()
	native := t.Detect(ctx)
	// The terminal's own modes are x/ansi's sequences, not hotty's.
	if err := t.Send(ansi.SetModeAltScreenSaveCursor, ansi.HideCursor, ansi.EraseEntireScreen, "\x1b[>1u"); err != nil {
		return err
	}
	defer t.Send("\x1b[<u", ansi.ShowCursor, ansi.ResetModeAltScreenSaveCursor)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if native {
		return host(ctx, t, s, winch)
	}
	surfui.SetKittyDefault(t.KittyGraphics(ctx))
	return inCells(ctx, t, s, winch)
}

// commands writes each command a surface writes to the terminal in a
// write of its own (Term.Send), so nothing lands between its parts.
type commands struct{ t *hottyterm.Term }

func (c commands) Write(p []byte) (int, error) {
	if err := c.t.Send(string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// host shows s on a hotty host, over the whole screen.
func host(ctx context.Context, t *hottyterm.Term, s *surf.Surface, winch <-chan os.Signal) error {
	name := s.Name()
	s.Attach(commands{t}, t.Caps())
	place := func() error {
		z := t.Size()
		if err := t.Send(hotty.Place(name, hotty.Placement{Cols: z.Cols, Rows: z.Rows, Press: hotty.On, Hover: hotty.On})); err != nil {
			return err
		}
		didPlace(s, z.Cols, z.Rows)
		return nil
	}
	show := func() error {
		doc, err := s.Document()
		if err != nil {
			return err
		}
		if err := t.Send(hotty.Doc(name, doc, hotty.Scroll(hotty.ScrollVertical|hotty.ScrollHorizontal))); err != nil {
			return err
		}
		return place()
	}
	if err := show(); err != nil {
		return err
	}
	defer func() {
		t.Send(hotty.Del(name))
		t.Fence(context.Background())
	}()
	events := t.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.Pending():
			if err := s.Update(); err != nil {
				return err
			}
		case <-winch:
			if err := place(); err != nil {
				return err
			}
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			switch ev := ev.(type) {
			case hottyterm.Message:
				if e, ok := hotty.EventOf(ev.Message); ok {
					if err := s.Event(e); err != nil {
						return err
					}
				} else if r, ok := hotty.ReplyOf(ev.Message); ok && !r.OK && r.Surface == name && r.Re == "delta" {
					// The host refused a delta: what it has is no longer
					// what the surface thinks, so it gets the document.
					if err := show(); err != nil {
						return err
					}
				}
			case uv.KeyPressEvent:
				k := hottyterm.KeyName(uv.Key(ev))
				if k == "" {
					continue
				}
				quit, err := key(s, k)
				if err != nil {
					return err
				}
				if quit {
					return nil
				}
			}
		}
	}
}

// inCells draws s in character cells (package cells), on a terminal that
// is no hotty host.
func inCells(ctx context.Context, t *hottyterm.Term, s *surf.Surface, winch <-chan os.Signal) error {
	c := newCellsHost(s, t.Size().Cols, t.Size().Rows)
	didPlace(s, t.Size().Cols, t.Size().Rows)
	if err := t.Send(ansi.SetModeMouseButtonEvent, ansi.SetModeMouseExtSgr, ansi.SetModeBracketedPaste); err != nil {
		return err
	}
	defer t.Send(ansi.ResetModeBracketedPaste, ansi.ResetModeMouseExtSgr, ansi.ResetModeMouseButtonEvent)
	if err := c.start(); err != nil {
		return err
	}
	draw := func() error { return t.Send(c.frame()) }
	if err := draw(); err != nil {
		return err
	}
	events := t.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.Pending():
			if err := s.Update(); err != nil {
				return err
			}
		case <-winch:
			z := t.Size()
			c.scr.Resize(z.Cols, z.Rows)
			didPlace(s, z.Cols, z.Rows)
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			switch ev := ev.(type) {
			case uv.KeyPressEvent:
				k := hottyterm.KeyName(uv.Key(ev))
				if k == "" {
					continue
				}
				quit, err := c.key(k)
				if err != nil || quit {
					return err
				}
			case uv.PasteEvent:
				c.scr.Paste(ev.Content)
			case uv.MouseClickEvent:
				if ev.Button == uv.MouseLeft {
					c.scr.Click(ev.X, ev.Y)
				}
			case uv.MouseWheelEvent:
				switch ev.Button {
				case uv.MouseWheelDown:
					c.scr.Wheel(ev.X, ev.Y, 3)
				case uv.MouseWheelUp:
					c.scr.Wheel(ev.X, ev.Y, -3)
				}
			}
		}
		if err := c.err; err != nil {
			return err
		}
		if c.stale {
			if err := c.start(); err != nil {
				return err
			}
		}
		if err := draw(); err != nil {
			return err
		}
	}
}
