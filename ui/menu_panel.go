package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
)

// MenuProps are a Menu's attributes.
type MenuProps struct {
	// ID names its elements; "" is "menu". Two on one surface need IDs of
	// their own.
	ID string
	// Items are the menu's, as a DropdownMenu's: groups, roles, shortcuts
	// shown, Disabled, OnSelect.
	Items []MenuItem
	// Highlight is the index in Items of the item highlighted, -1 for
	// none: the app's, as a List's selection is. The first item is
	// highlighted at 0, as a menu a key opens has it.
	Highlight int
	// OnHighlight asks for another highlight: the pointer onto an item
	// (surf.Hover), or the arrows, Home and End while the menu has the
	// keyboard. A nil OnHighlight changes nothing.
	OnHighlight func(i int)
	// Class is the menu's, for the app's own.
	Class string
}

// Menu is a menu's list of items in the kit's look, shown as it is: no
// trigger, no popover, no open state. Enter and Space on the highlighted
// item, or a click on an item, call its OnSelect. It is for a menu the app
// places itself, such as a surface of its own at the pointer; a menu that
// opens from a button is a DropdownMenu, and one that opens on a secondary
// press in an area is a ContextMenu.
func Menu(p MenuProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return menuPanel(newMenuPanel(p)).Render(ctx, w)
	})
}

// newMenuPanel is a Menu's view: a DropdownMenu's, always open, its
// highlight the app's. A highlight out of range, or on a disabled item, is
// none.
func newMenuPanel(p MenuProps) *menuView {
	st := &DropdownMenuState{open: true}
	v := newMenuView(st, DropdownMenuProps{ID: or(p.ID, "menu"), Items: p.Items, Class: p.Class})
	v.ctl = &p
	st.active = -1
	for k, o := range v.enabled {
		if v.items[o] == p.Highlight {
			st.active = k
		}
	}
	return v
}

// panelKey is a Menu's keys: the arrows, Home and End ask for a highlight
// among the items not disabled; Enter and Space choose the highlighted.
func (v *menuView) panelKey(k string) bool {
	n := len(v.enabled)
	if n == 0 {
		return false
	}
	cur, next := v.st.active, -1
	switch k {
	case "arrow_down":
		next = 0
		if cur >= 0 {
			next = min(cur+1, n-1)
		}
	case "arrow_up":
		next = n - 1
		if cur >= 0 {
			next = max(cur-1, 0)
		}
	case "home":
		next = 0
	case "end":
		next = n - 1
	case "enter", "space":
		if o := v.activeOpt(); o >= 0 {
			v.choose(o)
			return true
		}
		return false
	default:
		return false
	}
	if next != cur && v.ctl.OnHighlight != nil {
		v.ctl.OnHighlight(v.items[v.enabled[next]])
	}
	return true
}

// panelAttrs are a Menu's list's: its keys.
func (v *menuView) panelAttrs(ctx context.Context) templ.Attributes {
	return kitOn(ctx, v.id+"-menu", surf.Key(v.panelKey))
}

// panelCheck faults what a Menu doesn't take: an item's Variant or Role.
func (v *menuView) panelCheck(ctx context.Context) string {
	for _, it := range v.p.Items {
		checkVariant(ctx, "MenuItem", it.Variant, menuItemVariants)
		checkEnum(ctx, "MenuItem", "Role", it.Role, []string{"", "menuitem", "menuitemcheckbox", "menuitemradio"})
	}
	return ""
}

// panelFrame reads the frame number, which the items' handlers carry; a
// Menu takes no part in the surface's one open popup.
func (v *menuView) panelFrame(ctx context.Context) string {
	v.n = frameN(ctx)
	return ""
}
