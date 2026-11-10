package cells

import (
	"neuroplast.io/hotty-surf/pkg/cellblock"
	"neuroplast.io/hotty-surf/surfui"
)

// A custom element's own cells (surfui.Node.Cells, B2). Its box is a leaf
// to the layout: it takes the width it is given as any box does (a
// column's inner width, a row's natural or shared one), asks the block's
// func for the block at that width, and takes the block's height. The
// block's runs are drawn where the leaf's text would be; the children it
// places are laid out where it says, over the rows, and are the only
// children it has: an unplaced child isn't drawn and takes no press, but
// stays in the tree, which is where focus, Tab and keys are.

// The block's types are cellblock's, which surfui can import (see its
// doc); cells names them as it names a Screen's.
type (
	// Block is what a custom element draws: rows of Runs at most as
	// wide as it was given, the children it Places, the Hits it has, and
	// its Anchors.
	Block = cellblock.Block
	// Place is where a child of a custom element sits in its block.
	Place = cellblock.Place
	// BlockHit is a press in a block that acts on one of its element's
	// children.
	BlockHit = cellblock.Hit
	// Anchor is a place in a block a #fragment names.
	Anchor = cellblock.Anchor
	// Style is what a block may depend on beside its width.
	Style = cellblock.Style
	// BlockFunc makes a block at a width: Node.Cells' argument.
	BlockFunc = cellblock.Func
)

// Unbounded is the width a block is asked for to know its natural width.
const Unbounded = cellblock.Unbounded

// NewRun is a run of text from column col, in fg on bg ("" for the
// element's own colours) and attr, its clusters measured as the cells
// measure them.
func NewRun(col int, text string, fg, bg Role, attr Attr) Run {
	cs := clusters(text)
	return Run{Col: col, Width: width(cs), Text: joined(cs), Clusters: cs, Fg: fg, Bg: bg, Attr: attr}
}

// blockKey is what a block is kept by: the element, the width and the
// style, for the block is a function of those.
type blockKey struct {
	n  *surfui.Node
	w  int
	st Style
}

// blockCache keeps blocks across layouts: what a layout used is kept
// for the next, and the rest goes, so that it holds what is on the
// screen and no more. A cached render's nodes are the same nodes (the
// diff skips them), so their blocks are found; a node built fresh is
// made again.
type blockCache struct{ cur, prev map[blockKey]Block }

// next starts a layout: what the last one used is the previous.
func (c *blockCache) next() { c.prev, c.cur = c.cur, map[blockKey]Block{} }

// get is n's block at w and st, made by f if no layout has it.
func (c *blockCache) get(n *surfui.Node, f BlockFunc, w int, st Style) Block {
	k := blockKey{n, w, st}
	if b, ok := c.cur[k]; ok {
		return b
	}
	b, ok := c.prev[k]
	if !ok {
		b = f(w, st)
	}
	if c.cur == nil {
		c.cur = map[blockKey]Block{}
	}
	c.cur[k] = b
	return b
}

// custom makes b, whose element has a block func, a custom box: its
// children's boxes are slots, which place puts where the block says.
func (b *box) custom(f BlockFunc, d density, caps keyCaps, bc *blockCache) {
	b.kind, b.blk, b.bc = custom, f, bc
	for _, k := range surfui.Kids(b.n) {
		b.slots = append(b.slots, build(k, d, caps, bc))
	}
}

// block is the box's block at inner width w.
func (b *box) block(w int) Block {
	w = max(w, 1)
	if b.memoBW == w {
		return b.memoB
	}
	st := Style{Density: cellblock.Density(b.d)}
	var blk Block
	if b.bc != nil {
		blk = b.bc.get(b.n, b.blk, w, st)
	} else {
		blk = b.blk(w, st)
	}
	b.memoB, b.memoBW = blk, w // the layouts L4 keeps hold their boxes: this is cached with them
	return blk
}

// placeBlock puts a custom box's placed children where its block says,
// each in the rows it names, and no others.
func (b *box) placeBlock() {
	b.kids = b.kids[:0]
	blk := b.block(b.w - b.pl - b.pr)
	for _, p := range blk.Kids {
		if p.Kid < 0 || p.Kid >= len(b.slots) || p.W <= 0 || p.H <= 0 {
			continue
		}
		k := b.slots[p.Kid]
		k.place(b.x+b.pl+p.X, b.y+b.pt+p.Y, p.W, p.H)
		b.kids = append(b.kids, k)
	}
}

// block draws a custom box's rows in clip, then the presses its block
// has on its children. The runs' own colours win; where they have none
// the element's ink is theirs, and the ground below.
func (p *painter) block(b *box, r rect, clip rect, in ink) {
	x, y, w := r.x+b.pl, r.y+b.pt, r.w-b.pl-b.pr
	blk := b.block(w)
	for i, row := range blk.Rows {
		if i >= r.h-b.pt-b.pb {
			break
		}
		for _, run := range row {
			cs := run.Clusters
			if len(cs) == 0 {
				cs = clusters(run.Text)
			}
			st := style{fg: run.Fg, bg: run.Bg, attr: in.attr | run.Attr, ul: run.Ul}
			if st.fg == "" {
				st.fg = in.fg
			}
			p.write(x+run.Col, y+i, w-run.Col, cs, st, clip)
		}
	}
	view := rect{x, y, w, max(r.h-b.pt-b.pb, 0)}
	for _, h := range blk.Hits {
		if h.Kid < 0 || h.Kid >= len(b.slots) {
			continue
		}
		kn := b.slots[h.Kid].n
		if !acts(kn) {
			continue
		}
		vis := rect{x + h.X, y + h.Y, h.W, h.H}.cut(view).cut(clip)
		if !vis.empty() {
			p.r.rects[kn] = vis
			p.r.hits = append(p.r.hits, hit{r: vis, n: kn, list: p.list})
		}
	}
}

// walkSlots is walk that goes into a custom box's children whether or
// not its block places them (they are placed later, at layout).
func walkSlots(b *box, f func(*box) bool) {
	walk(b, func(k *box) bool {
		if !f(k) {
			return false
		}
		for _, s := range k.slots {
			walkSlots(s, f)
		}
		return true
	})
}
