package devcmd

import (
	"io"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"

	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/internal/dev/wire"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// cellsView is `hotty-surf run` in a terminal that is no hotty host: the
// app's surface and the report's each on a surftui.Screen here, in run's
// process, drawn into the terminal. The Screens outlive the app's
// processes, as a host's surfaces do, so a new build's first frame is a
// delta from what the old one drew.
//
// Every call is on run's loop: the app's commands come to it through outs.
type cellsView struct {
	h    *host
	app  *surftui.Screen
	dev  *surftui.Screen
	outs chan []byte

	cols, rows int
	devRows    int // the report's rows at the bottom; 0 while it is hidden
	devStale   bool
}

func newCellsView(h *host, cols, rows int) *cellsView {
	c := &cellsView{h: h, outs: make(chan []byte, 256), cols: cols, rows: rows}
	c.app = surftui.New(cols, rows, c.appEvent)
	c.dev = surftui.New(cols, 1, func(e hotty.Event) {
		if err := h.dev.Event(e); err != nil && h.devErr == nil {
			h.devErr = err
		}
	})
	return c
}

// appEvent sends the app an event its Screen sent, as a host would.
func (c *cellsView) appEvent(e hotty.Event) {
	in := devwire.GetInput()
	in.Msg = append(in.Msg, hotty.EncodeEvent(e)...)
	c.h.send(in)
}

// out applies one command the app sent. A refused delta is answered as a
// host answers it, so that the app sends its document again; the Screen's
// other refusals change nothing.
func (c *cellsView) out(p []byte) {
	if _, err := c.app.Write(p); err != nil && strings.HasPrefix(string(p), "\x1b]7279;a=delta") {
		in := devwire.GetInput()
		in.Msg = append(in.Msg, hotty.ReplyErr("", c.h.appName(), "delta", "EINVAL", err.Error())...)
		c.h.send(in)
	}
}

// devWriter is the report surface's writer: its Screen, and a refused
// delta sends its document again.
type devWriter struct{ c *cellsView }

func (w devWriter) Write(p []byte) (int, error) {
	if _, err := w.c.dev.Write(p); err != nil && strings.HasPrefix(string(p), "\x1b]7279;a=delta") {
		w.c.devStale = true
	}
	return len(p), nil
}

// Wrapped is the report's Screen, which the report surface offers its
// tree (rt.Wrapper).
func (w devWriter) Wrapped(rt.Seal) io.Writer { return w.c.dev }

// resize lays both out again at the terminal's size.
func (c *cellsView) resize(cols, rows int) {
	c.cols, c.rows = cols, rows
	c.app.Resize(cols, rows)
	c.placeDev(c.devRows)
}

// placeDev shows the report in its bottom rows (0 hides it).
func (c *cellsView) placeDev(rows int) {
	c.devRows = min(rows, c.rows)
	if c.devRows > 0 {
		c.dev.Resize(c.cols, c.devRows)
	}
}

// top is the report's first row.
func (c *cellsView) top() int { return c.rows - c.devRows }

// input takes what the terminal sent; quit says the user quit while no app
// runs.
func (c *cellsView) input(ev hottyterm.Event) (quit bool) {
	switch ev := ev.(type) {
	case uv.KeyPressEvent:
		k := hottyterm.KeyName(uv.Key(ev))
		if k == "" {
			return false
		}
		if c.app.Key(k) {
			return false
		}
		if !c.h.running() {
			return k == "ctrl+c" || k == "ctrl+q"
		}
		in := devwire.GetInput()
		in.Key = k
		c.h.send(in)
	case uv.PasteEvent:
		c.app.Paste(ev.Content)
	case uv.MouseClickEvent:
		if ev.Button != uv.MouseLeft {
			return false
		}
		if c.devRows > 0 && ev.Y >= c.top() {
			c.app.Focus(false)
			c.dev.Click(ev.X, ev.Y-c.top())
			return false
		}
		c.app.Focus(true)
		c.app.Click(ev.X, ev.Y)
	case uv.MouseWheelEvent:
		n := 3
		if ev.Button == uv.MouseWheelUp {
			n = -3
		} else if ev.Button != uv.MouseWheelDown {
			return false
		}
		if c.devRows > 0 && ev.Y >= c.top() {
			c.dev.Wheel(ev.X, ev.Y-c.top(), n)
		} else {
			c.app.Wheel(ev.X, ev.Y, n)
		}
	}
	return false
}

// frame is both Screens' lines as one write: the app's, the report's over
// its bottom rows, then the caret where a field has it.
func (c *cellsView) frame() string {
	var b strings.Builder
	b.WriteString(ansi.HideCursor)
	lines := c.app.Lines()
	if c.devRows > 0 {
		dev := c.dev.Lines()
		for i := range dev {
			if j := c.top() + i; j >= 0 && j < len(lines) {
				lines[j] = dev[i]
			}
		}
	}
	for i, l := range lines {
		b.WriteString(ansi.CursorPosition(1, i+1))
		b.WriteString(ansi.EraseLineRight)
		b.WriteString(l)
		b.WriteString(ansi.ResetStyle)
	}
	if col, row, ok := c.app.Cursor(); ok && (c.devRows == 0 || row < c.top()) {
		b.WriteString(ansi.CursorPosition(col+1, row+1))
		b.WriteString(ansi.ShowCursor)
	}
	return b.String()
}
