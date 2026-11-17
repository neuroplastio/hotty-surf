package cells

import "neuroplast.io/hotty-go"

// The pointer in cells (G-cells-hit, for plx). A Screen hears the
// terminal's mouse as the program decodes it, at its own cells, and sends
// the surface the events a hotty host sends for the same gestures (SPEC
// §9): a press, a click with its count and mods, a secondary press, a
// drag with its steps and its drop, and a hover. The surface's handlers
// (surf.Click, surf.Secondary, surf.Drag, surf.Drop, surf.Hover) hear
// them as they do on a host. Click stays: it is Press and Release at one
// cell with no mods.

// Press is a press of a mouse button at the cell col, row, from the
// Screen's top left, with mods held (hotty.ModShift, ModAlt, ModCtrl,
// ModSuper, as bits). button is numbered as the DOM and hotty.Event.Button
// number it: 0 the primary, 2 the secondary.
//
// The primary press does what a host's does: a press event (with the
// placement's Press), the keyboard to what takes it there or away from
// the surface, and a drag's start on an element with data-on drag
// (surf.Drag) unless Alt is held. Its click comes with the Release.
//
// The secondary press goes to the nearest element with data-on secondary
// (surf.Secondary) at or around the cell: a press event with Button 2,
// the cell as At and that element's cells as Area, and nothing else: no
// focus, no click, no drag. With Shift or Alt held, or on no such
// element, it is the program's.
//
// taken says whether the press landed on something of the surface's; a
// press the surface doesn't take (button 1, a secondary press on nothing
// that takes one) is the program's.
func (c *Screen) Press(col, row, button, mods int) (taken bool) {
	panic("cells: Press is in review (G-cells-hit)")
}

// Move is the pointer at the cell col, row, mods held, with a button
// down or none. During a drag it sends the drag events a host sends
// (SPEC §9.1): one where the element under the pointer, or its step in
// the dragged element or in the drop target (data-steps), changed, not
// one for every cell. With no button down, it sends a hover where the
// element under the pointer changed (SPEC §9.4), on a placement made
// with Hover, as surf's driver makes it (surf.Surface.Attach); a cell outside the
// Screen (a negative col or row, or one past its size) is the pointer
// leaving it, a hover with Out. taken says whether it sent anything.
func (c *Screen) Move(col, row, mods int) (taken bool) {
	panic("cells: Move is in review (G-cells-hit)")
}

// Release is the release of the button pressed, at the cell col, row,
// mods held. During a drag it sends the dragend, with the drop target
// as its Over; then, as a host does, the click: where the release is on
// what the press was on (the element a drag began on, for a drag), with
// its count in a run of clicks on that element (2 for a double click,
// counted as the platform counts them, within half a second) and mods.
// taken says whether it sent anything.
func (c *Screen) Release(col, row, mods int) (taken bool) {
	panic("cells: Release is in review (G-cells-hit)")
}

// At is what is drawn at the cell col, row: the innermost element with
// an id there, and its cells (Area), from the Screen's top left as the
// cell is. id is "" where no element with an id is. A program asks it
// for what the surface's events don't carry: the row under a carry that
// began outside the surface (which half: row < area.Row + area.H/2), or
// the element under a press the program keeps for itself.
func (c *Screen) At(col, row int) (id string, area hotty.Area) {
	panic("cells: At is in review (G-cells-hit)")
}
