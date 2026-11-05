package surfui

import "strconv"

// A widget's face and a popover, as the maintainer answered Q16
// (2026-11-05): a widget has two paths, as a tool has. By default it is
// data-first: the program gives an icon, a value, a unit, a tone and a
// meter, and the kit draws the face, so that every widget looks alike,
// on a host and in cells, and a prototype touches no styling. Or it
// draws its own: what it gives the face as children, in the same
// capsule. What it opens is a surface of its own, as a tool's.

// Widget is a widget's face: a capsule on the hover ground, 28px tall
// on a host, with its icon (a glyph, muted), its value (500, in the
// ink) and its unit (400, muted), each left out where "". Tone colours
// its icon and value (OK its icon only); Selected (its popover shows)
// is the accent's soft fill; Stale dims it all. The cells draw its
// glyph, value and unit in a row.
//
// kids is the custom path: the program's own widgets, after the glyph,
// value and unit, or in their place where those are "". The capsule,
// its states and the meter stay the kit's.
func Widget(glyph, value, unit string, kids ...*Node) *Node {
	n := El("div").Class("u-row", "u-wg")
	if glyph != "" {
		n.Add(Icon(glyph))
	}
	if value != "" {
		n.Add(Text(value).Class("u-wg-v"))
	}
	if unit != "" {
		n.Add(Text(unit).Class("u-wg-u"))
	}
	return n.Add(kids...)
}

// Meter adds a meter to a Widget's face: a bar 28px long and 4px tall
// on a host, filled to f, from 0 to 1.
func (n *Node) Meter(f float64) *Node {
	f = max(0, min(f, 1))
	return n.Add(El("span", El("span").Class("u-wg-fill")).Class("u-wg-meter").
		Var("u-m", strconv.FormatFloat(f, 'f', -1, 64)))
}

// Stale dims a Widget whose value is old: its source is gone.
func (n *Node) Stale() *Node { return n.Class("u-stale") }

// Popover makes a Box or a Stack a popover's card: the root of what a
// widget or a top strip's tool opens under it, in a surface of its own:
// the page's ground, radius 10, the input's hairline and the raised
// shadow, what it holds clipped to it. The cells draw it as a Box.
func (n *Node) Popover() *Node { return n.Class("u-pop") }
