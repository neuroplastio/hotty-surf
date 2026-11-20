package cells

import (
	"slices"

	"golang.org/x/net/html"
)

// What a pass laid out, kept for the next (laid): an element's layout is a
// function of its subtree and of what node is given (its width, what it
// inherits, its definite height and whether it grew), as long as the
// fields, the scrolled boxes and the focus it read are as they were. So a
// pass keeps, for each element it lays out, what that gave: its height
// and width, and what it drew, as the ops it did, in order and from its
// top left. A later pass that lays the element out with the same inputs,
// where nothing under it changed since (unlay), does those ops again
// where the element now is, through the same canvas, instead of laying it
// out: the cells, boxes and overlays are a fresh pass's, op for op. Only
// what changed is laid out, and the elements round it, whose places it
// moves.

// layMode is which of a pass's layouts lays an element out: the one that
// draws, a measuring one, or a measuring one that wants only widths.
type layMode uint8

const (
	mDrawn layMode = iota
	mMeasuring
	mWidthOnly
)

func (l *layout) mode() layMode {
	switch {
	case l.widthOnly:
		return mWidthOnly
	case l.measuring:
		return mMeasuring
	}
	return mDrawn
}

// laidKey is what node gives an element's layout from outside it. A
// drawn element's definite height (definite) is its parent's when it has
// h-full, or its column's share when it grew; a measuring one has none.
type laidKey struct {
	mode  layMode
	grown bool
	w     int
	def   int
	in    inh
}

// laid is an element's layout as a pass made it: its height and width
// (node's), the ops it drew from its top left, and what it read that its
// subtree doesn't hold: its fields as they were after (a field's
// state is changed by its layout, to its caret, and holds once it has
// been), the boxes' scroll it read, and the elements round it whose
// subtrees it read (depAt).
type laid struct {
	key     laidKey
	h, used int
	// rows is the Screen's height where the layout read it, 0 where it
	// didn't (screenRows).
	rows int
	ops  []lop
	// cells are the runs' (oRun).
	cells   []cell
	fields  []fieldAt
	scrolls []scrollAt
	deps    []depAt
}

type fieldAt struct {
	n *html.Node
	f field
}

type scrollAt struct {
	n   *html.Node
	off int
}

// depAt is an element whose subtree a layout read, beyond its own: it
// holds while nothing under it changed (its memo's ver).
type depAt struct {
	m   *nodeMemo
	ver uint64
}

// An op of a laid: what it did to the pass, from the element's top left.
const (
	oPut     uint8 = iota // canvas.put
	oDone                 // done: the box, and its empty cells
	oExt                  // ext: a box grown
	oClip                 // a scrolled box's clip, pushed
	oUnclip               // and popped
	oField                // a field's cells
	oScroll               // a scrolled box
	oCaret                // the focused field's caret
	oOverlay              // an open popover, drawn after the flow
	oAdjust               // a box scrolled past its end, scrolled back
	oHeight               // a column's growing item, given its rows
	oSub                  // a child's laid, done in turn
	oRun                  // puts of one cell each along a row: w of cells from v
)

type lop struct {
	n          *html.Node
	sub        *laid
	s          string
	x, y, w, h int32
	v          int32 // a scrolled box's content, an offset, a height
	in         inh
	look       look
	kind       uint8
}

// frame is a layout being recorded: where its element is, and what it
// did so far. Frames nest as node calls do, a pass's measuring layouts'
// with its own: what a measurement reads, the layout that asked for it
// reads too.
type frame struct {
	x, y    int
	drawn   bool
	rows    bool // it read the Screen's height
	ops     []lop
	fields  []*html.Node
	scrolls []scrollAt
	deps    []depAt
}

// pass is what a pass's layouts share: the frames being recorded,
// innermost last, spares, and its measuring layouts.
type pass struct {
	frames []*frame
	free   []*frame
	meas   *layout
	wide   *layout
}

// measurer is the pass's measuring layout, widthOnly or not.
func (l *layout) measurer(widthOnly bool) *layout {
	p := l.p
	m := &p.meas
	if widthOnly {
		m = &p.wide
	}
	if *m == nil {
		*m = &layout{c: l.c, measuring: true, widthOnly: widthOnly, nodes: l.nodes, gen: l.gen, p: p}
	}
	return *m
}

// top is the frame being recorded, nil for none.
func (l *layout) top() *frame {
	if n := len(l.p.frames); n > 0 {
		return l.p.frames[n-1]
	}
	return nil
}

// rec records an op in the frame being recorded, where it draws: kind
// for n at x, y, w wide and h high, with v, s and in.
func (l *layout) rec(kind uint8, n *html.Node, x, y, w, h, v int) {
	if f := l.top(); f != nil && f.drawn {
		f.ops = append(f.ops, lop{kind: kind, n: n, x: int32(x - f.x), y: int32(y - f.y), w: int32(w), h: int32(h), v: int32(v)})
	}
}

// recPut records a put (draw).
func (l *layout) recPut(x, y int, ch string, w int, lk look, own *html.Node) {
	if f := l.top(); f != nil && f.drawn {
		f.ops = append(f.ops, lop{kind: oPut, n: own, s: ch, look: lk, x: int32(x - f.x), y: int32(y - f.y), w: int32(w)})
	}
}

func (l *layout) push(x, y int, drawn bool) *frame {
	var f *frame
	if n := len(l.p.free); n > 0 {
		f = l.p.free[n-1]
		l.p.free = l.p.free[:n-1]
	} else {
		f = &frame{}
	}
	f.x, f.y, f.drawn = x, y, drawn
	l.p.frames = append(l.p.frames, f)
	return f
}

// pop ends f, the innermost frame, as a laid.
func (l *layout) pop(f *frame, key laidKey, h, used int) *laid {
	l.p.frames = l.p.frames[:len(l.p.frames)-1]
	r := &laid{key: key, h: h, used: used}
	if f.rows {
		r.rows = l.c.rows
	}
	if len(f.ops) > 0 {
		r.ops, r.cells = runs(f.ops)
	}
	if len(f.fields) > 0 {
		r.fields = make([]fieldAt, 0, len(f.fields))
		for i, n := range f.fields {
			if i > 0 && f.fields[i-1] == n {
				continue
			}
			if fs := l.c.fields[n]; fs != nil {
				r.fields = append(r.fields, fieldAt{n, *fs})
			}
		}
	}
	if len(f.scrolls) > 0 {
		r.scrolls = slices.Clone(f.scrolls)
	}
	if len(f.deps) > 0 {
		r.deps = slices.Clone(f.deps)
	}
	for i := range f.ops {
		f.ops[i] = lop{} // nothing of this pass held for the next
	}
	f.ops, f.fields, f.scrolls, f.deps = f.ops[:0], f.fields[:0], f.scrolls[:0], f.deps[:0]
	f.rows = false
	l.p.free = append(l.p.free, f)
	return r
}

// runs are ops with each run of puts of one cell, one after another along
// a row, made one op (oRun), and the runs' cells.
func runs(ops []lop) ([]lop, []cell) {
	// runEnd is where the run from i ends: i+1 for none.
	runEnd := func(i int) int {
		j := i + 1
		for j < len(ops) && extends(&ops[j-1], &ops[j]) {
			j++
		}
		return j
	}
	n, nc := 0, 0
	for i := 0; i < len(ops); n++ {
		j := runEnd(i)
		if j-i > 1 {
			nc += j - i
		}
		i = j
	}
	out := make([]lop, 0, n)
	var cells []cell
	if nc > 0 {
		cells = make([]cell, 0, nc)
	}
	for i := 0; i < len(ops); {
		j := runEnd(i)
		if j-i == 1 {
			out = append(out, ops[i])
			i++
			continue
		}
		out = append(out, lop{kind: oRun, x: ops[i].x, y: ops[i].y, w: int32(j - i), v: int32(len(cells))})
		for k := i; k < j; k++ {
			cells = append(cells, cell{s: ops[k].s, look: ops[k].look, own: ops[k].n})
		}
		i = j
	}
	return out, cells
}

// extends says whether put b draws the cell after put a's, both one cell
// of text.
func extends(a, b *lop) bool {
	return a.kind == oPut && b.kind == oPut && a.w == 1 && b.w == 1 && a.s != "" && b.s != "" &&
		a.y == b.y && b.x == a.x+1
}

// adopt tells the frame round r's element what r did, as if the element
// had been laid out inside it: its ops, where it draws, and what it read.
func (l *layout) adopt(r *laid, x, y int) {
	f := l.top()
	if f == nil {
		return
	}
	if f.drawn && r.key.mode == mDrawn {
		f.ops = append(f.ops, lop{kind: oSub, sub: r, x: int32(x - f.x), y: int32(y - f.y)})
	}
	for _, fa := range r.fields {
		f.fields = append(f.fields, fa.n)
	}
	f.scrolls = append(f.scrolls, r.scrolls...)
	f.deps = append(f.deps, r.deps...)
	f.rows = f.rows || r.rows != 0
}

// field is the Screen's field(n), read by the layout being recorded.
func (l *layout) field(n *html.Node) *field {
	if f := l.top(); f != nil {
		f.fields = append(f.fields, n)
	}
	return l.c.field(n)
}

// dependOn says the layout being recorded read n's subtree, n being no
// element inside it.
func (l *layout) dependOn(n *html.Node) {
	if f := l.top(); f != nil {
		m := l.memoOf(n)
		f.deps = append(f.deps, depAt{m, m.ver})
	}
}

// laidOf is the laid of m's element with key that still holds, nil for
// none.
func (l *layout) laidOf(m *nodeMemo, key laidKey) *laid {
	for i, r := range m.laid {
		if r.key != key {
			continue
		}
		if l.holds(r) {
			// The last used is the last evicted (keep).
			if last := len(m.laid) - 1; i < last {
				copy(m.laid[i:], m.laid[i+1:])
				m.laid[last] = r
			}
			return r
		}
		m.laid = slices.Delete(m.laid, i, i+1)
		return nil
	}
	return nil
}

// holds says whether what r read from outside its subtree is as it was.
func (l *layout) holds(r *laid) bool {
	if r.rows != 0 && r.rows != l.c.rows {
		return false
	}
	for i := range r.fields {
		f := l.c.fields[r.fields[i].n]
		if f == nil || *f != r.fields[i].f {
			return false
		}
	}
	for _, s := range r.scrolls {
		if l.c.boxScroll[s.n] != s.off {
			return false
		}
	}
	for _, d := range r.deps {
		if d.m.ver != d.ver {
			return false
		}
	}
	return true
}

// maxLaid is how many laids an element keeps: enough for each way a pass
// lays it out (drawn, measured at a width, measured for its width), and a
// second width.
const maxLaid = 4

// keep keeps r as m's element's laid for its key, in place of the one
// used longest ago (laidOf) where it has maxLaid.
func keep(m *nodeMemo, r *laid) {
	for i, o := range m.laid {
		if o.key == r.key {
			m.laid[i] = r
			return
		}
	}
	if len(m.laid) >= maxLaid {
		m.laid = slices.Delete(m.laid, 0, 1)
	}
	m.laid = append(m.laid, r)
}

// replay does r's ops again with its element at x, y.
func (l *layout) replay(r *laid, x, y int) {
	if r.key.mode != mDrawn {
		return
	}
	for i := range r.ops {
		o := &r.ops[i]
		ox, oy, w, h := x+int(o.x), y+int(o.y), int(o.w), int(o.h)
		switch o.kind {
		case oPut:
			l.cv.put(ox, oy, o.s, w, o.look, o.n)
		case oDone:
			l.boxes[o.n] = rect{ox, oy, w, h}
			l.cv.own(ox, oy, w, h, o.n)
		case oExt:
			l.boxes[o.n] = l.boxes[o.n].union(rect{ox, oy, w, h})
		case oClip:
			l.cv.clips = append(l.cv.clips, rect{ox, oy, w, h})
		case oUnclip:
			l.cv.clips = l.cv.clips[:len(l.cv.clips)-1]
		case oField:
			l.fields[o.n] = rect{ox, oy, w, h}
		case oScroll:
			l.scrolls[o.n] = scrollBox{view: rect{ox, oy, w, h}, content: int(o.v)}
		case oCaret:
			l.caret, l.hasCaret = [2]int{ox, oy}, true
		case oOverlay:
			l.overlays = append(l.overlays, overlay{n: o.n, x: ox, y: oy, w: w, in: o.in})
		case oAdjust:
			// Not reached: a pass that adjusts is laid out again at once
			// (layout), where every laid on the box's path is looked up
			// with the same key and doesn't hold, the offset it read
			// (scrolls) being the one adjusted. Done as the pass did it,
			// all the same.
			l.c.boxScroll[o.n] = int(o.v)
			l.adjusted = true
		case oHeight:
			l.setHeight(o.n, int(o.v))
		case oSub:
			l.replay(o.sub, ox, oy)
		case oRun:
			l.cv.putRun(ox, oy, r.cells[o.v:o.v+o.w])
		}
	}
}
