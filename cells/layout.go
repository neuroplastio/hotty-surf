package cells

import (
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
	memo      map[memoKey]int
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

type memoKey struct {
	n   *html.Node
	pre bool
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
// anything changed.
func (c *Screen) layout() *layout {
	if c.lay != nil {
		return c.lay
	}
	l := &layout{
		c:       c,
		cv:      &canvas{w: max(c.cols, 1)},
		boxes:   map[*html.Node]rect{},
		fields:  map[*html.Node]rect{},
		scrolls: map[*html.Node]scrollBox{},
		memo:    map[memoKey]int{},
	}
	if c.body != nil {
		l.height, _ = l.node(c.body, 0, 0, max(c.cols, 1), inh{})
		l.drawOverlays()
	}
	if l.adjusted {
		c.lay = nil
		return c.layout()
	}
	c.lay = l
	if t := c.revealAfter; t != nil {
		where := c.revealWhere
		c.revealAfter, c.revealWhere = nil, ""
		c.reveal(t, where)
		if c.lay == nil {
			return c.layout()
		}
		l = c.lay
	}
	c.scroll = min(c.scroll, max(l.height-c.rows, 0))
	c.scroll = max(c.scroll, 0)
	return l
}

// measure is the width n's content takes when nothing wraps.
func (l *layout) measure(n *html.Node, in inh) int {
	k := memoKey{n, in.pre}
	if w, ok := l.memo[k]; ok {
		return w
	}
	m := &layout{c: l.c, measuring: true, memo: l.memo}
	_, w := m.node(n, 0, 0, infinite, in)
	l.memo[k] = w
	return w
}

func (l *layout) ext(n *html.Node, r rect) {
	if l.cv == nil {
		return
	}
	l.boxes[n] = l.boxes[n].union(r)
}

// lookOf is the look of n's content: what it inherits, what its style
// adds, reverse while it has the keyboard, dim while disabled.
func (l *layout) lookOf(n *html.Node, in look, st box) look {
	lk := in | st.look
	if fg, _ := coloursOf(n); fg != "" && fg != "muted-foreground" {
		// A colour of its own: no longer the muted text around it.
		lk &^= lMuted
	}
	if st.look&lBold == 0 && in&lBold != 0 && strings.Contains(" "+attr(n, "class")+" ", " font-normal ") {
		lk &^= lBold
	}
	if l.c.active {
		// The keyboard's: the focused element, or the item it names its
		// active one (a menu's, a list's), drawn in its place.
		ad := l.activeItem()
		if n == ad || n == l.c.focused && ad == nil {
			lk |= lReverse
		}
	}
	if disabled(n) || attr(n, "aria-disabled") == "true" {
		lk |= lDim
	}
	if attr(n, "role") == "tab" && attr(n, "aria-selected") == "true" {
		// The chosen tab, among tabs drawn alike.
		lk |= lBold | lUnderline
	}
	if attr(n, "role") == "heading" && closest(n, func(a *html.Node) bool { return attr(a, "role") == "group" }) != nil {
		// A group's label in a menu or a list: not one of its items.
		lk |= lBold
	}
	return lk
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
		if styleOf(a).display == dNone {
			return nil
		}
	}
	l.active = t
	return t
}

// node lays n out as a block at x, y, w wide, and says how high it is and
// how wide its content is.
func (l *layout) node(n *html.Node, x, y, w int, in inh) (h, used int) {
	st := styleOf(n)
	if n.Type != html.ElementNode || st.display == dNone && n != l.c.body {
		return 0, 0
	}
	if n == l.c.body {
		st = box{}
	}
	// Its margins (mt-*, mb-*): rows above and below its box.
	h, used = l.box(n, st, x, y+st.marginT, w, in)
	return h + st.marginT + st.marginB, used
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
	if separator(n) == "vertical" {
		if l.cv != nil {
			l.cv.put(x, y, "│", 1, in.look|lRule, n)
		}
		l.done(n, rect{x, y, 1, 1})
		return 1, 1
	}
	if n.DataAtom == atom.Hr || separator(n) == "horizontal" {
		for i := 0; i < w && l.cv != nil; i++ {
			l.cv.put(x+i, y, "─", 1, in.look|lRule, n)
		}
		l.done(n, rect{x, y, w, 1})
		return 1, 1
	}
	cx, cy := x+st.padL, y+st.padT
	cw := max(w-st.padL-st.padR, 1)
	if m := marker(n); m != "" {
		mw := widthOf(charsOf(m))
		if l.cv != nil {
			px := cx
			for _, ch := range charsOf(m) {
				l.cv.put(px, cy, ch.s, ch.w, in.look, n)
				px += ch.w
			}
		}
		cx += mw
		cw = max(cw-mw, 1)
		used += mw
	}
	var ch, cu, lead, ew int
	if s, ok := empty(n); ok {
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
	if fh, fu, ok := l.figure(n, cx, cy, cw, in); ok {
		h = st.padT + fh + st.padB
		used += st.padL + fu + st.padR
		l.done(n, rect{x, y, w, h})
		return h, used
	}
	limit, bounded := scrollLimit(st.maxHeight, l.c.rows)
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
		cy -= offset
		if l.cv != nil {
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
			l.cv.clips = l.cv.clips[:len(l.cv.clips)-1]
			l.scrolls[n] = scrollBox{view: rect{cx, originalY, cw, min(ch, limit)}, content: ch}
			if maxOffset := max(ch-limit, 0); offset > maxOffset {
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
	if h == 0 && marker(n) != "" {
		h = 1
	}
	l.done(n, rect{x, y, w, h})
	return h, used
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
		return l.c.rows
	}
	if h, ok := l.heights[n]; ok {
		return h
	}
	h := 0
	switch st.height {
	case "":
	case "screen", "dvh", "svh", "lvh":
		h = l.c.rows
	case "full", "100%":
		if p := n.Parent; p != nil && p.Type == html.ElementNode {
			h = l.definite(p, styleOf(p))
		}
	default:
		h, _ = scrollLimit(st.height, l.c.rows)
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
	m := &layout{c: l.c, measuring: true, memo: l.memo}
	var run []*html.Node
	flush := func() {
		if len(run) > 0 {
			hh, _ := m.inline(run, n, 0, 0, w, in, false)
			used += hh
			items++
			run = nil
		}
	}
	for _, k := range kids(n) {
		switch k.Type {
		case html.TextNode:
			if strings.TrimSpace(k.Data) != "" {
				run = append(run, k)
				flush()
			}
		case html.ElementNode:
			st := styleOf(k)
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
	if l.heights == nil {
		l.heights = map[*html.Node]int{}
	}
	if l.grown == nil {
		l.grown = map[*html.Node]bool{}
	}
	for i, k := range growers {
		share := rest / (len(growers) - i)
		rest -= share
		l.heights[k], l.grown[k] = share, true
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
		m := &layout{c: l.c, measuring: true, memo: l.memo}
		h, pw := m.node(o.n, 0, 0, o.w, o.in)
		pw = max(min(pw, o.w), 1)
		for j := o.y; j < o.y+h; j++ {
			for i := o.x; i < o.x+pw; i++ {
				l.cv.put(i, j, " ", 1, 0, o.n)
			}
		}
		l.node(o.n, o.x, o.y, pw, o.in)
	}
}

// separator is the orientation of a separator with nothing in it
// (role=separator, Basecoat's and the kit's), "" for an element that is
// none: "horizontal" a rule row, "vertical" a bar a cell wide.
func separator(n *html.Node) string {
	if attr(n, "role") != "separator" || textOf(n) != "" {
		return ""
	}
	for _, k := range []string{"aria-orientation", "data-orientation"} {
		if attr(n, k) == "vertical" {
			return "vertical"
		}
	}
	return "horizontal"
}

// done records n's box, and makes its empty cells its own.
func (l *layout) done(n *html.Node, r rect) {
	if l.cv == nil {
		return
	}
	l.boxes[n] = r
	l.cv.own(r.x, r.y, r.w, r.h, n)
}

// marker is a list item's marker, where its list has one (list-disc,
// list-decimal: Tailwind's lists have none).
func marker(n *html.Node) string {
	switch attr(n, "role") {
	case "option":
		// A popup list's option: its chosen one checked (Basecoat's ✓).
		if closest(n, func(a *html.Node) bool { return has(a, "data-popover") }) != nil {
			if attr(n, "aria-selected") == "true" {
				return "✓ "
			}
			return "  "
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
	switch styleOf(n.Parent).list {
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
	for _, k := range kids(n) {
		switch k.Type {
		case html.TextNode:
			run = append(run, k)
		case html.ElementNode:
			st := styleOf(k)
			if st.display == dNone {
				continue
			}
			if !items && (st.display == dInline || st.display == dAtom) && !hasBlock(k) {
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
func hasBlock(n *html.Node) bool {
	if isAtom(n) {
		return false
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type != html.ElementNode {
			continue
		}
		switch st := styleOf(k); st.display {
		case dBlock, dFlex, dTable:
			return true
		case dInline:
			if hasBlock(k) {
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
	type item struct {
		n     *html.Node
		w     int
		grow  bool
		fixed bool // shrink-0: not made narrower
	}
	var items []item
	for _, k := range kids(n) {
		switch k.Type {
		case html.TextNode:
			if !in.pre && strings.TrimSpace(k.Data) == "" {
				continue
			}
			it := item{n: k}
			m := &layout{c: l.c, measuring: true, memo: l.memo}
			_, it.w = m.inline([]*html.Node{k}, n, 0, 0, infinite, in, false)
			items = append(items, it)
		case html.ElementNode:
			ks := styleOf(k)
			if ks.display == dNone {
				continue
			}
			items = append(items, item{n: k, w: l.measure(k, in), grow: ks.grow || ks.full && isTextField(k) || kitGrows(k), fixed: ks.noShrink})
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
	lead, between := 0, make([]int, len(items))
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
		if styleOf(tr).display == dNone {
			return
		}
		r := row{tr: tr, head: head}
		for k := tr.FirstChild; k != nil; k = k.NextSibling {
			if (k.DataAtom == atom.Td || k.DataAtom == atom.Th) && styleOf(k).display != dNone {
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
			if styleOf(k).display == dNone {
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
				l.cv.put(x+k, y+h, "─", 1, in.look|lRule, n)
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
	full  bool       // a field as wide as its line
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
	var ps []piece
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
		l.cv.put(cx, y, ch.s, ch.w, lk, own)
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
		return textPieces(ps, n.Data, in, n.Parent)
	case html.ElementNode:
	default:
		return ps
	}
	st := styleOf(n)
	if st.display == dNone {
		return ps
	}
	if n.DataAtom == atom.Br {
		return append(ps, piece{kind: pBreak, own: n})
	}
	if tokens(n, "class", "chart-legend-indicator") && n.FirstChild == nil {
		// The kit's swatch, drawn by its stylesheet.
		return append(ps, piece{kind: pWord, chars: charsOf(legendMark(n)), w: 1, look: l.lookOf(n, in.look, st), own: n})
	}
	if st.display == dAtom {
		p := l.atom(n, st, in)
		p.full = p.full || fill && isField(n)
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
		label := textOf(n)
		if n.DataAtom == atom.Input {
			label = attr(n, "value")
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
		switch attr(n, "aria-haspopup") {
		case "", "false":
		default:
			// It opens a menu or a list: the kit's chevron.
			return text("[ " + label + " ▾ ]")
		}
		return text("[ " + label + " ]")
	case n.DataAtom == atom.Input && typ == "checkbox":
		if l.c.field(n).checked {
			return text("[x]")
		}
		return text("[ ]")
	case n.DataAtom == atom.Input && typ == "radio":
		if l.c.field(n).checked {
			return text("(•)")
		}
		return text("( )")
	case n.DataAtom == atom.Select:
		v := l.c.field(n).ed.Value
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
		p.h = 2
		if v, err := strconv.Atoi(attr(n, "rows")); err == nil && v > 0 {
			p.h = v
		}
		if v, err := strconv.Atoi(attr(n, "cols")); err == nil && v > 0 {
			p.w = v
		}
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
		cs := p.chars
		if widthOf(cs) > w {
			cs = clip(cs, w)
		}
		l.paint(cs, x, y, p.look, n, stop)
		return
	}
	r := rect{x, y, w, p.h}
	l.fields[n] = r
	for m := n; m != nil && m != stop; m = m.Parent {
		l.ext(m, r)
	}
	f := l.c.field(n)
	base := p.look | lUnderline
	for j := 0; j < p.h; j++ {
		for i := 0; i < w; i++ {
			l.cv.put(x+i, y+j, " ", 1, base, n)
		}
	}
	focused := n == l.c.focused && l.c.active
	if f.ed.Value == "" {
		if ph := attr(n, "placeholder"); ph != "" {
			cs := clip(charsOf(ph), w)
			cx := x
			for _, ch := range cs {
				l.cv.put(cx, y, ch.s, ch.w, base|lMuted, n)
				cx += ch.w
			}
		}
		if focused {
			l.caret, l.hasCaret = [2]int{x, y}, true
		}
		return
	}
	chars := graphemesOf(f.ed.Value)
	if n.DataAtom == atom.Textarea {
		f.ed.Width, f.ed.Rows = w, p.h
		rows := f.ed.Wrap()
		f.top = f.ed.Scroll(f.top)
		for j := 0; j < p.h && f.top+j < len(rows); j++ {
			rw := rows[f.top+j]
			cx := x
			for _, g := range chars[rw.Start:rw.End] {
				for _, ch := range charsOf(g) {
					if cx+ch.w > x+w {
						break
					}
					l.cv.put(cx, y+j, ch.s, ch.w, base, n)
					cx += ch.w
				}
			}
		}
		if focused {
			row, col := f.ed.Cursor()
			l.caret, l.hasCaret = [2]int{x + col, y + row - f.top}, true
		}
		return
	}
	// An input: one line, scrolled along to keep the caret in view.
	var cs []char
	caretCol := 0
	for i, g := range chars {
		if i == f.ed.Caret {
			caretCol = widthOf(cs)
		}
		if f.ed.Password {
			cs = append(cs, char{"•", 1})
		} else {
			cs = append(cs, charsOf(g)...)
		}
	}
	if f.ed.Caret >= len(chars) {
		caretCol = widthOf(cs)
	}
	if focused {
		if caretCol < f.off {
			f.off = caretCol
		}
		if caretCol >= f.off+w {
			f.off = caretCol - w + 1
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
	cx := x
	for _, ch := range cs {
		if skip > 0 {
			skip -= ch.w
			continue
		}
		if cx+ch.w > x+w {
			break
		}
		l.cv.put(cx, y, ch.s, ch.w, base, n)
		cx += ch.w
	}
	if focused {
		l.caret, l.hasCaret = [2]int{x + caretCol - f.off, y}, true
	}
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
	"x": "×", "plus": "+", "check": "✓", "ellipsis": "…", "search": "⌕",
	"arrow-down": "↓", "arrow-up": "↑", "arrow-right": "→",
	"chevron-down": "▾", "chevron-left": "‹", "chevron-right": "›",
	"chevrons-left": "«", "chevrons-right": "»", "chevrons-up-down": "↕",
}

// iconLabel is the label of a control with no text, an icon's button: the
// icon's glyph if a terminal has one, else the first word of its
// aria-label or title (a row's [ Delete ], not [ Delete Water the plants ]
// on a line of its own); "" for none.
func iconLabel(n *html.Node) string {
	var icon string
	walk(n, func(c *html.Node) {
		if icon == "" && c.Data == "svg" {
			for _, cl := range strings.Fields(attr(c, "class")) {
				if g, ok := glyphs[strings.TrimPrefix(cl, "lucide-")]; ok && cl != "lucide" {
					icon = g
				}
			}
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
