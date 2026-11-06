package surfrun

import (
	"math"
	"slices"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// A menu has one look wherever it shows: an element's context menu
// (Surface.Menu, then Surface.MenuView) and one a host program opens at
// a point of its own, a pane's or a tab's (OpenMenu). Either is a
// MenuView: a small view in a surface of its own, which its host places
// at the point (MenuView.Place), shows above what it is of, and closes
// on a pick, Escape or a press outside it.

// MenuLook is what a menu a host program opens is drawn in: the theme
// and density of what it is of ("" for the defaults, neuroplastio and
// compact), the key style its keys are drawn in and the host's mod, and
// the host's cell, in CSS px, which its size in cells is taken in (0 for
// the 10×21 a host draws at by default).
type MenuLook struct {
	Theme, Density string
	KeyStyle       surfkeys.Style
	Mod            string
	CellW, CellH   float64
}

// MenuView is a menu as its host shows it.
type MenuView struct {
	// Tree is its surface's root (surfui.MenuCard), resolved under
	// surfui.MenuID: a raised card, a row for each item.
	Tree *surfui.Node
	// Items are its items, in order: a click on Tree's row of item i
	// (surfui.MenuItemID(i)) picks Items[i] (Item).
	Items []MenuItem
	// Cols and Rows are the cells it takes on a host: in Compact and
	// Roomy the card's width, at least its look's 220 CSS px, and its
	// height in the look's px, in cells; in Dense the cells' width, and
	// their rows less one, as a tooltip's.
	Cols, Rows int
	// Cells are what it draws in the cells: a line round the card, a row
	// an item.
	Cells cells.Screen
}

// HTML is the menu's document on a host, with the host's stylesheets
// after surf's (surfui.Document).
func (v MenuView) HTML(sheets ...string) string { return surfui.Document(v.Tree, sheets...) }

// Item is the item a press on target picks, target an element of Tree
// (a host event's target): its row's, or one in its row. ok is false
// for none, a separator, a heading, a disabled item, or no row.
func (v MenuView) Item(target string) (it MenuItem, ok bool) {
	ix := surfui.Resolve(v.Tree, surfui.MenuID)
	for n := ix[target]; n != nil; n = surfui.Parent(n) {
		for i := range v.Items {
			if surfui.IDOf(n) == surfui.MenuItemID(i) {
				it = v.Items[i]
				return it, it.Enabled && !it.Heading
			}
		}
	}
	return MenuItem{}, false
}

// Place is where a menu opened at cell (c, r) of a grid cols × rows
// goes, its top-left cell: under the point and to its right, above it
// where there is no room below, to its left where there is none to the
// right, and kept in the grid.
func (v MenuView) Place(c, r, cols, rows int) (x, y int) {
	return cells.PlaceAt(c, r, v.Cols, v.Rows, cols, rows)
}

// PlaceAtPointer is where a menu a press opened goes, its top-left cell,
// for a pointer at cell (c, r) of a grid cols × rows, as a native menu
// opens: at the pointer's cell, with its right edge at the pointer's cell
// where there is no room to the right, and its bottom row at the
// pointer's row where there is none below, kept in the grid. Use it for a
// menu opened by a pointer (an element's, Frame.Menu); use Place for one
// that opens under a button or a header, which it must not cover.
func (v MenuView) PlaceAtPointer(c, r, cols, rows int) (x, y int) {
	return cells.PlaceAtPointer(c, r, v.Cols, v.Rows, cols, rows)
}

// MenuView is the menu m (from Menu) as its host shows it, in the theme
// and density of its element, its keys in the surface's key style: in
// Compact, in cells of 10×21 CSS px. hi is the item the keyboard is on,
// -1 for none.
func (s *Surface) MenuView(m Menu, hi int) MenuView {
	s.fresh()
	from := s.index[m.ID]
	v := menuView(from, m.Items, hi, surfkeys.OnPlatform(s.keyStyle(), s.Platform.Mod), s.Hues(), 0, 0)
	// The surface takes the keyboard (Frame.Menu), so it can have it, and
	// gives the program the keys the menu uses (SPEC §10.2): they reach it
	// as keys, and Surface.Key takes them.
	v.Tree.Focusable().Give(hostMenuKeys...)
	return v
}

// hostMenuKeys are the keys the menu's document gives the program (its
// data-keys): the menu's keys but Escape, which the host never takes
// (SPEC §10.2), and characters, which reach it as keys.
var hostMenuKeys = slices.DeleteFunc(slices.Clone(menuKeys), func(k string) bool { return k == "escape" })

// OpenMenu is a menu a host program opens at a point of its own, not an
// element's: items, drawn in look, hi the item the keyboard is on (-1
// for none). Its pick is the host program's: Item says which item a
// press on its view picks.
func OpenMenu(items []MenuItem, hi int, look MenuLook) MenuView {
	root := surfui.Surface()
	if look.Theme != "" {
		root.Theme(look.Theme)
	}
	d := surfui.Compact
	if look.Density != "" {
		d = surfui.Density(look.Density)
	}
	root.Density(d)
	return menuView(root, items, hi, surfkeys.OnPlatform(look.KeyStyle, look.Mod), nil, look.CellW, look.CellH)
}

// menuView draws items as a menu of from's.
func menuView(from *surfui.Node, items []MenuItem, hi int, style surfkeys.Style, hues surfui.Palette, cw, ch float64) MenuView {
	its := make([]surfui.MenuItem, len(items))
	for i, it := range items {
		its[i] = surfui.MenuItem{ID: it.ID, Sep: it.Sep, Label: it.Label, Icon: it.Icon, Key: it.Key,
			Disabled: !it.Enabled && !it.Sep && !it.Heading, Danger: it.Danger, Heading: it.Heading}
	}
	tree := surfui.MenuCard(from, its, hi)
	surfkeys.Expand(tree, style, true)
	surfui.Resolve(tree, surfui.MenuID)
	scr := cells.DrawTip(tree, style, hues)
	if cw <= 0 {
		cw = 10
	}
	if ch <= 0 {
		ch = 21
	}
	// In Dense, as a tooltip: its room above and below is half a cell
	// each on a host, where the cells draw a line.
	cols, rows := scr.Cols, max(scr.Rows-1, 1)
	if hostLook(tree) {
		// The look's sizes, in CSS px: 4 of room round the rows, a row
		// 28, a heading 23.5, a rule 9, each 1 from the next.
		h := 8.0 + float64(max(len(items)-1, 0))
		for _, it := range items {
			switch {
			case it.Sep:
				h += 9
			case it.Heading:
				h += 23.5
			default:
				h += 28
			}
		}
		// Under the card and past its end, the room its surface keeps
		// for the raised shadow (surfui.MenuShadowBelow, MenuShadowSide).
		rows = int(math.Ceil((h + surfui.MenuShadowBelow) / ch))
		cols = max(cols, int(math.Ceil(220/cw))) + int(math.Ceil(surfui.MenuShadowSide/cw))
	}
	return MenuView{Tree: tree, Items: items, Cols: cols, Rows: rows, Cells: scr}
}

// hostLook reports whether a menu's tree is drawn in the look's sizes
// on a host (Compact, Roomy) rather than in cells' rows (Dense).
func hostLook(tree *surfui.Node) bool {
	for _, c := range surfui.Classes(tree) {
		if c == "u-dense" {
			return false
		}
	}
	return true
}
