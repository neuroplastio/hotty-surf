package cells

import (
	"github.com/clipperhouse/displaywidth"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The layout: the document drawn in cells, a subset of CSS's (SUBSET.md).
// Blocks stack; text and controls flow in lines and wrap; a flex row puts
// its items side by side, sized by their content, the room left going to
// those that grow; a flex column stacks them with its gap; a table sizes
// its columns by their cells. What it draws is a canvas as wide as the
// Screen, with the element under each cell, and each element's box.

// infinite is the width content is measured at: wide enough that nothing
// wraps.
const infinite = 1 << 20

type rect struct{ x, y, w, h int }

func (r rect) has(x, y int) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

func (r rect) union(o rect) rect {
	if r.w == 0 && r.h == 0 {
		return o
	}
	x0, y0 := min(r.x, o.x), min(r.y, o.y)
	x1, y1 := max(r.x+r.w, o.x+o.w), max(r.y+r.h, o.y+o.h)
	return rect{x0, y0, x1 - x0, y1 - y0}
}

type layout struct {
	c         *Screen
	cv        *canvas // nil while measuring
	boxes     map[*html.Node]rect
	fields    map[*html.Node]rect // each field's cells, its value's place
	scrolls   map[*html.Node]scrollBox
	adjusted  bool
	height    int
	caret     [2]int
	hasCaret  bool
	measuring bool
	// widthOnly is measure's layout, whose heights no one reads: a row
	// doesn't lay its items out again for them.
	widthOnly bool
	// nodes are what a pass knows of each node (kept), shared with the
	// measuring layouts: the tree doesn't change while it is laid out.
	// Widths of another gen than the pass's are not known. lastN and
	// lastM are the node last looked up and its memo: a pass asks of
	// one node several times in a row.
	nodes map[*html.Node]*nodeMemo
	gen   uint64
	lastN *html.Node
	lastM *nodeMemo
	// heights are the definite heights of the elements that have one
	// (h-screen, h-full, h-[…]), and of a column's growing items, which
	// it gives the rows its other items leave (flex-1).
	heights map[*html.Node]int
	grown   map[*html.Node]bool
	// overlays are the open popovers, drawn over the flow once it is
	// laid out (overlay).
	overlays []overlay
	// active is the element the focused one names its active one
	// (aria-activedescendant), where it is shown; adDone says it was
	// looked for.
	active *html.Node
	adDone bool
	// p is what the pass's layouts share: the frames it records what it
	// lays out in (laid.go), and its measuring layouts.
	p *pass
}

type scrollBox struct {
	view    rect
	content int
}

// scrollLimit reads the bounded overflow heights cells supports.
func scrollLimit(value string, screenRows int) (int, bool) {
	if strings.HasSuffix(value, "vh") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(value, "vh"), 64)
		if err == nil && n > 0 {
			return max(1, int(float64(screenRows)*n/100)), true
		}
	}
	if px, ok := length(value); ok && px > 0 {
		return max(1, int((px+15)/16)), true
	}
	return 0, false
}

// widths are a node's widths when nothing wraps (measure), as white
// space is collapsed and as it is kept (pre); has says which are known.
type widths struct {
	w   [2]int
	has [2]bool
}

func preIndex(pre bool) int {
	if pre {
		return 1
	}
	return 0
}

// inh is what an element's content inherits: its look, and whether white
// space is kept.
type inh struct {
	look look
	pre  bool
	// nowrap is a truncate's content: one line, cut with … where it
	// runs out of room.
	nowrap bool
}

// layout is the document laid out at the Screen's width, made again after
// anything changed, with the hovered chain what is under the pointer.
func (c *Screen) layout() *layout {
	l := c.layOut()
	if !c.rehit || c.laying > 0 {
		return l
	}
	// What is under a still pointer may have moved: the chain is hit
	// again, once, on the layout as it is, as a host hits it after each
	// pass (Blitz's refresh_hover), and laid out again where its looks
	// change. A hover that moves what is under the pointer isn't chased.
	c.rehit = false
	c.setHover(c.pointed(l))
	if c.lay == nil {
		l = c.layOut()
	}
	return l
}

// layOut is the Screen's layout as its tree, its chain and its size give
// it: the last pass's while nothing dropped it, else a new pass.
func (c *Screen) layOut() *layout {
	if c.lay != nil {
		return c.lay
	}
	c.laying++
	defer func() { c.laying-- }()
	l := &layout{c: c, cv: &canvas{w: max(c.cols, 1)}}
	boxes := 0
	if c.kept != nil {
		// As many as the last pass's, which this one most likely has.
		boxes = c.kept.boxes
	}
	l.boxes = make(map[*html.Node]rect, boxes)
	l.fields, l.scrolls = map[*html.Node]rect{}, map[*html.Node]scrollBox{}
	if c.kept == nil {
		c.kept = &kept{nodes: map[*html.Node]*nodeMemo{}}
	}
	// A text's widths are its pass's.
	c.kept.gen++
	l.nodes, l.gen = c.kept.nodes, c.kept.gen
	l.p = &pass{free: c.kept.frames}
	c.kept.frames = nil
	l.cv.hint = c.kept.rows
	l.cv.used, c.kept.used = c.kept.used, nil
	c.kept.settle(l)
	if c.body != nil {
		l.height, _ = l.node(c.body, 0, 0, max(c.cols, 1), inh{})
		l.drawOverlays()
		l.drawTip()
	}
	c.kept.frames, c.kept.rows = l.p.free, len(l.cv.rows)
	if l.adjusted {
		// Nothing reads a pass laid out again: its rows are the next's.
		c.kept.used = l.cv.chunks
		c.lay = nil
		return c.layOut()
	}
	if old := c.kept.canvas; old != nil {
		c.kept.used = old.chunks
	}
	c.kept.canvas = l.cv
	c.kept.boxes = len(l.boxes)
	c.lay = l
	if t := c.revealAfter; t != nil {
		where := c.revealWhere
		c.revealAfter, c.revealWhere = nil, ""
		c.reveal(t, where)
		if c.lay == nil {
			return c.layOut()
		}
		l = c.lay
	}
	c.scroll = min(c.scroll, max(l.height-c.rows, 0))
	c.scroll = max(c.scroll, 0)
	return l
}

// measure is the width n's content takes when nothing wraps, kept as
// node keeps a layout (laid).
func (l *layout) measure(n *html.Node, in inh) int {
	_, w := l.measurer(true).node(n, 0, 0, infinite, in)
	return w
}

// measureText is the width text k takes in n's line when nothing wraps,
// memoized as measure's.
func (l *layout) measureText(k, n *html.Node, in inh) int {
	i := preIndex(in.pre)
	if w, ok := l.widthOf(k, i); ok {
		return w
	}
	_, w := l.measurer(true).inline([]*html.Node{k}, n, 0, 0, infinite, in, false)
	l.setWidth(k, i, w)
	return w
}

// widthOf is n's width when nothing wraps (measure), if this pass's gen
// knows it.
func (l *layout) widthOf(n *html.Node, i int) (int, bool) {
	m := l.memoOf(n)
	if m.wGen != l.gen || !m.w.has[i] {
		return 0, false
	}
	return m.w.w[i], true
}

func (l *layout) setWidth(n *html.Node, i, w int) {
	m := l.memoOf(n)
	if m.wGen != l.gen {
		m.w, m.wGen = widths{}, l.gen
	}
	m.w.w[i], m.w.has[i] = w, true
}

// memoOf is n's memo, made where it has none.
func (l *layout) memoOf(n *html.Node) *nodeMemo {
	if n == l.lastN && l.lastM != nil {
		return l.lastM
	}
	m := l.nodes[n]
	if m == nil {
		m = &nodeMemo{}
		l.nodes[n] = m
	}
	l.lastN, l.lastM = n, m
	return m
}

func (l *layout) ext(n *html.Node, r rect) {
	if l.cv == nil {
		return
	}
	l.rec(oExt, n, r.x, r.y, r.w, r.h, 0)
	l.boxes[n] = l.boxes[n].union(r)
}

// draw is the canvas's put, recorded (laid).
func (l *layout) draw(x, y int, ch string, w int, lk look, own *html.Node) {
	l.recPut(x, y, ch, w, lk, own)
	l.cv.put(x, y, ch, w, lk, own)
}

// setHeight gives a column's growing item its rows (grow).
func (l *layout) setHeight(n *html.Node, h int) {
	if l.heights == nil {
		l.heights = map[*html.Node]int{}
	}
	if l.grown == nil {
		l.grown = map[*html.Node]bool{}
	}
	l.heights[n], l.grown[n] = h, true
}

// lookOf is the look of n's content: what it inherits, what its style
// adds, reverse while it has the keyboard, dim while disabled.
func (l *layout) lookOf(n *html.Node, in look, st box) look {
	lk := in | st.look
	if lk&lMuted != 0 {
		// A colour of its own changes its look here (inMuted).
		l.memoOf(n).inMuted = true
	}
	if fg, _ := l.colour(n); unmutes(fg) {
		// A colour of its own: no longer the muted text around it.
		lk &^= lMuted
	}
	if st.look&lBold == 0 && in&lBold != 0 && st.fx.normal {
		lk &^= lBold
	}
	if l.c.active {
		// The keyboard's: the focused element, or the item it names its
		// active one (a menu's, a list's), drawn in its place.
		ad := l.activeItem()
		if n == ad || n == l.c.focused && ad == nil {
			if n.DataAtom == atom.Input && attr(n, "type") == "checkbox" && attr(n, "data-mode") == "tui" {
				lk |= lBold
			} else {
				lk |= lReverse
			}
		}
	}
	// Dim while disabled; the chosen tab among tabs drawn alike; a
	// group's label in a menu or a list, not one of its items (facts).
	return lk | st.fx.add
}

// activeItem is the element the focused one names its active one
// (aria-activedescendant), if it is shown; nil for none.
func (l *layout) activeItem() *html.Node {
	if l.adDone {
		return l.active
	}
	l.adDone = true
	f := l.c.focused
	if f == nil {
		return nil
	}
	id := attr(f, "aria-activedescendant")
	if id == "" {
		return nil
	}
	t := l.c.byID(id)
	for a := t; a != nil && a.Type == html.ElementNode; a = a.Parent {
		if l.style(a).display == dNone {
			return nil
		}
	}
	l.active = t
	return t
}

// node lays n out as a block at x, y, w wide, and says how high it is and
// how wide its content is. Where a pass laid it out with the same inputs
// and nothing it read changed since, it does what that pass did (laid).
func (l *layout) node(n *html.Node, x, y, w int, in inh) (h, used int) {
	st := l.style(n)
	if n.Type != html.ElementNode || st.display == dNone && n != l.c.body {
		return 0, 0
	}
	if n == l.c.body {
		st = box{}
	}
	key := laidKey{mode: l.mode(), w: w, in: in}
	if key.mode == mDrawn {
		key.def, key.grown = l.definite(n, st), l.grown[n]
	}
	m := l.memoOf(n)
	if r := l.laidOf(m, key); r != nil {
		l.replay(r, x, y)
		l.adopt(r, x, y)
		return r.h, r.used
	}
	f := l.push(x, y, key.mode == mDrawn)
	// Its margins (mt-*, mb-*): rows above and below its box.
	h, used = l.box(n, st, x, y+st.marginT, w, in)
	h += st.marginT + st.marginB
	r := l.pop(f, key, h, used)
	keep(m, r)
	l.adopt(r, x, y)
	return h, used
}

// box lays n out, as node does, without its margins.
func (l *layout) box(n *html.Node, st box, x, y, w int, in inh) (h, used int) {
	// Its width (w-*, width) and its largest (max-w-*, max-width), as far
	// as its container has room.
	if st.width > 0 {
		w = min(w, st.width)
	}
	if st.maxWidth > 0 {
		w = min(w, st.maxWidth)
	}
	if st.width > 0 {
		defer func() { used = max(used, w) }()
	}
	in.look = l.lookOf(n, in.look, st)
	in.pre = in.pre || st.pre
	in.nowrap = in.nowrap || st.truncate
	sep := st.fx.sep
	if sep != "" && textOf(n) != "" {
		sep = ""
	}
	if sep == "vertical" {
		if l.cv != nil {
			l.draw(x, y, "│", 1, in.look|lRule, n)
		}
		l.done(n, rect{x, y, 1, 1})
		return 1, 1
	}
	if n.DataAtom == atom.Hr || sep == "horizontal" {
		for i := 0; i < w && l.cv != nil; i++ {
			l.draw(x+i, y, "─", 1, in.look|lRule, n)
		}
		l.done(n, rect{x, y, w, 1})
		return 1, 1
	}
	cx, cy := x+st.padL, y+st.padT
	cw := max(w-st.padL-st.padR, 1)
	if m := st.fx.marker; m != "" {
		mw := widthOf(charsOf(m))
		if l.cv != nil {
			px := cx
			for _, ch := range charsOf(m) {
				l.draw(px, cy, ch.s, ch.w, in.look, n)
				px += ch.w
			}
		}
		cx += mw
		cw = max(cw-mw, 1)
		used += mw
	}
	var ch, cu, lead, ew int
	if s, ok := st.fx.empty, st.fx.hasEmpty; ok {
		// The kit's ::before: a line of its own before the rest.
		cs := charsOf(s)
		if widthOf(cs) > cw {
			cs = clip(cs, cw)
		}
		l.paint(cs, cx, cy, in.look|lMuted, n, n)
		ew = widthOf(cs)
		cy++
		lead = 1
	}
	if fh, fu, ok := l.figure(n, st.fx.fig, cx, cy, cw, in); ok {
		h = st.padT + fh + st.padB
		used += st.padL + fu + st.padR
		if l.c.active && n == l.activeItem() && h > 0 && w > used && !l.measuring {
		for j := y; j < y+h; j++ {
			for i := x + used; i < x+w; i++ {
				l.draw(i, j, " ", 1, in.look, n)
			}
		}
	}
	if isKitTextarea(n) || tokens(n, "class", "select") {
			l.done(n, rect{x, y, used, h})
		} else {
			l.done(n, rect{x, y, w, h})
		}
		return h, used
	}
	limit, bounded := scrollLimit(st.maxHeight, l.screenRows(st.maxHeight))
	bounded = bounded && st.scrollY
	def := l.definite(n, st)
	if l.grown[n] && st.scrollY {
		// A growing item that scrolls: the rows its column gave it.
		limit, bounded = max(def-st.padT-st.padB-lead, 0), true
	}
	originalY := cy
	offset := 0
	if bounded {
		offset = l.c.boxScroll[n]
		if f := l.top(); f != nil {
			f.scrolls = append(f.scrolls, scrollAt{n, offset})
		}
		cy -= offset
		if l.cv != nil {
			l.rec(oClip, nil, cx, originalY, cw, limit, 0)
			l.cv.clips = append(l.cv.clips, rect{cx, originalY, cw, limit})
		}
	}
	switch st.display {
	case dFlex:
		if st.column {
			if def > 0 {
				l.grow(n, cw, in, max(def-st.padT-st.padB-lead, 0), st.gapY)
			}
			ch, cu = l.children(n, cx, cy, cw, in, true, st.gapY)
		} else {
			ch, cu = l.row(n, st, cx, cy, cw, in)
		}
	case dTable:
		ch, cu = l.table(n, cx, cy, cw, in)
	case dAtom:
		ch, cu = l.inline([]*html.Node{n}, n.Parent, cx, cy, cw, in, true)
	default:
		if n.DataAtom == atom.Details && !has(n, "open") {
			// Only its summary shows.
			var sum []*html.Node
			for k := n.FirstChild; k != nil; k = k.NextSibling {
				if k.DataAtom == atom.Summary {
					sum = append(sum, k)
					break
				}
			}
			for _, k := range sum {
				ch, cu = l.node(k, cx, cy, cw, in)
			}
		} else {
			ch, cu = l.children(n, cx, cy, cw, in, false, 0)
		}
	}
	if bounded {
		if l.cv != nil {
			l.rec(oUnclip, nil, 0, 0, 0, 0, 0)
			l.cv.clips = l.cv.clips[:len(l.cv.clips)-1]
			view := rect{cx, originalY, cw, min(ch, limit)}
			l.rec(oScroll, n, view.x, view.y, view.w, view.h, ch)
			l.scrolls[n] = scrollBox{view: view, content: ch}
			if maxOffset := max(ch-limit, 0); offset > maxOffset {
				l.rec(oAdjust, n, 0, 0, 0, 0, maxOffset)
				l.c.boxScroll[n] = maxOffset
				l.adjusted = true
			}
		}
		ch = min(ch, limit)
	}
	h = st.padT + lead + ch + st.padB
	if def > 0 && (st.display == dFlex && st.column || l.grown[n]) {
		h = max(h, def) // a column as high as it is, an item as it grew
	}
	used += st.padL + max(cu, ew) + st.padR
	if h == 0 && st.fx.marker != "" {
		h = 1
	}
	if l.c.active && n == l.activeItem() && h > 0 && w > used && !l.measuring {
		for j := y; j < y+h; j++ {
			for i := x + used; i < x+w; i++ {
				l.draw(i, j, " ", 1, in.look, n)
			}
		}
	}
	if inTUIPopover(n) && attr(n, "role") == "option" {
		if !l.measuring && h > 0 && w >= 3 && attr(n, "aria-selected") == "true" {
			l.draw(x+w-2, y, "✓", 1, in.look, n)
		}
		used += 3 // space + ✓ + right pad
	}
	if st.framed && !l.measuring && l.cv != nil && h >= 2 && w >= 4 {
		fw := w
		bl := in.look | lRule
		if tokens(n, "class", "alert") && attr(n, "data-variant") == "destructive" {
			bl = in.look
		}
		l.draw(x, y, "╭", 1, bl, n)
		l.draw(x+fw-1, y, "╮", 1, bl, n)
		l.draw(x, y+h-1, "╰", 1, bl, n)
		l.draw(x+fw-1, y+h-1, "╯", 1, bl, n)
		for i := 1; i < fw-1; i++ {
			l.draw(x+i, y, "─", 1, bl, n)
			l.draw(x+i, y+h-1, "─", 1, bl, n)
		}
		for j := 1; j < h-1; j++ {
			l.draw(x, y+j, "│", 1, bl, n)
			l.draw(x+fw-1, y+j, "│", 1, bl, n)
		}
		used = max(used, fw)
	}
	if isKitTextarea(n) || tokens(n, "class", "select") || tokens(n, "class", "combobox") {
		l.done(n, rect{x, y, used, h})
	} else {
		l.done(n, rect{x, y, w, h})
	}
	return h, used
}

// screenRows is the Screen's height, as a length v in vh reads it ("" for
// the height itself): the layout being recorded holds at that height
// only (laid).
func (l *layout) screenRows(v string) int {
	if v == "" || strings.HasSuffix(v, "vh") {
		if f := l.top(); f != nil {
			f.rows = true
		}
	}
	return l.c.rows
}

// definite is n's height where it has one, in rows, 0 where it is its
// content's: h-screen (the Screen's), h-full (its parent's, the body's
// the Screen's), h-[…] or an inline height, or the rows its column gave
// it as a growing item (grow).
func (l *layout) definite(n *html.Node, st box) int {
	if l.measuring {
		return 0
	}
	if n == l.c.body || n.Parent == l.c.body && attr(n, "id") == "root" {
		// The window, and the document's own root in it, which a host
		// makes as high as the window.
		return l.screenRows("")
	}
	if h, ok := l.heights[n]; ok {
		return h
	}
	h := 0
	switch st.height {
	case "":
	case "screen", "dvh", "svh", "lvh":
		h = l.screenRows("")
	case "full", "100%":
		if p := n.Parent; p != nil && p.Type == html.ElementNode {
			h = l.definite(p, l.style(p))
		}
	default:
		h, _ = scrollLimit(st.height, l.screenRows(st.height))
	}
	if h > 0 {
		if l.heights == nil {
			l.heights = map[*html.Node]int{}
		}
		l.heights[n] = h
	}
	return h
}

// grow gives a column's growing items (flex-1, grow) the rows of its
// height (fill) that its other items and its gaps leave, shared alike,
// measured w wide: a list that fills the screen above a pinned footer.
func (l *layout) grow(n *html.Node, w int, in inh, fill, gap int) {
	var growers []*html.Node
	used, items := 0, 0
	m := l.measurer(false)
	var run []*html.Node
	flush := func() {
		if len(run) > 0 {
			hh, _ := m.inline(run, n, 0, 0, w, in, false)
			used += hh
			items++
			run = nil
		}
	}
	for _, k := range l.kids(n) {
		switch k.Type {
		case html.TextNode:
			if strings.TrimSpace(k.Data) != "" {
				run = append(run, k)
				flush()
			}
		case html.ElementNode:
			st := l.style(k)
			if st.display == dNone {
				continue
			}
			items++
			if st.grow {
				growers = append(growers, k)
				continue
			}
			hh, _ := m.node(k, 0, 0, w, in)
			used += hh
		}
	}
	if len(growers) == 0 {
		return
	}
	rest := max(fill-used-gap*max(items-1, 0), 0)
	for i, k := range growers {
		share := rest / (len(growers) - i)
		rest -= share
		l.rec(oHeight, k, 0, 0, 0, 0, share)
		l.setHeight(k, share)
	}
}

// overlay is an open popover drawn over the flow, below its trigger, as
// a host draws it.
type overlay struct {
	n    *html.Node
	x, y int
	w    int
	in   inh
}

// overlay takes an open popover below its trigger out of the flow (it
// takes no rows) to draw it over what follows, once the rest is laid
// out; y is where its trigger ends. One above its trigger (data-side
// top) stays before it in the flow, which has no room above the top
// row, and one beside it (inline-*) a row with it.
func (l *layout) overlay(k *html.Node, x, y, w int, in inh) bool {
	if !has(k, "data-popover") {
		return false
	}
	switch attr(k, "data-side") {
	case "", "bottom", "block-end":
	default:
		return false
	}
	if l.cv != nil {
		if f := l.top(); f != nil && f.drawn {
			f.ops = append(f.ops, lop{kind: oOverlay, n: k, x: int32(x - f.x), y: int32(y - f.y), w: int32(w), in: in})
		}
		l.overlays = append(l.overlays, overlay{n: k, x: x, y: y, w: w, in: in})
	}
	return true
}

// drawOverlays draws the open popovers over the flow, each on a ground
// of its own (its cells blanked), as wide as its content: the last
// drawn is on top, and one opened inside another is drawn after it.
func (l *layout) drawOverlays() {
	for len(l.overlays) > 0 {
		o := l.overlays[0]
		l.overlays = l.overlays[1:]
		if attr(o.n, "data-mode") == "tui" && o.w >= 4 {
			l.drawTUIPopover(o)
			continue
		}
		h, pw := l.measurer(false).node(o.n, 0, 0, o.w, o.in)
		pw = max(min(pw, o.w), 1)
		l.cv.clear(o.x, o.y, pw, h)
		l.node(o.n, o.x, o.y, pw, o.in)
	}
}

// drawTUIPopover draws a TUI-mode popover in a rounded single-line box
// (╭─╮ / │…│ / ╰─╯) sized at least to its trigger width and aligned by
// data-align, with full-width interior rows for active/hover highlights.
func (l *layout) drawTUIPopover(o overlay) {
	maxInner := max(o.w-2, 1)
	h, cw := l.measurer(false).node(o.n, 0, 0, maxInner, o.in)
	var tb rect
	if o.n.Parent != nil {
		for ch := o.n.Parent.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != html.ElementNode || ch == o.n {
				continue
			}
			if ch.DataAtom == atom.Button || ch.DataAtom == atom.Input ||
				tokens(ch, "class", "input-group") || tokens(ch, "class", "combobox-chips") {
				if r, ok := l.boxes[ch]; ok {
					tb = r
					break
				}
			}
		}
	}
	iw := max(cw+1, 14)
	if tb.w > 2 {
		iw = max(iw, tb.w-2)
	}
	iw = max(min(iw, maxInner), 1)
	fw, fh := iw+2, h+2
	ox := o.x
	if tb.w > 0 {
		switch attr(o.n, "data-align") {
		case "end":
			ox = tb.x + tb.w - fw
		case "center":
			ox = tb.x + (tb.w-fw)/2
		default:
			ox = tb.x
		}
	}
	ox = max(0, min(ox, l.c.cols-fw))
	l.cv.clear(ox, o.y, fw, fh)
	l.node(o.n, ox+1, o.y+1, iw, o.in)
	l.draw(ox, o.y, "╭", 1, lRule, o.n)
	l.draw(ox+fw-1, o.y, "╮", 1, lRule, o.n)
	l.draw(ox, o.y+fh-1, "╰", 1, lRule, o.n)
	l.draw(ox+fw-1, o.y+fh-1, "╯", 1, lRule, o.n)
	for i := 1; i < fw-1; i++ {
		l.draw(ox+i, o.y, "─", 1, lRule, o.n)
		l.draw(ox+i, o.y+fh-1, "─", 1, lRule, o.n)
	}
	for j := 1; j < fh-1; j++ {
		l.draw(ox, o.y+j, "│", 1, lRule, o.n)
		l.draw(ox+fw-1, o.y+j, "│", 1, lRule, o.n)
	}
	l.done(o.n, rect{ox, o.y, fw, fh})
}

// drawTip draws the tooltip of the element the pointer is over
// (data-tooltip, ui.Tooltip's), as a host shows it on hover: its text, a
// cell in from each end, on one row beside the element, on its side
// (data-side: above unless it says otherwise; the other side where the
// screen has no room, as a browser flips one), aligned by data-align,
// over everything else, inside the screen and cut with an ellipsis where
// the screen is narrower.
func (l *layout) drawTip() {
	c := l.c
	n := c.tip
	if n == nil || !c.attached(n) {
		return
	}
	b, ok := l.boxes[n]
	if !ok {
		return
	}
	text := " " + strings.Join(strings.Fields(attr(n, "data-tooltip")), " ") + " "
	if displaywidth.String(text) > c.cols {
		text = displaywidth.TruncateString(text, c.cols, "…")
	}
	w := displaywidth.String(text)
	above, below := b.y-1, b.y+b.h
	x, y := b.x+(b.w-w)/2, above
	switch attr(n, "data-align") {
	case "start":
		x = b.x
	case "end":
		x = b.x + b.w - w
	}
	switch attr(n, "data-side") {
	case "bottom":
		y = below
	case "left", "inline-start":
		x, y = b.x-w-1, b.y
		if x < 0 {
			x = b.x + b.w + 1
		}
	case "right", "inline-end":
		x, y = b.x+b.w+1, b.y
		if x+w > c.cols && b.x-w-1 >= 0 {
			x = b.x - w - 1
		}
	}
	top, end := c.scroll, c.scroll+c.rows
	switch {
	case y < top:
		y = below
	case y >= end:
		y = above
	}
	if y < top || y >= end {
		return
	}
	x = max(0, min(x, c.cols-w))
	g := displaywidth.StringGraphemes(text)
	for g.Next() {
		l.draw(x, y, g.Value(), g.Width(), lTip, n)
		x += g.Width()
	}
}

// done records n's box, and makes its empty cells its own.
func (l *layout) done(n *html.Node, r rect) {
	if l.cv == nil {
		return
	}
	l.rec(oDone, n, r.x, r.y, r.w, r.h, 0)
	l.boxes[n] = r
	l.cv.own(r.x, r.y, r.w, r.h, n)
}

func inTUIPopover(n *html.Node) bool {
	return closest(n, func(a *html.Node) bool {
		return has(a, "data-popover") && attr(a, "data-mode") == "tui"
	}) != nil
}

// marker is a list item's marker, where its list has one (list-disc,
// list-decimal: Tailwind's lists have none).
func marker(n *html.Node) string {
	switch attr(n, "role") {
	case "option":
		// A popup list's option: TUI mode draws ✓ on the right with 1-cell
		// left padding; legacy popovers draw ✓ on the left.
		if pop := closest(n, func(a *html.Node) bool { return has(a, "data-popover") }); pop != nil {
			if attr(pop, "data-mode") == "tui" {
				if n.Parent != nil && attr(n.Parent, "role") == "group" {
					return "  "
				}
				return " "
			}
			if attr(n, "aria-selected") == "true" {
				return "✓ "
			}
			return "  "
		}
	case "heading":
		if inTUIPopover(n) {
			return " "
		}
	case "menuitemcheckbox", "menuitemradio":
		if attr(n, "aria-checked") == "true" {
			return "✓ "
		}
		return "  "
	}
	if n.DataAtom != atom.Li || n.Parent == nil {
		return ""
	}
	switch styleOf(n.Parent, false).list { // a list's hover: isn't read
	case "disc":
		return "• "
	case "decimal":
		i := 1
		for k := n.Parent.FirstChild; k != nil && k != n; k = k.NextSibling {
			if k.DataAtom == atom.Li {
				i++
			}
		}
		return strconv.Itoa(i) + ". "
	}
	return ""
}

// children lays out n's children as a block's: blocks stacked, and the
// text and controls between them in lines. As a flex column's (items),
// every child is an item of its own, gap rows apart.
func (l *layout) children(n *html.Node, x, y, w int, in inh, items bool, gap int) (h, used int) {
	var run []*html.Node
	count := 0
	add := func(hh, u int) {
		if hh > 0 {
			if count > 0 {
				h += gap
			}
			count++
		}
		h += hh
		used = max(used, u)
	}
	at := func() int {
		if count > 0 {
			return y + h + gap
		}
		return y + h
	}
	flush := func() {
		if len(run) > 0 {
			add(l.inline(run, n, x, at(), w, in, false))
			run = nil
		}
	}
	for _, k := range l.kids(n) {
		switch k.Type {
		case html.TextNode:
			run = append(run, k)
		case html.ElementNode:
			st := l.style(k)
			if st.display == dNone {
				continue
			}
			if !items && (st.display == dInline || st.display == dAtom) && !l.hasBlock(k) {
				run = append(run, k)
				continue
			}
			flush()
			if l.overlay(k, x, at(), w, in) {
				continue
			}
			add(l.node(k, x, at(), w, in))
		}
		if items {
			flush()
		}
	}
	flush()
	return h, used
}

// hasBlock says whether an inline element holds a block, which makes it
// one.
func (l *layout) hasBlock(n *html.Node) bool {
	if isAtom(n) || n.Data == "svg" {
		return false
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type != html.ElementNode {
			continue
		}
		switch st := l.style(k); st.display {
		case dBlock, dFlex, dTable:
			return true
		case dInline:
			if l.hasBlock(k) {
				return true
			}
		}
	}
	return false
}

// row lays out a flex row: its items side by side, each as wide as its
// content, the room left to those that grow, or between them
// (justify-between); too little room is taken from the widest.
func (l *layout) row(n *html.Node, st box, x, y, w int, in inh) (h, used int) {
	items := l.c.bufs.items()
	defer func() { l.c.bufs.putItems(items) }()
	for _, k := range l.kids(n) {
		switch k.Type {
		case html.TextNode:
			if !in.pre && strings.TrimSpace(k.Data) == "" {
				continue
			}
			items = append(items, item{n: k, w: l.measureText(k, n, in)})
		case html.ElementNode:
			ks := l.style(k)
			if ks.display == dNone {
				continue
			}
			items = append(items, item{n: k, w: l.measure(k, in), grow: ks.grow || ks.full && isTextField(k) || ks.fx.grows, fixed: ks.noShrink})
		}
	}
	if len(items) == 0 {
		return 0, 0
	}
	gaps := st.gapX * (len(items) - 1)
	sum := 0
	for _, it := range items {
		sum += it.w
	}
	used = sum + gaps
	if l.widthOnly {
		// measure's: the width alone.
		return 0, used
	}
	if l.measuring {
		for _, it := range items {
			hh := 0
			if it.n.Type == html.TextNode {
				hh = 1
			} else {
				hh, _ = l.node(it.n, 0, 0, it.w, in)
			}
			h = max(h, hh)
		}
		return h, used
	}
	if st.wrap && used > w && len(items) > 1 {
		// flex-wrap: as many items a line as fit, each as wide as its
		// content.
		cx, lineH := 0, 0
		for _, it := range items {
			iw := min(it.w, w)
			if cx > 0 && cx+iw > w {
				h += lineH + st.gapY
				cx, lineH = 0, 0
			}
			var hh int
			if it.n.Type == html.TextNode {
				hh, _ = l.inline([]*html.Node{it.n}, n, x+cx, y+h, iw, in, false)
			} else {
				hh, _ = l.node(it.n, x+cx, y+h, iw, in)
			}
			lineH = max(lineH, hh)
			cx += iw + st.gapX
		}
		return h + lineH, w
	}
	avail := max(w-gaps, len(items))
	lead, between := 0, l.c.bufs.ints(len(items))
	defer l.c.bufs.putInts(between)
	if sum <= avail {
		extra := avail - sum
		growers := 0
		for _, it := range items {
			if it.grow {
				growers++
			}
		}
		switch {
		case growers > 0:
			for i := range items {
				if items[i].grow {
					share := extra / growers
					extra -= share
					growers--
					items[i].w += share
				}
			}
		case st.between && len(items) > 1:
			for i := 0; extra > 0; i = (i + 1) % (len(items) - 1) {
				between[i]++
				extra--
			}
		case st.end:
			lead = extra
		case st.center:
			lead = extra / 2
		}
	} else {
		for over := sum - avail; over > 0; over-- {
			wi := -1
			for i := range items {
				if !items[i].fixed && (wi < 0 || items[i].w > items[wi].w) {
					wi = i
				}
			}
			if wi < 0 || items[wi].w <= 1 {
				break
			}
			items[wi].w--
		}
	}
	cx := x + lead
	if st.itemsCenter && len(items) > 1 && l.cv != nil {
		hs := make([]int, len(items))
		m := layout{c: l.c, measuring: true, nodes: l.nodes, gen: l.gen, p: &pass{}}
		for i, it := range items {
			if it.n.Type == html.TextNode {
				hs[i], _ = m.inline([]*html.Node{it.n}, n, cx, y, it.w, in, false)
			} else {
				hs[i], _ = m.node(it.n, cx, y, it.w, in)
			}
			h = max(h, hs[i])
		}
		for i, it := range items {
			dy := (h - hs[i]) / 2
			if it.n.Type == html.TextNode {
				l.inline([]*html.Node{it.n}, n, cx, y+dy, it.w, in, false)
			} else {
				l.node(it.n, cx, y+dy, it.w, in)
			}
			cx += it.w + st.gapX + between[i]
		}
		return h, used
	}
	for i, it := range items {
		var hh int
		if it.n.Type == html.TextNode {
			hh, _ = l.inline([]*html.Node{it.n}, n, cx, y, it.w, in, false)
		} else {
			hh, _ = l.node(it.n, cx, y, it.w, in)
		}
		h = max(h, hh)
		cx += it.w + st.gapX + between[i]
	}
	return h, used
}

// table lays out a table: its rows' cells in columns as wide as their
// widest cell, two cells apart, a rule under its head.
func (l *layout) table(n *html.Node, x, y, w int, in inh) (h, used int) {
	type row struct {
		tr    *html.Node
		cells []*html.Node
		head  bool
	}
	var rows []row
	addRow := func(tr *html.Node, head bool) {
		if l.style(tr).display == dNone {
			return
		}
		r := row{tr: tr, head: head}
		for k := tr.FirstChild; k != nil; k = k.NextSibling {
			if (k.DataAtom == atom.Td || k.DataAtom == atom.Th) && l.style(k).display != dNone {
				r.cells = append(r.cells, k)
			}
		}
		rows = append(rows, r)
	}
	var caption *html.Node
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		switch k.DataAtom {
		case atom.Tr:
			addRow(k, false)
		case atom.Thead, atom.Tbody, atom.Tfoot:
			if l.style(k).display == dNone {
				continue
			}
			for r := k.FirstChild; r != nil; r = r.NextSibling {
				if r.DataAtom == atom.Tr {
					addRow(r, k.DataAtom == atom.Thead)
				}
			}
		case atom.Caption:
			caption = k
		}
	}
	if caption != nil {
		ch, cu := l.node(caption, x, y, w, in)
		h += ch
		used = cu
	}
	// Each cell's first column, and how many it spans (colspan).
	span := func(td *html.Node) int {
		if v, err := strconv.Atoi(attr(td, "colspan")); err == nil && v > 1 {
			return min(v, 1000)
		}
		return 1
	}
	ncols := 0
	for _, r := range rows {
		c := 0
		for _, td := range r.cells {
			c += span(td)
		}
		ncols = max(ncols, c)
	}
	if ncols == 0 {
		return h, used
	}
	const gap = 2
	colW := make([]int, ncols)
	for _, r := range rows {
		c := 0
		for _, td := range r.cells {
			if span(td) == 1 {
				colW[c] = max(colW[c], l.measure(td, in))
			}
			c += span(td)
		}
	}
	// A spanning cell wider than its columns widens the last of them.
	for _, r := range rows {
		c := 0
		for _, td := range r.cells {
			if k := span(td); k > 1 {
				end := min(c+k, ncols)
				have := gap * (end - c - 1)
				for j := c; j < end; j++ {
					have += colW[j]
				}
				if need := l.measure(td, in); need > have {
					colW[end-1] += need - have
				}
			}
			c += span(td)
		}
	}
	sum := 0
	for _, cw := range colW {
		sum += cw
	}
	used = max(used, sum+gap*(ncols-1))
	if !l.measuring {
		for over := sum + gap*(ncols-1) - w; over > 0; over-- {
			wi := 0
			for j := range colW {
				if colW[j] > colW[wi] {
					wi = j
				}
			}
			if colW[wi] <= 1 {
				break
			}
			colW[wi]--
		}
	}
	total := gap * (ncols - 1)
	for _, cw := range colW {
		total += cw
	}
	for i, r := range rows {
		cx, rh, c := x, 0, 0
		rin := in
		if attr(r.tr, "data-state") == "selected" || attr(r.tr, "aria-selected") == "true" {
			rin.look |= lBold
		}
		for _, td := range r.cells {
			end := min(c+span(td), ncols)
			cw := gap * (end - c - 1)
			for j := c; j < end; j++ {
				cw += colW[j]
			}
			hh, _ := l.node(td, cx, y+h, cw, rin)
			rh = max(rh, hh)
			cx += cw + gap
			c = end
		}
		l.done(r.tr, rect{x, y + h, total, rh})
		h += rh
		if r.head && (i+1 == len(rows) || !rows[i+1].head) {
			for k := 0; k < total && l.cv != nil; k++ {
				l.draw(x+k, y+h, "─", 1, in.look|lRule, n)
			}
			h++
		}
	}
	return h, used
}

// piece is a piece of a line: a word, a space, a control, a break.
type piece struct {
	kind  uint8
	chars []char
	w, h  int
	look  look
	own   *html.Node // the innermost element it is of
	hard  bool       // a kept space, which a line may start with
	full    bool       // a field as wide as its line
	boxed   bool       // drawn as a 3-row rounded box (╭─╮ / │…│ / ╰─╯)
	tall    bool       // drawn as a 3-row filled block (lg filled buttons)
	outline bool       // 1-row outline button with muted border brackets
}

const (
	pWord uint8 = iota
	pSpace
	pAtom
	pBreak
)

// inline lays out text and inline elements in lines w wide, wrapping at
// spaces, and a word too long for a line wherever it must. fill makes a
// lone field as wide as the line (a flex item's).
func (l *layout) inline(nodes []*html.Node, stop *html.Node, x, y, w int, in inh, fill bool) (h, used int) {
	ps := l.c.bufs.pieces()
	defer func() { l.c.bufs.putPieces(ps) }()
	for _, n := range nodes {
		ps = l.pieces(ps, n, in, fill)
	}
	cx, cy, lineH := 0, 0, 1
	lineUsed := false
	newline := func() {
		cy += lineH
		cx, lineH, lineUsed = 0, 1, false
	}
	// cut ends a truncate's one line: … in its last cell, and nothing
	// after it.
	cut := false
	ellipsis := func(p piece) {
		at := max(min(cx, w-1), 0)
		l.paint([]char{{"…", 1}}, x+at, y+cy, p.look, p.own, stop)
		cx, cut, lineUsed = at+1, true, true
		used = max(used, cx)
	}
	for _, p := range ps {
		if cut {
			break
		}
		if in.nowrap && !l.measuring {
			switch {
			case p.kind == pBreak:
				continue
			case p.kind == pSpace && cx+1 > w,
				p.kind == pWord && cx+p.w > w,
				p.kind == pAtom && cx > 0 && cx+p.w > w:
				if p.kind == pWord {
					// As much of the word as leaves the last cell for …
					for _, ch := range p.chars {
						if cx+ch.w > w-1 {
							break
						}
						l.paint([]char{ch}, x+cx, y+cy, p.look, p.own, stop)
						cx += ch.w
					}
				}
				ellipsis(p)
				continue
			}
		}
		switch p.kind {
		case pBreak:
			newline()
			continue
		case pSpace:
			if cx == 0 && !p.hard {
				continue
			}
			if cx+1 > w {
				newline()
				if !p.hard {
					continue
				}
			}
			l.paint(p.chars, x+cx, y+cy, p.look, p.own, stop)
			cx++
		case pWord:
			if l.cv == nil && w >= infinite/2 {
				// Measured where nothing wraps and nothing is drawn:
				// the word's width, as its characters would add up.
				if len(p.chars) > 0 {
					cx += p.w
					lineUsed = true
					used = max(used, cx)
				}
				continue
			}
			if cx > 0 && cx+p.w > w {
				newline()
			}
			for _, ch := range p.chars {
				if cx > 0 && cx+ch.w > w {
					newline()
				}
				if ch.w > w {
					// A wide character in a box one cell wide: it would
					// run into the next box.
					ch = char{"…", 1}
				}
				l.paint([]char{ch}, x+cx, y+cy, p.look, p.own, stop)
				cx += ch.w
				lineUsed = true
				used = max(used, cx)
			}
			continue
		case pAtom:
			aw := p.w
			if p.full && !l.measuring {
				if w-cx < 8 && cx > 0 {
					newline()
				}
				aw = w - cx
			}
			if cx > 0 && cx+aw > w {
				newline()
			}
			aw = min(aw, w)
			l.paintAtom(p, x+cx, y+cy, aw, stop)
			cx += aw
			lineH = max(lineH, p.h)
		}
		lineUsed = true
		used = max(used, cx)
	}
	// A break that ends the text makes no line of its own.
	h = cy
	if lineUsed {
		h += lineH
	}
	return h, used
}

// paint draws characters at x, y as own's, and grows the boxes of own and
// its inline ancestors up to stop.
func (l *layout) paint(cs []char, x, y int, lk look, own, stop *html.Node) {
	if l.cv == nil {
		return
	}
	cx := x
	for _, ch := range cs {
		l.draw(cx, y, ch.s, ch.w, lk, own)
		cx += ch.w
	}
	r := rect{x, y, cx - x, 1}
	for n := own; n != nil && n != stop; n = n.Parent {
		l.ext(n, r)
	}
}

// pieces adds n's pieces to ps.
func (l *layout) pieces(ps []piece, n *html.Node, in inh, fill bool) []piece {
	switch n.Type {
	case html.TextNode:
		return l.textPieces(ps, n, in)
	case html.ElementNode:
	default:
		return ps
	}
	st := l.style(n)
	if n.Data == "svg" {
		inBadge := closest(n.Parent, func(a *html.Node) bool { return tokens(a, "class", "badge") }) != nil
		if inBadge || svgInGlyphHost(n) {
			if g := svgGlyph(n); g != "" {
				cs := charsOf(g)
				ps = append(ps, piece{kind: pWord, chars: cs, w: widthOf(cs), look: l.lookOf(n, in.look, st), own: n})
				if inBadge {
					ps = append(ps, piece{kind: pSpace, chars: []char{{" ", 1}}, w: 1, look: in.look, own: n})
				}
				return ps
			}
		}
	}
	if st.display == dNone {
		return ps
	}
	if n.DataAtom == atom.Br {
		return append(ps, piece{kind: pBreak, own: n})
	}
	if tokens(n, "class", "chart-legend-indicator") && n.FirstChild == nil {
		// The kit's swatch, drawn by its stylesheet, which reads its
		// figure.
		if p := n.Parent; p != nil && p.Parent != nil && p.Parent.Parent != nil {
			l.dependOn(p.Parent.Parent)
		}
		return append(ps, piece{kind: pWord, chars: charsOf(legendMark(n)), w: 1, look: l.lookOf(n, in.look, st), own: n})
	}
	if st.display == dAtom {
		p := l.atom(n, st, in)
		if !isKitTextarea(n) && !isTUICombobox(n) {
			p.full = p.full || fill && isField(n)
		}
		if p.w > 0 || p.full {
			ps = append(ps, p)
		}
		return ps
	}
	in.look = l.lookOf(n, in.look, st)
	in.pre = in.pre || st.pre
	in.nowrap = in.nowrap || st.truncate
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		ps = l.pieces(ps, k, in, false)
	}
	return ps
}

// textPieces are a text's words and spaces: white space collapsed to one
// space, or kept with pre, its line breaks breaks.
func textPieces(ps []piece, s string, in inh, own *html.Node) []piece {
	space := func() piece {
		return piece{kind: pSpace, chars: []char{{" ", 1}}, w: 1, look: in.look, own: own, hard: in.pre}
	}
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		switch {
		case in.pre && (r == '\n' || r == '\r'):
			if r == '\r' && strings.HasPrefix(s[1:], "\n") {
				size++
			}
			ps = append(ps, piece{kind: pBreak, own: own})
			s = s[size:]
		case in.pre && r == '\t':
			for range 4 {
				ps = append(ps, space())
			}
			s = s[size:]
		case unicode.IsSpace(r) && r != '\u00a0':
			if in.pre {
				ps = append(ps, space())
			} else if len(ps) == 0 || ps[len(ps)-1].kind != pSpace {
				ps = append(ps, space())
			}
			s = s[size:]
		default:
			end := strings.IndexFunc(s, func(r rune) bool { return unicode.IsSpace(r) && r != '\u00a0' })
			if end < 0 {
				end = len(s)
			}
			cs := charsOf(s[:end])
			ps = append(ps, piece{kind: pWord, chars: cs, w: widthOf(cs), look: in.look, own: own})
			s = s[end:]
		}
	}
	return ps
}

// atom is a control's piece: a button as [ label ], a box as [x] or (•), a
// field as wide as its size, a select as its option, an image as its alt.
func (l *layout) atom(n *html.Node, st box, in inh) piece {
	lk := l.lookOf(n, in.look, st)
	p := piece{kind: pAtom, h: 1, look: lk, own: n, full: st.full}
	text := func(s string) piece {
		p.chars = charsOf(s)
		p.w = widthOf(p.chars)
		p.full = false
		return p
	}
	typ := strings.ToLower(attr(n, "type"))
	switch {
	case n.DataAtom == atom.Img:
		if alt := attr(n, "alt"); alt != "" {
			p.look |= lDim
			return text("[" + alt + "]")
		}
		return text("")
	case n.DataAtom == atom.Button || n.DataAtom == atom.A ||
		n.DataAtom == atom.Input && (typ == "button" || typ == "submit" || typ == "reset" || typ == "image"):
		label := ""
		if n.DataAtom == atom.Input {
			label = attr(n, "value")
		} else {
			label = buttonLabel(n)
		}
		if label == "" {
			label = iconLabel(n)
		}
		if label == "" {
			label = "·"
		}
		switch attr(n, "role") {
		case "menuitem", "menuitemcheckbox", "menuitemradio", "option", "tab":
			return text(" " + label + " ")
		}
		hasPopup := false
		switch attr(n, "aria-haspopup") {
		case "", "false":
		default:
			hasPopup = true
		}
		if tokens(n, "class", "btn") {
			return btnPiece(n, p, label, hasPopup, text)
		}
		if hasPopup {
			// It opens a menu or a list: the kit's chevron.
			return text("[ " + label + " ▾ ]")
		}
		return text("[ " + label + " ]")
	case n.DataAtom == atom.Input && typ == "checkbox":
		if attr(n, "data-mode") == "tui" {
			if l.field(n).checked {
				return text("\U000f0c52") // mdi checkbox-outline
			}
			return text("\U000f0131") // mdi checkbox-blank-outline
		}
		if l.field(n).checked {
			return text("[x]")
		}
		return text("[ ]")
	case n.DataAtom == atom.Input && typ == "radio":
		if l.field(n).checked {
			return text("(•)")
		}
		return text("( )")
	case n.DataAtom == atom.Select:
		v := l.field(n).ed.Value
		label := ""
		for _, o := range options(n) {
			if o.value == v {
				label = o.label
				break
			}
		}
		return text("[" + label + " ▾]")
	case n.DataAtom == atom.Input && (typ == "range" || typ == "color" || typ == "file"):
		return text("[" + typ + "]")
	}
	// A text field.
	p.w = 20
	if v, err := strconv.Atoi(attr(n, "size")); err == nil && v > 0 {
		p.w = v
	}
	if n.DataAtom == atom.Textarea {
		if isKitTextarea(n) {
			p.h, p.w = 4, 36
			p.full = false
		} else {
			p.h = 2
		}
		if v, err := strconv.Atoi(attr(n, "rows")); err == nil && v > 0 {
			p.h = v
		}
		if v, err := strconv.Atoi(attr(n, "cols")); err == nil && v > 0 {
			p.w = v
			p.full = false
		}
		if f := l.c.fields[n]; f != nil {
			if f.rows > 0 {
				p.h = f.rows
			} else if attr(n, "data-autogrow") == "true" && f.ed.Value != "" {
				innerW := p.w
				if isKitTextarea(n) && p.w >= 4 {
					innerW = p.w - 2
				}
				f.ed.Width, f.ed.Rows = max(innerW, 1), p.h
				need := len(f.ed.Wrap())
				maxR := attrPosInt(n, "data-max-rows")
				if maxR == 0 {
					maxR = 12
				}
				if need > p.h {
					p.h = min(need, maxR)
				}
			}
			if f.cols > 0 {
				p.w = f.cols
				p.full = false
			}
		}
	}
	if isTUICombobox(n) {
		p.w = max(attrPosInt(n, "data-min-w"), 12) + 4 // " " + text + " ▾ "
		p.full = false
	}
	return p
}

// paintAtom draws a control's piece at x, y, w wide.
func (l *layout) paintAtom(p piece, x, y, w int, stop *html.Node) {
	if l.cv == nil {
		return
	}
	n := p.own
	if !isTextField(n) {
		if p.boxed && w >= 3 {
			l.paintBoxedAtom(p, x, y, w, stop)
			return
		}
		if p.tall && w >= 1 {
			l.paintTallAtom(p, x, y, w, stop)
			return
		}
		if p.outline && w >= 3 {
			l.paintOutlineAtom(p, x, y, w, stop)
			return
		}
		cs := p.chars
		if widthOf(cs) > w {
			cs = clip(cs, w)
		}
		l.paint(cs, x, y, p.look, n, stop)
		return
	}
	r := rect{x, y, w, p.h}
	l.rec(oField, n, r.x, r.y, r.w, r.h, 0)
	l.fields[n] = r
	for m := n; m != nil && m != stop; m = m.Parent {
		l.ext(m, r)
	}
	f := l.field(n)
	kitTA := isKitTextarea(n)
	tuiCB := isTUICombobox(n)
	base := p.look
	if !kitTA && !tuiCB {
		base |= lUnderline
	}
	for j := 0; j < p.h; j++ {
		for i := 0; i < w; i++ {
			l.draw(x+i, y+j, " ", 1, base, n)
		}
	}
	drawGrip := func() {
		if textareaResizable(n) && w >= 4 {
			l.draw(x+w-1, y+p.h-1, "◢", 1, p.look|lMuted, n)
		}
		if tuiCB && w >= 5 {
			l.draw(x+w-2, y, "▾", 1, p.look|lMuted, n)
		}
	}
	focused := n == l.c.focused && l.c.active
	padX := 0
	innerW := w
	if kitTA && w >= 4 {
		padX = 1
		innerW = w - 2
	} else if tuiCB && w >= 5 {
		padX = 1
		innerW = w - 4
	}
	if f.ed.Value == "" {
		if ph := attr(n, "placeholder"); ph != "" {
			if n.DataAtom == atom.Textarea {
				phEd := f.ed
				phEd.Value = ph
				phEd.Width, phEd.Rows = innerW, p.h
				phChars := graphemesOf(ph)
				phRows := phEd.Wrap()
				for j := 0; j < p.h && j < len(phRows); j++ {
					rw := phRows[j]
					cx := x + padX
					for gi := rw.Start; gi < rw.End; gi++ {
						for _, ch := range charsOf(phChars[gi]) {
							if cx+ch.w > x+padX+innerW {
								break
							}
							l.draw(cx, y+j, ch.s, ch.w, base|lMuted, n)
							cx += ch.w
						}
					}
				}
			} else {
				cs := clip(charsOf(ph), innerW)
				cx := x + padX
				for _, ch := range cs {
					l.draw(cx, y, ch.s, ch.w, base|lMuted, n)
					cx += ch.w
				}
			}
		}
		drawGrip()
		if focused {
			l.setCaret(x+padX, y)
		}
		return
	}
	chars := graphemesOf(f.ed.Value)
	selLo, selHi := f.ed.Selection()
	hasSel := focused && selHi > selLo
	if n.DataAtom == atom.Textarea {
		f.ed.Width, f.ed.Rows = innerW, p.h
		rows := f.ed.Wrap()
		if !f.scrolled {
			f.top = f.ed.Scroll(f.top)
		} else {
			f.top = max(0, min(f.top, max(len(rows)-p.h, 0)))
		}
		for j := 0; j < p.h && f.top+j < len(rows); j++ {
			rw := rows[f.top+j]
			cx := x + padX
			for gi := rw.Start; gi < rw.End; gi++ {
				g := chars[gi]
				lk := base
				if hasSel && gi >= selLo && gi < selHi {
					lk |= lSel
				}
				for _, ch := range charsOf(g) {
					if cx+ch.w > x+padX+innerW {
						break
					}
					l.draw(cx, y+j, ch.s, ch.w, lk, n)
					cx += ch.w
				}
			}
			// Show a 1-cell selection indicator for selected newlines (so
			// selected blank lines and line endings read clearly in TUI).
			if hasSel && rw.End < len(chars) && chars[rw.End] == "\n" && rw.End >= selLo && rw.End < selHi && cx < x+padX+innerW {
				l.draw(cx, y+j, " ", 1, base|lSel, n)
			}
		}
		if kitTA && w >= 4 && len(rows) > p.h {
			barH := p.h
			if textareaResizable(n) && barH > 1 {
				barH--
			}
			total := len(rows)
			last := total - p.h
			thumbLook := (p.look | lMuted) | lReverse
			if focused {
				thumbLook = p.look | lReverse
			}
			trackLook := p.look | lRule
			trackStart, trackH := 0, barH
			if barH >= 3 {
				upLook := p.look | lMuted
				if f.top == 0 {
					upLook = trackLook
				}
				downLook := p.look | lMuted
				if f.top >= last {
					downLook = trackLook
				}
				l.draw(x+w-1, y, "▴", 1, upLook, n)
				l.draw(x+w-1, y+barH-1, "▾", 1, downLook, n)
				trackStart, trackH = 1, barH-2
			}
			size := max(1, (trackH*p.h+total/2)/total)
			if size > trackH {
				size = trackH
			}
			at := 0
			if last > 0 && trackH > size {
				at = (f.top*(trackH-size) + last/2) / last
			}
			for j := range trackH {
				if j >= at && j < at+size {
					l.draw(x+w-1, y+trackStart+j, " ", 1, thumbLook, n)
				} else {
					l.draw(x+w-1, y+trackStart+j, "·", 1, trackLook, n)
				}
			}
		}
		drawGrip()
		if focused {
			row, col := f.ed.Cursor()
			if row >= f.top && row < f.top+p.h {
				l.setCaret(x+padX+col, y+row-f.top)
			}
		}
		return
	}
	// An input: one line, scrolled along to keep the caret in view.
	type cellChar struct {
		char
		sel bool
	}
	var cs []cellChar
	caretCol := 0
	for i, g := range chars {
		if i == f.ed.Caret {
			caretCol = len(cs)
		}
		isSel := hasSel && i >= selLo && i < selHi
		if f.ed.Password {
			cs = append(cs, cellChar{char{"•", 1}, isSel})
		} else {
			for _, ch := range charsOf(g) {
				cs = append(cs, cellChar{ch, isSel})
			}
		}
	}
	if f.ed.Caret >= len(chars) {
		caretCol = 0
		for _, ch := range cs {
			caretCol += ch.w
		}
	} else {
		// Recompute display width up to caretCol.
		w0 := 0
		for _, ch := range cs[:caretCol] {
			w0 += ch.w
		}
		caretCol = w0
	}
	if focused {
		if caretCol < f.off {
			f.off = caretCol
		}
		if caretCol >= f.off+innerW {
			f.off = caretCol - innerW + 1
		}
	} else {
		f.off = 0
	}
	// The first column shown is a character's first: an offset inside a
	// wide one moves past it.
	skip := 0
	for _, ch := range cs {
		if skip >= f.off {
			break
		}
		skip += ch.w
	}
	f.off = skip
	cx := x + padX
	for _, ch := range cs {
		if skip > 0 {
			skip -= ch.w
			continue
		}
		if cx+ch.w > x+padX+innerW {
			break
		}
		lk := base
		if ch.sel {
			lk |= lSel
		}
		l.draw(cx, y, ch.s, ch.w, lk, n)
		cx += ch.w
	}
	drawGrip()
	if focused {
		l.setCaret(x+padX+caretCol-f.off, y)
	}
}

// setCaret puts the focused field's caret at x, y.
func (l *layout) setCaret(x, y int) {
	l.rec(oCaret, nil, x, y, 0, 0, 0)
	l.caret, l.hasCaret = [2]int{x, y}, true
}

// clip is cs in w cells, ending in … where it was cut.
func clip(cs []char, w int) []char {
	if widthOf(cs) <= w {
		return cs
	}
	if w <= 0 {
		return nil
	}
	var out []char
	n := 0
	for _, c := range cs {
		if n+c.w > w-1 {
			break
		}
		out = append(out, c)
		n += c.w
	}
	return append(out, char{"…", 1})
}

// glyphs are the icons a terminal has a character for that reads the same.
var glyphs = map[string]string{
	"x": "×", "plus": "+", "minus": "−", "check": "✓", "ellipsis": "…",
	"ellipsis-vertical": "⋮", "search": "⌕", "menu": "≡",
	"arrow-left": "←", "arrow-right": "→", "arrow-up": "↑", "arrow-down": "↓",
	"chevron-up": "▴", "chevron-down": "▾", "chevron-left": "‹", "chevron-right": "›",
	"chevrons-left": "«", "chevrons-right": "»", "chevrons-up-down": "↕",
	"settings": "⚙", "settings-2": "⚙", "trash": "×", "trash-2": "×",
	"pencil": "✎", "play": "▶", "star": "★", "heart": "♥",
	"info": "ⓘ", "circle-help": "?", "circle-alert": "!", "circle-check-big": "✓", "triangle-alert": "⚠",
	"external-link": "↗", "trending-up": "↗", "trending-down": "↘", "refresh-cw": "↻", "download": "↓", "upload": "↑",
	"filter": "▽", "sun": "☀", "moon": "☾", "eye": "◉", "send": "➤",
	"inbox": "✉", "folder": "▤", "bell": "◈", "user": "◉",
}

func svgInGlyphHost(n *html.Node) bool {
	return closest(n.Parent, func(a *html.Node) bool {
		if tokens(a, "class", "badge") || has(a, "data-tooltip") {
			return true
		}
		if attr(a, "data-mode") == "tui" {
			return tokens(a, "class", "alert") || tokens(a, "class", "empty") || tokens(a, "class", "item")
		}
		return false
	}) != nil
}

// btnPiece formats a Basecoat .btn according to its data-variant and
// data-size: filled pills (" Label ") for primary, secondary and
// destructive; bracketed "[ Label ]" for outline; clean " Label " for
// ghost; underlined Label for link. Primary buttons are bold.
// paintBoxedAtom draws a 3-row rounded box (╭─╮ / │…│ / ╰─╯) for boxed
// controls such as outline buttons. Border cells take lMuted
// (muted-foreground role) so the outline reads clearly without overpowering
// the label; interior cells take the control's own look and colour roles.
func (l *layout) paintBoxedAtom(p piece, x, y, w int, stop *html.Node) {
	n := p.own
	border := p.look | lMuted
	l.draw(x, y, "╭", 1, border, n)
	l.draw(x, y+2, "╰", 1, border, n)
	for i := 1; i < w-1; i++ {
		l.draw(x+i, y, "─", 1, border, n)
		l.draw(x+i, y+1, " ", 1, p.look, n)
		l.draw(x+i, y+2, "─", 1, border, n)
	}
	l.draw(x+w-1, y, "╮", 1, border, n)
	l.draw(x+w-1, y+2, "╯", 1, border, n)
	l.draw(x, y+1, "│", 1, border, n)
	l.draw(x+w-1, y+1, "│", 1, border, n)
	inner := p.chars
	if widthOf(inner) > w-2 {
		inner = clip(inner, w-2)
	}
	cx := x + 1
	for _, ch := range inner {
		l.draw(cx, y+1, ch.s, ch.w, p.look, n)
		cx += ch.w
	}
	r := rect{x, y, w, 3}
	for m := n; m != nil && m != stop; m = m.Parent {
		l.ext(m, r)
	}
}

// paintTallAtom draws a 3-row filled block for lg filled buttons so all
// lg buttons share the same 3-row height as lg outline buttons.
func (l *layout) paintTallAtom(p piece, x, y, w int, stop *html.Node) {
	n := p.own
	for j := range 3 {
		for i := range w {
			l.draw(x+i, y+j, " ", 1, p.look, n)
		}
	}
	inner := p.chars
	if widthOf(inner) > w {
		inner = clip(inner, w)
	}
	cx := x
	for _, ch := range inner {
		l.draw(cx, y+1, ch.s, ch.w, p.look, n)
		cx += ch.w
	}
	r := rect{x, y, w, 3}
	for m := n; m != nil && m != stop; m = m.Parent {
		l.ext(m, r)
	}
}

// paintOutlineAtom draws a 1-row outline button with muted-foreground
// brackets so it matches the height of other 1-row button variants.
func (l *layout) paintOutlineAtom(p piece, x, y, w int, stop *html.Node) {
	n := p.own
	border := p.look | lMuted
	l.draw(x, y, "[", 1, border, n)
	for i := 1; i < w-1; i++ {
		l.draw(x+i, y, " ", 1, p.look, n)
	}
	inner := p.chars
	if widthOf(inner) > w-2 {
		inner = clip(inner, w-2)
	}
	cx := x + 1
	for _, ch := range inner {
		l.draw(cx, y, ch.s, ch.w, p.look, n)
		cx += ch.w
	}
	l.draw(x+w-1, y, "]", 1, border, n)
	r := rect{x, y, w, 1}
	for m := n; m != nil && m != stop; m = m.Parent {
		l.ext(m, r)
	}
}

// btnPiece formats a Basecoat .btn by variant and size:
//   - xs: compact (tight padding, normal weight), 1 row.
//   - sm: 1-space padding, normal weight, 1 row.
//   - default: 1-space padding, bold weight, 1 row.
//   - lg / icon-lg: 3 rows tall, 2-space padding, bold weight.
func btnPiece(n *html.Node, p piece, label string, hasPopup bool, text func(string) piece) piece {
	if minW := attrPosInt(n, "data-min-w"); minW > 0 {
		if w := displaywidth.String(label); w < minW {
			label += strings.Repeat(" ", minW-w)
		}
	}
	if hasPopup {
		label += " ▾"
	}
	sz := attr(n, "data-size")
	isXs := sz == "xs" || sz == "icon-xs"
	isSm := sz == "sm" || sz == "icon-sm"
	isLg := sz == "lg" || sz == "icon-lg"
	iconOnly := strings.HasPrefix(sz, "icon")

	if !isXs && !isSm {
		p.look |= lBold
	}
	pad := " "
	switch {
	case isXs:
		pad = ""
	case isLg:
		pad = "  "
	}

	switch attr(n, "data-variant") {
	case "", "primary":
		p.chars = charsOf(pad + label + pad)
		p.w = widthOf(p.chars)
		if isLg {
			p.tall = true
			p.h = 3
		}
	case "secondary", "destructive":
		p.chars = charsOf(pad + label + pad)
		p.w = widthOf(p.chars)
		if isLg {
			p.tall = true
			p.h = 3
		}
	case "outline":
		if isLg {
			p.boxed = true
			p.h = 3
			innerPad := "  "
			if iconOnly {
				innerPad = " "
			}
			p.chars = charsOf(innerPad + label + innerPad)
			p.w = widthOf(p.chars) + 2
		} else {
			p.outline = true
			if iconOnly && (isXs || isSm) {
				p.chars = charsOf(label)
			} else {
				p.chars = charsOf(pad + label + pad)
			}
			p.w = widthOf(p.chars) + 2
		}
	case "link":
		p.chars = charsOf(label)
		p.w = widthOf(p.chars)
	default: // ghost
		p.chars = charsOf(pad + label + pad)
		p.w = widthOf(p.chars)
	}
	p.full = false
	return p
}

// buttonLabel collects a button's text and any Lucide icon glyphs in
// document order so icons beside labels (e.g. "+ New") stay visible in
// cells. Returns "" if neither text nor a known icon glyph is present.
func buttonLabel(n *html.Node) string {
	var parts []string
	var collect func(*html.Node)
	collect = func(cur *html.Node) {
		switch cur.Type {
		case html.TextNode:
			if s := strings.Join(strings.Fields(cur.Data), " "); s != "" {
				parts = append(parts, s)
			}
			return
		case html.ElementNode:
			if cur != n && (has(cur, "hidden") || tokens(cur, "class", "sr-only") || tokens(cur, "class", "hidden")) {
				return
			}
			if cur.Data == "svg" {
				if g := svgGlyph(cur); g != "" {
					parts = append(parts, g)
				}
				return
			}
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(n)
	return strings.Join(parts, " ")
}

func svgGlyph(n *html.Node) string {
	for _, cl := range strings.Fields(attr(n, "class")) {
		if g, ok := glyphs[strings.TrimPrefix(cl, "lucide-")]; ok && cl != "lucide" {
			return g
		}
	}
	return ""
}

// iconLabel is the label of a control with no text, an icon's button: the
// icon's glyph if a terminal has one, else the first word of its
// aria-label or title (a row's [ Delete ], not [ Delete Water the plants ]
// on a line of its own); "" for none.
func iconLabel(n *html.Node) string {
	var icon string
	walk(n, func(c *html.Node) {
		if icon == "" && c.Data == "svg" {
			icon = svgGlyph(c)
		}
	})
	if icon != "" {
		return icon
	}
	for _, k := range []string{"aria-label", "title"} {
		if f := strings.Fields(attr(n, k)); len(f) > 0 {
			return f[0]
		}
	}
	return ""
}

// style is styleOf(n), hovered or not, once a pass.
func (l *layout) style(n *html.Node) box {
	m := l.memoOf(n)
	if m.hasSt {
		return m.st
	}
	st := styleOf(n, l.c.hoverOn(n))
	if n.Type == html.ElementNode {
		st.fx = factsOf(n)
	}
	m.st, m.hasSt = st, true
	return st
}

// unmutes says whether an element's own text colour fg ends the muted
// text around it (lookOf).
func unmutes(fg Role) bool {
	return fg != "" && fg != "muted-foreground"
}

// colour is coloursOf(n), hovered or not, once a pass.
func (l *layout) colour(n *html.Node) (fg, bg Role) {
	m := l.memoOf(n)
	if m.hasCol {
		return m.fg, m.bg
	}
	fg, bg = coloursOf(n, l.c.hoverOn(n))
	m.fg, m.bg, m.hasCol = fg, bg, true
	return fg, bg
}

// textPass is a text's pieces under what it inherits.
type textPass struct {
	in inh
	ps []piece
}

// textPieces adds a text's pieces to ps, as textPieces does, making them
// once for each look it is laid out in. Only its first white space depends
// on ps: collapsed into a space ps ends with.
func (l *layout) textPieces(ps []piece, n *html.Node, in inh) []piece {
	var t []piece
	found := false
	m := l.memoOf(n)
	for _, e := range m.texts {
		if e.in == in {
			t, found = e.ps, true
			break
		}
	}
	if !found {
		t = textPieces(nil, n.Data, in, n.Parent)
		m.texts = append(m.texts, textPass{in, t})
	}
	if !in.pre && len(t) > 0 && t[0].kind == pSpace && len(ps) > 0 && ps[len(ps)-1].kind == pSpace {
		t = t[1:]
	}
	return append(ps, t...)
}

// kids are kids(n), once a pass.
func (l *layout) kids(n *html.Node) []*html.Node {
	if m := l.memoOf(n); m.hasKids {
		return m.kids
	}
	ks := kids(n, l.style(n))
	m := l.memoOf(n)
	m.kids, m.hasKids = ks, true
	return ks
}

// kept is what a layout pass knows of the tree that holds until it
// changes (settle), node by node. A text's widths are its pass's (gen).
// focus is what had the keyboard, which the laids draw (settle).
type kept struct {
	nodes map[*html.Node]*nodeMemo
	gen   uint64
	focus focusState
	// boxes is how many the last pass drew, and rows how many rows its
	// canvas had.
	boxes, rows int
	// frames are the last pass's, free, which the next records in.
	frames []*frame
	// canvas is the last layout's canvas, and used the chunks of the
	// one before, which the next pass's rows take: no layout two
	// layouts old is read (a caller holds the layout it asked for only
	// until it asks again). A pass laid out again (adjusted) is no
	// layout: the pass after takes its chunks.
	canvas *canvas
	used   [][]cell
	// recoloured are the elements whose colours were forgotten since the
	// last snapshot shown (paint), and recolourGen counts the forgetting.
	recoloured  map[*html.Node]struct{}
	recolourGen uint64
}

// focusState is what the layout reads of the keyboard: whether the
// Screen has it, the focused element, and the item it names its active
// one (activeItem).
type focusState struct {
	active        bool
	focused, item *html.Node
}

// settle forgets the laids that l's pass can't use: those of what had
// the keyboard or has it now, which draw it.
func (k *kept) settle(l *layout) {
	c := l.c
	fs := focusState{active: c.active, focused: c.focused}
	if c.body != nil {
		fs.item = l.activeItem()
	}
	if fs == k.focus {
		return
	}
	for _, n := range [...]*html.Node{k.focus.focused, k.focus.item, fs.focused, fs.item} {
		if n != nil {
			k.unlay(n)
		}
	}
	k.focus = fs
}

// nodeMemo is what a pass knows of a node: an element's style, colours
// and children, a text's pieces in each look it is laid out in (as
// textPieces makes them after no pieces), and its widths as of wGen; an
// element's laids (laid), and ver, which counts the changes under it
// (unlay).
type nodeMemo struct {
	st      box
	hasSt   bool
	fg, bg  Role
	hasCol  bool
	kids    []*html.Node
	hasKids bool
	texts   []textPass
	w       widths
	wGen    uint64
	// inMuted says a layout drew the element where the text around it
	// is muted, so that a colour of its own changes its look (lookOf).
	// It is kept until the styles are forgotten: a look that comes to
	// be muted lays the element out again, which sets it.
	inMuted bool
	laid    []*laid
	ver     uint64
	// roles are the element's roles in Cells (rolesAt): its colours, or
	// the nearest ancestor's that has them.
	roles    [2]Role
	hasRoles bool
}

// unlay forgets the laids of n and of the elements round it: something
// under them changed.
func (k *kept) unlay(n *html.Node) {
	for a := n; a != nil; a = a.Parent {
		if m := k.nodes[a]; m != nil {
			m.laid = nil
			m.ver++
		}
	}
}

// unlayAll forgets the laids of n, everything under it, and the elements
// round it.
func (k *kept) unlayAll(n *html.Node) {
	walk(n, func(d *html.Node) {
		if m := k.nodes[d]; m != nil {
			m.laid = nil
			m.ver++
		}
	})
	if n.Parent != nil {
		k.unlay(n.Parent)
	}
}

// forgetStyles forgets every node's style and children, and the layouts
// they make: an attribute changed, which any element's style may read
// (kit, filtered, barLabels).
func (k *kept) forgetStyles() {
	for _, m := range k.nodes {
		m.hasSt, m.hasKids = false, false
		m.kids = nil
		m.inMuted = false
		m.laid = nil
		m.ver++
	}
	k.gen++
}

// forgetStylesAround forgets the style and children of every node whose
// style may read n's attributes, as forgetStyles does for all: n's
// ancestors (which read their descendants', hasShown and kids), and
// everything under n's parent (n's descendants read their ancestors',
// closest; its children their parent's, kitGrows; its siblings their
// parent's children, barLabels; a legend's marks its figure's children,
// legendMark). Their layouts go with them.
func (k *kept) forgetStylesAround(n *html.Node) {
	forget := func(d *html.Node) {
		if m := k.nodes[d]; m != nil {
			m.hasSt, m.hasKids = false, false
			m.kids = nil
			m.inMuted = false
			m.laid = nil
			m.ver++
		}
	}
	top := n
	if n.Parent != nil {
		top = n.Parent
	}
	walk(top, forget)
	for a := top.Parent; a != nil; a = a.Parent {
		forget(a)
	}
	k.gen++
}

// forgetLooks forgets the colours of n and its descendants, the pieces of
// their texts, and the layouts they are in.
func (k *kept) forgetLooks(n *html.Node) {
	k.recolour()
	walk(n, func(d *html.Node) {
		if m := k.nodes[d]; m != nil {
			m.hasCol, m.hasRoles = false, false
			m.texts = nil
			// Its callers forget the styles round n too, which drops these
			// laids already; dropped here as well, so that forgetLooks alone
			// is whole.
			m.laid = nil
			m.ver++
			k.recoloured[d] = struct{}{}
		}
	})
	if n.Parent != nil {
		k.unlay(n.Parent)
	}
}

// forgetColours forgets the colours of n and its descendants. Their
// texts' pieces hold: they are kept by the look they were made in. So do
// their layouts, which read no colour but where a look changes
// (attrsChanged).
func (k *kept) forgetColours(n *html.Node) {
	k.recolour()
	walk(n, func(d *html.Node) {
		if m := k.nodes[d]; m != nil {
			m.hasCol, m.hasRoles = false, false
			k.recoloured[d] = struct{}{}
		}
	})
}

// recolour counts a forgetting of colours, and makes the set of the
// elements it reaches.
func (k *kept) recolour() {
	k.recolourGen++
	if k.recoloured == nil {
		k.recoloured = map[*html.Node]struct{}{}
	}
}

// forgetText forgets a text's pieces, and the layouts it is in.
func (k *kept) forgetText(n *html.Node) {
	if m := k.nodes[n]; m != nil {
		m.texts = nil
	}
	k.unlay(n)
}

// item is a flex row's item: its element or text, its width, whether
// it grows, and whether it is made no narrower (shrink-0).
type item struct {
	n     *html.Node
	w     int
	grow  bool
	fixed bool
}

// scratch are slices a layout pass uses and gives back, taken and given
// back in turn as the calls that use them nest: pieces for a run of
// inline content, a row's items and its gaps.
type scratch struct {
	ps  [][]piece
	its [][]item
	is  [][]int
}

func (s *scratch) pieces() []piece {
	if n := len(s.ps); n > 0 {
		b := s.ps[n-1]
		s.ps = s.ps[:n-1]
		return b[:0]
	}
	return nil
}

func (s *scratch) putPieces(b []piece) {
	for i := range b {
		b[i] = piece{} // nothing of this pass held for the next
	}
	s.ps = append(s.ps, b[:0])
}

func (s *scratch) items() []item {
	if n := len(s.its); n > 0 {
		b := s.its[n-1]
		s.its = s.its[:n-1]
		return b[:0]
	}
	return nil
}

func (s *scratch) putItems(b []item) {
	for i := range b {
		b[i] = item{}
	}
	s.its = append(s.its, b[:0])
}

// ints is n zeros.
func (s *scratch) ints(n int) []int {
	if k := len(s.is); k > 0 {
		b := s.is[k-1]
		s.is = s.is[:k-1]
		if cap(b) >= n {
			b = b[:n]
			for i := range b {
				b[i] = 0
			}
			return b
		}
	}
	return make([]int, n)
}

func (s *scratch) putInts(b []int) { s.is = append(s.is, b) }
