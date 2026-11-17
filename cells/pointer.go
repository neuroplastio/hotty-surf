package cells

import (
	"math"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The pointer in cells (G-cells-hit, for plx). A Screen hears the
// terminal's mouse as the program decodes it, at its own cells, and sends
// the surface the events a hotty host sends for the same gestures (SPEC
// §9): a press, a click with its count and mods, a secondary press, a
// drag with its steps and its drop, and a hover. The surface's handlers
// (surf.Click, surf.Secondary, surf.Drag, surf.Drop, surf.Hover) hear
// them as they do on a host. Click stays: it is Press and Release at one
// cell with no mods. Keys during a drag go where they go without one
// (Key): Escape doesn't end it, as on a host.
//
// taken, in Press, Move and Release alike, says whether the gesture is
// the surface's: a press it took, the moves and release of that press,
// and a move with no button down over the Screen's cells. What it
// doesn't take is the program's.

// Press is a press of a mouse button at the cell col, row, from the
// Screen's top left, with mods held (hotty.ModShift, ModAlt, ModCtrl,
// ModSuper, as bits), of button, numbered as hotty.Event.Button numbers
// it: 0 the primary, 2 the secondary.
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
// element, it is the program's, as is any other button.
//
// A Press while a drag is under way ends it first, with no drop and no
// click, as the terminal lost its release.
func (c *Screen) Press(col, row, mods, button int) (taken bool) {
	c.cutDrag()
	// A press of any button ends what an earlier one left: its click,
	// whose release was lost, is no longer due.
	c.down, c.held, c.pressing = nil, false, true
	switch button {
	case 0:
		taken = c.pressAt(col, row, mods)
	case 2:
		taken = c.secondary(col, row, mods)
	}
	c.held = taken
	c.flush()
	return taken
}

// Move is the pointer at the cell col, row, mods held, with a button
// down or none. During a drag it sends the drag events a host sends
// (SPEC §9.1): one where the element under the pointer, or its step in
// the dragged element or in the drop target (data-steps), changed, not
// one for every cell; a drag only moves with the primary button. With
// no button down, or another one, it sends a hover where the element
// under the pointer changed (SPEC §9.4), on a placement made with Hover,
// as surf's driver makes it (surf.Surface.Attach); a cell outside the
// Screen (a negative col or row, or one past its size) is the pointer
// leaving it, a hover with Out. Lines changes after a Move only through
// what the surface writes (a hover's look, the dragging class), so a
// program redraws on Writes, and on the presses and releases it takes,
// not on every motion.
func (c *Screen) Move(col, row, mods int) (taken bool) {
	switch {
	case c.drag != nil:
		c.dragTo(hotty.EventDrag, col, row, mods)
	case c.down == nil:
		c.hoverAt(col, row)
	}
	taken = c.held || !c.pressing && c.body != nil && col >= 0 && row >= 0 && col < c.cols && row < c.rows
	c.flush()
	return taken
}

// Release is the release of the button pressed, at the cell col, row,
// mods held. During a drag it sends the dragend, with the drop target
// as its Over; then, as a host does, the click: where the release is on
// what the press was on (the element a drag began on, for a drag), with
// its count in a run of clicks on that element (2 for a double click,
// counted as the platform counts them, within half a second) and mods.
// A drag also ends, with no drop and no click, when the Screen loses
// focus (Focus(false)), takes a new document, loses the dragged element,
// or hears a Press before this Release.
func (c *Screen) Release(col, row, mods int) (taken bool) {
	taken, c.held, c.pressing = c.held, false, false
	if d := c.drag; d != nil {
		c.dragTo(hotty.EventDragEnd, col, row, mods)
		c.drag = nil
		if d.t != d.from {
			// A drag that ends elsewhere clicks nothing, and ends the run
			// of clicks.
			c.down, c.clicks = nil, clickRun{}
			c.flush()
			return taken
		}
	}
	c.releaseAt(col, row, mods)
	c.flush()
	return taken
}

// At is what is drawn at the cell col, row: the innermost element there
// with an id of the app's (its markup's, or surf.On's; not one the
// surface generated), and that element's cells (Area), from the
// Screen's top left as the cell is. id is "" where none is. A program
// asks it for what the surface's events don't carry: the row under a
// carry that began outside the surface (which half: row < area.Row +
// area.H/2), or the element under a press the program keeps for itself.
func (c *Screen) At(col, row int) (id string, area hotty.Area) {
	n := closest(c.hitAt(col, row), func(n *html.Node) bool {
		v, ok := get(n, "id")
		return ok && v != "" && !rt.RuntimeID(v)
	})
	if n == nil || n == c.body {
		return "", hotty.Area{}
	}
	if a := c.area(n); a != nil {
		area = *a
	}
	return attr(n, "id"), area
}

// cutDrag ends a drag under way short (SPEC §9.1): a dragend with no
// target, the steps as they last were, no drop and no click.
func (c *Screen) cutDrag() {
	d := c.drag
	if d == nil {
		return
	}
	c.drag, c.down, c.held = nil, nil, false
	g := hotty.Drag{Col: d.c, Row: d.r, X: d.x, Y: d.y, HasX: d.sx > 0, HasY: d.sy > 0}
	c.emit(hotty.Event{Kind: hotty.EventDragEnd, Drag: &g})
}

// pressed is a primary press whose release may click: the innermost
// element pressed, and what a click acts on (a label's control).
type pressed struct {
	hit, target *html.Node
}

// dragging is a drag in progress (SPEC §9.1): the handle it began on, its
// box and steps as they were at the start, and what the last event said
// (its target, its cell, its steps), so that a move that changes none of
// it sends nothing.
type dragging struct {
	from   string
	box    rect
	sx, sy int
	t      string
	c, r   int
	x, y   int
	tx, ty int
	hasT   bool
}

// clickRun is the run of clicks a double click counts: on one element,
// each within half a second of the last.
type clickRun struct {
	n     *html.Node
	at    time.Time
	count int
}

// now is the clock a run of clicks is counted by; a test sets it.
var now = time.Now

// count is a click's place in its run of clicks on n.
func (c *Screen) count(n *html.Node) int {
	t := now()
	if c.clicks.n == n && t.Sub(c.clicks.at) < 500*time.Millisecond {
		c.clicks.count++
	} else {
		c.clicks.count = 1
	}
	c.clicks.n, c.clicks.at = n, t
	return c.clicks.count
}

// hitAt is the innermost element drawn at the cell col, row of the
// Screen, nil for none or a cell off it.
func (c *Screen) hitAt(col, row int) *html.Node {
	if c.body == nil || col < 0 || row < 0 || col >= c.cols || row >= c.rows {
		return nil
	}
	l := c.layout()
	y := row + c.scroll
	if y >= len(l.cv.rows) {
		return nil
	}
	return l.cv.rows[y][col].own
}

// within says whether n is a or inside it.
func within(n, a *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == a {
			return true
		}
	}
	return false
}

// secondary is a secondary press at col, row (SPEC §9): to the nearest
// element with data-on secondary there, unless Shift or Alt makes it the
// terminal's or the program's.
func (c *Screen) secondary(col, row, mods int) bool {
	if mods&(hotty.ModShift|hotty.ModAlt) != 0 {
		return false
	}
	hit := c.hitAt(col, row)
	if closest(hit, func(n *html.Node) bool { return has(n, "data-hotty-modal-inert") }) != nil {
		return true
	}
	t := closest(hit, func(n *html.Node) bool { return has(n, "id") && tokens(n, "data-on", "secondary") })
	if t == nil {
		return false
	}
	c.emit(hotty.Event{Kind: hotty.EventPress, Button: 2, Target: attr(t, "id"),
		At: &hotty.Cell{Col: col, Row: row}, Area: c.area(t), Mods: mods})
	return true
}

// startDrag starts a drag where a primary press at col, row on hit is on
// a drag handle with an id, Alt not held: dragstart, before what else the
// press does.
func (c *Screen) startDrag(hit *html.Node, col, row, mods int) {
	c.drag = nil
	h := closest(hit, func(n *html.Node) bool { return tokens(n, "data-on", "drag") })
	if h == nil || !has(h, "id") || mods&hotty.ModAlt != 0 {
		return
	}
	d := &dragging{from: attr(h, "id"), t: attr(h, "id"), c: col, r: row}
	if r, ok := c.layout().boxes[h]; ok {
		d.box = rect{r.x, r.y - c.scroll, r.w, r.h}
	}
	d.sx, d.sy = stepsOf(h)
	g := hotty.Drag{Col: col, Row: row}
	g.X, g.HasX = dragStep(col, d.box.x, d.box.w, d.sx)
	g.Y, g.HasY = dragStep(row, d.box.y, d.box.h, d.sy)
	d.x, d.y = g.X, g.Y
	c.drag = d
	c.emit(hotty.Event{Kind: hotty.EventDragStart, Target: d.from, Mods: mods &^ hotty.ModAlt, Drag: &g})
}

// dragTo is the pointer at col, row during the drag: a drag (or the
// dragend, which is always sent) where its target, its cell with no
// target, or a step changed.
func (c *Screen) dragTo(kind string, col, row, mods int) bool {
	d := c.drag
	g := hotty.Drag{Col: col, Row: row}
	g.X, g.HasX = dragStep(col, d.box.x, d.box.w, d.sx)
	g.Y, g.HasY = dragStep(row, d.box.y, d.box.h, d.sy)
	var t string
	tn := closest(c.hitAt(col, row), func(n *html.Node) bool {
		return has(n, "id") && (tokens(n, "data-on", "drag") || tokens(n, "data-on", "drop"))
	})
	if tn != nil {
		t = attr(tn, "id")
		if tokens(tn, "data-on", "drop") {
			if r, ok := c.layout().boxes[tn]; ok {
				sx, sy := stepsOf(tn)
				g.TX, g.HasTX = dragStep(col, r.x, r.w, sx)
				g.TY, g.HasTY = dragStep(row, r.y-c.scroll, r.h, sy)
			}
		}
	}
	same := t == d.t && (t != "" || col == d.c && row == d.r) &&
		g.X == d.x && g.Y == d.y && g.TX == d.tx && g.TY == d.ty && (g.HasTX || g.HasTY) == d.hasT
	if kind == hotty.EventDrag && same {
		return true
	}
	d.t, d.c, d.r, d.x, d.y, d.tx, d.ty, d.hasT = t, col, row, g.X, g.Y, g.TX, g.TY, g.HasTX || g.HasTY
	c.emit(hotty.Event{Kind: kind, Target: t, Mods: mods, Drag: &g})
	return true
}

// stepsOf are an element's data-steps across and down, 0 for none.
func stepsOf(n *html.Node) (x, y int) {
	f := strings.Fields(attr(n, "data-steps"))
	if len(f) == 0 || len(f) > 2 {
		return 0, 0
	}
	v := make([]int, 2)
	for i, s := range f {
		k, err := strconv.Atoi(s)
		if err != nil || k < 0 {
			return 0, 0
		}
		v[i] = k
	}
	return v[0], v[1]
}

// dragStep is the pointer's step along one axis of a box from at, size
// cells long, divided into count steps (SPEC §9.1): from the middle of
// the pointer's cell, rounded half up, clamped to 0 and count. ok is
// false with no steps.
func dragStep(p, at, size, count int) (int, bool) {
	if count == 0 {
		return 0, false
	}
	if size <= 0 {
		return 0, true
	}
	v := math.Floor((float64(p-at)+0.5)/float64(size)*float64(count) + 0.5)
	return int(max(0, min(v, float64(count)))), true
}

// hoverAt is the pointer at col, row with no button down (SPEC §9.4): a
// hover where the element with an id under it changed, Out where it left
// the Screen.
func (c *Screen) hoverAt(col, row int) bool {
	if c.body == nil || !c.hover {
		return false
	}
	if col < 0 || row < 0 || col >= c.cols || row >= c.rows {
		if c.hoverOut {
			return false
		}
		c.hoverOut, c.hoverID = true, ""
		c.emit(hotty.Event{Kind: hotty.EventHover, Hover: &hotty.Hover{Out: true}})
		return true
	}
	id := ""
	if n := withID(c.hitAt(col, row)); n != nil && n != c.body {
		id = attr(n, "id")
	}
	if !c.hoverOut && id == c.hoverID {
		return false
	}
	c.hoverOut, c.hoverID = false, id
	c.emit(hotty.Event{Kind: hotty.EventHover, Target: id, Hover: &hotty.Hover{Col: col, Row: row}})
	return true
}
