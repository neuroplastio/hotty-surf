package surfrun

import (
	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// A drag (SPEC §9.1) of a drag handle (surfui.Node.Drag) onto a place to
// drop on (surfui.Node.Drop): the host reports dragstart, drag and
// dragend (the cells' renderer synthesizes them), and the surface keeps
// the drag. It runs the handle's "dragstart" handler as it starts; marks
// the target under the pointer where a drop would land (surfui.DropMark),
// which the next Update sends; and as it ends over a target, runs the
// target's "drop", then the handle's "dragend". Escape ends it first:
// "dragend" runs with Cancel, no "drop" does, and what the host says of
// the drag after it, and the click its release would be, change nothing.
// A drop on the dragged element itself is none: its release is its click.

// Drag is what a drag's handlers hear (Event.Drag): From, the dragged
// element's key (surfui.Node.Key, a template's :key), its id where it has
// none; Where, where in the target it was dropped (surfui.WhereBefore,
// WhereAfter, WhereInto, or "" for a target split in steps the host gave
// no step of); Cancel, whether Escape ended it. A template's handler has
// them as $drag.from, $drag.where and $drag.cancel.
type Drag struct {
	From   string
	Where  string
	Cancel bool
}

// dragging is the drag in progress: the dragged element's id and key
// (Drag.From), the target under the pointer and where in it, "" for none,
// and whether Escape ended it.
type dragging struct {
	from, key   string
	over, where string
	cancelled   bool
}

// Dragging reports whether a drag is in progress, which Escape would end
// (Key): for a host that shows what Escape does.
func (s *Surface) Dragging() bool { return s.drag != nil && !s.drag.cancelled }

// dragEvent routes one of the host's drag's events (SPEC §9.1), and reports
// whether the surface took it.
func (s *Surface) dragEvent(e hotty.Event) bool {
	switch e.Kind {
	case hotty.EventDragStart:
		if s.drag != nil {
			s.due, s.stale = true, true // the root's mark goes, or stays
		}
		s.drag = nil
		n := s.index[e.Target]
		if n == nil || !surfui.Draggable(n) {
			return false
		}
		g := &dragging{from: e.Target, key: dragKey(n)}
		s.drag = g
		s.due, s.stale = true, true // the root is marked
		s.dragRun(n, hotty.EventDragStart, Drag{From: g.key}, e.Mods)
		return true
	case hotty.EventDrag:
		g := s.drag
		if g == nil || g.cancelled {
			return g != nil
		}
		over, where := s.dropAt(e, g.from)
		if over != g.over || where != g.where {
			g.over, g.where = over, where
			s.due, s.stale = true, true // the mark moves
		}
		return true
	case hotty.EventDragEnd:
		g := s.drag
		s.drag = nil
		if g == nil {
			return false
		}
		s.due, s.stale = true, true // the marks go
		if g.cancelled {
			s.noClick = g.from
			return true
		}
		if over, where := s.dropAt(e, g.from); over != "" {
			s.dragRun(s.index[over], "drop", Drag{From: g.key, Where: where}, e.Mods)
		}
		s.fresh() // the handle as the drop left it
		if n := s.index[g.from]; n != nil {
			s.dragRun(n, hotty.EventDragEnd, Drag{From: g.key}, e.Mods)
		}
		return true
	}
	return false
}

// dropAt is the target of a drag's event and where in it the pointer is:
// t where it is a place to drop on other than the dragged element, and
// where by its steps (surfui.WhereOf); "" for none. from is the dragged
// element's id.
func (s *Surface) dropAt(e hotty.Event, from string) (over, where string) {
	n := s.index[e.Target]
	if n == nil || e.Target == from || surfui.DropOf(n) == surfui.DropNone {
		return "", ""
	}
	ty, has := 0, false
	if d := e.Drag; d != nil {
		ty, has = d.TY, d.HasTY
	}
	return e.Target, surfui.WhereOf(surfui.DropOf(n), ty, has)
}

// dragRun runs n's handler of a drag's event kind, or tells Unhandled.
func (s *Surface) dragRun(n *surfui.Node, kind string, d Drag, mods int) {
	ev := Event{Kind: kind, Target: surfui.IDOf(n), Mods: mods, Drag: d}
	act, ok := surfui.Handler(n, kind)
	if !ok {
		if s.Unhandled != nil {
			s.Unhandled(ev)
			s.due, s.stale = true, true
		}
		return
	}
	s.do(act, ev)
}

// cancelDrag is Escape during a drag: the handle's "dragend", with Cancel,
// and no "drop". The host's dragend after it, and the click of its
// release, are dropped (dragEvent, noClick).
func (s *Surface) cancelDrag() {
	g := s.drag
	g.cancelled = true
	g.over, g.where = "", ""
	s.due, s.stale = true, true // the marks go
	s.fresh()
	if n := s.index[g.from]; n != nil {
		s.dragRun(n, hotty.EventDragEnd, Drag{From: g.key, Cancel: true}, 0)
	}
}

// markDrop marks the target of the drag in progress in the tree indexed
// by ix, where a drop would land (surfui.DropMark).
func (s *Surface) markDrop(ix surfui.Index) {
	g := s.drag
	if g == nil || g.over == "" || g.cancelled {
		return
	}
	if n := ix[g.over]; n != nil {
		surfui.DropMark(n, g.where)
	}
}

// markDragging marks the root of the tree indexed by ix while a drag is
// in progress (surfui.DraggingClass). A Fine model's tree is its own and
// gets no mark, as it gets no drop mark.
func (s *Surface) markDragging(ix surfui.Index) {
	if !s.Dragging() {
		return
	}
	if n := ix["r"]; n != nil {
		n.Class(surfui.DraggingClass)
	}
}

// dragKey is n's key, or its id where it has none: what Drag.From says.
func dragKey(n *surfui.Node) string {
	if k := surfui.KeyOf(n); k != "" {
		return k
	}
	return surfui.IDOf(n)
}
