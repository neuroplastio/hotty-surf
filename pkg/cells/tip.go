package cells

import (
	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surfui"
)

// DrawTip draws a tooltip in cells: tree is its surface's root
// (surfui.TipCard, resolved under surfui.TipID). Its card has a line
// round it, as a Select's open list has, and is as wide as what it holds
// wants, up to surfui.TipCols, its text wrapped there, and as tall as
// that takes: the Screen's Cols and Rows. Its keys are drawn in style,
// and its hues in the terminal colours palette names, as a Renderer's
// are. A host in the cells draws its Lines where it draws a context
// menu: they act on no press (it has no Hits).
func DrawTip(tree *surfui.Node, style surfkeys.Style, palette surfui.Palette) Screen {
	r := New(nil)
	r.KeyStyle, r.Palette = style, palette
	r.tree = tree
	b := build(tree, densityOf(tree), r.keyCaps())
	if len(b.kids) == 1 {
		b = b.kids[0] // the surface holds the card: the card is the tip
	}
	w := max(min(b.natural(), surfui.TipCols), 1)
	return r.Draw(w, max(b.height(w), 1))
}
