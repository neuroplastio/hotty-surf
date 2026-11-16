// Package cells draws a surface in character cells, for a terminal that is
// no hotty host. A Screen is a hotty host inside the program, for one
// surface: the surface writes it the commands it would write a terminal, and
// the Screen lays the document out in cells (blocks, flex, tables, text),
// keeps its focus and its fields, and gives the surface the events and keys
// a host would. It has to work, not look good: the CSS it reads is a subset,
// written down in docs/coverage.md.
//
// Package run uses a Screen when its terminal is no host. A program that
// owns its screen (a Bubble Tea program in a plain terminal, through
// surftea) draws a Screen's lines where the surface goes.
package cells

import (
	"neuroplast.io/hotty-surf"
)

// A Screen is one surface drawn in a rectangle of cells.
type Screen struct {
	_ struct{}
}

// New makes a Screen cols wide and rows high, and shows s on it: the Screen
// attaches itself to s as its host and lays out its document. Later changes
// reach it as s's deltas, when the program calls s.Update.
func New(s *surf.Surface, cols, rows int) (*Screen, error) {
	panic("L0")
}

// Resize makes the Screen cols wide and rows high, and lays the document out
// again.
func (c *Screen) Resize(cols, rows int) {
	panic("L0")
}

// Key presses a key on the Screen, named as hotty names keys. The focused
// field or control takes it if it can (typing, Tab between controls, Enter on
// a button), and the surface's Key is offered the rest; taken says whether
// either took it. A key neither takes is the program's.
func (c *Screen) Key(key string) (taken bool, err error) {
	panic("L0")
}

// Click clicks the cell at col, row of the Screen, from its top left.
func (c *Screen) Click(col, row int) error {
	panic("L0")
}

// Wheel scrolls what is under the cell at col, row by lines, down for a
// positive number.
func (c *Screen) Wheel(col, row, lines int) error {
	panic("L0")
}

// Lines are the Screen's rows, each its cells as text with ANSI colours and
// attributes (SGR), for the program to draw where the surface goes.
func (c *Screen) Lines() []string {
	panic("L0")
}

// Cursor is where the focused field's caret is, from the top left, and
// whether there is one.
func (c *Screen) Cursor() (col, row int, ok bool) {
	panic("L0")
}
