package ui

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
)

// CommandProps are a Command's attributes.
type CommandProps struct {
	// ID names its elements; "" is "command". Two on one surface need IDs
	// of their own.
	ID string
	// Items are what it lists, as a DropdownMenu's: consecutive items with
	// one Group are shown under its label. The search finds an item by its
	// Label and its Keywords.
	Items []MenuItem
	// Placeholder is the search field's, shown while it is empty; "" is
	// "Type a command or search...".
	Placeholder string
	// Empty is what it shows when no item matches; "" is "No results
	// found." (data-empty).
	Empty string
	// Class is more classes for its root, after its own.
	Class string
}

// CommandState is what a Command remembers between renders: the search
// and the item highlighted. The zero value searches for nothing, which
// shows every item, and highlights the first.
type CommandState struct {
	query string
	// active is the item the user moved to (the arrows, the pointer), as
	// an index into the view's items; it means something while moved is
	// set and it is shown. Otherwise the first item shown is highlighted,
	// as command.js's filterItems leaves it after every search.
	active int
	moved  bool
}

// Search sets the search, as typing it does, and highlights the first
// item it shows. A palette that opens afresh clears it: Search("").
func (st *CommandState) Search(q string) { st.query, st.moved = q, false }

// Command is Basecoat's command menu: a search field over a list of items,
// its markup generated from command.html.jinja and its behaviour
// command.js's, translated to Go. Typing searches: an item shows while its
// Label or one of its Keywords holds the search, ignoring case, and the
// first item shown is highlighted. The arrows, Home and End move the
// highlight over the items shown and not disabled, Enter chooses it, and a
// click chooses an item (its OnSelect). The keyboard stays in the field
// throughout. A list that shows nothing shows Empty.
func Command(st *CommandState, p CommandProps) templ.Component {
	// The view is made at each render: what the search shows is the
	// state's then, not when the component was made.
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return command(newCommandView(st, or(p.ID, "command"), p)).Render(ctx, w)
	})
}

// CommandDialogProps are a CommandDialog's attributes.
type CommandDialogProps struct {
	// ID names its elements; "" is "command-dialog". It is unique on the
	// surface among Dialogs and CommandDialogs, open or closed, as a
	// Dialog's ID is.
	ID string
	// Open shows it. The app keeps it, and sets it false in OnClose.
	Open bool
	// Items, Placeholder and Empty are as a Command's.
	Items       []MenuItem
	Placeholder string
	Empty       string
	// OnClose hears that the dialog closes: an item chosen (after its
	// OnSelect), Escape, or a click on the backdrop.
	OnClose func()
	// Class is more classes for the dialog element, after its own.
	Class string
}

// CommandDialog is Basecoat's command dialog: a Command in a modal
// dialog, the command palette that a shortcut opens. The keyboard goes to
// its search field when it opens and back where it was when it closes, as
// a Dialog's does. Choosing an item closes it.
func CommandDialog(st *CommandState, p CommandDialogProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		id := or(p.ID, "command-dialog")
		d := newDialogView(ctx, DialogProps{ID: id, Open: p.Open, OnClose: p.OnClose})
		if !p.Open {
			return nil
		}
		v := newCommandView(st, id, CommandProps{ID: id, Items: p.Items, Placeholder: p.Placeholder, Empty: p.Empty})
		v.dialog = d
		var b strings.Builder
		if err := commandBody(v).Render(ctx, &b); err != nil {
			return err
		}
		// The dialog finds its field in the body, as a Dialog finds the
		// first element that takes the keyboard in its children.
		body, err := d.body(ctx, templ.Raw(b.String()))
		if err != nil {
			return err
		}
		return commandDialog(v, d, p.Class, body).Render(ctx, w)
	})
}

// commandView is one render of a Command or a CommandDialog's.
type commandView struct {
	st      *CommandState
	p       CommandProps
	id      string
	entries []listEntry
	opts    []listOpt   // every item, in order; as p.Items
	shown   []bool      // by opt: the search holds it (aria-hidden="false")
	visible []int       // command.js's visibleItems: shown and not disabled, as opts
	dialog  *dialogView // the dialog it is in, or nil
}

func newCommandView(st *CommandState, id string, p CommandProps) *commandView {
	v := &commandView{st: st, p: p, id: id}
	v.entries, v.opts = grouped(id+"-items", len(p.Items),
		func(i int) string { return p.Items[i].Group },
		func(i int, id string) listOpt {
			return listOpt{id: id, label: p.Items[i].Label, disabled: p.Items[i].Disabled}
		})
	term := strings.ToLower(strings.TrimSpace(st.query))
	v.shown = make([]bool, len(v.opts))
	for o, it := range p.Items {
		v.shown[o] = commandMatch(it, term)
		if v.shown[o] && !it.Disabled {
			v.visible = append(v.visible, o)
		}
	}
	if st.moved && !v.isVisible(st.active) {
		// The item moved to went away, or the search hid it:
		// refreshCommand's filterItems highlights the first shown.
		st.moved = false
	}
	return v
}

// commandMatch is filterItems' test: the item's text or one of its
// keywords holds term.
func commandMatch(it MenuItem, term string) bool {
	if strings.Contains(strings.ToLower(strings.TrimSpace(it.Label)), term) {
		return true
	}
	for _, k := range strings.FieldsFunc(strings.ToLower(it.Keywords), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
	}) {
		if strings.Contains(k, term) {
			return true
		}
	}
	return false
}

func (v *commandView) isVisible(o int) bool {
	for _, x := range v.visible {
		if x == o {
			return true
		}
	}
	return false
}

// activeOpt is the item highlighted, as an index into opts; -1 for none.
func (v *commandView) activeOpt() int {
	if v.st.moved {
		return v.st.active
	}
	if len(v.visible) > 0 {
		return v.visible[0]
	}
	return -1
}

func (v *commandView) placeholder() string {
	return or(v.p.Placeholder, "Type a command or search...")
}

func (v *commandView) empty() string { return or(v.p.Empty, "No results found.") }

// fieldKeys are the keys the field gives the program: Enter always
// (handleKeyNavigation prevents it), and the arrows, Home and End while
// an item is shown, which otherwise are the field's.
func (v *commandView) fieldKeys() string {
	if len(v.visible) == 0 {
		return "enter=program"
	}
	return "arrow_down=program arrow_up=program home=program end=program enter=program"
}

// input is handleInput: a new search, which highlights its first item.
func (v *commandView) input(s string) {
	v.st.query, v.st.moved = s, false
}

// move highlights item o.
func (v *commandView) move(o int) { v.st.active, v.st.moved = o, true }

// key is handleKeyNavigation, heard where the keyboard is: the field.
func (v *commandView) key(k string) bool {
	if v.dialog != nil && v.dialog.key(k) {
		return true
	}
	switch k {
	case "enter":
		if o := v.activeOpt(); o >= 0 {
			v.choose(o)
		}
		return true
	case "arrow_down", "arrow_up", "home", "end":
	default:
		return false
	}
	n := len(v.visible)
	if n == 0 {
		return false
	}
	cur := -1
	for i, o := range v.visible {
		if o == v.activeOpt() {
			cur = i
		}
	}
	next := cur
	switch k {
	case "arrow_down":
		if cur < n-1 {
			next = cur + 1
		}
	case "arrow_up":
		next = max(cur-1, 0)
	case "home":
		next = 0
	case "end":
		next = n - 1
	}
	if next != cur {
		v.move(v.visible[next])
	}
	return true
}

// choose is the click on item o, which Enter makes too: its OnSelect, and
// in a dialog handleMenuClick's close.
func (v *commandView) choose(o int) {
	if !v.isVisible(o) {
		return
	}
	if f := v.p.Items[o].OnSelect; f != nil {
		f()
	}
	if v.dialog != nil {
		v.dialog.dismiss()
	}
}

// hover is handleMenuMousemove over item o.
func (v *commandView) hover(o int) func(bool) {
	return func(over bool) {
		if over && v.isVisible(o) && v.activeOpt() != o {
			v.move(o)
		}
	}
}

// itemAttrs are item o's: its id, and while it can be chosen its click
// and hover.
func (v *commandView) itemAttrs(ctx context.Context, o int) templ.Attributes {
	if v.p.Items[o].Disabled {
		return templ.Attributes{"id": v.opts[o].id}
	}
	return surf.On(ctx, v.opts[o].id, surf.Click(func() { v.choose(o) }), surf.Hover(v.hover(o)))
}

// check faults an item's Variant the list doesn't take, as a
// DropdownMenu's.
func (v *commandView) check(ctx context.Context) string {
	for _, it := range v.p.Items {
		checkVariant(ctx, "MenuItem", it.Variant, menuItemVariants)
	}
	return ""
}
