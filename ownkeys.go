package surf

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// The keys a native control uses (SPEC §10.2), each also with shift.
var (
	activateKeys = []string{"space", "enter"}
	selectKeys   = []string{"space", "arrow_up", "arrow_down", "home", "end", "page_up", "page_down", "enter"}
	dateKeys     = []string{"space", "backspace", "delete", "arrow_left", "arrow_right", "arrow_up", "arrow_down",
		"home", "end", "page_up", "page_down", "enter"}
)

// ownKeys is what OwnKeys says of n, focused, on platform p, with on the
// frame's handlers.
func ownKeys(n *html.Node, on *rt.Frame, p hotty.Platform, legacy bool) []string {
	if n == nil || hasAttr(n, "disabled") {
		return nil
	}
	kind, _ := rt.NativeKind(n)
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
		// platform default, ancestor inner maps, and the field's own
		// map. A key it gives the program, a prefix or a scroll is not
		// the field's; a move is, with shift too (it selects).
		m := rt.ResolveKeys(p, n, nil, rt.KeyReader(n), legacy)
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
	if kind != "text" {
		// The keys its keymap binds to a scroll action (surf.Bind): the
		// host scrolls with them, ahead of the program, unless a nearer
		// binding gives one the program. A character is listed too, as a
		// declared one is: the element doesn't type it.
		m := rt.ResolveKeys(p, n, nil, rt.KeyReader(n), legacy)
		for _, b := range strings.Fields(m.Format()) {
			k, a, _ := strings.Cut(b, "=")
			if !hotty.Action(a).Scrolls() {
				continue
			}
			c, ok := p.ReadKey(k)
			if !ok {
				continue
			}
			if rt.Printable(c) {
				set[c] = true
				continue
			}
			add(c)
			if !strings.Contains(c, "shift+") && m.Scroll("shift+"+c) != "" {
				add("shift+" + c)
			}
		}
	}
	// The keys the Key handlers of it and the elements around it declare:
	// those the host gives the program there, for them to take (a list's
	// arrows, a combobox's Enter, an app's mod+s on its main), with shift
	// too where it isn't a character (SPEC §10.2 looks a key with shift up
	// again without it). A declared character is listed, but not on an
	// element that types, which keeps keys it types. Escape a host ignores in
	// data-keys, but a handler that declares it takes it. A KeyScope's
	// bindings are Bindings', not these.
	types := rt.Types(n)
	for e := n; e != nil; e = e.Parent {
		h := on.Get(attr(e, "id"))
		if h == nil || e.Type != html.ElementNode {
			continue
		}
		for _, k := range h.Keys {
			if rt.Printable(k) {
				if !types {
					set[k] = true
				}
				continue
			}
			add(k)
			// Escape reaches the program anyway: its shift is noise.
			if k != "escape" && !strings.Contains(k, "shift+") {
				add("shift+" + k)
			}
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

// fieldTakes is rt.Peek.FieldTakes: the focused element is a text field,
// not a kit control under a KeyScope (whose scope's keys the kit forwards),
// and its keymap binds key to an edit action, which it does itself.
func (s *Surface) fieldTakes(key string) bool {
	n := s.byID[s.focused]
	if n == nil || s.keyMarks[s.focused].local > 0 {
		return false
	}
	kind, _ := rt.NativeKind(n)
	if kind != "text" {
		return false
	}
	m := rt.ResolveKeys(s.platform, n, nil, rt.KeyReader(n), s.legacy)
	switch a := m.Lookup(key); a {
	case "", hotty.Program, hotty.Prefix, hotty.Insert:
		return false
	}
	return true
}
