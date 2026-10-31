package cells

import (
	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Draw lays the tree out at cols×rows and draws it: the screen, with the
// open list over it, under its Select. It brings into view what the
// program revealed and what took the keyboard since the last draw.
func (r *Renderer) Draw(cols, rows int) Screen {
	cols, rows = max(cols, 1), max(rows, 1)
	r.cols, r.rows = cols, rows
	r.layout()
	r.keepInView()
	g := newGrid(cols, rows)
	r.hits, r.ports, r.rects = nil, nil, map[*surfui.Node]rect{}
	p := r.painter(g)
	screen := rect{0, 0, cols, rows}
	if r.root != nil {
		dy := r.offsets[""]
		p.paint(r.root, dy, screen, ink{fg: Fg})
		if r.pageBar {
			p.bar(cols-1, screen, screen, r.root.h, dy, false)
		}
		if r.scrolls() && r.root.h > rows {
			r.ports = append([]port{{id: "", view: screen, b: r.root}}, r.ports...)
		}
	}
	if o := r.over; o != nil {
		p.list = true
		p.focus = nil
		p.paint(o, 0, rect{o.x, o.y, o.w, o.h}.cut(screen), ink{fg: Fg})
	}
	s := Screen{Cols: cols, Rows: rows, Lines: g.lines(), Caret: p.caret}
	for _, h := range r.hits {
		s.Hits = append(s.Hits, Hit{X: h.r.x, Y: h.r.y, W: h.r.w, H: h.r.h, ID: surfui.IDOf(h.n)})
	}
	s.Mouse = len(s.Hits) > 0 || len(r.ports) > 0
	if !r.drawn || !s.same(r.screen) {
		r.gen++
	}
	r.drawn = true
	s.Gen = r.gen
	r.screen = s
	return s
}

// scrolls reports whether the document scrolls down.
// keyCaps draws a key's sequence in KeyStyle, as cells draw keys: text
// caps, unless KeyStyle says glyphs, labelled for the Platform's system
// where KeyStyle leaves it.
func (r *Renderer) keyCaps() keyCaps {
	st := surfkeys.OnPlatform(r.KeyStyle, r.Platform.Mod)
	return func(seq string) ([]string, string) { return surfkeys.Caps(st, false, seq), surfkeys.Between(st) }
}

func (r *Renderer) scrolls() bool { return r.Scroll&hotty.ScrollVertical != 0 }

// layout builds the boxes and places them: the root as tall as the
// screen at least (min-height: 100vh), the open list under its Select.
func (r *Renderer) layout() {
	r.root, r.over, r.pageBar = nil, nil, false
	if r.tree == nil {
		return
	}
	d := densityOf(r.tree)
	r.root = build(r.tree, d, r.keyCaps())
	w := r.cols
	h := r.root.height(w)
	if r.scrolls() && h > r.rows && w > 1 {
		// The page's bar takes its last column.
		r.pageBar = true
		w--
		r.root = build(r.tree, d, r.keyCaps())
		h = r.root.height(w)
	}
	r.root.place(0, 0, w, max(h, r.rows))
	if !r.scrolls() && r.endAtScreen() {
		walk(r.root, func(b *box) bool { b.memoW = -1; return true })
		r.root.place(0, 0, w, max(r.root.height(w), r.rows))
	}
	r.offsets[""] = r.clamp(r.offsets[""], r.root.h, r.rows)
	if r.list != nil && r.list.tree != nil {
		r.placeList(d)
	}
}

// endAtScreen ends each box grown around a scaled element past the
// screen's bottom at the bottom, where the document doesn't scroll: it
// holds as many rows as are left, as a Scroll box, its bar showing what
// is cut, so that nothing draws outside its border. It reports whether
// one ended.
func (r *Renderer) endAtScreen() bool {
	ended := false
	in := map[*box]bool{} // what a Scroll box holds, whose rows are its own
	walk(r.root, func(b *box) bool {
		if in[b] || b.f&fScroll != 0 {
			for _, k := range b.kids {
				in[k] = true
			}
			return true
		}
		if b.f&fHolds != 0 && b.fh >= 0 && b.h > b.fh && b.y+b.h > r.rows {
			b.fh = max(r.rows-b.y, b.fh)
			b.f = b.f&^fHolds | fScroll
			ended = true
			for _, k := range b.kids {
				in[k] = true
			}
		}
		return true
	})
	return ended
}

// placeList puts the open list under its Select, as wide as it at least,
// or above it where there is more room there; it scrolls where it
// doesn't fit, with option At in view.
func (r *Renderer) placeList(d density) {
	o := build(r.list.tree, densityOf(r.list.tree), r.keyCaps())
	// The list's surface holds the box of options: the box is the list.
	if len(o.kids) == 1 {
		o = o.kids[0]
	}
	w, h := o.natural(), o.height(o.natural())
	sel := r.selectBox()
	x, y := (r.cols-w)/2, (r.rows-h)/2
	if sel != nil {
		top := sel.y - r.offsets[""]
		w = max(w, sel.w)
		x = sel.x
		below, above := r.rows-(top+sel.h), top
		if h <= below || below >= above {
			y, h = top+sel.h, min(h, below)
		} else {
			h = min(h, above)
			y = top - h
		}
	}
	w = max(min(w, r.cols), 1)
	h = max(h, 1)
	x = max(min(x, r.cols-w), 0)
	y = max(min(y, r.rows-h), 0)
	if o.height(w) > h {
		o.f |= fScroll // too long for the room: it scrolls
	}
	o.place(x, y, w, h)
	r.over = o
}

// selectBox is the box of the Select whose list is open.
func (r *Renderer) selectBox() *box {
	var found *box
	walk(r.root, func(b *box) bool {
		if b.kind == choose {
			if v, _ := surfui.AttrOf(b.n, "aria-expanded"); v == "true" {
				found = b
				return false
			}
		}
		return true
	})
	return found
}

// walk calls f on each box under b, b first, in tree order, until f
// returns false; with no recursion, for TinyGo's stacks.
func walk(b *box, f func(*box) bool) {
	if b == nil {
		return
	}
	todo := []*box{b}
	for len(todo) > 0 {
		k := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if !f(k) {
			return
		}
		for i := len(k.kids) - 1; i >= 0; i-- {
			todo = append(todo, k.kids[i])
		}
	}
}

// boxOf is the box of the element id, nil where it has none.
func (r *Renderer) boxOf(id string) *box {
	var found *box
	walk(r.root, func(b *box) bool {
		if surfui.Tag(b.n) != "" && surfui.IDOf(b.n) == id {
			found = b
			return false
		}
		return true
	})
	return found
}

// densityOf is a surface's density, from its root's class.
func densityOf(n *surfui.Node) density {
	switch {
	case hasClass(n, "u-compact"):
		return compact
	case hasClass(n, "u-roomy"):
		return roomy
	}
	return dense
}

// clamp keeps a scroll offset in range: content rows in a view of view.
func (r *Renderer) clamp(off, content, view int) int {
	return max(min(off, content-view), 0)
}

// keepInView brings into view what the program revealed and what took
// the keyboard, by the least scroll that does, where the document
// scrolls; and keeps every Scroll box's offset in range.
func (r *Renderer) keepInView() {
	walk(r.root, func(b *box) bool {
		if b.f&fScroll != 0 {
			id := surfui.IDOf(b.n)
			r.offsets[id] = r.clamp(r.offsets[id], b.contentH, b.view(0).h)
		}
		return true
	})
	if r.scrolls() {
		ids := r.reveals
		if r.revealFocus && r.focused != "" {
			ids = append(ids, r.focused)
		}
		for _, id := range ids {
			if b := r.boxOf(id); b != nil {
				r.reveal(b)
			}
		}
	}
	r.reveals, r.revealFocus = nil, false
	if o := r.over; o != nil && o.f&fScroll != 0 && r.list.at >= 0 && r.list.at < len(o.kids) {
		// The option the keys are on, in view.
		id, v, k := surfui.IDOf(o.n), o.view(0), o.kids[r.list.at]
		r.offsets[id] = r.clamp(into(r.offsets[id], k.y-v.y, k.h, v.h), o.contentH, v.h)
	}
}

// reveal scrolls each Scroll box b is in, from the innermost, and the
// page, by the least that brings b into view, its top where it is taller
// than the view.
func (r *Renderer) reveal(b *box) {
	y, h := b.y, b.h
	for q := surfui.Parent(b.n); q != nil; q = surfui.Parent(q) {
		if !hasClass(q, "u-scroll") {
			continue
		}
		s := r.boxOf(surfui.IDOf(q))
		if s == nil {
			continue
		}
		v := s.view(0)
		id := surfui.IDOf(q)
		off := into(r.offsets[id], y-v.y, h, v.h)
		r.offsets[id] = r.clamp(off, s.contentH, v.h)
		// What shows of b, for the boxes around this one.
		y -= r.offsets[id]
		top, bottom := max(y, v.y), min(y+h, v.y+v.h)
		y, h = top, max(bottom-top, 0)
	}
	if r.root != nil {
		r.offsets[""] = r.clamp(into(r.offsets[""], y, h, r.rows), r.root.h, r.rows)
	}
}

// into is the offset nearest off that shows rows at..at+h of a view h
// rows tall.
func into(off, at, h, view int) int {
	switch {
	case at < off:
		return at
	case at+h > off+view:
		return min(at, at+h-view)
	}
	return off
}
