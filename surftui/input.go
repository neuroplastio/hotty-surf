package surftui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyedit"
	"neuroplast.io/hotty-surf/internal/core/keydefault"
)

// What the user does, as a host has it done (SPEC §9, §10): focus, keys,
// fields, clicks, and the events they send.

// field is what a control holds once shown, which the user changes and
// a=field sets (SPEC §6.4): a text field's value and caret, a box's state,
// a select's value; and its rev, one more for each change of the user's.
type field struct {
	ed        hottyedit.Field
	checked   bool
	rev       int
	committed string // the value the last change said, or the markup's
	off       int    // an input's first column shown
	top       int    // a textarea's first row shown
	scrolled  bool   // user scrolled textarea via wheel; cleared on next key/edit
	rows      int    // user-resized textarea height (0 = default)
	cols      int    // user-resized textarea width (0 = default)
	def       string // the markup's default when last read (defaultOf)
}

// defaultOf is what n's markup sets a control to: a box's checked, a
// select's selected option, a field's value (SPEC §6.2).
func defaultOf(n *html.Node) string {
	switch {
	case isCheck(n):
		return strconv.FormatBool(has(n, "checked"))
	case n.DataAtom == atom.Select:
		return selected(n)
	case n.DataAtom == atom.Textarea:
		var b strings.Builder
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.TextNode {
				b.WriteString(k.Data)
			}
		}
		return b.String()
	}
	return strings.NewReplacer("\r", "", "\n", "").Replace(attr(n, "value"))
}

// field is n's state, made from its markup the first time it is asked for.
func (c *Screen) field(n *html.Node) *field {
	if f := c.fields[n]; f != nil {
		return f
	}
	f := &field{}
	switch {
	case isCheck(n):
		f.checked = has(n, "checked")
	case n.DataAtom == atom.Select:
		f.ed.Value = selected(n)
	case n.DataAtom == atom.Textarea:
		f.ed.Multiline = true
		var b strings.Builder
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.TextNode {
				b.WriteString(k.Data)
			}
		}
		f.ed.Value = b.String()
	default:
		f.ed.Value = strings.NewReplacer("\r", "", "\n", "").Replace(attr(n, "value"))
		f.ed.Password = strings.EqualFold(attr(n, "type"), "password")
	}
	f.ed.Caret = len(graphemesOf(f.ed.Value))
	f.committed = f.ed.Value
	f.def = defaultOf(n)
	c.fields[n] = f
	return f
}

// isField is a control (SPEC §6.4): an input of a type that holds a value,
// a textarea, a select.
func isField(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Textarea, atom.Select:
		return true
	case atom.Input:
		switch strings.ToLower(attr(n, "type")) {
		case "hidden", "button", "submit", "reset", "image", "file":
			return false
		}
		return true
	}
	return false
}

// isTextField is a field that edits text (SPEC §10.2).
func isTextField(n *html.Node) bool {
	if n == nil {
		return false
	}
	switch n.DataAtom {
	case atom.Textarea:
		return true
	case atom.Input:
		switch strings.ToLower(attr(n, "type")) {
		case "hidden", "checkbox", "radio", "file", "submit", "image", "reset", "button", "color", "range":
			return false
		}
		return true
	}
	return false
}

func isCheck(n *html.Node) bool {
	t := strings.ToLower(attr(n, "type"))
	return n.DataAtom == atom.Input && (t == "checkbox" || t == "radio")
}

// disabled is a control that is disabled, or in a disabled fieldset.
func disabled(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Button, atom.Input, atom.Select, atom.Textarea, atom.Option:
	default:
		return false
	}
	if has(n, "disabled") {
		return true
	}
	for child, parent := n, n.Parent; parent != nil; child, parent = parent, parent.Parent {
		if parent.DataAtom != atom.Fieldset || !has(parent, "disabled") {
			continue
		}
		var legend *html.Node
		for el := parent.FirstChild; el != nil; el = el.NextSibling {
			if el.DataAtom == atom.Legend {
				legend = el
				break
			}
		}
		if child != legend {
			return true
		}
	}
	return false
}

// focusable is an element a click focuses (SPEC §10.1).
func focusable(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode || disabled(n) || closest(n, func(el *html.Node) bool { return has(el, "inert") }) != nil {
		return false
	}
	switch n.DataAtom {
	case atom.Button, atom.Select, atom.Textarea:
		return true
	case atom.Input:
		return !strings.EqualFold(attr(n, "type"), "hidden")
	case atom.A:
		if has(n, "href") && attr(n, "target") != "_blank" {
			return true
		}
	case atom.Summary:
		if n.Parent != nil && n.Parent.DataAtom == atom.Details {
			for k := n.Parent.FirstChild; k != nil; k = k.NextSibling {
				if k.DataAtom == atom.Summary {
					return k == n
				}
			}
		}
	}
	if v, ok := get(n, "contenteditable"); ok && !strings.EqualFold(v, "false") {
		return true
	}
	_, err := strconv.Atoi(strings.TrimSpace(attr(n, "tabindex")))
	return err == nil
}

// tabbable is a focusable element Tab goes to: one without a negative
// tabindex.
func tabbable(n *html.Node) bool {
	if !focusable(n) {
		return false
	}
	v, err := strconv.Atoi(strings.TrimSpace(attr(n, "tabindex")))
	return err != nil || v >= 0
}

// activatable is an element space and enter click (SPEC §10.2): a
// button, a link, a summary, an input that is a button. An element that
// only reports clicks (data-on~=click) takes no keys.
func activatable(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Button, atom.A, atom.Summary:
		return true
	case atom.Input:
		switch strings.ToLower(attr(n, "type")) {
		case "button", "submit", "reset", "image":
			return true
		}
	}
	return false
}

// clickable is an element that reports a click (SPEC §9).
func clickable(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Button, atom.A, atom.Summary:
		return true
	case atom.Input:
		switch strings.ToLower(attr(n, "type")) {
		case "button", "submit", "reset", "image":
			return true
		}
	}
	return tokens(n, "data-on", "click")
}

// keymaps are the data-keys values from the root down to el (SPEC §10.2).
func keymaps(el *html.Node) []string {
	var vs []string
	for n := el; n != nil; n = n.Parent {
		if v, ok := get(n, "data-keys"); ok {
			vs = append(vs, v)
		}
	}
	slices.Reverse(vs)
	return vs
}

type option struct {
	value, label string
	disabled     bool
}

func options(sel *html.Node) []option {
	var out []option
	walk(sel, func(o *html.Node) {
		if o.DataAtom != atom.Option {
			return
		}
		v, ok := get(o, "value")
		if !ok {
			v = textOf(o)
		}
		label, ok := get(o, "label")
		if !ok {
			label = textOf(o)
		}
		out = append(out, option{v, label, has(o, "disabled")})
	})
	return out
}

// selected is a select's value as its markup gives it.
func selected(sel *html.Node) string {
	var first, pick *html.Node
	walk(sel, func(o *html.Node) {
		if o.DataAtom != atom.Option {
			return
		}
		if first == nil {
			first = o
		}
		if pick == nil && has(o, "selected") {
			pick = o
		}
	})
	if pick == nil {
		pick = first
	}
	if pick == nil {
		return ""
	}
	if v, ok := get(pick, "value"); ok {
		return v
	}
	return textOf(pick)
}

// emit queues an event for the program, sent once the act is done.
func (c *Screen) emit(e hotty.Event) {
	e.Surface = c.name
	c.queue = append(c.queue, e)
}

// flush sends the queued events, in order. An event's callback may act on
// the Screen (write the surface's delta): what that queues is sent after
// it, by the same loop.
func (c *Screen) flush() {
	if c.flushing {
		return
	}
	c.flushing = true
	defer func() { c.flushing = false }()
	for len(c.queue) > 0 {
		e := c.queue[0]
		c.queue = c.queue[1:]
		if c.events != nil {
			c.events(e)
		}
	}
}

// area is n's cells, as an event reports them (SPEC §9).
func (c *Screen) area(n *html.Node) *hotty.Area {
	r, ok := c.layout().boxes[n]
	if !ok {
		return nil
	}
	return &hotty.Area{Col: r.x, Row: r.y - c.scroll, W: r.w, H: r.h}
}

// focusTo moves the keyboard to n. The user's move is told (focus); the
// program's (a=focus) is not (SPEC §10.1). A text field it leaves commits.
func (c *Screen) focusTo(n *html.Node, user bool) {
	c.prefixed = false
	if n == c.focused {
		return
	}
	if c.focused != nil {
		c.commit(c.focused)
	}
	c.focused, c.remembered = n, nil
	c.lay = nil
	c.revealAfter, c.revealWhere = n, hotty.RevealNearest
	if user {
		e := hotty.Event{Kind: hotty.EventFocus}
		if t := withID(n); t != nil {
			e.Target = attr(t, "id")
		}
		c.emit(e)
	}
}

// fixFocus commits a field still in the document and clears element focus.
func (c *Screen) fixFocus() {
	c.commit(c.focused)
	c.focused, c.prefixed, c.lay = nil, false, nil
}

// blur takes the keyboard from the focused element: it commits, and the
// surface hears blur.
func (c *Screen) blur() {
	if c.focused == nil {
		return
	}
	c.commit(c.focused)
	c.focused = nil
	c.lay = nil
	c.emit(hotty.Event{Kind: hotty.EventBlur})
}

// commit sends a text field's change, when its value isn't what the last
// change said (SPEC §9).
func (c *Screen) commit(n *html.Node) {
	if !isTextField(n) {
		return
	}
	f := c.field(n)
	if f.ed.Value == f.committed {
		return
	}
	f.committed = f.ed.Value
	if id := attr(n, "id"); id != "" {
		c.emit(hotty.Event{Kind: hotty.EventChange, Target: id, Value: f.ed.Value, Rev: f.rev})
	}
}

// edited is an edit of the user's: one more rev, and input where the field
// asks for it (data-on~=input).
func (c *Screen) edited(n *html.Node, f *field) {
	f.rev++
	c.lay = nil
	if id := attr(n, "id"); id != "" && tokens(n, "data-on", "input") {
		e := hotty.Event{Kind: hotty.EventInput, Target: id, Value: f.ed.Value, Rev: f.rev}
		if tokens(n, "data-on", "caret") {
			sel := fieldSelection(f)
			e.Sel = &sel
		}
		c.emit(e)
	}
}

// fieldSelection converts the editor's grapheme positions to the byte
// offsets carried by hotty field events.
func fieldSelection(f *field) [2]int {
	anchor, _ := f.ed.Anchor()
	bytesAt := func(index int) int {
		off := 0
		for i, g := range graphemesOf(f.ed.Value) {
			if i >= index {
				break
			}
			off += len(g)
		}
		return off
	}
	return [2]int{bytesAt(anchor), bytesAt(f.ed.Caret)}
}

// group is a radio button's group: the radios of its name in its form.
func (c *Screen) group(n *html.Node) []*html.Node {
	name := attr(n, "name")
	if name == "" || !strings.EqualFold(attr(n, "type"), "radio") {
		return []*html.Node{n}
	}
	form := closest(n, func(m *html.Node) bool { return m.DataAtom == atom.Form })
	var out []*html.Node
	walk(c.body, func(m *html.Node) {
		if m.DataAtom == atom.Input && strings.EqualFold(attr(m, "type"), "radio") && attr(m, "name") == name &&
			closest(m, func(x *html.Node) bool { return x.DataAtom == atom.Form }) == form {
			out = append(out, m)
		}
	})
	return out
}

// check sets a box, unchecking the rest of a radio's group.
func (c *Screen) check(n *html.Node, on bool) {
	if on {
		for _, m := range c.group(n) {
			c.field(m).checked = false
		}
	}
	c.field(n).checked = on
	c.lay = nil
}

// toggle is the user's check: a checkbox flips, a radio is chosen; change
// says so (SPEC §9).
func (c *Screen) toggle(n *html.Node) {
	f := c.field(n)
	on, changed := keydefault.Check(f.checked, strings.EqualFold(attr(n, "type"), "radio"))
	if !changed {
		return
	}
	c.check(n, on)
	rev := 0
	for _, m := range c.group(n) {
		rev = max(rev, c.field(m).rev)
	}
	for _, m := range c.group(n) {
		c.field(m).rev = rev + 1
	}
	if id := attr(n, "id"); id != "" {
		v := attr(n, "value")
		if !has(n, "value") {
			v = "on"
		}
		c.emit(hotty.Event{Kind: hotty.EventChange, Target: id, Checked: &on, Value: v, Rev: rev + 1})
	}
}

// pick is the user's pick of a select's option: input (data-on~=input),
// then change, at once.
func (c *Screen) pick(n *html.Node, v string) {
	f := c.field(n)
	f.ed.Value = v
	f.committed = v
	f.rev++
	c.lay = nil
	id := attr(n, "id")
	if id == "" {
		return
	}
	if tokens(n, "data-on", "input") {
		c.emit(hotty.Event{Kind: hotty.EventInput, Target: id, Value: v, Rev: f.rev})
	}
	c.emit(hotty.Event{Kind: hotty.EventChange, Target: id, Value: v, Rev: f.rev})
}

// activate is a click on an element that reports one, by the pointer or
// the keyboard: a summary opens its details first, a submit button
// submits its form after.
func (c *Screen) activate(n *html.Node) { c.activateBy(n, 1, 0) }

// activateBy is activate with the click's count in its run and its mods.
func (c *Screen) activateBy(n *html.Node, count, mods int) {
	if n.DataAtom == atom.Summary && n.Parent != nil && n.Parent.DataAtom == atom.Details {
		if has(n.Parent, "open") {
			unset(n.Parent, "open")
		} else {
			set(n.Parent, "open", "")
		}
		c.lay, c.kept = nil, nil
	}
	e := hotty.Event{Kind: hotty.EventClick, Count: count, Mods: mods, Area: c.area(n)}
	if v, ok := get(n, "value"); ok {
		e.Value = v
	}
	switch id := attr(n, "id"); {
	case id != "":
		e.Target = id
		if n.DataAtom == atom.A && has(n, "href") {
			e.Link = &hotty.Link{Href: attr(n, "href")}
		}
		c.emit(e)
	case n.DataAtom == atom.A && has(n, "href"):
		e.Link = &hotty.Link{Href: attr(n, "href")}
		c.emit(e)
	}
	typ := strings.ToLower(attr(n, "type"))
	if n.DataAtom == atom.Button && (typ == "" || typ == "submit") || n.DataAtom == atom.Input && (typ == "submit" || typ == "image") {
		if form := closest(n, func(m *html.Node) bool { return m.DataAtom == atom.Form }); form != nil {
			c.submit(form, n)
		}
	}
}

// submit sends a form's fields (SPEC §9).
func (c *Screen) submit(form, by *html.Node) {
	e := hotty.Event{Kind: hotty.EventSubmit, Fields: []hotty.FormField{}}
	if id := attr(form, "id"); id != "" {
		e.Target = id
	}
	walk(form, func(n *html.Node) {
		name := attr(n, "name")
		if name == "" || disabled(n) {
			return
		}
		switch {
		case n == by:
			e.Fields = append(e.Fields, hotty.FormField{Name: name, Value: attr(n, "value")})
		case isCheck(n):
			if c.field(n).checked {
				v, ok := get(n, "value")
				if !ok {
					v = "on"
				}
				e.Fields = append(e.Fields, hotty.FormField{Name: name, Value: v})
			}
		case isField(n):
			e.Fields = append(e.Fields, hotty.FormField{Name: name, Value: c.field(n).ed.Value})
		case n.DataAtom == atom.Input && strings.EqualFold(attr(n, "type"), "hidden"):
			e.Fields = append(e.Fields, hotty.FormField{Name: name, Value: attr(n, "value")})
		}
	})
	c.emit(e)
}

// splitKey is a canonical key name's modifiers and its key.
func splitKey(k string) (mods []string, name string) {
	for {
		i := strings.IndexByte(k, '+')
		if i <= 0 || i == len(k)-1 {
			return mods, k
		}
		switch m := k[:i]; m {
		case "ctrl", "alt", "shift", "super", "mod":
			mods = append(mods, m)
			k = k[i+1:]
		default:
			return mods, k
		}
	}
}

// key does what a host does with a key while the surface has the keyboard
// (SPEC §10.2), and says whether it used it.
func (c *Screen) key(key string) bool {
	if c.body == nil || !c.active {
		return false
	}
	if c.focused != nil {
		c.layout()
	}
	k, ok := c.plat.ReadKey(key)
	if !ok || c.prefixed {
		// After a prefix every key is the program's, Tab and Escape too,
		// until it ends the sequence (SPEC §10.5).
		return false
	}
	if c.focused != nil && hotty.ParseKeymap(c.plat, strings.Join(keymaps(c.focused), " ")).Prefix(key) {
		c.prefixed = true
		return false
	}
	mods, name := splitKey(k)
	shiftOnly := true
	for _, m := range mods {
		if m != "shift" {
			shiftOnly = false
		}
	}
	el := c.focused
	if name == "tab" && shiftOnly {
		if len(mods) == 0 && isTextField(el) && c.fieldKeymap(el).Program(key) {
			return false
		}
		return c.tab(len(mods) > 0)
	}
	if el == nil {
		// No element focused, no keymap: every key is the program's.
		return false
	}
	if isTextField(el) {
		return c.fieldKey(el, key)
	}
	km := hotty.ParseKeymap(c.plat, strings.Join(keymaps(el), " "))
	if km.Program(key) {
		return false
	}
	if shiftOnly {
		switch {
		case isCheck(el):
			switch name {
			case "space":
				c.toggle(el)
				return true
			case "enter":
				if form := closest(el, func(m *html.Node) bool { return m.DataAtom == atom.Form }); form != nil {
					c.submit(form, nil)
				}
				return true
			}
		case el.DataAtom == atom.Select:
			if c.selectKey(el, key, name) {
				return true
			}
		case activatable(el):
			if name == "space" || name == "enter" {
				c.activate(el)
				return true
			}
		}
	}
	if a := km.Scroll(key); a != "" {
		c.scrollBy(a)
		return true
	}
	return false
}

func (c *Screen) fieldKeymap(el *html.Node) *hotty.Keymap {
	return hotty.Resolve(c.plat, el.DataAtom == atom.Textarea, keymaps(el)...)
}

// fieldKey gives a key to a text field, which does what its keymap says.
// The Screen has no clipboard: copy, cut and paste are the program's (it
// pastes with Paste, and copies and cuts with Selection and Cut).
//
// A readonly field takes the keys that would edit it, and does nothing
// with them; it moves its caret and selects. A disabled one takes nothing.
func (c *Screen) fieldKey(el *html.Node, key string) bool {
	if disabled(el) {
		return false
	}
	f := c.field(el)
	km := c.fieldKeymap(el)
	if has(el, "readonly") {
		switch a := km.Lookup(key); {
		case a == "":
			return false
		case a.Moves() || a == hotty.SelectAll || a == hotty.Copy || a == hotty.Submit:
		default:
			return true
		}
	}
	before := fieldSelection(f)
	f.scrolled = false
	a, changed := f.ed.Key(km, key)
	switch {
	case a == "":
		return false
	case a == hotty.Submit:
		c.commit(el)
		if form := closest(el, func(m *html.Node) bool { return m.DataAtom == atom.Form }); form != nil {
			c.submit(form, nil)
		}
	case a == hotty.Copy || a == hotty.Cut || a == hotty.Paste:
		return false
	case changed:
		c.edited(el, f)
	}
	if !changed && tokens(el, "data-on", "caret") {
		if sel := fieldSelection(f); sel != before && attr(el, "id") != "" {
			c.emit(hotty.Event{Kind: hotty.EventCaret, Target: attr(el, "id"), Rev: f.rev, Sel: &sel})
		}
	}
	c.lay = nil
	return true
}

// selectKey picks a select's option with the arrows, home and end, and
// with a character: the next option whose label starts with it, ignoring
// case, going round to the first (SPEC §10.2 Selects).
func (c *Screen) selectKey(el *html.Node, key, name string) bool {
	opts := options(el)
	choices := make([]keydefault.Option, len(opts))
	for i, o := range opts {
		choices[i] = keydefault.Option{Value: o.value, Label: o.label, Disabled: o.disabled}
	}
	value, changed, taken := keydefault.Select(choices, c.field(el).ed.Value, key, name)
	if changed {
		c.pick(el, value)
	}
	return taken
}

// drawn says whether the layout drew n, or would at no size: a focusable
// element with no cells (an empty span with tabindex) is a Tab stop, as
// in a browser, unless it or an element round it draws nothing
// (display:none, hidden, a closed details' body).
func (l *layout) drawn(n *html.Node) bool {
	for m := n; m != nil; m = m.Parent {
		if _, ok := l.boxes[m]; ok {
			return true
		}
		if m.Type != html.ElementNode {
			continue
		}
		if styleOf(m, false).display == dNone { // whatever the pointer is over
			return false
		}
		if p := m.Parent; p != nil && p.DataAtom == atom.Details && !has(p, "open") && m.DataAtom != atom.Summary {
			return false
		}
	}
	return false
}

// tab moves focus to the next or the previous element Tab goes to. Past
// the last, or before the first, nothing is focused (blur); the Screen
// keeps the keyboard, so the next Tab starts again.
func (c *Screen) tab(back bool) bool {
	l := c.layout()
	var order []*html.Node
	at := -1
	walk(c.body, func(n *html.Node) {
		if n == c.focused {
			at = len(order)
		}
		if (l.drawn(n) || has(n, "data-hotty-modal-guard")) && tabbable(n) {
			order = append(order, n)
		}
	})
	if len(order) == 0 {
		return false
	}
	in := at >= 0 && at < len(order) && order[at] == c.focused
	i := at
	switch {
	case at < 0 && back:
		i = len(order) - 1
	case at < 0:
		i = 0
	case back:
		i--
	case in:
		i++
	}
	if i < 0 || i >= len(order) {
		c.blur()
		return true
	}
	c.focusTo(order[i], true)
	return true
}

// scrollBy scrolls the document as a scroll action does (SPEC §10.2).
func (c *Screen) scrollBy(a hotty.Action) {
	h := c.layout().height
	switch a {
	case hotty.ScrollUp:
		c.scroll--
	case hotty.ScrollDown:
		c.scroll++
	case hotty.ScrollPageUp:
		c.scroll -= max(c.rows-1, 1)
	case hotty.ScrollPageDown:
		c.scroll += max(c.rows-1, 1)
	case hotty.ScrollHalfPageUp:
		c.scroll -= max(c.rows/2, 1)
	case hotty.ScrollHalfPageDown:
		c.scroll += max(c.rows/2, 1)
	case hotty.ScrollStart:
		c.scroll = 0
	case hotty.ScrollEnd:
		c.scroll = h
	}
	c.scroll = max(min(c.scroll, h-c.rows), 0)
}

// reveal scrolls n into view through each bounded scrolling ancestor,
// from the innermost out, and then through the document.
func (c *Screen) reveal(n *html.Node, where string) {
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		l := c.layout()
		box, ok := l.scrolls[parent]
		if !ok {
			continue
		}
		r, ok := l.boxes[n]
		if !ok {
			continue
		}
		old := c.boxScroll[parent]
		to := revealOffset(r, box.view, old, box.content-box.view.h, where)
		if to != old {
			if c.boxScroll == nil {
				c.boxScroll = map[*html.Node]int{}
			}
			c.boxScroll[parent] = to
			c.lay = nil
		}
	}
	l := c.layout()
	r, ok := l.boxes[n]
	if !ok {
		return
	}
	c.scroll = revealOffset(r, rect{y: c.scroll, h: c.rows}, c.scroll, l.height-c.rows, where)
}

func revealOffset(target, view rect, old, limit int, where string) int {
	to := old
	switch where {
	case hotty.RevealStart:
		to += target.y - view.y
	case hotty.RevealEnd:
		to += target.y + target.h - (view.y + view.h)
	default:
		if target.y+target.h > view.y+view.h {
			to += target.y + target.h - (view.y + view.h)
		}
		if target.y < view.y {
			to += target.y - view.y
		}
	}
	return max(0, min(to, limit))
}

// click is a press of the primary button at a cell, as a host has it
// (SPEC §9, §10.1): press, then focus or blur, then the click or the
// toggle.
func (c *Screen) pressAt(col, row, mods int) bool {
	c.down = nil
	if c.body == nil || col < 0 || row < 0 || col >= c.cols || row >= c.rows {
		return false
	}
	hit := c.hitAt(col, row)
	// The runtime marks the background of an open modal as pointer-inert.
	// It is still drawn, but must not focus or activate a control.
	if closest(hit, func(n *html.Node) bool { return has(n, "data-hotty-modal-inert") }) != nil {
		return true
	}
	if c.press {
		e := hotty.Event{Kind: hotty.EventPress, Mods: mods}
		if t := withID(hit); t != nil && t != c.body {
			e.Target = attr(t, "id")
			e.Area = c.area(t)
		}
		c.emit(e)
	}
	// A press on a drag handle starts its drag first (SPEC §9.1).
	c.startDrag(hit, col, row, mods)
	target := hit
	f := closest(hit, focusable)
	if f == nil {
		// A click on a label is a click on its control.
		if lab := closest(hit, func(n *html.Node) bool { return n.DataAtom == atom.Label }); lab != nil {
			if ctl := c.labelled(lab); ctl != nil {
				target = ctl
				if focusable(ctl) {
					f = ctl
				}
			}
		}
	}
	if f != nil {
		if isTextField(f) && target == hit {
			// Against the cells clicked: focus scrolls an input to its caret.
			c.startTextSelect(f, col, row, mods)
		}
		c.active = true
		c.focusTo(f, true)
	} else {
		c.blur()
	}
	c.down = &pressed{hit: hit, target: target}
	c.lay = nil
	return true
}

// releaseAt is a primary release's click: on what the press was on,
// where the release is over it too.
func (c *Screen) releaseAt(col, row, mods int) bool {
	p := c.down
	c.down = nil
	if p == nil || c.body == nil {
		return false
	}
	if p.hit != nil && !c.attached(p.hit) {
		return false // the press drew the tree anew without it
	}
	over := c.hitAt(col, row)
	if p.hit == nil || over == nil || !within(over, p.hit) && !within(p.hit, over) {
		return false
	}
	target := p.target
	switch {
	case target != nil && isCheck(target) && !disabled(target):
		c.toggle(target)
	default:
		if k := closest(target, clickable); k != nil && !disabled(k) {
			c.activateBy(k, c.count(k), mods)
		}
	}
	c.lay = nil
	return true
}

// labelled is a label's control: the one its for names, or the first in
// it.
func (c *Screen) labelled(lab *html.Node) *html.Node {
	if id := attr(lab, "for"); id != "" {
		return c.byID(id)
	}
	var ctl *html.Node
	walk(lab, func(n *html.Node) {
		if ctl == nil && n != lab && (isField(n) || n.DataAtom == atom.Button) {
			ctl = n
		}
	})
	return ctl
}

// caretAt puts a text field's caret at the cell clicked.
func (c *Screen) caretAt(n *html.Node, x, y int) {
	l := c.layout()
	r, ok := l.fields[n]
	if !ok || !r.has(x, y) {
		return
	}
	f := c.field(n)
	chars := graphemesOf(f.ed.Value)
	start, end, col := 0, len(chars), x-r.x
	if n.DataAtom == atom.Textarea {
		rows := f.ed.Wrap()
		i := y - r.y + f.top
		if i >= len(rows) {
			f.ed.Select(len(chars), len(chars))
			return
		}
		start, end = rows[i].Start, rows[i].End
	} else {
		col += f.off
	}
	p, w := start, 0
	for ; p < end; p++ {
		cw := widthOf(charsOf(chars[p]))
		if f.ed.Password {
			cw = 1
		}
		if w+cw > col {
			break
		}
		w += cw
	}
	f.ed.Select(p, p)
}

// setField takes an a=field (SPEC §6.4). A change made against a rev the
// user has moved past changes nothing, as a host's ESTALE.
func (c *Screen) setField(m hotty.Message) error {
	ch, err := hotty.FieldOf(m)
	if err != nil {
		return err
	}
	n := c.byID(hotty.Get(m.Control, "t"))
	if n == nil {
		return errNoTarget
	}
	if !isField(n) {
		return fmt.Errorf("not a control (%s)", hotty.EINVAL)
	}
	f := c.field(n)
	check := isCheck(n)
	if ch.Checked != nil && !check || (ch.Value != nil || ch.Text != nil || ch.Sel != nil) && check ||
		(ch.Text != nil || ch.Sel != nil) && !isTextField(n) {
		return fmt.Errorf("a change the control doesn't take (%s)", hotty.EINVAL)
	}
	if ch.Rev != f.rev {
		return nil
	}
	c.lay = nil
	if ch.Checked != nil {
		c.check(n, *ch.Checked)
		return nil
	}
	v := f.ed.Value
	caret := -1
	switch {
	case ch.Value != nil:
		v = *ch.Value
		if n.DataAtom == atom.Input {
			v = strings.NewReplacer("\r", "", "\n", "").Replace(v)
		}
	case ch.Text != nil:
		text := *ch.Text
		if n.DataAtom == atom.Input {
			text = strings.NewReplacer("\r", "", "\n", "").Replace(text)
		}
		var from, to int
		if ch.From != nil {
			a, ok1 := charAt(v, *ch.From)
			b, ok2 := charAt(v, *ch.To)
			if !ok1 || !ok2 {
				return fmt.Errorf("a position outside the value (%s)", hotty.EINVAL)
			}
			from, to = a, b
		} else {
			from, to = f.ed.Selection()
		}
		fb, tb := byteAt(v, from), byteAt(v, to)
		v = v[:fb] + text + v[tb:]
		caret = from + len(graphemesOf(text))
	}
	f.ed.Value = v
	f.committed = v
	n2 := len(graphemesOf(v))
	if caret < 0 {
		caret = n2
	}
	if ch.Sel != nil {
		a, ok1 := charAt(v, ch.Sel[0])
		b, ok2 := charAt(v, ch.Sel[1])
		if !ok1 || !ok2 {
			return fmt.Errorf("a selection outside the value (%s)", hotty.EINVAL)
		}
		f.ed.Select(a, b)
	} else if ch.Value != nil || ch.Text != nil {
		f.ed.Select(caret, caret)
	}
	return nil
}
