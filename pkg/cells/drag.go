package cells

import (
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Drags in the cells (SPEC §9.1). A cell frame has no elements on the
// wire, only the press, the pointer's moves while the button is down
// (Move) and its release; the renderer synthesizes from them the events
// a host sends, against what it drew, so that the program hears a drag
// as it does on a host. A press on a drag handle (data-on~=drag) starts
// the drag at once, as a host's does: dragstart, then what else the
// press does (the keyboard). Its click waits for the release, which
// sends dragend, then the click only where the drag ends on the handle
// it began on: a row that is both a handle and a button is clicked by a
// press and release on it, and dragged by anything else. A press with
// Alt held starts no drag (SPEC §9.2).
//
// The pointer's step in an element (data-steps) is measured from the
// middle of its cell, against the cells the element shows: in a row two
// cells tall split in halves, the top cell is before it and the bottom
// one after it.

// dragging is a drag in progress: the handle it began on, and what the
// last event said of the pointer (its target, its cell, its steps), so
// that a move that changes none of it sends nothing.
type dragging struct {
	from   string
	t      string
	c, r   int
	x, y   int
	tx, ty int
}

// Move is the pointer at x, y while the button pressed is down, mods
// held: during a drag, a drag event where the element under it changed
// (or, over none, the cell), or its step in the dragged element or in
// the target did. It reports whether the drag took it.
func (r *Renderer) Move(x, y, mods int) (taken, redraw bool) {
	r.changed = false
	d := r.drag
	if r.menu != nil {
		return true, false // the menu has the pointer
	}
	if d == nil {
		return false, false
	}
	e := r.dragEvent(hotty.EventDrag, x, y, mods)
	g := e.Drag
	if e.Target == d.t && (e.Target != "" || x == d.c && y == d.r) && g.X == d.x && g.Y == d.y && g.TX == d.tx && g.TY == d.ty {
		return true, false
	}
	d.t, d.c, d.r, d.x, d.y, d.tx, d.ty = e.Target, x, y, g.X, g.Y, g.TX, g.TY
	r.settle(r.send(e))
	return true, true
}

// startDrag starts a drag where a press at x, y on n, mods held, is on a
// drag handle: it sends dragstart, and reports whether it did and whether
// an update is due.
func (r *Renderer) startDrag(n *surfui.Node, x, y, mods int) (started, due bool) {
	h := closest(n, surfui.Draggable)
	if h == nil || surfui.IDOf(h) == "" || mods&hotty.ModAlt != 0 {
		return false, false
	}
	id := surfui.IDOf(h)
	r.drag = &dragging{from: id, t: id, c: x, r: y}
	e := r.dragEvent(hotty.EventDragStart, x, y, mods)
	e.Target = id
	g := e.Drag
	g.HasTX, g.HasTY, g.TX, g.TY = false, false, 0, 0
	r.drag.x, r.drag.y = g.X, g.Y
	return true, r.send(e)
}

// endDrag ends the drag in progress at x, y, mods held: dragend, and
// whether the release ends it on the handle it began on, and so clicks.
func (r *Renderer) endDrag(x, y, mods int) (clicks, due bool) {
	d := r.drag
	e := r.dragEvent(hotty.EventDragEnd, x, y, mods)
	r.drag = nil
	due = r.send(e)
	own := ""
	if h := closest(r.under(x, y), surfui.Draggable); h != nil {
		own = surfui.IDOf(h)
	}
	return own == d.from, due
}

// dragEvent is a drag's event of kind with the pointer at x, y: its target,
// the nearest element from the one under the pointer outward with an id
// and drag or drop in its data-on ("" for none); the cell; the step in the
// dragged element, and in a target that is a place to drop on, where they
// have steps.
func (r *Renderer) dragEvent(kind string, x, y, mods int) hotty.Event {
	g := &hotty.Drag{Col: x, Row: y}
	e := hotty.Event{Kind: kind, Mods: mods, Drag: g}
	if n := r.index[r.drag.from]; n != nil {
		g.X, g.HasX, g.Y, g.HasY = r.stepIn(n, x, y)
	} else {
		g.X, g.HasX, g.Y, g.HasY = r.drag.x, true, r.drag.y, true // gone: as it last was
	}
	t := closest(r.under(x, y), func(n *surfui.Node) bool {
		return surfui.IDOf(n) != "" && (surfui.Draggable(n) || surfui.DropOf(n) != surfui.DropNone)
	})
	if t == nil {
		return e
	}
	e.Target = surfui.IDOf(t)
	if surfui.DropOf(t) != surfui.DropNone {
		g.TX, g.HasTX, g.TY, g.HasTY = r.stepIn(t, x, y)
	}
	return e
}

// under is the innermost element drawn at x, y, nil for none.
func (r *Renderer) under(x, y int) *surfui.Node {
	if id := r.At(x, y); id != "" {
		return r.index[id]
	}
	return nil
}

// stepIn is where x, y is in n by its data-steps (SPEC §9.1): along each
// axis it counts, from the middle of the cell, over the cells n shows,
// rounded half up and clamped; has says the axis counts.
func (r *Renderer) stepIn(n *surfui.Node, x, y int) (sx int, hasX bool, sy int, hasY bool) {
	counts := strings.Fields(surfui.DataSteps(n))
	if len(counts) == 0 {
		if v, ok := surfui.AttrOf(n, "data-steps"); ok {
			counts = strings.Fields(v)
		}
	}
	if len(counts) == 0 || len(counts) > 2 {
		return 0, false, 0, false
	}
	var cs [2]int
	for i, c := range counts {
		v, err := strconv.Atoi(c)
		if err != nil || v < 0 {
			return 0, false, 0, false
		}
		cs[i] = v
	}
	vis := r.rects[n]
	if cs[0] > 0 {
		sx, hasX = step(cs[0], x-vis.x, vis.w), true
	}
	if cs[1] > 0 {
		sy, hasY = step(cs[1], y-vis.y, vis.h), true
	}
	return sx, hasX, sy, hasY
}

// step is the step of count at cell at of size cells, from the cell's
// middle: (at + ½) / size × count, rounded half up, clamped to 0 and
// count; 0 for no size.
func step(count, at, size int) int {
	if size <= 0 {
		return 0
	}
	v := ((2*at+1)*count + size) / (2 * size)
	if 2*at+1 < 0 {
		v = 0
	}
	return max(0, min(v, count))
}

// WhereStep is a cell of the element id where the pointer is at step ty of
// its data-steps' second count (a place to drop on split in halves or
// thirds): the first such cell, top to bottom, of those it shows. ok is
// false where id is not on the screen, or no cell of it is at that step.
// For a test, or a host that drags by id.
func (r *Renderer) WhereStep(id string, ty int) (x, y int, ok bool) {
	r.ensure()
	n := r.index[id]
	vis, seen := r.rects[n]
	if n == nil || !seen {
		return 0, 0, false
	}
	for yy := vis.y; yy < vis.y+vis.h; yy++ {
		if _, _, sy, has := r.stepIn(n, vis.x, yy); has && sy == ty {
			return vis.x, yy, true
		}
	}
	return 0, 0, false
}

// dropLines draws where a drop would land (surfui.DropMarked), as a line
// in the accent that recolours no text: after an element, an underline
// of its bottom row; before it, an underline of the bottom row of what is
// drawn directly above it, or where nothing is, an overline of its top
// row; into it, both of its own.
func (r *Renderer) dropLines(g *grid) {
	walkNodes(r.tree, func(n *surfui.Node) {
		where := surfui.DropMarked(n)
		vis, ok := r.rects[n]
		if where == "" || !ok {
			return
		}
		x0, x1 := vis.x, vis.x+vis.w
		bottom := vis.y + vis.h - 1
		switch where {
		case surfui.WhereAfter:
			g.line(x0, x1, bottom, Underline)
		case surfui.WhereInto:
			g.line(x0, x1, vis.y, Overline)
			g.line(x0, x1, bottom, Underline)
		case surfui.WhereBefore:
			if above, ok := r.above(vis); ok {
				g.line(x0, x1, above.y+above.h-1, Underline)
			} else {
				g.line(x0, x1, vis.y, Overline)
			}
		}
	})
}

// above is what is drawn directly above vis: of the elements whose last
// row is the row over vis's first and whose columns meet vis's, the
// largest. An element round vis can't end there, and one nested in the
// row above is smaller than it.
func (r *Renderer) above(vis rect) (rect, bool) {
	var best rect
	found := false
	for _, o := range r.rects {
		if o.y+o.h != vis.y || o.x+o.w <= vis.x || o.x >= vis.x+vis.w {
			continue
		}
		if !found || o.w*o.h > best.w*best.h || o.w*o.h == best.w*best.h && (o.y < best.y || o.y == best.y && o.x < best.x) {
			best, found = o, true
		}
	}
	return best, found
}

// line draws a line of attr (Underline, Overline) in the accent along row
// y, from column x0 to x1.
func (g *grid) line(x0, x1, y int, attr Attr) {
	if y < 0 || y >= g.rows {
		return
	}
	for x := max(x0, 0); x < min(x1, g.cols); x++ {
		c := &g.cells[y*g.cols+x]
		c.attr |= attr
		c.ul = Accent
	}
}

// walkNodes calls f with each element of the tree under n, in document
// order.
func walkNodes(n *surfui.Node, f func(*surfui.Node)) {
	stackprobe.Probe()
	if n == nil || surfui.Tag(n) == "" {
		return
	}
	f(n)
	for _, k := range surfui.Kids(n) {
		walkNodes(k, f)
	}
}
