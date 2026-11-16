package ui

import (
	"context"
	"fmt"
	"slices"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// DropdownMenuProps are a DropdownMenu's attributes.
type DropdownMenuProps struct {
	ID string
	// Trigger is what opens the menu: Basecoat's button, holding it.
	Trigger templ.Component
	Items   []MenuItem
	// Align is the menu's side along its trigger: "start" (the default),
	// "center" or "end". surftest fails any other.
	Align string
	Class string
}

// A MenuItem is one of a menu's items. Consecutive items with the same
// Group are shown under its label, the groups apart.
type MenuItem struct {
	Label string
	// Icon is a Lucide icon's name, shown before the label.
	Icon string
	// Keys is the item's shortcut, shown as Kbd shows it. The menu shows it
	// only: the app binds the key itself.
	Keys     string
	Group    string
	Variant  Variant // Destructive, or Primary
	Disabled bool
	// OnSelect runs when the user chooses the item; the menu closes.
	OnSelect func()
}

// DropdownMenuState is what a DropdownMenu remembers between renders:
// whether it is open and the item highlighted. The zero value is closed.
type DropdownMenuState struct {
	open bool
	// active is dropdown-menu.js's activeIndex: an index into the items
	// not disabled; -1 for none. It means something only while open.
	active int
	// pick is open()'s initialSelection, which the next render applies
	// once it knows the items: pickNone, pickFirst or pickLast.
	pick uint8
	// focus is trigger.focus(): the next render asks for the keyboard on
	// the trigger.
	focus bool
}

const (
	pickNone uint8 = iota
	pickFirst
	pickLast
)

// Open opens the menu with its first item highlighted, as a shortcut that
// opens it does.
func (st *DropdownMenuState) Open() {
	st.open, st.active, st.pick = true, -1, pickFirst
	// The menu's keys are heard where the keyboard is: on its trigger.
	st.focus = true
}

// DropdownMenu is Basecoat's dropdown menu: its markup generated from
// dropdown-menu.html.jinja, its behaviour dropdown-menu.js's, translated to
// Go (the arrows, Home, End, Enter, Escape, a click outside closing it).
func DropdownMenu(st *DropdownMenuState, p DropdownMenuProps) templ.Component {
	return dropdownMenu(newMenuView(st, p))
}

var (
	menuAligns       = []string{"", "start", "center", "end"}
	menuItemVariants = []Variant{Primary, Destructive}
)

// menuView is one render of a DropdownMenu.
type menuView struct {
	st      *DropdownMenuState
	p       DropdownMenuProps
	id      string
	entries []listEntry
	opts    []listOpt // every item, in order; value is its index into p.Items
	items   []int     // the items' indexes into p.Items, as opts
	enabled []int     // the items not disabled, as indexes into opts (getItems)
}

func newMenuView(st *DropdownMenuState, p DropdownMenuProps) *menuView {
	v := &menuView{st: st, p: p, id: or(p.ID, "dropdown-menu")}
	v.entries, v.opts = grouped(v.id+"-items", len(p.Items),
		func(i int) string { return p.Items[i].Group },
		func(i int, id string) listOpt {
			v.items = append(v.items, i)
			return listOpt{id: id, label: p.Items[i].Label, disabled: p.Items[i].Disabled}
		})
	for i, o := range v.opts {
		if !o.disabled {
			v.enabled = append(v.enabled, i)
		}
	}
	if !st.open || st.active >= len(v.enabled) {
		// refreshDropdownMenu: an active item that went away is none.
		st.active = -1
	}
	if st.open && len(v.enabled) > 0 {
		switch st.pick {
		case pickFirst:
			st.active = 0
		case pickLast:
			st.active = len(v.enabled) - 1
		}
	}
	st.pick = pickNone
	return v
}

// check faults the values the menu doesn't take: an Align, an item's
// Variant.
func (v *menuView) check(ctx context.Context) string {
	if !slices.Contains(menuAligns, v.p.Align) {
		rt.Fault(ctx, fmt.Sprintf("ui.DropdownMenu: no align %q (it takes %s)", v.p.Align, list(menuAligns[1:])))
	}
	for _, it := range v.p.Items {
		checkVariant(ctx, "MenuItem", it.Variant, menuItemVariants)
	}
	return ""
}

func (v *menuView) item(o int) MenuItem { return v.p.Items[v.items[o]] }

func (v *menuView) pos(o int) int {
	for k, j := range v.enabled {
		if j == o {
			return k
		}
	}
	return -1
}

// activeOpt is the highlighted item, into opts; -1 for none.
func (v *menuView) activeOpt() int {
	if !v.st.open || v.st.active < 0 {
		return -1
	}
	return v.enabled[v.st.active]
}

// open is openPopover: initial is the item highlighted, if any.
func (v *menuView) open(initial uint8) {
	v.st.open = true
	if n := len(v.enabled); n > 0 {
		switch initial {
		case pickFirst:
			v.st.active = 0
		case pickLast:
			v.st.active = n - 1
		}
	}
}

// close is closePopover: focus says whether the trigger takes the keyboard.
func (v *menuView) close(focus bool) {
	if !v.st.open {
		return
	}
	v.st.open, v.st.active = false, -1
	if focus {
		v.st.focus = true
	}
}

// toggle is the trigger's click: root.toggle().
func (v *menuView) toggle() {
	if v.st.open {
		v.close(true)
	} else {
		v.open(pickNone)
	}
}

func (v *menuView) outside() { v.close(false) }

// choose is handleMenuClick on item o: it runs, and the menu closes and
// gives the trigger the keyboard.
func (v *menuView) choose(o int) {
	it := v.item(o)
	if it.Disabled {
		return
	}
	if it.OnSelect != nil {
		it.OnSelect()
	}
	v.close(true)
	// As a Select's option: the click blurred the surface, and the blur
	// closed the menu before the click came.
	v.st.focus = true
}

// key is handleKeydown, on the menu's root.
func (v *menuView) key(k string) bool {
	if k == "escape" {
		if !v.st.open {
			return false
		}
		v.close(true)
		return true
	}
	if !v.st.open {
		switch k {
		case "enter", "space":
			v.open(pickNone)
		case "arrow_down":
			v.open(pickFirst)
		case "arrow_up":
			v.open(pickLast)
		default:
			return false
		}
		return true
	}
	n := len(v.enabled)
	if n == 0 {
		return false
	}
	cur := v.st.active
	next := cur
	switch k {
	case "arrow_down":
		if cur == -1 {
			next = 0
		} else {
			next = min(cur+1, n-1)
		}
	case "arrow_up":
		if cur == -1 {
			next = n - 1
		} else {
			next = max(cur-1, 0)
		}
	case "home":
		next = 0
	case "end":
		next = n - 1
	case "enter", "space":
		if o := v.activeOpt(); o >= 0 {
			v.choose(o)
		}
		v.close(true)
		return true
	default:
		return false
	}
	v.st.active = next
	return true
}

// hover is handleMenuMousemove over item o.
func (v *menuView) hover(o int) func(bool) {
	return func(over bool) {
		if over && v.st.open {
			v.st.active = v.pos(o)
		}
	}
}

// leave is handleMenuMouseleave.
func (v *menuView) leave(over bool) {
	if !over && v.st.open {
		v.st.active = -1
	}
}

func (v *menuView) refocus() bool {
	f := v.st.focus
	v.st.focus = false
	return f
}

// rootAttrs are the root's: its keys, and while open the click outside.
func (v *menuView) rootAttrs(ctx context.Context) templ.Attributes {
	if !v.st.open {
		return surf.On(ctx, v.id, surf.Key(v.key))
	}
	return surf.On(ctx, v.id, surf.Key(v.key), surf.Outside(v.outside))
}
