package ui

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// DropdownMenuProps are a DropdownMenu's attributes.
type DropdownMenuProps struct {
	// ID names its elements; "" is "dropdown-menu". Two on one surface
	// need IDs of their own.
	ID string
	// Trigger is what opens the menu: Basecoat's button, holding it. An
	// icon alone takes text beside it for who can't see it:
	// <span class="sr-only">More</span>.
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
	Keys  string
	Group string
	// Keywords are more words a Command's search finds the item by,
	// separated by spaces or commas (data-keywords), case-insensitively, in
	// addition to Label. Keys is display text, not a search keyword.
	// A DropdownMenu has no search, and doesn't use Keywords.
	Keywords string
	Variant  Variant // Destructive, or Primary
	Disabled bool
	// Role is "" or "menuitem" for an action, "menuitemcheckbox" for a
	// toggle, or "menuitemradio" for a choice. Command ignores it.
	Role string
	// Checked and OnCheckedChange control a checkbox or radio item. Radio
	// items in the same rendered group are exclusive; selecting one calls
	// changed, enabled items' callbacks in item order. Choosing an item calls
	// these callbacks before OnSelect, then closes the menu. Command ignores
	// these fields.
	Checked         bool
	OnCheckedChange func(bool)
	// OnSelect runs when the user chooses the item; the menu closes.
	OnSelect func()
	// Leading is drawn in the icon's place, in place of Icon: a colour
	// dot, a Switch showing a choice's state, an icon of the app's own.
	// Trailing is drawn in the shortcut's place, in place of Keys: a note,
	// a count. Content is drawn in the label's place: the label with an
	// aside after it, a name being typed with its caret. Label still names
	// the item (its aria-label, then) and is what a pick reports. They are
	// the app's markup: nothing in them takes a press or a key, which are
	// the item's. Command ignores them.
	Leading, Content, Trailing templ.Component
	// Separator makes the item a rule between the items around it, and
	// nothing else: its other fields are ignored, and it is never
	// highlighted. Group changes draw their own.
	Separator bool
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
	// focus is trigger.focus(): render focusAt (0 for the next) asks for
	// the keyboard on the trigger, and the ask goes with it (oneShot).
	focus   bool
	focusAt uint64
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
	// The menu's keys are heard where the keyboard is: on its trigger,
	// at whichever render comes next, since the app asks.
	st.focus, st.focusAt = true, 0
}

// DropdownMenu is Basecoat's dropdown menu: its markup generated from
// dropdown-menu.html.jinja, its behaviour dropdown-menu.js's, translated to
// Go (the arrows, Home, End, Enter, Escape, a click outside closing it).
func DropdownMenu(st *DropdownMenuState, p DropdownMenuProps) templ.Component {
	// The view is made at each render, from the state as it is then.
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return dropdownMenu(newMenuView(st, p)).Render(ctx, w)
	})
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
	n       uint64    // the frame rendering it (rt.Frame.N), its handlers'
	popups  *popup    // the surface's open popup
	// ctl is a Menu's props: the highlight is the app's, and a choice
	// closes nothing.
	ctl *MenuProps
}

// join reads the frame: its number, and the surface's popups.
func (v *menuView) join(ctx context.Context) string {
	v.n = frameN(ctx)
	v.popups = popups(ctx)
	v.popups.track(v.st, v.st.open, v.shut)
	return ""
}

// shut closes it for another popup's opening.
func (v *menuView) shut() { v.st.open, v.st.active = false, -1 }

func newMenuView(st *DropdownMenuState, p DropdownMenuProps) *menuView {
	v := &menuView{st: st, p: p, id: or(p.ID, "dropdown-menu")}
	v.entries, v.opts = grouped(v.id+"-items", len(p.Items),
		func(i int) string { return p.Items[i].Group },
		func(i int, id string) listOpt {
			v.items = append(v.items, i)
			it := p.Items[i]
			return listOpt{id: id, label: it.Label, disabled: it.Disabled || it.Separator}
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
		checkEnum(ctx, "MenuItem", "Role", it.Role, []string{"", "menuitem", "menuitemcheckbox", "menuitemradio"})
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
	v.popups.opened(v.st, v.shut)
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
	v.popups.closed(v.st)
	if focus {
		v.ask()
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
	if it.Disabled || it.Separator {
		return
	}
	if v.ctl != nil {
		v.run(o, it)
		return
	}
	v.run(o, it)
	v.close(true)
	// As a Select's option: the click blurred the surface, and the blur
	// closed the menu before the click came.
	v.ask()
}

// run runs item o's callbacks: its check's, then OnSelect.
func (v *menuView) run(o int, it MenuItem) {
	switch it.Role {
	case "menuitemcheckbox":
		if it.OnCheckedChange != nil {
			it.OnCheckedChange(!it.Checked)
		}
	case "menuitemradio":
		v.checkRadio(o)
	}
	if it.OnSelect != nil {
		it.OnSelect()
	}
}

// checkRadio follows dropdown-menu.js's nearest group, or the whole menu
// for an ungrouped radio; disabled radios are left alone.
func (v *menuView) checkRadio(chosen int) {
	indices := make([]int, 0, len(v.opts))
	for _, e := range v.entries {
		if e.kind == entryGroup && slices.Contains(e.opts, chosen) {
			indices = e.opts
			break
		}
	}
	if len(indices) == 0 {
		for i := range v.opts {
			indices = append(indices, i)
		}
	}
	for _, i := range indices {
		it := v.item(i)
		if it.Role != "menuitemradio" || it.Disabled || it.Checked == (i == chosen) {
			continue
		}
		if it.OnCheckedChange != nil {
			it.OnCheckedChange(i == chosen)
		}
	}
}

// ask is trigger.focus(), from a handler of this view's frame: for the
// frame after it.
func (v *menuView) ask() { v.st.focus, v.st.focusAt = true, v.n+1 }

// key is handleKeydown, on the menu's root.
func (v *menuView) triggerKeys() string {
	if v.st.open {
		return "enter=program space=program"
	}
	return ""
}

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
		if v.ctl != nil {
			if over && v.ctl.OnHighlight != nil && v.items[o] != v.ctl.Highlight {
				v.ctl.OnHighlight(v.items[o])
			}
			return
		}
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
	return oneShot(&v.st.focus, v.st.focusAt, v.n)
}

// rootAttrs are the root's: its keys, and while open the click outside.
func (v *menuView) rootAttrs(ctx context.Context) templ.Attributes {
	if !v.st.open {
		return kitOn(ctx, v.id, surf.Key(v.key))
	}
	return kitOn(ctx, v.id, surf.Key(v.key), surf.Outside(v.outside))
}
