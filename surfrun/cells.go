package surfrun

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottyterm"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Cells is the surface as a cells.Program, for a renderer that hosts it
// in a terminal's cells (cells.New): the runtime's own where there is no
// host, and a host's that paints its own cells.
func (s *Surface) Cells() cells.Program { return program{s} }

// program is a Surface as cells.Program has it.
type program struct{ s *Surface }

func (p program) Show() cells.Frame                           { return cellsFrame(p.s.Show()) }
func (p program) Update() cells.Frame                         { return cellsFrame(p.s.Update()) }
func (p program) Tree() *surfui.Node                          { return p.s.Tree() }
func (p program) Event(e hotty.Event) (took, redraw bool)     { return p.s.Event(e) }
func (p program) ListEvent(e hotty.Event) (took, redraw bool) { return p.s.ListEvent(e) }
func (p program) Key(name string) (took, redraw bool)         { return p.s.Key(name) }
func (p program) Keys() ([]cells.KeyUse, cells.Chars)         { return p.s.Keys() }

// Hues are the surface's, which the renderer draws each in its terminal
// colour (cells.Hued).
func (p program) Hues() surfui.Palette { return p.s.Hues() }

// cellsFrame is f for a renderer: its ops are in the tree it draws, and
// its reveals the ids it brings into view.
func cellsFrame(f Frame) cells.Frame {
	out := cells.Frame{Blur: f.Blur, Focus: f.Focus}
	for _, fl := range f.Fields {
		out.Fields = append(out.Fields, cells.FieldSet{ID: fl.ID, Change: fl.Change})
	}
	if l := f.List; l != nil {
		out.List = &cells.List{Closed: l.Closed, Tree: l.Tree, At: l.At}
	}
	for _, o := range slices.Concat(f.Ops, f.Reveals) {
		if o.Op == hotty.OpReveal(o.Target, "").Op {
			out.Reveals = append(out.Reveals, o.Target)
		}
	}
	return out
}

// toCells draws the model in cells: with its own cells UI (Cells), or
// else with a renderer that hosts its surface there, which takes the
// keys, the mouse and pastes as a host does.
func (r *Runtime) toCells() {
	if _, ok := r.m.(Cells); ok {
		r.leaveCells()
		r.render()
		r.draw()
		return
	}
	r.s.Platform = hotty.Platform{}
	r.rend = cells.New(r.s.Cells())
	r.rend.Scroll = r.opts.Scroll
	r.rend.Show()
	r.gen = 0
	r.draw()
}

// leaveCells stops the renderer, the terminal's mouse and the caret
// with it: for the host, or the end.
func (r *Runtime) leaveCells() {
	var b strings.Builder
	if r.mouse {
		b.WriteString(mouseOff)
		r.mouse = false
	}
	if r.caret {
		b.WriteString("\x1b[?25l\x1b[0 q")
		r.caret = false
	}
	r.write(b.String())
	r.rend = nil
}

// Renderer is what draws the model's tree in cells, nil while nothing
// does (on a host, or for a model with its own Cells): for a test, on
// the runtime's goroutine (Call).
func (r *Runtime) Renderer() *cells.Renderer { return r.rend }

// The terminal's mouse reporting, its presses, releases and wheel in SGR's
// encoding (DECSET 1000 and 1006), and its moves while a button is down
// (1002), which drag an element (cells.Renderer.Move): hottyterm reads
// them as ultraviolet's mouse events. hotty-go has no call for it: it is a mode of the
// program's own screen, as the alternate screen is, and the runtime
// writes it itself, only while the renderer has something a press or the
// wheel acts on, so that the terminal's own selection works otherwise.
const (
	mouseOn  = "\x1b[?1000h\x1b[?1006h\x1b[?1002h"
	mouseOff = "\x1b[?1002l\x1b[?1006l\x1b[?1000l"
)

// drawTree writes the renderer's screen, unless it is the one written
// last: each row's runs in the terminal's colours (cellsSGR), then the
// caret, a bar where the focused field has it. The runtime writes these
// bytes itself, as it writes a cells UI's lines: they are the terminal's
// screen, not hotty's protocol.
func (r *Runtime) drawTree() {
	r.rend.KeyStyle = r.s.keyStyle()
	s := r.rend.Draw(r.cols, r.surfaceRows())
	// An open menu is drawn over the screen, and reported the mouse for.
	cm, open := r.cellsMenu()
	sig := ""
	if open {
		sig = cm.sig()
		s.Lines = cm.overlay(s.Lines)
		s.Mouse = true
	}
	if s.Gen == r.gen && sig == r.menuSig {
		return
	}
	r.gen, r.menuSig = s.Gen, sig
	var b strings.Builder
	b.WriteString("\x1b[?2026h")
	r.headerTo(&b)
	if s.Mouse != r.mouse {
		r.mouse = s.Mouse
		if s.Mouse {
			b.WriteString(mouseOn)
		} else {
			b.WriteString(mouseOff)
		}
	}
	for i, row := range Paint(s) {
		fmt.Fprintf(&b, "\x1b[%d;1H", r.opts.Top+1+i)
		b.WriteString(row)
	}
	switch c := s.Caret; {
	case c.Visible:
		shape := "6" // a steady bar
		if c.Shape == cells.Block {
			shape = "2"
		}
		fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[%s q\x1b[?25h", r.opts.Top+1+c.Y, c.X+1, shape)
		r.caret = true
	case r.caret:
		b.WriteString("\x1b[?25l\x1b[0 q")
		r.caret = false
	}
	b.WriteString("\x1b[?2026l")
	r.write(b.String())
}

// Lines is a screen as the runtime paints it, one string a row (Paint).
// It is for a program that prints its tree once among its output, as a
// command's result or a shell's greeting is, and leaves the terminal to
// what runs next: no cursor moves, no modes. The rows at the screen's
// end that show nothing are left out.
func Lines(s cells.Screen) []string {
	end := len(s.Lines)
	for end > 0 && blank(s.Lines[end-1]) {
		end--
	}
	s.Lines = s.Lines[:end]
	return Paint(s)
}

// Paint is a screen as the runtime paints it, one string a row: each run
// in the terminal's colours (cellsSGR), the row ending in a reset. It is
// for a program that places a screen's rows itself, such as a View drawn
// in lines. It keeps every row, so the screen's height holds; Lines
// leaves out the blank ones at the end. No cursor moves, no modes.
func Paint(s cells.Screen) []string {
	out := make([]string, len(s.Lines))
	for i, line := range s.Lines {
		var b strings.Builder
		for _, run := range line {
			b.WriteString(cellsSGR(run))
			b.WriteString(run.Text)
		}
		b.WriteString("\x1b[0m")
		out[i] = b.String()
	}
	return out
}

// blank reports whether a row shows nothing in the sixteen colours:
// spaces, with no fill cellsSGR paints and no line.
func blank(line []cells.Run) bool {
	for _, run := range line {
		if strings.Trim(run.Text, " ") != "" || run.Attr&(cells.Reverse|cells.Underline|cells.Strike|cells.Overline) != 0 {
			return false
		}
		if s := cellsSGR(cells.Run{Fg: run.Fg, Bg: run.Bg}); strings.Contains(s, ";7") || strings.Contains(s, ";100") {
			return false
		}
	}
	return true
}

// cellsEvent is the mouse, or the terminal's focus, for the renderer: at
// the surface's cells, below the header.
func (r *Runtime) cellsEvent(ev hottyterm.Event) {
	if r.rend == nil {
		return
	}
	r.settled = true
	if r.cellsMenuEvent(ev) {
		r.settled = false // the menu may have opened, closed or picked
		return
	}
	switch e := ev.(type) {
	case uv.MouseClickEvent:
		if e.Button == uv.MouseLeft {
			r.rend.Press(e.X, e.Y-r.opts.Top, mods(e.Mod))
		}
	case uv.MouseMotionEvent:
		if e.Button == uv.MouseLeft {
			r.rend.Move(e.X, e.Y-r.opts.Top, mods(e.Mod))
		}
	case uv.MouseReleaseEvent:
		if e.Button == uv.MouseLeft || e.Button == uv.MouseNone {
			r.rend.Release(e.X, e.Y-r.opts.Top, mods(e.Mod))
		}
	case uv.MouseWheelEvent:
		switch e.Button {
		case uv.MouseWheelUp:
			r.rend.Wheel(e.X, e.Y-r.opts.Top, -1)
		case uv.MouseWheelDown:
			r.rend.Wheel(e.X, e.Y-r.opts.Top, 1)
		}
	case uv.FocusEvent:
		r.rend.Focus(true)
	case uv.BlurEvent:
		r.rend.Focus(false)
	default:
		r.settled = false
	}
}

// mods are a mouse event's modifiers as hotty's bits.
func mods(m uv.KeyMod) int {
	bits := 0
	for _, k := range [...]struct {
		uv  uv.KeyMod
		bit int
	}{{uv.ModShift, hotty.ModShift}, {uv.ModAlt, hotty.ModAlt}, {uv.ModCtrl, hotty.ModCtrl}, {uv.ModSuper, hotty.ModSuper}} {
		if m.Contains(k.uv) {
			bits |= k.bit
		}
	}
	return bits
}

// cellsSGR is a run's look in the terminal's palette, as surf.css's
// terminal theme maps the roles: ok to ansi 10, warn to 11, error to 9,
// info to 14, the border and a rule's line to 8; the accent to ansi 5
// (magenta), since a terminal names no accent of its own; muted and faint
// dim. The fills: field and line on ansi 8 (the terminal's grey); tonal
// and a box's surface on the ground, since tonal is a surface too (a
// Compact box's), and in the one grey the terminal has, a pane of it is
// a slab; a list's selection while the list has the keyboard (selected)
// on ansi 8, its text in its own colour, so that a focused entry's ▎
// shows on it, as the host's quiet selected fill: a row of the grey is
// no slab; a control with the keyboard (sel) in reverse video with its
// tones dropped;
// and a tone's fill, as text in the ground's colour on any fill
// (on-accent, a Compact selection's and a focused button's), the fill's
// colour in reverse video: its text is the ground's own colour, where
// ansi black on the fill is the tone on itself in many palettes.
func cellsSGR(run cells.Run) string {
	ps := []string{"0"}
	fg := run.Fg
	switch bg := run.Bg; {
	case bg == cells.Sel:
		ps = append(ps, "7")
		fg = cells.Fg
	case fg == cells.OnAccent || fg == cells.Bg || tone(bg):
		if c := ansiFg(bg); c != "" {
			ps = append(ps, c)
		}
		ps = append(ps, "7")
		fg = ""
	case bg == cells.Selected || bg == cells.Field || bg == cells.Line || bg == cells.Border || bg == cells.Muted || bg == cells.Faint:
		ps = append(ps, "100")
	}
	if c := ansiFg(fg); c != "" {
		ps = append(ps, c)
	}
	for _, a := range [...]struct {
		attr cells.Attr
		sgr  string
	}{{cells.Bold, "1"}, {cells.Dim, "2"}, {cells.Italic, "3"}, {cells.Underline, "4"}, {cells.Reverse, "7"}, {cells.Strike, "9"}, {cells.Overline, "53"}} {
		if run.Attr&a.attr != 0 {
			ps = append(ps, a.sgr)
		}
	}
	if c := ansiIndex(run.Ul); c != "" && run.Attr&(cells.Underline|cells.Overline) != 0 {
		ps = append(ps, "58;5;"+c) // the line's colour; a terminal without it draws it in the text's
	}
	return "\x1b[" + strings.Join(ps, ";") + "m"
}

// ansiFg is a role's colour as text's SGR, "" for the terminal's own.
func ansiFg(r cells.Role) string {
	if n, ok := r.Ansi(); ok {
		// A hue: one of the terminal's 16 colours, as its palette has it.
		if n < 8 {
			return strconv.Itoa(30 + n)
		}
		return strconv.Itoa(90 + n - 8)
	}
	switch r {
	case cells.Accent:
		return "35"
	case cells.OK:
		return "92"
	case cells.Warn:
		return "93"
	case cells.Error:
		return "91"
	case cells.Info:
		return "96"
	case cells.Border, cells.Line:
		return "90"
	case cells.Muted, cells.Faint:
		return "2"
	}
	return ""
}

// ansiIndex is a role's colour as an index of the terminal's palette, for
// a line's colour (SGR 58;5), as ansiFg maps it; "" for the text's.
func ansiIndex(r cells.Role) string {
	switch r {
	case cells.Accent:
		return "5"
	case cells.OK:
		return "10"
	case cells.Warn:
		return "11"
	case cells.Error:
		return "9"
	case cells.Info:
		return "14"
	case cells.Border, cells.Line:
		return "8"
	}
	return ""
}

// tone reports whether r is a tone, which fills as its colour.
func tone(r cells.Role) bool {
	switch r {
	case cells.Accent, cells.OK, cells.Warn, cells.Error, cells.Info:
		return true
	}
	return false
}
