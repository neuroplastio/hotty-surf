package ui

import (
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Bindings are the actions s offers now, from its last render's
// [KeyScope]s, in the order they were rendered: the order a menu or a
// palette shows them. A Name is listed once (see KeyBinding.Name).
//
// A binding keeps its Key only while the key reaches it: the keyboard is
// inside its scope, and no nearer scope's enabled binding takes the same
// key. Any other keyed binding is listed with Key "" when it has a Name,
// since a pick still runs it, and left out when it has none. Disabled
// bindings are listed, Disabled. Their Run is nil: call [Pick] to run
// one, so that Disabled and a Name's nearest binding hold.
//
// It reads the surface's state: call it on the surface driver's goroutine
// (in a handler or in [surf.Surface.Do]). It is nil before s first
// renders.
func Bindings(s *surf.Surface) []KeyBinding {
	var out []KeyBinding
	for _, o := range offered(s) {
		out = append(out, o.binding)
	}
	return out
}

// HostMenu is s's [Bindings] as a hotty menu (SPEC §10.7), for hotty.Menu:
// one top-level item each, with its Key, Label, Name and whether it is
// enabled, under label. It is nil when s offers none, which hotty.Menu
// sends as no menu. An app that sends its own menu, with key sequences or
// groups of its own, adds these items to it. It runs on the driver's
// goroutine, as Bindings does.
func HostMenu(s *surf.Surface, label string) *hotty.MenuTree {
	bs := Bindings(s)
	if len(bs) == 0 {
		return nil
	}
	t := &hotty.MenuTree{Label: label, Items: make([]hotty.MenuItem, 0, len(bs))}
	for _, b := range bs {
		t.Items = append(t.Items, hotty.MenuItem{Key: b.Key, Label: b.Label, Name: b.Name, Enabled: !b.Disabled})
	}
	return t
}

// Pick runs the action named name of s's [Bindings], as a host's menu or
// an app's palette picks it (a hotty command event's name): the binding of
// that name nearest the keyboard, else the first rendered. taken is what
// its Run returns, and false for a name s does not offer or a disabled
// binding. It runs on the driver's goroutine: call it in
// [surf.Surface.Do], whose Update renders what Run changed.
func Pick(s *surf.Surface, name string) (taken bool) {
	if name == "" {
		return false
	}
	for _, o := range offered(s) {
		if o.binding.Name == name {
			return !o.binding.Disabled && o.at.scope.Pick(o.at.index)
		}
	}
	return false
}

// at is a binding: its scope and its index there.
type at struct {
	scope *rt.KeyScope
	index int
}

type offer struct {
	binding KeyBinding
	at      at
}

// offered is Bindings with where each came from.
func offered(s *surf.Surface) []offer {
	if s == nil || rt.Peek.Frame == nil {
		return nil
	}
	f := rt.Peek.Frame(s)
	if f == nil || len(f.Scopes) == 0 {
		return nil
	}
	reach := rt.Peek.Scopes(s) // nearest the keyboard first
	// A field that types keeps its printable keys (K27): none reaches a
	// binding while it has the keyboard.
	typing := false
	if n := byID(rt.Peek.Tree(s), rt.Peek.Focused(s)); n != nil {
		typing = rt.Types(n)
	}
	// winner is each key's binding: the nearest enabled one, else the
	// nearest disabled one. nearest is each name's.
	winner, nearest := map[string]at{}, map[string]at{}
	for _, enabled := range []bool{true, false} {
		for _, sc := range reach {
			for i, b := range sc.Bindings {
				if b.Physical != "" && b.Disabled != enabled && !(typing && rt.Printable(b.Physical)) {
					if _, ok := winner[b.Physical]; !ok {
						winner[b.Physical] = at{sc, i}
					}
				}
			}
		}
	}
	for _, sc := range reach {
		for i, b := range sc.Bindings {
			if _, ok := nearest[b.Name]; b.Name != "" && !ok {
				nearest[b.Name] = at{sc, i}
			}
		}
	}
	listed := map[string]bool{}
	var out []offer
	for _, sc := range f.Scopes {
		for i, b := range sc.Bindings {
			here := at{sc, i}
			if b.Name != "" {
				if listed[b.Name] {
					continue
				}
				listed[b.Name] = true
				if n, ok := nearest[b.Name]; ok {
					here, b = n, n.scope.Bindings[n.index]
				}
			}
			kb := KeyBinding{Key: b.Key, Label: b.Label, Name: b.Name, Disabled: b.Disabled}
			if b.Physical != "" && winner[b.Physical] != here {
				if b.Name == "" {
					continue
				}
				kb.Key = ""
			}
			out = append(out, offer{kb, here})
		}
	}
	return out
}

// byID is the element of root's tree whose id is id, nil for none.
func byID(root *html.Node, id string) *html.Node {
	if root == nil || id == "" {
		return nil
	}
	for _, a := range root.Attr {
		if a.Key == "id" && a.Val == id {
			return root
		}
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if n := byID(c, id); n != nil {
			return n
		}
	}
	return nil
}
