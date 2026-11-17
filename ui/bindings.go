package ui

import (
	"strconv"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Bindings are the actions s offers now, from its last render's
// [KeyScope]s, in the order a menu or a palette shows them: first those of
// the scopes around the focused kit control, nearest scope first, then
// every other scope's in the order of the render, each scope's in the
// order it was given them.
//
// A binding keeps its Key only while the key reaches it: in a scope around
// the focused kit control, and not taken by a nearer scope's enabled
// binding of the same key. Any other keyed binding is listed with its Key
// empty when it has a Name, since a pick still runs it, and is left out
// when it has none. Disabled bindings are listed, Disabled.
//
// It reads the surface's state: call it on the surface driver's goroutine
// (in a handler or in [surf.Surface.Do]). It is nil before s first
// renders.
func Bindings(s *surf.Surface) []KeyBinding {
	var out []KeyBinding
	for _, b := range offered(s) {
		out = append(out, b.binding)
	}
	return out
}

// Menu is s's [Bindings] as a hotty menu (SPEC §10.7), for hotty.Menu:
// one top-level item each, with its Key, Label, Name and whether it is
// enabled, under label. It is nil when s offers none, which hotty.Menu
// sends as no menu. An app that sends its own menu, with key sequences of
// its own, adds these items to it. It runs on the driver's goroutine, as
// Bindings does.
func Menu(s *surf.Surface, label string) *hotty.MenuTree {
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

// Pick runs the binding named name of s's [Bindings], as a host's menu or
// an app's palette picks it (a hotty command event's name): its scope's fn
// gets it, and taken is what fn returns. A name s does not offer, or a
// disabled binding, runs nothing. It runs on the driver's goroutine: call
// it in [surf.Surface.Do], whose Update renders what fn changed.
func Pick(s *surf.Surface, name string) (taken bool) {
	if name == "" {
		return false
	}
	for _, b := range offered(s) {
		if b.binding.Name == name {
			return !b.binding.Disabled && b.scope.Pick(b.index)
		}
	}
	return false
}

type offer struct {
	binding KeyBinding
	scope   *rt.KeyScope
	index   int
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
	var chain []*rt.KeyScope
	if n := byID(rt.Peek.Tree(s), rt.Peek.Focused(s)); n != nil {
		for _, a := range n.Attr {
			if a.Key == "data-hotty-keyscope" {
				if i, err := strconv.Atoi(a.Val); err == nil && i > 0 && i <= len(f.KeyScopes) {
					for sc := f.KeyScopes[i-1]; sc != nil; sc = sc.Parent {
						chain = append(chain, sc)
					}
				}
			}
		}
	}
	// winner is each key's binding: the nearest enabled one, else the
	// nearest disabled one.
	type at struct {
		scope *rt.KeyScope
		index int
	}
	winner := map[string]at{}
	for _, enabled := range []bool{true, false} {
		for _, sc := range chain {
			for i, b := range sc.Bindings {
				if b.Physical == "" || b.Disabled == enabled {
					continue
				}
				if _, ok := winner[b.Physical]; !ok {
					winner[b.Physical] = at{sc, i}
				}
			}
		}
	}
	var out []offer
	add := func(sc *rt.KeyScope) {
		for i, b := range sc.Bindings {
			kb := KeyBinding{Key: b.Key, Label: b.Label, Name: b.Name, Disabled: b.Disabled}
			if b.Physical != "" && winner[b.Physical] != (at{sc, i}) {
				if b.Name == "" {
					continue
				}
				kb.Key = ""
			}
			out = append(out, offer{kb, sc, i})
		}
	}
	inChain := map[*rt.KeyScope]bool{}
	for _, sc := range chain {
		inChain[sc] = true
		add(sc)
	}
	for _, sc := range f.Scopes {
		if !inChain[sc] {
			add(sc)
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
