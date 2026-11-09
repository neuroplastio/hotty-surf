package cells

import (
	"maps"

	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfui"
)

// The natural height, where an element is, and what is kept.
//
// Draw(cols, 0) lays the tree out to its full height: the screen it
// returns is as tall as the content, and every element's box is in
// document coordinates (before any scroll) over that height. Measure is
// that height alone. Box, BoxOf, Rows and RowsOf say where an element is
// in the last draw's layout, any element, not only what a press reaches
// (Where).
//
// A lookup is made by a match on the element, one place that finds
// them all: Box by id, BoxOf by the :key path. Finding by another
// attribute (a heading's anchor, A1) is one more match, not another walk.

// layoutKey is what a layout depends on besides the tree: the width, the
// height it was laid out for, whether the page scrolls, and how keys are
// capped.
type layoutKey struct {
	cols, rows int
	scrolls    bool
	mod        string
}

// sameStyle is whether a and b cap keys alike.
func sameStyle(a, b surfkeys.Style) bool {
	return a.OS == b.OS && a.Mod == b.Mod && a.Caps == b.Caps && a.Case == b.Case &&
		a.Sep == b.Sep && a.Then == b.Then && a.Glue == b.Glue && maps.Equal(a.Labels, b.Labels)
}

// laid is a tree laid out: its root, and whether the page's bar took a
// column.
type laid struct {
	root    *box
	pageBar bool
}

// layouts are the layouts of one tree, which stay while the program
// gives the same tree: a draw of an unchanged tree at a width it was
// drawn at is a lookup. The tree's identity is its root's pointer: a
// program that changes a tree gives another.
type layouts struct {
	tree  *surfui.Node
	style surfkeys.Style // the keys' caps they were laid out with
	lays  map[layoutKey]laid
	heigh map[layoutKey]int // natural heights, by width alone
	spare map[layoutKey]*box
}

// maxLayouts bounds what is kept for a tree: a page at many widths
// (a site's prerender has six) fits, a window dragged wide and narrow
// doesn't grow it.
const maxLayouts = 16

// of is the layouts for tree, empty where the tree is another's, or the
// keys are capped otherwise.
func (c *layouts) of(tree *surfui.Node, style surfkeys.Style) *layouts {
	if c.tree != tree || c.lays == nil || !sameStyle(c.style, style) {
		*c = layouts{tree: tree, style: style, lays: map[layoutKey]laid{}, heigh: map[layoutKey]int{}, spare: map[layoutKey]*box{}}
	}
	if len(c.lays) >= maxLayouts {
		clear(c.lays)
	}
	if len(c.heigh) >= maxLayouts {
		clear(c.heigh)
		clear(c.spare)
	}
	return c
}

func (r *Renderer) layoutKey(cols, rows int) layoutKey {
	return layoutKey{cols: cols, rows: rows, scrolls: r.scrolls(), mod: r.Platform.Mod}
}

// Measure is the natural height of the tree at cols cells: how many rows
// it takes whole, which Draw(cols, 0) draws. 0 where there is no tree.
// It changes nothing the renderer shows, and costs a lookup where the
// tree is the one it measured at this width.
func (r *Renderer) Measure(cols int) int {
	cols = max(cols, 1)
	if r.tree == nil {
		return 0
	}
	c := r.cache.of(r.tree, r.KeyStyle)
	k := r.layoutKey(cols, 0)
	if h, ok := c.heigh[k]; ok {
		return h
	}
	root := build(r.tree, densityOf(r.tree), r.keyCaps())
	h := root.height(cols)
	c.heigh[k], c.spare[k] = h, root
	return h
}

// Box is where the element id is in the last draw's layout, in document
// coordinates: x, y from the document's top left, w and h in cells,
// before any scroll and over the whole laid-out height (a Scroll box's
// content is laid out in its own, before its own scroll). It is
// the box that covers the element where a wrap split it over rows: Rows
// says which rows. ok is false for an id that is no element, or one not
// drawn (a list's options are, while open, in their own surface).
func (r *Renderer) Box(id string) (x, y, w, h int, ok bool) {
	b, _, _, ok := r.find(func(n *surfui.Node) bool { return surfui.IDOf(n) == id })
	return b.x, b.y, b.w, b.h, ok
}

// BoxOf is Box for the element a :key path leads to, from the root:
// BoxOf("comments", "c12", "body") is the element keyed body, in the one
// keyed c12, in the one keyed comments, wherever each is in the tree
// (elements without a key between them are passed). A caller drawing
// :each rows knows their keys, not their ids.
func (r *Renderer) BoxOf(keys ...string) (x, y, w, h int, ok bool) {
	n := r.byKeys(keys)
	if n == nil {
		return 0, 0, 0, 0, false
	}
	return r.Box(surfui.IDOf(n))
}

// Rows is the first and the last row, in document coordinates, of the
// element id: the same row for an element of one row, and two for one a
// wrap split over rows, whose Box covers them all, rows first to last.
func (r *Renderer) Rows(id string) (first, last int, ok bool) {
	b, _, _, ok := r.find(func(n *surfui.Node) bool { return surfui.IDOf(n) == id })
	return b.y, b.y + b.h - 1, ok
}

// RowsOf is Rows for the element a :key path leads to (BoxOf).
func (r *Renderer) RowsOf(keys ...string) (first, last int, ok bool) {
	n := r.byKeys(keys)
	if n == nil {
		return 0, 0, false
	}
	return r.Rows(surfui.IDOf(n))
}

// byKeys is the element at the :key path from the root, nil for none.
// Each key is looked for under the element before it, nearest first.
func (r *Renderer) byKeys(keys []string) *surfui.Node {
	n := r.tree
	if n == nil || len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		n = under(n, func(k *surfui.Node) bool { return surfui.KeyOf(k) == key })
		if n == nil {
			return nil
		}
	}
	return n
}

// under is the nearest element below n for which match holds, breadth
// first, nil for none.
func under(n *surfui.Node, match func(*surfui.Node) bool) *surfui.Node {
	level := surfui.Kids(n)
	for len(level) > 0 {
		var next []*surfui.Node
		for _, k := range level {
			if surfui.Tag(k) == "" {
				continue
			}
			if match(k) {
				return k
			}
			next = append(next, surfui.Kids(k)...)
		}
		level = next
	}
	return nil
}

// find is the box that covers the elements matching match in the
// layout, which every box of one covers: its own and those a wrap
// made of its runs. ok is false where none is laid out.
func (r *Renderer) find(match func(*surfui.Node) bool) (b box, first, last int, ok bool) {
	r.ensure()
	if r.root == nil {
		return b, 0, 0, false
	}
	x0, y0, x1, y1 := 0, 0, 0, 0
	walk(r.root, func(k *box) bool {
		if k.f&fGone != 0 || k.n == nil || surfui.Tag(k.n) == "" || !match(k.n) {
			return true
		}
		if !ok {
			x0, y0, x1, y1, ok = k.x, k.y, k.x+k.w, k.y+k.h, true
			return true
		}
		x0, y0, x1, y1 = min(x0, k.x), min(y0, k.y), max(x1, k.x+k.w), max(y1, k.y+k.h)
		return true
	})
	b.x, b.y, b.w, b.h = x0, y0, x1-x0, y1-y0
	return b, b.y, b.y + b.h - 1, ok
}
