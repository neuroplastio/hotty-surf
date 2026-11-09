package surfrun

import (
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfui"
)

// A tooltip is an element's (surfui.Node.Tip, a template's title= or
// <title>): the tree never draws it. A surface whose view has one asks
// its host for hover (Frame.Hover, hotty SPEC §9.4). The host keeps the
// dwell and the placement: once the pointer has rested on an element
// for about 500 ms it asks the surface for the tooltip there (Tip), and
// shows it in a surface of its own, above the view, until the pointer
// moves to another element, leaves, presses, or a key comes
// (docs/guide.md, "Tooltips"). The runtime (Runtime) shows none, as it
// shows no context menu, and asks for no hover.

// TipAt is where the pointer rests.
type TipAt struct {
	// ID is the element's id: a hover's target (hotty.Event.Target of a
	// hotty.EventHover), or, in the cells, the element at the pointer's
	// cell (cells.Renderer.At).
	ID string
}

// Tip is an element's tooltip as its host shows it: a small view of its
// own, in surf's look.
type Tip struct {
	// ID is the element's whose tooltip it is: at's, or the nearest one
	// around it with one. A host that shows a tooltip keeps it while the
	// pointer moves to an element whose Tip has the same ID.
	ID string
	// Tree is its surface's root (surfui.TipCard), resolved under
	// surfui.TipID: a card, its text small, holding what the element's
	// tooltip shows, at most surfui.TipCols wide and wrapped there.
	Tree *surfui.Node
	// Cols and Rows are the cells it takes on a host, the placement's c
	// and r: Cells' width, and its rows less one, since the card's room
	// above and below on a host is half a cell each, where the cells
	// draw a line.
	Cols, Rows int
	// Cells are what it draws in the cells: Cells.Lines, Cells.Cols by
	// Cells.Rows, a line round the card.
	Cells cells.Screen
}

// HTML is the tooltip's document on a host, with the host's stylesheets
// after surf's (surfui.Document): a host gives it one that keeps it from
// the pointer, html { pointer-events: none }, where it has passthrough.
func (t Tip) HTML(sheets ...string) string { return surfui.Document(t.Tree, sheets...) }

// Tip is the tooltip at at: that of the innermost element that has one,
// from the element at up through those around it. ok is false where none
// has one, or at names no element of the tree, as a hover from before a
// change may. Its keys are drawn in the surface's key style and its hues
// as the surface's palette has them.
//
// It is the tree the model has now (fresh), so that a host that asks
// again after an Update while the tooltip shows draws what changed.
func (s *Surface) Tip(at TipAt) (t Tip, ok bool) {
	s.fresh()
	for n := s.index[at.ID]; n != nil; n = surfui.Parent(n) {
		content := surfui.TipOf(n)
		if content == nil {
			continue
		}
		tree := surfui.TipCard(n, content)
		surfkeys.Expand(tree, surfkeys.OnPlatform(s.keyStyle(), s.Platform.Mod), true)
		surfui.Resolve(tree, surfui.TipID)
		scr := cells.DrawTip(tree, s.keyStyle(), s.Hues())
		return Tip{ID: surfui.IDOf(n), Tree: tree, Cols: scr.Cols, Rows: max(scr.Rows-1, 1), Cells: scr}, true
	}
	return Tip{}, false
}
