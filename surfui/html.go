package surfui

import (
	"html"
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
)

// HTML is the markup of a resolved tree.
func HTML(n *Node) string {
	var b strings.Builder
	write(&b, n)
	return b.String()
}

// KidsHTML is the markup of n's children.
func KidsHTML(n *Node) string {
	var b strings.Builder
	for _, k := range n.kids {
		write(&b, k)
	}
	return b.String()
}

var void = map[string]bool{"input": true, "br": true, "hr": true, "img": true, "meta": true, "link": true}

func write(b *strings.Builder, n *Node) {
	stackprobe.Probe()
	if n.tag == "" {
		b.WriteString(html.EscapeString(n.text))
		return
	}
	b.WriteByte('<')
	b.WriteString(n.tag)
	if n.id != "" {
		writeAttr(b, "id", n.id)
	}
	for _, a := range attrsOf(n) {
		writeAttr(b, a.k, a.v)
	}
	b.WriteByte('>')
	if void[n.tag] {
		return
	}
	if n.isRaw {
		b.WriteString(n.raw)
	}
	for _, k := range n.kids {
		write(b, k)
	}
	b.WriteString("</")
	b.WriteString(n.tag)
	b.WriteByte('>')
}

func writeAttr(b *strings.Builder, k, v string) {
	b.WriteByte(' ')
	b.WriteString(k)
	if v == "" && k != "value" && k != "class" {
		return
	}
	b.WriteString(`="`)
	b.WriteString(html.EscapeString(v))
	b.WriteByte('"')
}

// attrsOf is every attribute n renders with but its id, in a fixed order:
// class, the author's, the style that holds its vars, data-on and
// data-keys.
func attrsOf(n *Node) []attr {
	out := make([]attr, 0, len(n.attrs)+4)
	if len(n.class) > 0 {
		out = append(out, attr{"class", strings.Join(n.class, " ")})
	}
	out = append(out, n.attrs...)
	if s := styleOf(n); s != "" {
		out = append(out, attr{"style", s})
	}
	if on := dataOn(n); on != "" {
		out = append(out, attr{"data-on", on})
	}
	if keys := dataKeys(n); keys != "" {
		out = append(out, attr{"data-keys", keys})
	}
	return out
}

func styleOf(n *Node) string {
	if len(n.vars) == 0 {
		return ""
	}
	var b strings.Builder
	for i, v := range n.vars {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString("--")
		b.WriteString(v.k)
		b.WriteByte(':')
		b.WriteString(v.v)
	}
	return b.String()
}

// clicks are the elements whose click a host reports without data-on
// (SPEC §9).
var clicks = map[string]bool{"button": true, "a": true, "summary": true}

// DataOn is n's data-on, as it renders: the events it asks a host for
// (SPEC §9), for a host of the program's own, such as the cells'.
func DataOn(n *Node) string { return dataOn(n) }

// DataKeys is n's data-keys, as it renders: its keymap (SPEC §10.2), for
// a host of the program's own, such as the cells'.
func DataKeys(n *Node) string { return dataKeys(n) }

// dataOn asks for the events n has handlers for that a host doesn't send
// by itself: a click on an element that isn't a button, every edit of a
// field. A Cycle's parts report their clicks, which the runtime turns
// into the Cycle's change. A field that commits (Commits) reports its
// edits, so that the runtime has what Enter commits.
func dataOn(n *Node) string {
	var on []string
	if n.part != 0 {
		on = append(on, "click")
	}
	for _, h := range n.on {
		switch h.kind {
		case "click":
			if !clicks[n.tag] && !(n.tag == "input" && isButtonInput(n)) {
				on = appendOnce(on, "click")
			}
		case "input":
			on = appendOnce(on, "input")
		}
	}
	if Commits(n) {
		on = appendOnce(on, "input")
	}
	return strings.Join(on, " ")
}

// Commits reports whether n is a text field with an OnChange, which the
// runtime runs when the user commits what they typed: on Enter, and when
// they leave the field (the host's change, SPEC §9), once for each
// value. Such a field gives the program Enter, and reports its edits.
func Commits(n *Node) bool {
	if !IsTextField(n) {
		return false
	}
	_, ok := Handler(n, "change")
	return ok
}

func isButtonInput(n *Node) bool {
	t, _ := AttrOf(n, "type")
	return t == "button" || t == "submit" || t == "reset"
}

func appendOnce(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

// IsTextField reports whether n is a field a host edits as text, whose
// keymap would take keys from the program (SPEC §10.2).
func IsTextField(n *Node) bool {
	switch n.tag {
	case "textarea":
		return true
	case "input":
		t, _ := AttrOf(n, "type")
		switch t {
		case "", "text", "search", "url", "tel", "email", "password", "number":
			return true
		}
	}
	return false
}

// ValueOf is the value a control shows as the program last drew it, and
// whether n is one the runtime keeps with a=field (SPEC §6.4): a text
// field's value, or a select's (its own value, as a binding sets it, or
// else its selected option's).
func ValueOf(n *Node) (string, bool) {
	switch {
	case IsTextField(n):
		v, _ := AttrOf(n, "value")
		return v, true
	case n.tag == "select":
		if v, ok := AttrOf(n, "value"); ok {
			return v, true
		}
		for _, o := range n.kids {
			if _, sel := AttrOf(o, "selected"); sel && o.tag == "option" {
				v, _ := AttrOf(o, "value")
				return v, true
			}
		}
		return "", true
	}
	return "", false
}

// dataKeys is n's keymap (SPEC §10.2), which a host reads from the root
// down to the focused element, each element's overriding what came
// before it key by key: so a binding of n's holds in every element in
// it, as a Bind does. n gives the program the keys it binds that a host
// would scroll with (ScrollKey), so that the binding hears them first,
// in a document that scrolls too; the other keys reach the program
// anyway. A Cycle and a Select give the keys that step them which a host
// would scroll with, and an open Select Space and Enter too.
//
// A binding to a scroll action (BindScroll) n writes as it is: the key is
// then the host's, who scrolls with it.
//
// What n uses itself it keeps. A button keeps Space, which presses it:
// in an element that binds Space it binds Space back to the page scroll
// it has in a browser, which a button doesn't do. A text field is
// fieldKeys'. An element with a text field in it gives no Space, which
// the field types and no action gives back.
func dataKeys(n *Node) string {
	if IsTextField(n) {
		return fieldKeys(n)
	}
	var keys []string
	switch {
	case n.cycle != nil:
		keys = append(keys, cycleKeys...)
	case n.choice != nil && n.choice.open:
		keys = append(keys, selectOpenKeys...)
	case n.choice != nil:
		keys = append(keys, selectKeys...)
	}
	press := presses(n) && (n.choice == nil || !n.choice.open)
	for _, b := range n.keys {
		switch {
		case b.Scroll != "":
			keys = appendOnce(keys, b.Key+"="+string(b.Scroll))
		case ScrollKey(b.Key) && !(spaceKey(b.Key) && (press || holdsField(n))):
			keys = appendOnce(keys, b.Key+"=program")
		}
	}
	if press {
		for _, k := range [...]struct {
			key string
			to  hotty.Action
		}{{"space", hotty.ScrollPageDown}, {"shift+space", hotty.ScrollPageUp}} {
			if bindsAbove(n, k.key) {
				keys = appendOnce(keys, k.key+"="+string(k.to))
			}
		}
	}
	return strings.Join(keys, " ")
}

// fieldKeys gives the program, from a text field, the keys that the
// field and its parents bind (S12), and only those a field can do
// without: a key with Ctrl, Alt or Mod, the vertical arrows and pages,
// Enter, the function keys. A field that commits gives Enter, which
// commits it (Commits). A letter, Backspace, the left and right arrows,
// Home and End stay the field's, even where a parent binds them: the
// arrows, Home and End a parent gives the program (dataKeys) the field
// binds back to their editing actions. A binding to a scroll action is
// none of a field's: its keymap leaves them out (SPEC §10.2).
func fieldKeys(n *Node) string {
	var keys []string
	if Commits(n) {
		keys = append(keys, "enter=program")
	}
	for p := n; p != nil; p = p.parent {
		for _, b := range p.keys {
			if b.Scroll != "" {
				continue
			}
			if Giveable(b.Key) {
				keys = appendOnce(keys, b.Key+"=program")
			} else if a, ok := fieldKeeps[b.Key]; ok && p != n {
				keys = appendOnce(keys, b.Key+"="+string(a))
			}
		}
	}
	return strings.Join(keys, " ")
}

// fieldKeeps are the keys a host scrolls with that a text field edits
// with, by their actions in every host's default keymap (SPEC §10.2).
// Space and Shift+Space type, which no action does.
var fieldKeeps = map[string]hotty.Action{
	"arrow_left": hotty.CharBackward, "arrow_right": hotty.CharForward,
	"home": hotty.LineStart, "end": hotty.LineEnd,
}

// ScrollKey reports whether key is one a host scrolls with, where the
// document scrolls (SPEC §5.3): an arrow, Page Up and Down, Home, End,
// Space and Shift+Space, with no other modifier, as a browser has them.
// A binding of one gives it to the program (dataKeys), which would
// otherwise hear it only once nothing can scroll.
func ScrollKey(key string) bool {
	switch key {
	case "arrow_up", "arrow_down", "arrow_left", "arrow_right",
		"page_up", "page_down", "home", "end", "space", "shift+space":
		return true
	}
	return false
}

func spaceKey(key string) bool { return key == "space" || key == "shift+space" }

// ScrollAction is the scroll action (SPEC §10.2) that a <bind>'s scroll
// names, the action's name without its "scroll-": "down" is
// hotty.ScrollDown, "page-up" hotty.ScrollPageUp. ok is false for a name
// that is none.
func ScrollAction(name string) (a hotty.Action, ok bool) {
	a = hotty.Action("scroll-" + name)
	return a, a.Scrolls()
}

// presses reports whether Space presses n on a host (SPEC §10.2): a
// button (a Button, a Switch, a Select), a link, a summary, a box.
func presses(n *Node) bool {
	switch n.tag {
	case "button", "summary":
		return true
	case "a":
		_, ok := AttrOf(n, "href")
		return ok
	case "input":
		t, _ := AttrOf(n, "type")
		return t == "button" || t == "submit" || t == "reset" || t == "checkbox" || t == "radio"
	}
	return false
}

// TakesFocus reports whether a host gives n the keyboard on a click or
// with Tab (SPEC §10.1): an input, a select, a textarea or a button, a
// link with an href that is no hyperlink (target="_blank"), a summary,
// an editing host, or an element with a tabindex of 0 or more; none that
// is disabled.
func TakesFocus(n *Node) bool {
	if n == nil || n.tag == "" {
		return false
	}
	if _, off := AttrOf(n, "disabled"); off {
		return false
	}
	switch n.tag {
	case "input":
		t, _ := AttrOf(n, "type")
		return !strings.EqualFold(t, "hidden")
	case "select", "textarea", "button", "summary":
		return true
	case "a":
		_, href := AttrOf(n, "href")
		t, _ := AttrOf(n, "target")
		if href && t != "_blank" {
			return true
		}
	}
	if e, ok := AttrOf(n, "contenteditable"); ok && e != "false" {
		return true
	}
	t, _ := AttrOf(n, "tabindex")
	i, err := strconv.Atoi(t)
	return err == nil && i >= 0
}

// bindsAbove reports whether an element around n binds key to an action
// of the program's.
func bindsAbove(n *Node, key string) bool {
	for p := n.parent; p != nil; p = p.parent {
		for _, b := range p.keys {
			if b.Key == key && b.Scroll == "" {
				return true
			}
		}
	}
	return false
}

// holdsField reports whether a text field is in n, by a walk of its own:
// no recursion, for TinyGo's stacks.
func holdsField(n *Node) bool {
	todo := append([]*Node(nil), n.kids...)
	for len(todo) > 0 {
		k := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if IsTextField(k) {
			return true
		}
		todo = append(todo, k.kids...)
	}
	return false
}

// Giveable reports whether a text field may give key to the program
// without losing what it needs to edit.
func Giveable(key string) bool {
	if i := strings.LastIndexByte(key, '+'); i > 0 && i < len(key)-1 {
		mods := key[:i]
		if strings.Contains(mods, "ctrl") || strings.Contains(mods, "alt") ||
			strings.Contains(mods, "mod") || strings.Contains(mods, "super") {
			return true
		}
		key = key[i+1:]
		if !strings.Contains(mods, "shift") {
			return false
		}
		// Shift+Delete, an old way to cut, which a field can do without.
		if key == "delete" {
			return true
		}
	}
	switch key {
	case "arrow_up", "arrow_down", "page_up", "page_down", "enter":
		return true
	}
	return len(key) >= 2 && key[0] == 'f' && key[1] >= '1' && key[1] <= '9'
}

// Document is a whole surface's document: the stylesheet, by reference
// to a resource the session sends once (StyleID), and the tree. sheets
// are the host's own stylesheets, each an href ("cid:plx-theme"), linked
// after surf's so that the host's rules win where both set a property.
func Document(root *Node, sheets ...string) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><link rel="stylesheet" href="cid:` + StyleID + `">`)
	for _, s := range sheets {
		b.WriteString(`<link rel="stylesheet" href="` + html.EscapeString(s) + `">`)
	}
	b.WriteString(`</head><body>`)
	write(&b, root)
	b.WriteString(`</body></html>`)
	return b.String()
}

// StandaloneDocument is Document with the stylesheet inline, for a file
// that `hotty render` or a browser opens.
func StandaloneDocument(root *Node, hostCSS string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><style>` + hostCSS + `</style><style>` + CSS + `</style></head><body>` +
		HTML(root) + `</body></html>`
}
