package surfrun

import (
	"strconv"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// A context menu is an element's (surfui.Node.Menu, a template's <menu>):
// the tree never draws it. Its host asks the surface for it where the
// user asks for one (Menu), draws it itself at the pointer or the
// focused element, and hands back the item picked (Pick).

// MenuAt is where the user asks for a context menu: at an element, or
// where the keys go.
type MenuAt struct {
	// ID is the element's id: a host event's target, or, in the cells,
	// the element at the cell the user pressed (cells.Renderer.At).
	ID string
	// Focus asks at the focused element instead, or at the keys' home
	// with none focused, as a key goes: for a key that asks for the menu
	// (the Menu key, shift+F10). ID is then not read.
	Focus bool
}

// Menu is an element's context menu as its host draws it. Its fields are
// values only, which a host may relay as they are.
type Menu struct {
	// ID is the element's id, its key path in the tree (surfui.Resolve),
	// which stays the same across renders while the element is there: a
	// :each's row keyed by its :key keeps it as rows before it come and
	// go. Pick takes it.
	ID string
	// Items are the menu's items, in order.
	Items []MenuItem
}

// MenuItem is an item of a Menu: an action, or a separator (Sep), which
// has nothing but its ID.
type MenuItem struct {
	// ID names it among its menu's items, which Pick takes: a
	// template's item's is its place among its <menu>'s in the source,
	// so that an item a :if leaves out moves no other's.
	ID  string
	Sep bool
	// Label is what it says; Icon an MDI symbol's name ("content-copy"),
	// "" for none; Key a key's name to show beside it, which it binds
	// nothing to.
	Label string
	Icon  string
	Key   string
	// Enabled is false for an item that can't be picked (:disabled):
	// shown, and its pick does nothing.
	Enabled bool
	// Danger is an action that destroys, drawn in the error's colour;
	// Heading a caption over the items after it, its Label alone, which
	// can't be picked.
	Danger, Heading bool
}

// Menu is the context menu at at: that of the innermost element that has
// one, from the element at up through those around it. ok is false where
// none has one, or at names no element of the tree. An element whose
// menu draws no item (each left out by its :if) has none.
//
// It is the tree the model has now (fresh), as an event is routed by: the
// menu the user sees is the one a pick then finds.
func (s *Surface) Menu(at MenuAt) (m Menu, ok bool) {
	s.fresh()
	n := s.menuStart(at)
	for ; n != nil; n = surfui.Parent(n) {
		if items := surfui.MenuOf(n); len(items) > 0 {
			return Menu{ID: surfui.IDOf(n), Items: menuItems(items)}, true
		}
	}
	return Menu{}, false
}

// menuStart is the element a menu at at is looked for from.
func (s *Surface) menuStart(at MenuAt) *surfui.Node {
	if !at.Focus {
		return s.index[at.ID]
	}
	id := s.focused
	if id == "" || s.index[id] == nil {
		id = s.home
	}
	if id == "" {
		id = "r"
	}
	return s.index[id]
}

// menuItems are a node's menu items as its host has them.
func menuItems(items []surfui.MenuItem) []MenuItem {
	out := make([]MenuItem, len(items))
	for i, it := range items {
		out[i] = MenuItem{ID: itemID(it, i), Sep: it.Sep, Label: it.Label, Icon: it.Icon, Key: it.Key,
			Enabled: !it.Disabled && !it.Sep && !it.Heading, Danger: it.Danger, Heading: it.Heading}
	}
	return out
}

// itemID is item i's id: its own, or its place.
func itemID(it surfui.MenuItem, i int) string {
	if it.ID != "" {
		return it.ID
	}
	return strconv.Itoa(i)
}

// Pick runs the item of the menu (Menu.ID, MenuItem.ID) the user picked,
// as a click runs its target's handler: the model's Do, with an Event of
// Kind "pick", its Target the menu's element and its Value the item's ID;
// an Update is then due. It reports whether it ran the item, and whether
// an Update is due, as Event does.
//
// A pick of an item that is no longer there to pick runs nothing, and is
// no error: the element is gone, or its menu has no such item now, or
// the item is a separator or disabled. The tree it looks in is the one
// the model has now (fresh), so a pick after a change the user has not
// yet seen finds the item as the change left it.
func (s *Surface) Pick(menu, item string) (took, redraw bool) {
	s.fresh()
	n := s.index[menu]
	if n == nil {
		return false, s.due
	}
	for i, it := range surfui.MenuOf(n) {
		if itemID(it, i) != item {
			continue
		}
		if it.Sep || it.Heading || it.Disabled || it.Act == nil {
			return false, s.due
		}
		s.do(it.Act, Event{Kind: "pick", Target: menu, Value: item})
		return true, s.due
	}
	return false, s.due
}
