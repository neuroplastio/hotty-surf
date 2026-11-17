package surf

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The keys a native control uses (SPEC §10.2), each also with shift.
var (
	activateKeys = []string{"space", "enter"}
	selectKeys   = []string{"space", "arrow_up", "arrow_down", "home", "end", "page_up", "page_down", "enter"}
	dateKeys     = []string{"space", "backspace", "delete", "arrow_left", "arrow_right", "arrow_up", "arrow_down",
		"home", "end", "page_up", "page_down", "enter"}
)

// ownKeys is what OwnKeys says of n, focused, on platform p.
func ownKeys(n *html.Node, p hotty.Platform) []string {
	if n == nil || hasAttr(n, "disabled") {
		return nil
	}
	kind, multiline := nativeKind(n)
	set := map[string]bool{}
	add := func(k string) {
		c, ok := p.ReadKey(k)
		if !ok || c == "tab" || c == "shift+tab" {
			return
		}
		// Characters are Chars' to say. Space is one in a text field,
		// which types it; elsewhere it is a key the control uses.
		if rt.Printable(c) && (kind == "text" || c != "space" && c != "shift+space") {
			return
		}
		set[c] = true
	}
	addShifted := func(keys []string) {
		for _, k := range keys {
			add(k)
			add("shift+" + k)
		}
	}
	switch kind {
	case "text":
		// The field's keymap as a host and cells.Screen resolve it: the
		// default for the platform, then each data-keys from the root
		// down. A key it gives the program, a prefix or a scroll is not
		// the field's; a move is, with shift too (it selects).
		m := hotty.Resolve(p, multiline, keymapsOf(n)...)
		for _, b := range strings.Fields(m.Format()) {
			k, a, _ := strings.Cut(b, "=")
			act := hotty.Action(a)
			if m.Program(k) || m.Prefix(k) || act.Scrolls() || act == "default" {
				continue
			}
			add(k)
			if act.Moves() {
				add("shift+" + k)
			}
		}
	case "select":
		addShifted(selectKeys)
	case "date":
		addShifted(dateKeys)
	case "activate":
		addShifted(activateKeys)
	}
	// A kit control's own keys: those it gives the program on itself,
	// which its Key handler takes (a list's arrows, a combobox's Enter).
	// Under a KeyScope its own are kept apart from the scope's bindings.
	decl, ok := get(n, "data-hotty-default-keys")
	if !ok && !hasAttr(n, "data-hotty-keyscope") {
		decl = attr(n, "data-keys")
	}
	for _, b := range strings.Fields(decl) {
		if k, a, _ := strings.Cut(b, "="); a == "program" {
			add(k)
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// nativeKind is the row of SPEC §10.2's table n is in: "text" (multiline
// for a textarea or an editing host), "select", "date", "activate", or ""
// for an element that uses no keys itself.
func nativeKind(n *html.Node) (kind string, multiline bool) {
	if hasAttr(n, "contenteditable") && !strings.EqualFold(attr(n, "contenteditable"), "false") {
		return "text", true
	}
	switch n.Data {
	case "textarea":
		return "text", true
	case "select":
		if hasAttr(n, "multiple") {
			return "", false
		}
		return "select", false
	case "button", "summary":
		return "activate", false
	case "a":
		if hasAttr(n, "href") {
			return "activate", false
		}
	case "input":
		switch t := strings.ToLower(attr(n, "type")); t {
		case "hidden", "file", "color", "range", "image":
			return "", false
		case "checkbox", "radio", "submit", "reset", "button":
			return "activate", false
		case "date", "time", "datetime-local", "month", "week":
			return "date", false
		}
		return "text", false
	}
	return "", false
}

// keymapsOf are the data-keys values from the root down to n (SPEC §10.2),
// as the host and cells.Screen read them.
func keymapsOf(n *html.Node) []string {
	var vs []string
	for e := n; e != nil; e = e.Parent {
		if e.Type != html.ElementNode {
			continue
		}
		if v, ok := get(e, "data-keys"); ok {
			vs = append(vs, v)
		}
	}
	slices.Reverse(vs)
	return vs
}
