package surfui

import (
	"strconv"

	"github.com/neuroplastio/hotty-surf/pkg/mdi"
)

// A menu, drawn: every menu in one look, an element's context menu
// (Node.Menu) and one a host program opens at a point of its own (a
// pane's, a tab's). MenuCard is the root of the surface its host shows
// it in.

// MenuID is the id of a menu's root, in its own surface (MenuCard).
const MenuID = "m"

// MenuCard is a menu as its host shows it, in a surface of its own: the
// root of that surface, in the theme and density nearest from (a node
// of the tree the menu is of, or nil for the defaults), holding a card,
// raised, with a hairline round it; in it a row for each item, in
// order: an action its icon, its label and its key, muted where it is
// Disabled and in the error where it is Danger; a Heading its label, a
// caption; a Sep a rule. hi is the index of the item the keyboard is
// on, -1 for none: the selected fill. Resolve it under MenuID: item i's
// row is then MenuItemID(i). On a host the surface is clear round the
// card, with room under it and at its end for the raised shadow
// (MenuShadowBelow, MenuShadowSide), which its host's view counts in.
func MenuCard(from *Node, items []MenuItem, hi int) *Node {
	card := El("div").Class("u-menu").Attr("role", "menu")
	icons := false
	for _, it := range items {
		if _, ok := mdi.Glyph(it.Icon); ok && !it.Sep && !it.Heading {
			icons = true
		}
	}
	for i, it := range items {
		switch {
		case it.Sep:
			card.Add(Rule().Class("u-menu-sep"))
		case it.Heading:
			card.Add(Text(it.Label).Class("u-menu-head"))
		default:
			icon := Icon("")
			if g, ok := mdi.Glyph(it.Icon); ok {
				icon = Icon(string(g))
			} else if icons {
				// In the cells a row with none keeps the icons' slot, a cell,
				// so that the labels line up (a host's CSS gives the slot).
				icon.cells = " "
			}
			row := Item(icon, Text(it.Label).Grow()).Class("u-menu-item").Attr("role", "menuitem")
			// Its click is a pick (surfrun.Surface.MenuEvent), which a
			// disabled row has none of, so it asks the host for none.
			if !it.Disabled {
				row.On("click", nil)
			}
			if it.Key != "" {
				row.Add(Key(it.Key))
			}
			if it.Danger {
				row.Class("u-danger")
			}
			row.Disabled(it.Disabled).Selected(i == hi)
			card.Add(row)
		}
	}
	s := El("div", card).Class("u-surface", "u-menu-surface")
	density, theme := false, false
	for n := from; n != nil; n = n.parent {
		for _, cl := range n.class {
			switch {
			case !density && (cl == "u-dense" || cl == "u-roomy" || cl == "u-compact"):
				density = true
				s.Class(cl)
			case !theme && len(cl) > 8 && cl[:8] == "u-theme-":
				theme = true
				s.Class(cl)
			}
		}
	}
	return s
}

// MenuShadowBelow and MenuShadowSide are the CSS px the raised shadow
// reaches under a menu's card and past its end (--u-shadow-raised: 6 down,
// 16 of blur, 6 drawn in), the room its surface keeps clear for it.
const (
	MenuShadowBelow = 16
	MenuShadowSide  = 10
)

// MenuItemID is the id of item i's row in a MenuCard resolved under
// MenuID: a click's target there names the item it picks.
func MenuItemID(i int) string { return MenuID + ".0." + strconv.Itoa(i) }
