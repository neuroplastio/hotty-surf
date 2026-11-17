package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// ListProps are a List's attributes.
type ListProps struct {
	// OnChange requests selection of the item at this zero-based index.
	// It runs on the surface driver; the app supplies the new Selected
	// value on its next render. nil leaves selection to the app.
	OnChange func(index int)
	// Attrs are attributes of the listbox, including its accessible
	// name, e.g. templ.Attributes{"aria-label": "Sessions"} or
	// aria-labelledby, and any surf.On handlers. The component owns
	// role and class; repeating either is a fault. The list is single
	// selection and owns aria-multiselectable.
	Attrs templ.Attributes
	// Class adds classes after the list's own.
	Class string
}

// List is a single-selection listbox of ListItem children. The app owns
// selection: it sets Selected on at most one item and applies OnChange
// by changing its state. OnChange receives the zero-based index of the
// item in rendered ListItem order; non-item children do not count.
//
// Tab enters at the selected enabled item, or the first enabled item
// if none is selected. Up and Down move one enabled item; Home and End
// move to the first and last. Movement stops at either end, requests
// selection through OnChange, and moves focus to that item. A click
// requests selection the same way. Enter and Space activate the focused
// item through its surf.Click handler. Disabled items are skipped.
// Tab and Shift+Tab leave the list by the host's focus traversal.
//
// KeyScope can take a named key before this default. A declined key
// keeps the default, and a key the list does not take goes to the app.
// The list does not require OnChange: without it navigation moves focus
// but the app's Selected values do not change.
func List(p ListProps) templ.Component {
	return scoped(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		checkAttrs(ctx, "List", p.Attrs, "role", "class", "aria-multiselectable")
		if !hasName(p.Attrs) {
			rt.Fault(ctx, "ui.List: Attrs must give the list an accessible name (aria-label or aria-labelledby)")
		}
		v := &listView{p: p}
		v.tag = fmt.Sprintf("ui-list-%p-", v)
		id := attrString(p.Attrs, "id")
		v.base = or(id, "list")
		if f := rt.From(ctx); f != nil {
			k := "ui.list:" + v.base
			v.memo, _ = f.Memo[k].(*listMemo)
			if v.memo == nil {
				v.memo = &listMemo{}
				f.Memo[k] = v.memo
			}
		} else {
			v.memo = &listMemo{}
		}
		// The items' tab stop is known once all of them have rendered:
		// each writes a placeholder, filled in below.
		var body bytes.Buffer
		if kids := templ.GetChildren(ctx); kids != nil {
			if err := kids.Render(context.WithValue(templ.ClearChildren(ctx), listKey{}, v), &body); err != nil {
				return err
			}
		}
		if n := v.selectedCount(); n > 1 {
			rt.Fault(ctx, fmt.Sprintf("ui.List: %d items are Selected (a List selects at most one)", n))
		}
		stop := v.stop()
		out := body.String()
		for i := range v.items {
			val := "-1"
			if i == stop {
				val = "0"
			}
			out = strings.Replace(out, `tabindex="`+v.tag+strconv.Itoa(i)+`"`, `tabindex="`+val+`"`, 1)
		}
		if ask := v.memo.focus; ask != "" {
			v.memo.focus = ""
			if i := v.index(ask); i >= 0 && !v.items[i].disabled {
				surf.Autofocus(ctx, ask)
			}
		}
		return listBox(p, out).Render(ctx, w)
	}), false)
}

// listKey is the context key of the List an item renders in.
type listKey struct{}

// listView is one render of a List: its items in render order.
type listView struct {
	p     ListProps
	tag   string // the items' tabindex placeholder, before their index
	base  string // the list's id, or "list": its memo and generated item ids
	memo  *listMemo
	items []listRow
}

type listRow struct {
	id                 string
	selected, disabled bool
}

// listMemo is what a List keeps between renders, by its id.
type listMemo struct {
	current string // the item the keyboard or a click last reached
	focused string // the item that has the keyboard, by its Focus events
	focus   string // the item to give the keyboard at the next render
}

func (v *listView) index(id string) int {
	for i, it := range v.items {
		if it.id == id {
			return i
		}
	}
	return -1
}

func (v *listView) selectedCount() int {
	n := 0
	for _, it := range v.items {
		if it.selected {
			n++
		}
	}
	return n
}

func (v *listView) selectedIndex() int {
	for i, it := range v.items {
		if it.selected {
			return i
		}
	}
	return -1
}

// stop is the item Tab reaches: where the keyboard or a click last was,
// else the selected enabled item, else the first enabled one; -1 with
// none enabled.
func (v *listView) stop() int {
	if i := v.index(v.memo.current); i >= 0 && !v.items[i].disabled {
		return i
	}
	if i := v.selectedIndex(); i >= 0 && !v.items[i].disabled {
		return i
	}
	for i, it := range v.items {
		if !it.disabled {
			return i
		}
	}
	return -1
}

// choose requests selection of item i, unless it is selected already.
func (v *listView) choose(i int) {
	if v.items[i].disabled || v.p.OnChange == nil || i == v.selectedIndex() {
		return
	}
	v.p.OnChange(i)
}

// key is item i's navigation: it answers only while i has the keyboard,
// so that with nothing focused the items don't all move at once.
func (v *listView) key(i int) func(string) bool {
	return func(k string) bool {
		if v.memo.focused != v.items[i].id {
			return false
		}
		var on []int
		cur := -1
		for j, it := range v.items {
			if it.disabled {
				continue
			}
			if j == i {
				cur = len(on)
			}
			on = append(on, j)
		}
		if cur < 0 {
			return false
		}
		var next int
		switch k {
		case "arrow_down":
			next = on[min(cur+1, len(on)-1)]
		case "arrow_up":
			next = on[max(cur-1, 0)]
		case "home":
			next = on[0]
		case "end":
			next = on[len(on)-1]
		default:
			return false
		}
		if next == i {
			return true // at an end: taken, nothing to request
		}
		v.choose(next)
		// The keyboard goes there at the next render (Autofocus), which
		// brings no Focus event of its own.
		v.memo.current, v.memo.focus, v.memo.focused = v.items[next].id, v.items[next].id, v.items[next].id
		return true
	}
}

// listItemKeys are the keys a List's item sends to the program.
const listItemKeys = "arrow_up=program arrow_down=program home=program end=program"

// listItemAttrs are a ListItem's button attributes: the app's, with the
// item's id (one of its own when the app gives none), its tabindex
// placeholder, and the List's click, focus and keys joined to the
// handlers the app registered for that id.
func listItemAttrs(ctx context.Context, p ListItemProps) templ.Attributes {
	checkAttrs(ctx, "ListItem", p.Attrs, "type", "role", "aria-selected", "tabindex", "class")
	a := make(templ.Attributes, len(p.Attrs)+3)
	for k, val := range p.Attrs {
		a[k] = val
	}
	v, _ := ctx.Value(listKey{}).(*listView)
	if v == nil {
		rt.Fault(ctx, "ui.ListItem: outside a List (an option needs its listbox)")
		a["tabindex"] = "-1"
		return a
	}
	i := len(v.items)
	id := attrString(p.Attrs, "id")
	disabled := attrOn(p.Attrs, "disabled")
	if id == "" {
		id = v.base + "-item-" + strconv.Itoa(i)
		a["id"] = id
	}
	v.items = append(v.items, listRow{id: id, selected: p.Selected, disabled: disabled})
	a["tabindex"] = v.tag + strconv.Itoa(i)
	f := rt.From(ctx)
	if f == nil {
		return a
	}
	h := f.On[id]
	if h == nil {
		kitOn(ctx, id)
		h = f.On[id]
	}
	if disabled {
		h.Click = nil
		return a
	}
	m := v.memo
	click := h.Click
	h.Click = func() {
		v.choose(i)
		m.current = id
		if click != nil {
			click()
		}
	}
	focus := h.Focus
	h.Focus = func(in bool) {
		if in {
			m.focused, m.current = id, id
		} else if m.focused == id {
			m.focused = ""
		}
		if focus != nil {
			focus(in)
		}
	}
	if len(h.Key) == 0 {
		h.KitKey = true
	}
	h.Key = append([]func(string) bool{v.key(i)}, h.Key...)
	a["data-on"] = addWord(attrString(a, "data-on"), "click")
	keys := listItemKeys
	if old := attrString(a, "data-keys"); old != "" {
		keys = old + " " + keys
	}
	return scopeWithKeys(ctx, a, keys)
}

// attrString is attrs' value for name, whatever its case ("" for none).
func attrString(attrs templ.Attributes, name string) string {
	for k, val := range attrs {
		if strings.EqualFold(k, name) {
			s, _ := val.(string)
			return s
		}
	}
	return ""
}

// attrOn is whether attrs sets the boolean attribute name.
func attrOn(attrs templ.Attributes, name string) bool {
	for k, val := range attrs {
		if strings.EqualFold(k, name) {
			return val != nil && val != false
		}
	}
	return false
}

// hasName is whether attrs give an element an accessible name.
func hasName(attrs templ.Attributes) bool {
	return strings.TrimSpace(attrString(attrs, "aria-label")) != "" ||
		strings.TrimSpace(attrString(attrs, "aria-labelledby")) != ""
}

// addWord adds w to a space-separated list, once.
func addWord(list, w string) string {
	for _, x := range strings.Fields(list) {
		if x == w {
			return list
		}
	}
	return strings.TrimSpace(list + " " + w)
}
