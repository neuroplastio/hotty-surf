package ui

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// SelectProps are a Select's attributes.
type SelectProps struct {
	// ID names its elements; "" is "select". Two on one surface need IDs
	// of their own.
	ID    string
	Items []SelectItem
	// Value is the value of the item shown as chosen; "" is none, and the
	// trigger shows Placeholder. With neither, the trigger is blank, which
	// is a fault: a Select with nothing chosen takes a Placeholder.
	// Value and OnChange are used in single mode; Values and OnValuesChange
	// are used in multiple mode. The other pair is ignored.
	// A single choice closes the list; multiple choices keep it open unless
	// CloseOnSelect is set. An absent selected value or a repeated value in
	// Values is a fault and is omitted outside tests.
	Value string
	// Multiple selects more than one value, reported through OnValuesChange.
	Multiple bool
	// Values are the selected item values when Multiple is true; nil or empty
	// means none. The callback receives a fresh slice in item order, unlike
	// Combobox's selection order. Disabled selected items stay selected.
	Values []string
	// CloseOnSelect closes a multiple-select list after each choice.
	CloseOnSelect bool
	// Align places the list along its trigger: "" or "start" (the default),
	// "center" or "end". surftest fails any other. It does not automatically
	// flip or clamp the list to the surface.
	Align       string
	Placeholder string
	Disabled    bool
	// OnChange hears the value the user chose.
	OnChange func(value string)
	// OnValuesChange hears selected values in item order when Multiple.
	OnValuesChange func(values []string)
	Class          string
}

// A SelectItem is an option of a Select, NativeSelect or Combobox.
// Consecutive items with the same Group are shown under its label, the
// groups apart. Values uniquely identify items in Select and Combobox;
// those components fault on duplicate item values. Labels may repeat.
// NativeSelect does not check for duplicate option values.
type SelectItem struct {
	Value, Label string
	Group        string
	Disabled     bool
	// Keywords are additional search words for Combobox, separated by
	// spaces or commas. Select and NativeSelect ignore them.
	Keywords string
}

// SelectState is what a Select remembers between renders: whether its list
// is open and the item the arrows highlighted. The zero value is closed.
type SelectState struct {
	open bool
	// active is select.js's activeIndex: the highlighted option, an index
	// into the options not disabled; -1 for none. It means something only
	// while open.
	active int
	// focus is select.js's trigger.focus(): render focusAt asks for the
	// keyboard on the trigger, and the ask goes with it (oneShot).
	focus   bool
	focusAt uint64
}

// Select is Basecoat's select: its markup generated from select.html.jinja,
// its behaviour select.js's, translated to Go (opening, the arrows, Home,
// End, choosing, Escape, a click outside closing it).
func Select(st *SelectState, p SelectProps) templ.Component {
	// The view is made at each render, from the state as it is then.
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return selectBox(newSelectView(st, p)).Render(ctx, w)
	})
}

// selectView is one render of a Select: the macro's entries for its items,
// and select.js's handlers over them.
type selectView struct {
	st      *SelectState
	p       SelectProps
	id      string
	entries []listEntry // the listbox's children: options, groups, separators
	opts    []listOpt   // every option, in order (select.js's allOptions)
	enabled []int       // the options not disabled, as indexes into opts (its options)
	chosen  int         // the option whose value is Value, into opts; -1 for none
	n       uint64      // the frame rendering it (rt.Frame.N), its handlers'
	popups  *popup      // the surface's open popup
}

// triggerKeys are the keys key takes ahead of the trigger: Enter while
// open, which the button would otherwise press.
func (v *selectView) triggerKeys() []string {
	if v.st.open {
		return []string{"enter"}
	}
	return nil
}

// join reads the frame: its number, and the surface's popups.
func (v *selectView) join(ctx context.Context) string {
	values := map[string]bool{}
	for _, item := range v.p.Items {
		if values[item.Value] {
			rt.Fault(ctx, fmt.Sprintf("ui.Select: duplicate item value %q", item.Value))
		}
		values[item.Value] = true
	}
	selected := v.p.Values
	if !v.p.Multiple && v.p.Value != "" {
		selected = []string{v.p.Value}
	}
	if !v.p.Multiple && v.p.Value == "" {
		selected = nil
	}
	seen := map[string]bool{}
	for _, value := range selected {
		if !values[value] || seen[value] {
			rt.Fault(ctx, fmt.Sprintf("ui.Select: selected value %q is absent or repeated", value))
		}
		seen[value] = true
	}
	v.n = frameN(ctx)
	v.popups = popups(ctx)
	v.popups.track(ctx, v.st, v.id, v.st.open, v.shut)
	return ""
}

// shut closes it for another popup's opening.
func (v *selectView) shut() { v.st.open, v.st.active = false, -1 }

// A listEntry is a child of a select's listbox or a menu's menu, as the
// macros render their items: an option, a group of them under a heading,
// or a separator.
type listEntry struct {
	kind    entryKind
	opt     int    // an option's index (kind entryOpt)
	group   string // a group's label
	labelID string // a group's heading's id
	opts    []int  // a group's options
}

type entryKind uint8

const (
	entryOpt entryKind = iota
	entryGroup
	entrySeparator
)

// A listOpt is an option of a list with its element's id.
type listOpt struct {
	id       string
	value    string
	label    string
	disabled bool
}

// grouped lays out n items as the macros' items would be written for them:
// consecutive items of one group are a group under its label, with a
// separator between a group and what is beside it. The ids are the
// macro's: <prefix>-<n> at the top, <prefix>-<n>-<m> in a group, its
// heading group-label-<prefix>-<n>; n counts the separators, as the
// macro's loop.index does.
func grouped(prefix string, n int, group func(i int) string, opt func(i int, id string) listOpt) ([]listEntry, []listOpt) {
	var (
		entries []listEntry
		opts    []listOpt
	)
	for i := 0; i < n; {
		g := group(i)
		if len(entries) > 0 && (g != "" || entries[len(entries)-1].kind == entryGroup) {
			entries = append(entries, listEntry{kind: entrySeparator})
		}
		at := prefix + "-" + strconv.Itoa(len(entries)+1)
		if g == "" {
			opts = append(opts, opt(i, at))
			entries = append(entries, listEntry{kind: entryOpt, opt: len(opts) - 1})
			i++
			continue
		}
		e := listEntry{kind: entryGroup, group: g, labelID: "group-label-" + at}
		for m := 1; i < n && group(i) == g; i, m = i+1, m+1 {
			opts = append(opts, opt(i, at+"-"+strconv.Itoa(m)))
			e.opts = append(e.opts, len(opts)-1)
		}
		entries = append(entries, e)
	}
	return entries, opts
}

func newSelectView(st *SelectState, p SelectProps) *selectView {
	v := &selectView{st: st, p: p, id: or(p.ID, "select"), chosen: -1}
	v.entries, v.opts = grouped(v.id+"-items", len(p.Items),
		func(i int) string { return p.Items[i].Group },
		func(i int, id string) listOpt {
			it := p.Items[i]
			return listOpt{id: id, value: it.Value, label: it.Label, disabled: it.Disabled}
		})
	for i, o := range v.opts {
		if v.chosen < 0 && v.selected(i) {
			v.chosen = i
		}
		if o.disabled {
			continue
		}
		v.enabled = append(v.enabled, i)
		// select.js finds the chosen among the options not disabled. ""
		// is none, as the Props say, where select.js would show the
		// first option when there is no placeholder.
	}
	if !st.open || st.active >= len(v.enabled) {
		// refreshSelect: an active option that went away is none.
		st.active = -1
	}
	if p.Disabled {
		// A disabled trigger takes no click and no key: it shows no list.
		st.open, st.active = false, -1
	}
	return v
}

// pos is option i's place among the options not disabled, -1 for none.
func (v *selectView) pos(i int) int {
	for k, j := range v.enabled {
		if j == i {
			return k
		}
	}
	return -1
}

// activeOpt is the highlighted option, into opts; -1 for none.
func (v *selectView) activeOpt() int {
	if !v.st.open || v.st.active < 0 {
		return -1
	}
	return v.enabled[v.st.active]
}

// label is what the trigger shows: the chosen option's label, or the
// placeholder (muted).
func (v *selectView) label() (string, bool) {
	if v.p.Multiple {
		var labels []string
		for i := range v.opts {
			if v.selected(i) {
				labels = append(labels, v.opts[i].label)
			}
		}
		if len(labels) > 0 {
			return strings.Join(labels, ", "), false
		}
		return v.p.Placeholder, v.p.Placeholder != ""
	}
	if v.chosen >= 0 {
		return v.opts[v.chosen].label, false
	}
	return v.p.Placeholder, v.p.Placeholder != ""
}

// value is the hidden input's: the chosen value, or "".
func (v *selectView) value() string {
	if v.p.Multiple {
		var values strings.Builder
		values.WriteByte('[')
		first := true
		for i := range v.opts {
			if v.selected(i) {
				if !first {
					values.WriteByte(',')
				}
				writeJSONText(&values, v.opts[i].value)
				first = false
			}
		}
		values.WriteByte(']')
		return values.String()
	}
	if v.chosen >= 0 {
		return v.opts[v.chosen].value
	}
	return ""
}

func (v *selectView) selected(i int) bool {
	if v.p.Multiple {
		return slices.Contains(v.p.Values, v.opts[i].value)
	}
	return v.p.Value != "" && v.opts[i].value == v.p.Value
}

// open is root.open(): the list shows, the chosen option highlighted.
func (v *selectView) open() {
	v.st.open = true
	v.st.active = v.pos(v.chosen)
	v.popups.opened(v.st, v.shut)
}

// close is closePopover: focus says whether the trigger takes the keyboard.
func (v *selectView) close(focus bool) {
	if !v.st.open {
		return
	}
	if focus {
		v.ask()
	}
	v.st.open, v.st.active = false, -1
	v.popups.closed(v.st)
}

// toggle is the trigger's click.
func (v *selectView) toggle() {
	if v.st.open {
		v.close(true)
	} else {
		v.open()
	}
}

// outside is handleDocumentClick: a click outside closes the list, and
// leaves the keyboard where it went.
func (v *selectView) outside() { v.close(false) }

// rootAttrs are the root's id, and while the list is open the click
// outside it, which select.js hears on the document.
func (v *selectView) rootAttrs(ctx context.Context) templ.Attributes {
	if !v.st.open {
		return templ.Attributes{"id": v.id}
	}
	return surf.On(ctx, v.id, surf.Outside(v.outside))
}

// choose is a click on option i, or Enter on it: the change (when the
// value is new, or for the placeholder's empty option), then the list
// closes and the trigger takes the keyboard.
func (v *selectView) choose(i int) {
	o := v.opts[i]
	if v.p.Multiple {
		values := make([]string, 0)
		for j := range v.opts {
			selected := v.selected(j)
			if j == i {
				selected = !selected
			}
			if selected {
				values = append(values, v.opts[j].value)
			}
		}
		if v.p.OnValuesChange != nil {
			v.p.OnValuesChange(values)
		}
		if v.p.CloseOnSelect {
			v.close(true)
		} else {
			v.st.active = v.pos(i)
			v.ask()
		}
		return
	}
	if v.p.OnChange != nil {
		if v.p.Placeholder != "" && o.value == "" {
			v.p.OnChange("")
		} else if v.value() != o.value {
			v.p.OnChange(o.value)
		}
	}
	v.close(true)
	// A click on an option takes the keyboard off the surface (an option
	// takes no focus, SPEC §10.1), and that blur closed the list before
	// the click came (Outside hears a blur): the trigger takes it back
	// all the same.
	v.ask()
}

// ask is trigger.focus(), from a handler of this view's frame: for the
// frame after it.
func (v *selectView) ask() { v.st.focus, v.st.focusAt = true, v.n+1 }

// key is handleKeyNavigation, for the keys the trigger hears.
func (v *selectView) key(k string) bool {
	switch k {
	case "arrow_down", "arrow_up", "enter", "home", "end", "escape":
	default:
		return false
	}
	if !v.st.open {
		if k == "enter" || k == "escape" {
			return false
		}
		v.open()
		return true
	}
	switch k {
	case "escape":
		v.close(true)
		return true
	case "enter":
		if i := v.activeOpt(); i >= 0 {
			v.choose(i)
		}
		return true
	}
	n := len(v.enabled)
	if n == 0 {
		return true
	}
	cur := v.st.active
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
	v.st.active = next
	return true
}

// hover is the listbox's mousemove over option i.
func (v *selectView) hover(i int) func(bool) {
	return func(over bool) {
		if over && v.st.open {
			v.st.active = v.pos(i)
		}
	}
}

// leave is the listbox's mouseleave: the chosen option is highlighted
// again.
func (v *selectView) leave(over bool) {
	if !over && v.st.open {
		v.st.active = v.pos(v.chosen)
	}
}

// refocus says whether this render gives the trigger the keyboard, as
// select.js's trigger.focus() would have, once.
func (v *selectView) refocus() bool {
	return oneShot(&v.st.focus, v.st.focusAt, v.n)
}

// frameN is the number of the frame ctx renders (rt.Frame.N), 0 outside a
// surface: what a view's handlers know their frame by.
func frameN(ctx context.Context) uint64 {
	if f := rt.From(ctx); f != nil {
		return f.N
	}
	return 0
}

// oneShot says whether frame n takes a one-shot ask for the keyboard made
// for frame at (0 for the next render, as an app's Open is), and clears
// it: an ask whose frame went by without the component in it expires,
// rather than take the keyboard whenever the component is next shown.
func oneShot(asked *bool, at, n uint64) bool {
	ok := *asked && (at == 0 || at == n)
	*asked = false
	return ok
}

// or is s, or def when s is "".
func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func bstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
