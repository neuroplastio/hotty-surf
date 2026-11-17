package ui

import (
	"context"
	"fmt"
	"github.com/a-h/templ"
	"io"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// ComboboxProps are a Combobox's attributes. Selection is by item value.
type ComboboxProps struct {
	// ID names the elements; "" is "combobox". Two on one surface need IDs.
	ID string
	// Items are the options; consecutive items with one Group share a heading.
	// Values must uniquely identify options; duplicate labels are fine.
	Items []SelectItem
	// Value is the selected item's value, or "" for none, in single mode.
	Value string
	// Values are selected item values in multiple mode, in selection order;
	// nil or empty means none. The callback receives a fresh slice in that
	// order, unlike Select's item order. Unknown or repeated values are kept
	// as supplied, rendered as chips and carried into callbacks.
	Values []string
	// Multiple permits more than one selected item.
	Multiple bool
	// Placeholder is the text field's placeholder.
	Placeholder string
	// Disabled prevents editing and selection.
	Disabled bool
	// Clear adds the clear-selection button in single mode.
	Clear bool
	// AutoHighlight highlights the first matching enabled item after a search.
	AutoHighlight bool
	// CloseOnSelect closes a multiple combobox when an item is selected.
	CloseOnSelect bool
	// OnChange reports a changed single selection; "" means no selection.
	// Choosing the already-selected value does not report a change.
	OnChange func(string)
	// OnValuesChange reports multiple selections as item values.
	OnValuesChange func([]string)
	// InputAttrs are additional attributes for the search input, for
	// example templ.Attributes{"aria-label": "Choose a country"} or
	// aria-labelledby. The component owns id, class, type, value,
	// placeholder, disabled, role, autocomplete, autocorrect, spellcheck,
	// aria-autocomplete, aria-expanded, aria-controls,
	// aria-activedescendant, data-on, data-keys and its handlers;
	// repeating them is a fault.
	InputAttrs templ.Attributes
	// Class adds classes to the root.
	Class string
}

// ComboboxState holds the search, the open list and its highlighted item.
// Keep one state per combobox instance.
type ComboboxState struct {
	open            bool
	text            string
	query           string // the filter the script last applied, retained on close
	editing         bool
	lastValue       string
	seenValue       bool
	clearedByInput  bool
	changedByChoice bool
	active          int
	moved           bool
	focus           bool
	focusAt         uint64
}

// Combobox is Basecoat's searchable selection field. Typing filters its
// items by label or keyword, without case; arrows, Home and End move the highlight,
// Enter chooses it, Escape closes the list, and a click chooses an item.
// In multiple mode selected items become removable chips. The app keeps
// the chosen value or values. Single mode uses Value/OnChange; typing after
// a selection first reports "". Multiple mode uses Values/OnValuesChange;
// selecting or removing a chip reports the resulting values. Each mode
// ignores the other's fields. A nil callback changes no app-owned value.
func Combobox(st *ComboboxState, p ComboboxProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return combobox(newComboboxView(st, p)).Render(ctx, w)
	})
}

type comboboxView struct {
	st      *ComboboxState
	p       ComboboxProps
	id      string
	entries []listEntry
	opts    []listOpt
	shown   []bool
	visible []int
	n       uint64
	popups  *popup
}

func newComboboxView(st *ComboboxState, p ComboboxProps) *comboboxView {
	if !p.Multiple && st.seenValue && p.Value != st.lastValue && !(st.clearedByInput && p.Value == "") && !st.changedByChoice {
		st.text, st.query, st.editing = "", "", false
	}
	st.lastValue, st.seenValue, st.clearedByInput, st.changedByChoice = p.Value, true, false, false
	v := &comboboxView{st: st, p: p, id: or(p.ID, "combobox")}
	v.entries, v.opts = grouped(v.id+"-items", len(p.Items), func(i int) string { return p.Items[i].Group }, func(i int, id string) listOpt {
		it := p.Items[i]
		return listOpt{id: id, value: it.Value, label: it.Label, disabled: it.Disabled}
	})
	if p.Disabled {
		st.open = false
		st.moved = false
	}
	search := strings.ToLower(strings.TrimSpace(v.query()))
	v.shown = make([]bool, len(v.opts))
	for i, o := range v.opts {
		v.shown[i] = search == "" || strings.Contains(strings.ToLower(strings.TrimSpace(o.label)), search)
		if !v.shown[i] {
			for _, word := range strings.FieldsFunc(strings.ToLower(p.Items[i].Keywords), func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
				if strings.Contains(word, search) {
					v.shown[i] = true
					break
				}
			}
		}
		if v.shown[i] && !o.disabled {
			v.visible = append(v.visible, i)
		}
	}
	if st.moved && !slices.Contains(v.visible, st.active) {
		st.moved = false
	}
	return v
}

func (v *comboboxView) inputAttrs(ctx context.Context) templ.Attributes {
	owned := []string{"id", "class", "type", "value", "placeholder", "disabled", "role", "autocomplete", "autocorrect", "spellcheck", "aria-autocomplete", "aria-expanded", "aria-controls", "aria-activedescendant", "data-on", "data-keys"}
	checkAttrs(ctx, "Combobox", v.p.InputAttrs, owned...)
	a := make(templ.Attributes, len(v.p.InputAttrs)+4)
	for k, value := range v.p.InputAttrs {
		key := strings.ToLower(k)
		if strings.HasPrefix(key, "on") {
			rt.Fault(ctx, fmt.Sprintf("ui.Combobox: InputAttrs cannot install handler %q", k))
			continue
		}
		if strings.HasPrefix(key, "data-hotty-") {
			rt.Fault(ctx, fmt.Sprintf("ui.Combobox: InputAttrs cannot replace internal attribute %q", k))
			continue
		}
		if !slices.Contains(owned, key) {
			a[k] = value
		}
	}
	for k, value := range scopeCommandAttrs(ctx, surf.On(ctx, v.id+"-input", surf.Input(v.input), surf.Focus(v.focus), surf.Click(v.open)), v.fieldKeys()) {
		a[k] = value
	}
	return a
}
func (v *comboboxView) join(ctx context.Context) string {
	seen := map[string]bool{}
	for _, item := range v.p.Items {
		if seen[item.Value] {
			rt.Fault(ctx, fmt.Sprintf("ui.Combobox: duplicate item value %q", item.Value))
		}
		seen[item.Value] = true
	}
	v.n = frameN(ctx)
	v.popups = popups(ctx)
	v.popups.track(ctx, v.st, v.id, v.st.open, v.shut)
	return ""
}
func (v *comboboxView) shut() { v.st.open = false; v.st.moved = false }
func (v *comboboxView) selected(value string) bool {
	if v.p.Multiple {
		return slices.Contains(v.p.Values, value)
	}
	return value != "" && v.p.Value == value
}
func (v *comboboxView) query() string {
	return v.st.query
}
func (v *comboboxView) fieldValue() string {
	if v.p.Multiple || v.st.editing {
		return v.st.text
	}
	for _, o := range v.opts {
		if o.value != "" && o.value == v.p.Value {
			return o.label
		}
	}
	return ""
}
func (v *comboboxView) active() int {
	if !v.st.open {
		return -1
	}
	if v.st.moved && slices.Contains(v.visible, v.st.active) {
		return v.st.active
	}
	if v.p.AutoHighlight && len(v.visible) > 0 {
		return v.visible[0]
	}
	return -1
}
func (v *comboboxView) open() {
	if v.p.Disabled || v.st.open {
		return
	}
	v.st.open = true
	if v.p.Multiple || v.st.editing {
		v.st.query = v.st.text
	} else {
		v.st.query = ""
	}
	v.st.moved = false
	v.popups.opened(v.st, v.shut)
}
func (v *comboboxView) close(focus bool) {
	if !v.st.open {
		return
	}
	v.shut()
	v.popups.closed(v.st)
	if focus {
		v.ask()
	}
}
func (v *comboboxView) ask()          { v.st.focus = true; v.st.focusAt = v.n + 1 }
func (v *comboboxView) refocus() bool { return oneShot(&v.st.focus, v.st.focusAt, v.n) }
func (v *comboboxView) focus(in bool) {
	if in {
		v.open()
	}
}
func (v *comboboxView) input(text string) {
	v.st.text = text
	v.st.query = text
	v.st.editing = true
	v.st.moved = false
	v.open()
	if !v.p.Multiple && v.p.Value != "" && v.p.OnChange != nil {
		v.st.clearedByInput = true
		v.p.OnChange("")
	}
}
func (v *comboboxView) outside() { v.close(false) }
func (v *comboboxView) rootAttrs(ctx context.Context) templ.Attributes {
	if v.st.open {
		return kitOn(ctx, v.id, surf.Outside(v.outside), surf.Key(v.key))
	}
	return kitOn(ctx, v.id, surf.Key(v.key))
}
func (v *comboboxView) choose(i int) {
	if i < 0 || i >= len(v.opts) || v.opts[i].disabled {
		return
	}
	o := v.opts[i]
	if v.p.Multiple {
		values := slices.Clone(v.p.Values)
		if j := slices.Index(values, o.value); j >= 0 {
			values = slices.Delete(values, j, j+1)
		} else {
			values = append(values, o.value)
		}
		if v.p.OnValuesChange != nil {
			v.p.OnValuesChange(values)
		}
		v.st.text = ""
		v.st.query = ""
		v.st.editing = true
		if v.p.CloseOnSelect {
			v.close(true)
		} else {
			v.ask()
		}
	} else {
		if v.p.OnChange != nil && v.p.Value != o.value {
			v.st.changedByChoice = true
			v.p.OnChange(o.value)
		}
		v.st.text = ""
		v.st.editing = false
		v.close(true)
		v.ask()
	}
}
func (v *comboboxView) clear() {
	if v.p.Multiple {
		if v.p.OnValuesChange != nil {
			v.p.OnValuesChange(nil)
		}
	} else if v.p.OnChange != nil {
		v.p.OnChange("")
	}
	v.st.text = ""
	v.st.query = ""
	v.st.editing = true
	v.st.moved = false
	v.ask()
}
func (v *comboboxView) remove(value string) {
	if i := slices.Index(v.p.Values, value); i >= 0 {
		values := slices.Clone(v.p.Values)
		values = slices.Delete(values, i, i+1)
		if v.p.OnValuesChange != nil {
			v.p.OnValuesChange(values)
		}
		v.ask()
	}
}
func (v *comboboxView) hover(i int) func(bool) {
	return func(over bool) {
		if over && v.st.open && slices.Contains(v.visible, i) {
			v.st.active = i
			v.st.moved = true
		}
	}
}
func (v *comboboxView) key(k string) bool {
	if k == "escape" && v.st.open {
		v.close(true)
		return true
	}
	if k == "backspace" && v.p.Multiple && v.st.text == "" && len(v.p.Values) > 0 {
		v.remove(v.p.Values[len(v.p.Values)-1])
		return true
	}
	if !v.st.open {
		switch k {
		case "arrow_down", "arrow_up", "home", "end":
			v.open()
		default:
			return false
		}
	}
	if k == "enter" {
		if i := v.active(); i >= 0 {
			v.choose(i)
		}
		return true
	}
	if len(v.visible) == 0 {
		return false
	}
	pos := slices.Index(v.visible, v.active())
	next := pos
	switch k {
	case "arrow_down":
		next = min(pos+1, len(v.visible)-1)
	case "arrow_up":
		next = max(pos-1, 0)
	case "home":
		next = 0
	case "end":
		next = len(v.visible) - 1
	default:
		return false
	}
	v.st.active = v.visible[next]
	v.st.moved = true
	return true
}
func (v *comboboxView) fieldKeys() string {
	keys := "enter=program"
	if len(v.visible) > 0 {
		keys += " arrow_down=program arrow_up=program home=program end=program"
	}
	if v.p.Multiple && v.st.text == "" && len(v.p.Values) > 0 {
		keys += " backspace=program"
	}
	return keys
}

// hiddenValue is the macro's value-format hidden input.
func (v *comboboxView) hiddenValue() string {
	if !v.p.Multiple {
		return v.p.Value
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, value := range v.p.Values {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONText(&b, value)
	}
	b.WriteByte(']')
	return b.String()
}
func (v *comboboxView) label(value string) string {
	for _, item := range v.p.Items {
		if item.Value == value {
			return item.Label
		}
	}
	return value
}
func (v *comboboxView) removeClick(value string) func() { return func() { v.remove(value) } }
func (v *comboboxView) chipID(i int) string             { return v.id + "-chip-" + strconv.Itoa(i) }
func (v *comboboxView) selectClick(i int) func()        { return func() { v.choose(i) } }

func writeJSONText(b *strings.Builder, value string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte(hex[byte(r)>>4])
				b.WriteByte(hex[byte(r)&15])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

func (v *comboboxView) toggle() {
	if v.st.open {
		v.close(false)
	} else {
		v.open()
		v.ask()
	}
}
