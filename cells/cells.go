// Package cells draws a surface in character cells, for a terminal or a
// viewer that is no hotty host. A Screen is a hotty host inside the
// program: it is the writer a surface is attached to, so the surface writes
// it the same commands it would write a terminal, and the Screen lays the
// document out in cells (blocks, flex, tables, text), keeps its focus and
// its fields, and gives the program the events a host would send. It has to
// work, not look good: the CSS it reads is a subset, written down in
// docs/coverage.md.
//
// The driver's loop is the same as with a host: the program writes the
// surface's document to the Screen (hotty-go's Doc), hands the events the
// Screen sends to the surface's Event, and offers the surface the keys the
// Screen doesn't take. A stream of commands can be teed, so one surface can
// show on a host and in cells at once (plx's viewers with and without
// hotty), and the Screen can draw a surface in another process.
//
// Package run uses a Screen when its terminal is no host. A program that
// owns its screen (a Bubble Tea program in a plain terminal, through
// surftea; a plx tool) draws a Screen's lines where the surface goes.
package cells

import (
	"neuroplast.io/hotty-go"
)

// A Screen is one surface drawn in a rectangle of cells.
type Screen struct {
	_ struct{}
}

// New makes a Screen cols wide and rows high. events hears each event the
// Screen sends the surface, on the goroutine of the call that caused it
// (Write, Key, Paste, Click, Wheel, Focus); the program hands it to the
// surface's Event, and handles what Event returns.
func New(cols, rows int, events func(hotty.Event)) *Screen {
	panic("L0")
}

// Write takes hotty commands, as a surface writes them to a terminal, one
// whole command per Write: the document, its placement, and the deltas
// after. A command the Screen can't apply is an error, as a host's refusal
// is: the program sends the surface's document again.
func (c *Screen) Write(p []byte) (int, error) {
	panic("L0")
}

// Resize makes the Screen cols wide and rows high, and lays the document out
// again.
func (c *Screen) Resize(cols, rows int) {
	panic("L0")
}

// Key presses a key on the Screen, named as hotty names keys. The focused
// field or control takes it if it can (typing, Tab between controls, Enter
// on a button), and taken says whether it did. A key it doesn't take is
// what a host passes the program: the program offers it to the surface's
// Key, and keeps it if the surface doesn't take it either.
func (c *Screen) Key(key string) (taken bool) {
	panic("L0")
}

// Paste puts text into the focused field, as a bracketed paste reaches a
// host; taken is false when no field has the keyboard.
func (c *Screen) Paste(text string) (taken bool) {
	panic("L0")
}

// Click clicks the cell at col, row of the Screen, from its top left; taken
// says whether it landed on something of the surface's.
func (c *Screen) Click(col, row int) (taken bool) {
	panic("L0")
}

// Wheel scrolls what is under the cell at col, row by lines, down for a
// positive number; taken says whether anything scrolled.
func (c *Screen) Wheel(col, row, lines int) (taken bool) {
	panic("L0")
}

// Focus gives the Screen the keyboard (in) or takes it away, as the
// program's own focus moves onto the surface's rectangle or off it. Taking
// it commits a field being edited and blurs the surface, as a host's blur
// does.
func (c *Screen) Focus(in bool) {
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
