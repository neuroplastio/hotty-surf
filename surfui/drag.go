package surfui

// Drags (SPEC §9.1). An element is a drag handle (Drag), a place to drop
// on (Drop), or both, as a reorderable row is: data-on asks the host for
// the drag's events, and data-steps splits a drop target in halves or
// thirds, so that a drag's events say where in it the pointer is. The
// runtime keeps the drag (surfrun.Surface) and marks where a drop would
// land (DropMark), which surf.css and the cells draw alike.

// DropMode is how a place to drop on is split: whole, in halves (before,
// after) or in thirds (before, into, after). The zero DropMode is no place
// to drop on.
type DropMode uint8

// The modes.
const (
	DropNone DropMode = iota
	// DropWhole is the whole element: a drop on it is into it.
	DropWhole
	// DropHalves is before or after it: data-steps="0 1".
	DropHalves
	// DropThirds is before, into or after it: data-steps="0 2", a quarter,
	// a half and a quarter of its height.
	DropThirds
)

// Drag makes n a drag handle: a press on it starts a drag of it, and its
// click waits for the release, which clicks it only where the drag ended
// on it (SPEC §9.1). Its handlers of "dragstart" and "dragend" (On) run
// as the drag starts and ends (surfrun.Surface). n needs an id, which every
// element of a resolved tree has.
func (n *Node) Drag() *Node { n.drag = true; return n }

// Drop makes n a place to drop on, split as m says; DropNone makes it
// none again. Its handler of "drop" (On) runs when a drag ends over it.
func (n *Node) Drop(m DropMode) *Node { n.drop = m; return n }

// Draggable reports whether n is a drag handle (Drag).
func Draggable(n *Node) bool { return n.drag }

// DropOf is how n is split as a place to drop on, DropNone for none
// (Drop).
func DropOf(n *Node) DropMode { return n.drop }

// Where is where in a place to drop on a drag's pointer is: before it,
// after it, into it, or "" for none (a target that isn't split, or no
// target). A drop on a whole target is into it.
const (
	WhereBefore = "before"
	WhereAfter  = "after"
	WhereInto   = "into"
)

// WhereOf is where a drag's pointer is in a target split as m, from the
// step ty the host reports (SPEC §9.1's ty: data-steps' second count), and
// whether it has one: halves 0 and 1 are before and after; thirds 0, 1 and
// 2 before, into and after; a whole target is into, with no step. A split
// target with no step is none.
func WhereOf(m DropMode, ty int, hasTY bool) string {
	switch m {
	case DropWhole:
		return WhereInto
	case DropHalves:
		if !hasTY {
			return ""
		}
		if ty <= 0 {
			return WhereBefore
		}
		return WhereAfter
	case DropThirds:
		if !hasTY {
			return ""
		}
		switch {
		case ty <= 0:
			return WhereBefore
		case ty == 1:
			return WhereInto
		}
		return WhereAfter
	}
	return ""
}

// The classes DropMark gives an element: where a drop would land.
const (
	DropBeforeClass = "u-drop-before"
	DropAfterClass  = "u-drop-after"
	DropIntoClass   = "u-drop-into"
)

// DraggingClass is the class the runtime gives the tree's root (the
// .u-surface element) from a drag's start until its end or Escape's
// cancel: surf.css shows the closed hand over the whole surface while it
// is there. The cells ignore it.
const DraggingClass = "u-dragging"

// DropMark marks n where a drop would land, by where (WhereBefore,
// WhereAfter, WhereInto): a line at its top or bottom edge, which takes no
// room, or an outline round it. surf.css draws it on a host, and the cells
// draw it in the cells (cells.Renderer). "" marks nothing.
func DropMark(n *Node, where string) *Node {
	switch where {
	case WhereBefore:
		n.Class(DropBeforeClass)
	case WhereAfter:
		n.Class(DropAfterClass)
	case WhereInto:
		n.Class(DropIntoClass)
	}
	return n
}

// DropMarked is where DropMark marked n, "" for nowhere.
func DropMarked(n *Node) string {
	for _, c := range n.class {
		switch c {
		case DropBeforeClass:
			return WhereBefore
		case DropAfterClass:
			return WhereAfter
		case DropIntoClass:
			return WhereInto
		}
	}
	return ""
}

// dataSteps is n's data-steps: a split drop target's, "" for none.
func dataSteps(n *Node) string {
	if n.slider != nil {
		return sliderSteps(n)
	}
	switch n.drop {
	case DropHalves:
		return "0 1"
	case DropThirds:
		return "0 2"
	}
	return ""
}

// DataSteps is n's data-steps as it renders (SPEC §9.1): a split drop
// target's, "" for none; for a host of the program's own, such as the
// cells'.
func DataSteps(n *Node) string { return dataSteps(n) }
