package surfrun

import (
	"strings"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// The runtime shows an element's open menu (Frame.Menu) in a surface of
// its own, as it shows a Select's list: placed at a higher z than the
// list's, with presses on, at the pointer's cell or under the element's
// cells, and given the keyboard, so that its keys are the program's and a
// click elsewhere, which takes the keyboard from it, closes it
// (Surface.MenuEvent). Its document is sent fresh on each open and the
// surface is deleted on close: a detached surface would take no input, so
// it is not kept ready that way. The menu closes on a resize, when the
// surface is hidden or the run ends, and on the host losing it. The
// runtime sees no scroll of the surface (the host scrolls it, and reports
// none), so a menu stays through one until a click or a key.

// MenuSurface is the name of the surface an element's open menu shows in:
// for a test that reads it (surftest's Menu).
func (r *Runtime) MenuSurface() string { return r.menuSurface }

// menu writes what changed of the open menu to b: its surface deleted, its
// document, placement and the keyboard, or its delta.
func (r *Runtime) menu(b *strings.Builder, m *MenuFrame) {
	if m == nil {
		return
	}
	if m.Closed {
		b.WriteString(hotty.Del(r.menuSurface, hotty.Q(hotty.NoReply)))
	}
	switch {
	case m.View.Tree == nil:
	case m.New:
		x, y, p := r.menuPlacement(m)
		b.WriteString(hotty.Doc(r.menuSurface, m.View.HTML()))
		b.WriteString(hotty.PlaceAt(r.menuSurface, x, y, p))
		b.WriteString(hotty.Focus(r.menuSurface, surfui.MenuID))
	case len(m.Ops) > 0:
		b.WriteString(hotty.Delta(r.menuSurface, m.Ops))
	}
}

// dropMenu takes the open menu off at once: the surface is going, or
// being sent again, or it moved.
func (r *Runtime) dropMenu(refocus bool) {
	if r.s.DropMenu(refocus) {
		r.write(hotty.Del(r.menuSurface, hotty.Q(hotty.NoReply)))
	}
}

// menuPlacement is where the menu goes, in the screen's grid: for a
// press, View.PlaceAtPointer at the pointer's cell, from the surface's own top
// left; for a key, under the element's cells (placeBelow); with neither,
// at the surface's top left.
func (r *Runtime) menuPlacement(m *MenuFrame) (x, y int, p hotty.Placement) {
	top := r.opts.Top
	switch {
	case m.Point != nil:
		x, y = m.View.PlaceAtPointer(m.Point.Col, top+m.Point.Row, r.cols, r.rows)
	case m.Area != nil:
		a := m.Area
		x, y = placeBelow(m.View, a.Col, top+a.Row, a.H, r.cols, r.rows)
	default:
		x, y = placeBelow(m.View, 0, top-1, 1, r.cols, r.rows)
	}
	cols, rows := max(min(m.View.Cols, r.cols), 1), max(min(m.View.Rows, r.rows), 1)
	return x, y, hotty.Placement{Cols: cols, Rows: rows, Z: 2, Press: hotty.On}
}

// placeBelow is where a menu opened under an element goes, the element at
// (c, r), h rows tall, on a grid cols × rows: under it, above it where
// there is no room below and more above, and kept in the grid.
func placeBelow(v MenuView, c, r, h, cols, rows int) (x, y int) {
	return cells.PlaceBelow(c, r, h, v.Cols, v.Rows, cols, rows)
}
