// Package cellblock is the types of what a custom element draws in the
// cells: a Block of rows of runs, and what the layout needs to know of
// it. They are types only, with nothing to run.
//
// They are a package of their own because two packages need them that
// can't import each other: surfui (a node carries its element's block
// func, Node.Cells) and pkg/cells (which lays a tree out and draws it,
// and so imports surfui). pkg/cells re-exports every name here
// (cells.Block, cells.Run, cells.Role, ...) as an alias, so a program
// that draws blocks imports pkg/cells alone. A host build that never
// draws cells takes in this package, a few types and no code, and not
// pkg/cells with its text measuring.
package cellblock

import (
	"strconv"
	"strings"
)

// Role is a colour of the theme, named as surf.css's tokens name it: a
// host maps each to its palette, as the runtime maps them to the
// terminal's (surfrun, the cells) and plx to its theme. "" in a Run is
// the colour its element has: its text's ink, or the ground below.
type Role string

// The roles.
const (
	Fg       Role = "fg"        // the text
	Bg       Role = "bg"        // the ground
	Accent   Role = "accent"    // what has the keyboard, a match
	Muted    Role = "muted"     // secondary text
	Faint    Role = "faint"     // a placeholder, what is disabled
	OK       Role = "ok"        // the tones
	Warn     Role = "warn"      //
	Error    Role = "error"     //
	Info     Role = "info"      //
	Border   Role = "border"    // a dialog's line, a scrollbar
	Surface  Role = "surface"   // a box's fill, a step from the ground
	Field    Role = "field"     // a control's fill
	Tonal    Role = "tonal"     // a selection's, a button's, a Compact box's fill
	Line     Role = "line"      // a rule
	Selected Role = "selected"  // a list's selection, while it or what drives it has the keyboard
	Sel      Role = "sel"       // a control with the keyboard; an open Select's highlight
	OnAccent Role = "on-accent" // text on an accent fill

	// A code block's syntax roles, and its ground (surf.css's
	// --u-syn-* and --u-code-ground). A host that has no colour of its
	// own for one draws Base's: the theme's default, from the roles
	// above.
	SynKeyword   Role = "syn-keyword"   // accent
	SynTag       Role = "syn-tag"       // accent
	SynString    Role = "syn-string"    // ok
	SynNumber    Role = "syn-number"    // warn
	SynComment   Role = "syn-comment"   // muted
	SynType      Role = "syn-type"      // info
	SynAttribute Role = "syn-attribute" // fg
	SynPlain     Role = "syn-plain"     // fg
	CodeGround   Role = "code-ground"   // surface
)

// Base is the role a host that doesn't know r draws it as: a syntax
// role's and the code ground's default in every theme, from the roles
// that every host has. Any other role is its own.
func (r Role) Base() Role {
	switch r {
	case SynKeyword, SynTag:
		return Accent
	case SynString:
		return OK
	case SynNumber:
		return Warn
	case SynComment:
		return Muted
	case SynType:
		return Info
	case SynAttribute, SynPlain:
		return Fg
	case CodeGround:
		return Surface
	}
	return r
}

// AnsiRole is the role of the terminal's colour n, 0 to 15, "ansi-9": a
// hue's (surfui.Palette), which the terminal's palette draws, so that its
// theme keeps the control of colour in the cells too.
func AnsiRole(n int) Role { return Role("ansi-" + strconv.Itoa(n)) }

// Ansi is the terminal colour a role names, if it is AnsiRole's.
func (r Role) Ansi() (n int, ok bool) {
	rest, ok := strings.CutPrefix(string(r), "ansi-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil && n >= 0 && n <= 15
}

// Attr is a set of text attributes.
type Attr uint8

// The attributes.
const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	Reverse
	Strike
	// Overline is a line over the cell, SGR 53: where a drop would land
	// before an element with nothing drawn above it.
	Overline
)

var attrNames = [...]string{"bold", "dim", "italic", "underline", "reverse", "strike", "overline"}

// String is the attributes' names, joined by "+": "bold+underline".
func (a Attr) String() string {
	var out []string
	for i, n := range attrNames {
		if a&(1<<i) != 0 {
			out = append(out, n)
		}
	}
	return strings.Join(out, "+")
}

// Cluster is a grapheme cluster and the cells it takes: 1, or 2 for a
// wide one (an ideograph, most emoji).
type Cluster struct {
	Text  string
	Width int
}

// Run is a row's run of cells in one style, from column Col, Width cells
// wide: its text, and the clusters it is made of.
type Run struct {
	Col, Width int
	Text       string
	Clusters   []Cluster
	Fg, Bg     Role
	Attr       Attr
	// Ul is the colour of an Underline or an Overline, "" for the text's:
	// a drop's line is the accent's whatever the text's colour (SGR 58).
	Ul Role
	// Href is the link the cells belong to, "" for none: a hyperlink
	// (surfrun.Paint writes it as OSC 8, unless NoLinks takes it off).
	// Runs split where it changes.
	Href string
}

// Density is the surface's: the steps of its gaps and padding.
type Density uint8

// The densities. Compact is hotty-surf's default.
const (
	Dense Density = iota
	Compact
	Roomy
)

// Style is what the layout knows of where a block is drawn, and so all a
// block may depend on besides its width and its node: the surface's
// density. The ink is not in it: the colours a block leaves "" are
// filled by the element's own, as the painter has them (tone, hue,
// disabled), so that they follow the element's look without the block
// asking.
type Style struct {
	Density Density
}

// Place is where a child of the element sits in its block: the child is
// still a node in the tree (its id, its focus, its keys and events are
// hotty-surf's, as everywhere), and the block only says where. The
// layout lays it out W cells wide and H rows tall at X, Y from the
// block's top left and draws it, hits and caret and all, over the
// block's rows. A child no Place names isn't drawn in the cells and
// takes no press, though it stays in the tree and the order of Tab.
type Place struct {
	Kid        int // the index among the element's children (surfui.Kids)
	X, Y, W, H int
}

// Hit is a rectangle of the block, X and Y from its top left, that a
// press acts on as a press on the child Kid: for a child the block drew
// itself, in its rows (a bar's segment), which the layout then doesn't
// draw. Where two hits hold a cell, the later is on top.
type Hit struct {
	Kid        int
	X, Y, W, H int
}

// Anchor is a place in the block a #fragment names (a heading's anchor
// inside a drawn document): Name, at row Y.
type Anchor struct {
	Name string
	Y    int
}

// Block is what an element draws in the cells, at the width it was
// given: Rows, each a row of runs that reach at most that width, Col
// from the block's left; the children it Places; the Hits it has; and
// its Anchors. Its height is the rows it has, or where a Place or a Hit
// ends, if lower: its natural height, which the layout takes.
//
// A Run's Fg and Bg may be "": the colour the element's own look has
// there. Cells with no run are the element's ground.
type Block struct {
	Rows    [][]Run
	Kids    []Place
	Hits    []Hit
	Anchors []Anchor
}

// Height is the rows the block takes.
func (b Block) Height() int {
	h := len(b.Rows)
	for _, p := range b.Kids {
		h = max(h, p.Y+p.H)
	}
	for _, p := range b.Hits {
		h = max(h, p.Y+p.H)
	}
	return h
}

// Width is the widest the block's rows and places reach.
func (b Block) Width() int {
	w := 0
	for _, row := range b.Rows {
		for _, r := range row {
			w = max(w, r.Col+r.Width)
		}
	}
	for _, p := range b.Kids {
		w = max(w, p.X+p.W)
	}
	for _, p := range b.Hits {
		w = max(w, p.X+p.W)
	}
	return w
}

// Func makes the block of an element at width w, at least 1, in style
// in. It is pure: the block is a function of the element it was set on
// (what the func closes over) and w and in, and the layout keeps it by
// them, as the diff skips a cached subtree. Natural width is asked at
// w = Unbounded: say the width you want, don't fill it.
type Func func(w int, in Style) Block

// Unbounded is the width a block is asked for to know its natural width.
const Unbounded = 1 << 20
