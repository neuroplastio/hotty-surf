//go:build !js

package run

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// cellsHost is a surface on a cells.Screen: the host inside the program.
type cellsHost struct {
	s   *surf.Surface
	scr *cells.Screen
	err error // the first error an event the Screen sent led to
	// stale is a delta the Screen refused: it gets the document again.
	stale bool
}

// Write is the surface's writer. The Screen's refusal is a host's error
// reply, not the app's error: a refused delta sends the document again
// (as inHost's show), and the rest change nothing.
func (c *cellsHost) Write(p []byte) (int, error) {
	if _, err := c.scr.Write(p); err != nil && bytes.HasPrefix(p, []byte("\x1b]7279;a=delta")) {
		c.stale = true
	}
	return len(p), nil
}

func newCellsHost(s *surf.Surface, cols, rows int) *cellsHost {
	c := &cellsHost{s: s}
	c.scr = cells.New(cols, rows, func(e hotty.Event) {
		if err := s.Event(e); err != nil && c.err == nil {
			c.err = err
		}
	})
	return c
}

// start sends the surface's first document to the Screen.
func (c *cellsHost) start() error {
	c.s.Attach(c, hotty.Caps{})
	c.stale = false
	doc, err := c.s.Document()
	if err != nil {
		return err
	}
	_, err = c.scr.Write([]byte(hotty.Doc(c.s.Name(), doc)))
	return err
}

// key offers k to the Screen, then to the surface; quit says whether it
// quits the app.
func (c *cellsHost) key(k string) (quit bool, err error) {
	if c.scr.Key(k) {
		return false, nil
	}
	taken, err := c.s.Key(k)
	if err != nil {
		return false, err
	}
	return !taken && quits(k), nil
}

// frame is the Screen's lines as one write: each row from its first
// column, then the caret where a field has it.
func (c *cellsHost) frame() string {
	var b strings.Builder
	b.WriteString(ansi.HideCursor)
	for i, l := range c.scr.Lines() {
		b.WriteString(ansi.CursorPosition(1, i+1))
		b.WriteString(ansi.EraseLineRight) // before: after a full line it erases the last cell
		b.WriteString(l)
		b.WriteString(ansi.ResetStyle)
	}
	if col, row, ok := c.scr.Cursor(); ok {
		b.WriteString(ansi.CursorPosition(col+1, row+1))
		b.WriteString(ansi.ShowCursor)
	}
	return b.String()
}
