package surftea

import (
	"strings"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// A Select's open list and an element's menu are surfaces of their own,
// which surfrun.WriteFrame asks the host to write where they sit in a
// frame's order. A hottytea Session places the surfaces the program lists,
// so the Host does not write their documents: it keeps each popup (its
// document, where it goes) and Surfaces lists it, above the Host's own, for
// as long as it is open. A deltas goes out at once, to a surface the Session
// has sent. A popup that closes is deleted, first where another opens in
// the same frame, so that the new document goes out whole.

// writeList is the open list's turn in a frame: it opens, changes or
// closes.
func (h *Host) writeList(s *hottytea.Session, b *strings.Builder, l *surfrun.List) {
	name := h.name + "-list"
	if l.Closed || (l.New && h.list != nil) {
		s.Delete(name)
		h.list = nil
	}
	switch {
	case l.Tree == nil:
	case l.New:
		r := h.listRect(l)
		h.list = &popup{name: name, doc: surfui.Document(l.Tree), rect: r, z: h.opts.Z + 1, scroll: r.H < l.Rows}
	case len(l.Ops) > 0 && h.list != nil:
		ops := l.Ops
		if h.list.scroll {
			ops = append(ops, hotty.OpReveal(surfui.OptionID(l.At), ""))
		}
		h.list.doc = surfui.Document(l.Tree)
		b.WriteString(hotty.Delta(name, ops))
	}
}

// writeMenu is the open menu's turn: its surface is new and takes the
// keyboard once placed (a click elsewhere takes it from the menu and closes
// it), or its highlighted row changes, or it closes.
func (h *Host) writeMenu(s *hottytea.Session, b *strings.Builder, m *surfrun.MenuFrame) {
	name := h.name + "-menu"
	if m.Closed || (m.New && h.menu != nil) {
		s.Delete(name)
		h.menu = nil
	}
	switch {
	case m.View.Tree == nil:
	case m.New:
		x, y, cols, rows := h.menuRect(m)
		h.menu = &popup{name: name, doc: m.View.HTML(), rect: hottytea.Rect{X: x, Y: y, W: cols, H: rows},
			z: h.opts.Z + 2, focus: surfui.MenuID}
	case len(m.Ops) > 0 && h.menu != nil:
		h.menu.doc = m.View.HTML()
		b.WriteString(hotty.Delta(name, m.Ops))
	}
}

// listRect is where the open list goes: under its Select, as wide as it at
// least, or above it where there is more room there; in the middle of the
// surface where no click said where the Select is. It is the rows it wants,
// and scrolls where that doesn't fit.
func (h *Host) listRect(l *surfrun.List) hottytea.Rect {
	room, at := h.room, h.at
	cols, rows := l.Cols, l.Rows
	var x, y int
	if a := l.Anchor; a != nil {
		cols = max(cols, a.W)
		x = at.X + a.Col
		below := room.Y + room.H - (at.Y + a.Row + a.H)
		above := at.Y + a.Row - room.Y
		if rows <= below || below >= above {
			y, rows = at.Y+a.Row+a.H, min(rows, below)
		} else {
			rows = min(rows, above)
			y = at.Y + a.Row - rows
		}
	} else {
		rows = min(rows, at.H)
		x, y = at.X+(at.W-cols)/2, at.Y+(at.H-rows)/2
	}
	cols = max(min(cols, room.W), 1)
	x = max(min(x, room.X+room.W-cols), room.X)
	rows = max(rows, 1)
	y = max(min(y, room.Y+room.H-rows), room.Y)
	return hottytea.Rect{X: x, Y: y, W: cols, H: rows}
}

// menuRect is where the open menu goes: at the pointer's cell, under the
// element's cells, or under the surface's top row, kept in the room.
func (h *Host) menuRect(m *surfrun.MenuFrame) (x, y, cols, rows int) {
	room, at := h.room, h.at
	cols, rows = max(min(m.View.Cols, room.W), 1), max(min(m.View.Rows, room.H), 1)
	// The room's own grid: cells from its top left.
	ox, oy := room.X, room.Y
	switch {
	case m.Point != nil:
		x, y = cells.PlaceAtPointer(at.X+m.Point.Col-ox, at.Y+m.Point.Row-oy, cols, rows, room.W, room.H)
	case m.Area != nil:
		a := m.Area
		x, y = cells.PlaceBelow(at.X+a.Col-ox, at.Y+a.Row-oy, a.H, cols, rows, room.W, room.H)
	default:
		x, y = cells.PlaceBelow(at.X-ox, at.Y-oy-1, 1, cols, rows, room.W, room.H)
	}
	return x + ox, y + oy, cols, rows
}
