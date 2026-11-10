package surfui

import "neuroplast.io/hotty-surf/pkg/cellblock"

// Cells gives n its own drawing in the cells (B2): for an element whose
// look on a host is the host's own (a custom element's, a graphic's),
// which the cells can't draw from its tags and classes. f makes the
// block n draws at the width the layout gives it (see cellblock.Func).
// The layout gives n its width as it would a Stack's, and takes the
// block's height. f is pure, a function of what it closes over and of w
// and the style, so that the cells keep a block by n and the width and
// draw it again from there, as the diff skips a cached subtree. A host
// ignores it; its tree is as n's tags and children make it.
//
// n's children stay nodes in the tree: their ids, their focus and Tab
// order, their keys and events are as everywhere. The block only says
// where in it they sit (cellblock.Place), and a child it doesn't place
// isn't drawn in the cells. The types are cellblock's, a package of
// types only, because pkg/cells, which imports surfui, re-exports them
// (cells.Block) and surfui can't import it back.
func (n *Node) Cells(f cellblock.Func) *Node { n.block = f; return n }

// BlockOf is the func n draws its cells by (Cells), nil for none.
func BlockOf(n *Node) cellblock.Func { return n.block }

// Graphic is a drawing both ways: svg on a host (a Raw span in the page
// as it is, which the framework never rewrites), and in the cells the
// block cells makes at the width the layout gives, the cells' picture
// of it: a chart's bars, a QR code's blocks, big text. cells is pure, as
// Cells' func is, and takes the width alone: a graphic has no ink or
// density of its own to look at.
func Graphic(svg string, cells func(w int) cellblock.Block) *Node {
	n := Raw(svg).Class("u-graphic")
	return n.Cells(func(w int, _ cellblock.Style) cellblock.Block { return cells(w) })
}
