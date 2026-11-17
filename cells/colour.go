package cells

// Colour in cells (G-cells-colour, for plx). The Screen reads which of the
// kit's theme colours the stylesheet would paint an element's text and
// fill in, as it reads its layout: from its classes, its inline style and
// what it knows of the kit's components (SUBSET.md, "Colours"). It names
// them, and the program, which knows its theme and its terminal, paints
// them.

// A Role is one of the kit's theme colours, named as its CSS variable is
// without the leading dashes: "primary", "primary-foreground",
// "muted-foreground", "destructive", "accent", "card", "popover",
// "border", "input", "chart-1". A variable an element names in its inline
// style or an arbitrary class (`color: var(--plx-warn)`,
// `bg-[var(--plx-warn)]`) keeps its name too: "plx-warn". A role may
// carry Tailwind's opacity, "destructive/10": that much of the colour
// over what is under it, as Basecoat's tinted fills are. "" is none: the
// terminal's own text or ground.
type Role string

// Attr is a cell's text attributes, as SGR draws them.
type Attr uint8

// The attributes.
const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	// Reverse is the element that has the keyboard, as Lines draws it.
	Reverse
	Strike
)

// A Cell is one cell of the Screen: its text, a grapheme cluster ("" for
// the second cell of a wide one), the roles of its text and its fill, and
// its attributes.
type Cell struct {
	Text   string
	Fg, Bg Role
	Attr   Attr
}

// Cells are the Screen's rows as Lines draws them, each its Cols cells,
// with their colours as roles: for a program that paints the surface in
// its own theme, as plx does in its grid. Lines is Cells with the roles
// left out but for one: text in "muted-foreground" is Dim there, and
// not in Cells, where its role says it.
func (c *Screen) Cells() [][]Cell {
	panic("cells: Cells is in review (G-cells-colour)")
}
