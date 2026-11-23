package surftui

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// The pointer in cells (for plx). A Screen hears the
// terminal's mouse as the program decodes it, at its own cells, and sends
// the surface the events a hotty host sends for the same gestures (SPEC
// §9): a press, a click with its count and mods, context menu and auxiliary clicks, a
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
// Other buttons go to contextmenu, aux or rightbutton subscriptions.
// A second button pressed during a gesture follows the first one's route.
func (c *Screen) Press(col, row, mods, button int) (taken bool) {
	if button < 0 {
		return false
	}
	if c.pressing && button == c.button && len(c.otherButtons) > 0 {
		return c.held
	}
	if c.pressing && button != c.button || len(c.otherButtons) > 0 {
		taken = c.held
		for _, routed := range c.otherButtons {
			taken = taken || routed
		}
		if c.otherButtons == nil {
			c.otherButtons = map[int]bool{}
		}
		c.otherButtons[button] = taken
		return taken
	}
	c.cutDrag()
	c.button = button
	c.resizing = nil
	c.textSel = nil
	c.point(col, row)
	// A press of any button ends what an earlier one left: its click,
	// whose release was lost, is no longer due.
	c.down, c.held, c.pressing = nil, false, true
	switch button {
	case 0:
		if c.startResize(col, row) || c.pressTextareaScrollbar(col, row) {
			taken = true
		} else {
			taken = c.pressAt(col, row, mods)
		}
	default:
		taken = c.pressOther(col, row, mods, button)
	}
	c.held = taken
	c.rehit = true // what the press did (the focus, a reveal) may move what is under it
	c.flush()
	return taken
}

// Move is the pointer at the cell col, row, mods held, with a button
// down or none. During a drag it sends the drag events a host sends
// (SPEC §9.1): one where the element under the pointer, or its step in
// the dragged element or in the drop target (data-steps), changed, not
// one for every cell. With no button down, it sends a hover where the element
// under the pointer changed (SPEC §9.4), on a placement made with Hover,
// as surf's driver makes it (surf.Surface.Attach); a cell outside the
// Screen (a negative col or row, or one past its size) is the pointer
// leaving it, a hover with Out. Lines changes after a Move through what
// the surface writes (the dragging class), and by itself where what the
// pointer is over looks other hovered (its hover: classes, the kit's
// :hover) and where a tooltip shows or goes, with a button down or none;
// Dirty says when, so a program needn't redraw on every motion.
func (c *Screen) Move(col, row, mods int) (taken bool) {
	if c.resizing != nil {
		c.resizeTo(col, row)
		c.point(col, row)
		c.rehit = true
		c.flush()
		return true
	}
	if c.textSel != nil {
		c.extendTextSelect(col, row)
		c.point(col, row)
		c.rehit = true
		c.flush()
		return true
	}
	c.point(col, row)
	switch {
	case c.drag != nil:
		c.dragTo(hotty.EventDrag, col, row, mods)
	case c.aux != nil:
		c.moveAux(col, row, mods)
	case !c.pressing && len(c.otherButtons) == 0:
		c.hoverAt(col, row)
	}
	taken = c.held
	for _, routed := range c.otherButtons {
		taken = taken || routed
	}
	taken = taken || !c.pressing && len(c.otherButtons) == 0 && c.body != nil && col >= 0 && row >= 0 && col < c.cols && row < c.rows
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
// or hears another press of the same button before this Release.
func (c *Screen) Release(col, row, mods int) (taken bool) {
	return c.ReleaseButton(col, row, mods, c.button)
}

// ReleaseButton is like [Screen.Release], and identifies the released DOM
// button so a second button's release leaves the active gesture held.
func (c *Screen) ReleaseButton(col, row, mods, button int) (taken bool) {
	if taken, ok := c.otherButtons[button]; ok {
		delete(c.otherButtons, button)
		return taken
	}
	if button != c.button {
		return false
	}
	if c.resizing != nil {
		c.point(col, row)
		c.resizeTo(col, row)
		c.resizing = nil
		taken, c.held, c.pressing = c.held, false, false
		c.rehit = true
		c.flush()
		return taken
	}
	if c.textSel != nil {
		ts := c.textSel
		c.point(col, row)
		if ts.moved {
			c.extendTextSelect(col, row)
		}
		c.textSel = nil
		taken, c.held, c.pressing = c.held, false, false
		if !ts.moved {
			c.releaseAt(col, row, mods)
		} else {
			c.down = nil
		}
		c.rehit = true
		c.flush()
		return taken
	}
	taken, c.held, c.pressing = c.held, false, false
	c.point(col, row)
	// What the release does (a click's focus, a reveal) may move what is
	// under it.
	c.rehit = true
	if button != hotty.ButtonPrimary {
		if c.drag != nil {
			c.dragTo(hotty.EventDragEnd, col, row, mods)
			c.drag = nil
		} else if a := c.aux; a != nil && c.auxTarget(col, row, button) == a.target {
			t := a.target
			e := hotty.Event{Kind: hotty.EventAuxClick, Target: attr(t, "id"), Button: button,
				At: &hotty.Cell{Col: col, Row: row}, Mods: mods, Value: attr(t, "value")}
			if e.Target != "" {
				e.Area = c.area(t)
			}
			if t.DataAtom == atom.A && has(t, "href") {
				e.Link = &hotty.Link{Href: attr(t, "href")}
				e.Area = c.area(t)
			}
			c.emit(e)
		}
		c.aux = nil
		c.flush()
		return taken
	}
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
	c.rehit = true
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
	hit := c.hitAt(col, row)
	var n *html.Node
	for cur := hit; cur != nil && cur != c.body; cur = cur.Parent {
		if v, ok := get(cur, "id"); ok && v != "" && !rt.RuntimeID(v) {
			n = cur
			break
		}
		if cur.DataAtom == atom.Label {
			if ctl := c.labelled(cur); ctl != nil && !disabled(ctl) && ctl.DataAtom == atom.Input {
				typ := attr(ctl, "type")
				if typ == "checkbox" || typ == "radio" {
					if v, ok := get(ctl, "id"); ok && v != "" && !rt.RuntimeID(v) {
						n = ctl
						break
					}
				}
			}
		}
	}
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
	c.resizing, c.textSel, c.aux = nil, nil, nil
	d := c.drag
	if d == nil {
		return
	}
	c.drag, c.down, c.held = nil, nil, false
	g := hotty.Drag{Col: d.c, Row: d.r, X: d.x, Y: d.y, HasX: d.sx > 0, HasY: d.sy > 0}
	c.emit(hotty.Event{Kind: hotty.EventDragEnd, Drag: &g})
}

// fieldResize tracks a live drag of a textarea's bottom-right resize grip.
type fieldResize struct {
	n        *html.Node
	col, row int
	rows     int
	cols     int
	mode     string
}

func isKitTextarea(n *html.Node) bool {
	return n != nil && n.DataAtom == atom.Textarea && (tokens(n, "class", "textarea") || attr(n, "data-slot") == "textarea")
}

func isTUICombobox(n *html.Node) bool {
	return n != nil && n.DataAtom == atom.Input && attr(n, "role") == "combobox" && attr(n, "data-mode") == "tui"
}

func textareaResizable(n *html.Node) bool {
	if !isKitTextarea(n) || disabled(n) {
		return false
	}
	return attr(n, "data-resize") != "none"
}

// ResizeMode reports the resize axis ("both", "vertical", "horizontal") of the
// textarea whose corner grip is at col, row, or "" if none.
func (c *Screen) ResizeMode(col, row int) string {
	if c.body == nil {
		return ""
	}
	l := c.layout()
	y := row + c.scroll
	for n, r := range l.fields {
		if !textareaResizable(n) || r.w < 4 {
			continue
		}
		if col == r.x+r.w-1 && y == r.y+r.h-1 {
			if m := attr(n, "data-resize"); m != "" {
				return m
			}
			return "both"
		}
	}
	return ""
}

func (c *Screen) pressTextareaScrollbar(col, row int) bool {
	if c.body == nil {
		return false
	}
	l := c.layout()
	y := row + c.scroll
	for n, r := range l.fields {
		if !isKitTextarea(n) || r.w < 4 || col != r.x+r.w-1 || y < r.y || y >= r.y+r.h {
			continue
		}
		barH := r.h
		if textareaResizable(n) && barH > 1 {
			barH--
		}
		if y >= r.y+barH {
			continue
		}
		f := c.field(n)
		f.ed.Width, f.ed.Rows = max(r.w-2, 1), r.h
		total := len(f.ed.Wrap())
		last := total - r.h
		if last <= 0 {
			continue
		}
		rel := y - r.y
		next := f.top
		switch {
		case barH >= 3 && rel == 0:
			next = max(0, f.top-1)
		case barH >= 3 && rel == barH-1:
			next = min(last, f.top+1)
		default:
			trackH := barH
			pos := rel
			if barH >= 3 {
				trackH = barH - 2
				pos = rel - 1
			}
			if trackH > 1 {
				next = (pos*last + (trackH-1)/2) / (trackH - 1)
			}
		}
		f.top = max(0, min(next, last))
		f.scrolled = true
		c.lay = nil
		return true
	}
	return false
}

func (c *Screen) startResize(col, row int) bool {
	if c.body == nil {
		return false
	}
	l := c.layout()
	y := row + c.scroll
	for n, r := range l.fields {
		if !textareaResizable(n) || r.w < 4 {
			continue
		}
		if col == r.x+r.w-1 && y == r.y+r.h-1 {
			c.resizing = &fieldResize{
				n:    n,
				col:  col,
				row:  row,
				rows: r.h,
				cols: r.w,
				mode: attr(n, "data-resize"),
			}
			return true
		}
	}
	return false
}

func attrPosInt(n *html.Node, key string) int {
	v, err := strconv.Atoi(attr(n, key))
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func (c *Screen) resizeTo(col, row int) {
	z := c.resizing
	if z == nil || !c.attached(z.n) {
		c.resizing = nil
		return
	}
	f := c.field(z.n)
	changed := false
	if z.mode != "horizontal" {
		minR := attrPosInt(z.n, "data-min-rows")
		if minR == 0 {
			minR = 2
		}
		maxR := max(minR, c.rows-2)
		if mr := attrPosInt(z.n, "data-max-rows"); mr > 0 && mr < maxR {
			maxR = max(minR, mr)
		}
		nr := max(minR, min(z.rows+(row-z.row), maxR))
		if f.rows != nr {
			f.rows = nr
			changed = true
		}
	}
	if z.mode != "vertical" {
		minC := attrPosInt(z.n, "data-min-cols")
		if minC == 0 {
			minC = 12
		}
		maxC := max(minC, c.cols-2)
		if mc := attrPosInt(z.n, "data-max-cols"); mc > 0 && mc < maxC {
			maxC = max(minC, mc)
		}
		nc := max(minC, min(z.cols+(col-z.col), maxC))
		if f.cols != nc {
			f.cols = nc
			changed = true
		}
	}
	if changed {
		c.lay = nil
	}
}

// fieldSelect tracks pointer-driven caret placement and selection in a text
// field or textarea (click, shift+click, double-click word, triple-click line,
// and drag-to-select).
type fieldSelect struct {
	n         *html.Node
	anchor    int
	anchorEnd int
	mode      int // 1 = char, 2 = word, 3 = line
	moved     bool
}

func (c *Screen) startTextSelect(n *html.Node, col, row, mods int) {
	f := c.field(n)
	chars := graphemesOf(f.ed.Value)
	idx := c.fieldIndexAt(n, col, row)
	clicks := c.count(n)
	switch {
	case mods&hotty.ModShift != 0:
		a, ok := f.ed.Anchor()
		if !ok {
			a = f.ed.Caret
		}
		f.ed.Select(a, idx)
		c.textSel = &fieldSelect{n: n, anchor: a, anchorEnd: a, mode: 1}
	case clicks == 2:
		lo, hi := wordBounds(chars, idx)
		f.ed.Select(lo, hi)
		c.textSel = &fieldSelect{n: n, anchor: lo, anchorEnd: hi, mode: 2}
	case clicks >= 3:
		lo, hi := lineBounds(chars, idx)
		f.ed.Select(lo, hi)
		c.textSel = &fieldSelect{n: n, anchor: lo, anchorEnd: hi, mode: 3}
	default:
		f.ed.Select(idx, idx)
		c.textSel = &fieldSelect{n: n, anchor: idx, anchorEnd: idx, mode: 1}
	}
}

func (c *Screen) extendTextSelect(col, row int) {
	ts := c.textSel
	if ts == nil || !c.attached(ts.n) {
		c.textSel = nil
		return
	}
	ts.moved = true
	f := c.field(ts.n)
	chars := graphemesOf(f.ed.Value)
	idx := c.fieldIndexAt(ts.n, col, row)
	switch ts.mode {
	case 2:
		lo, hi := wordBounds(chars, idx)
		if idx < ts.anchor {
			f.ed.Select(ts.anchorEnd, lo)
		} else {
			f.ed.Select(ts.anchor, hi)
		}
	case 3:
		lo, hi := lineBounds(chars, idx)
		if idx < ts.anchor {
			f.ed.Select(ts.anchorEnd, lo)
		} else {
			f.ed.Select(ts.anchor, hi)
		}
	default:
		f.ed.Select(ts.anchor, idx)
	}
	c.lay = nil
}

func (c *Screen) fieldIndexAt(n *html.Node, col, row int) int {
	l := c.layout()
	r, ok := l.fields[n]
	if !ok {
		return 0
	}
	f := c.field(n)
	chars := graphemesOf(f.ed.Value)
	if len(chars) == 0 {
		return 0
	}
	padX := 0
	innerW := r.w
	if isKitTextarea(n) && r.w >= 4 {
		padX = 1
		innerW = r.w - 2
	} else if isTUICombobox(n) && r.w >= 5 {
		padX = 1
		innerW = r.w - 4
	}
	localX := max(0, col-(r.x+padX))
	if n.DataAtom == atom.Textarea {
		f.ed.Width, f.ed.Rows = innerW, r.h
		rows := f.ed.Wrap()
		if len(rows) == 0 {
			return 0
		}
		localY := row + c.scroll - r.y
		vi := max(0, min(f.top+localY, len(rows)-1))
		rw := rows[vi]
		end := rw.End
		if end > rw.Start && chars[end-1] == "\n" {
			end--
		}
		cx := 0
		for gi := rw.Start; gi < end; gi++ {
			gw := widthOf(charsOf(chars[gi]))
			if localX < cx+gw {
				return gi
			}
			cx += gw
		}
		return end
	}
	// Single-line input.
	target := f.off + localX
	cx := 0
	for gi, g := range chars {
		gw := 1
		if !f.ed.Password {
			gw = widthOf(charsOf(g))
		}
		if target < cx+gw {
			return gi
		}
		cx += gw
	}
	return len(chars)
}

func isWordChar(g string) bool {
	r, _ := utf8.DecodeRuneInString(g)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func wordBounds(chars []string, idx int) (lo, hi int) {
	n := len(chars)
	if n == 0 {
		return 0, 0
	}
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	word := isWordChar(chars[idx])
	lo, hi = idx, idx+1
	for lo > 0 && isWordChar(chars[lo-1]) == word && chars[lo-1] != "\n" {
		lo--
	}
	for hi < n && isWordChar(chars[hi]) == word && chars[hi] != "\n" {
		hi++
	}
	return lo, hi
}

func lineBounds(chars []string, idx int) (lo, hi int) {
	n := len(chars)
	if n == 0 {
		return 0, 0
	}
	if idx > n {
		idx = n
	}
	if idx < 0 {
		idx = 0
	}
	lo = idx
	for lo > 0 && chars[lo-1] != "\n" {
		lo--
	}
	hi = idx
	for hi < n && chars[hi] != "\n" {
		hi++
	}
	return lo, hi
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

// auxPress is held until release or until it moves into a drag.
type auxPress struct {
	target *html.Node
	drag   *dragging
	mods   int
}

func (c *Screen) auxTarget(col, row, button int) *html.Node {
	hit := takesPointer(c.hitAt(col, row))
	return closest(hit, func(n *html.Node) bool {
		if button == hotty.ButtonSecondary {
			return tokens(n, "data-on", "rightbutton") || tokens(n, "data-on", "contextmenu")
		}
		return tokens(n, "data-on", "aux") && !(n.DataAtom == atom.A && attr(n, "target") == "_blank")
	})
}

func (c *Screen) pressOther(col, row, mods, button int) bool {
	if mods&hotty.ModAlt != 0 {
		return false
	}
	hit := c.hitAt(col, row)
	if closest(hit, func(n *html.Node) bool { return has(n, "data-hotty-modal-inert") }) != nil {
		return true
	}
	t := c.auxTarget(col, row, button)
	if t == nil {
		return false
	}
	if button == hotty.ButtonSecondary && !tokens(t, "data-on", "rightbutton") {
		if mods&hotty.ModShift != 0 {
			return false
		}
		e := hotty.Event{Kind: hotty.EventContextMenu, Target: attr(t, "id"),
			At: &hotty.Cell{Col: col, Row: row}, Mods: mods}
		if e.Target != "" {
			e.Area = c.area(t)
		}
		c.emit(e)
		return true
	}
	c.aux = &auxPress{target: t, mods: mods}
	if has(t, "id") && tokens(t, "data-on", "drag") {
		c.aux.drag = c.dragFrom(t, col, row)
	}
	return true
}

func (c *Screen) moveAux(col, row, mods int) {
	a := c.aux
	d := a.drag
	if d == nil || !c.attached(a.target) || d.c == col && d.r == row {
		return
	}
	c.beginDrag(d, a.mods, c.button)
	c.aux = nil
	c.dragTo(hotty.EventDrag, col, row, mods)
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
	c.beginDrag(c.dragFrom(h, col, row), mods, hotty.ButtonPrimary)
}

func (c *Screen) dragFrom(h *html.Node, col, row int) *dragging {
	d := &dragging{from: attr(h, "id"), t: attr(h, "id"), c: col, r: row}
	if r, ok := c.layout().boxes[h]; ok {
		d.box = rect{r.x, r.y - c.scroll, r.w, r.h}
	}
	d.sx, d.sy = stepsOf(h)
	return d
}

func (c *Screen) beginDrag(d *dragging, mods, button int) {
	g := hotty.Drag{Col: d.c, Row: d.r}
	g.X, g.HasX = dragStep(d.c, d.box.x, d.box.w, d.sx)
	g.Y, g.HasY = dragStep(d.r, d.box.y, d.box.h, d.sy)
	d.x, d.y = g.X, g.Y
	c.drag = d
	c.emit(hotty.Event{Kind: hotty.EventDragStart, Target: d.from, Mods: mods &^ hotty.ModAlt, Button: button, Drag: &g})
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

// point is the pointer at col, row, on the Screen or off it: the hovered
// chain is what is under it, with a button down or none, as a host's
// follows the pointer (Blitz's, on every move).
func (c *Screen) point(col, row int) {
	c.ptrCol, c.ptrRow, c.ptrIn = col, row, true
	c.rehit = true
	if c.body != nil {
		c.layout()
	}
}

// pointed is what the hovered chain starts at in l: the innermost
// element drawn at the pointer's cell that takes the pointer
// (takesPointer); nil with no pointer, off the Screen or over none.
func (c *Screen) pointed(l *layout) *html.Node {
	col, row := c.ptrCol, c.ptrRow
	if !c.ptrIn || c.body == nil || col < 0 || row < 0 || col >= c.cols || row >= c.rows {
		return nil
	}
	y := row + c.scroll
	if y >= len(l.cv.rows) {
		return nil
	}
	return takesPointer(l.cv.rows[y][col].own)
}
