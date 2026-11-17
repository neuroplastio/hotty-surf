package rt

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
)

// DispatchKeyScope offers a key to its lexical scopes, nearest first.
// It is referenced only by the KeyScope component that installs it.
func DispatchKeyScope(scopes []*KeyScope, marker, physical, key string) (intercepted, taken bool, fallback func(KeyDefault) KeyResult, native func(*KeyEnvironment, string, func(KeyDefault) KeyResult) bool) {
	index, err := strconv.Atoi(marker)
	if err != nil || index <= 0 || index > len(scopes) {
		return
	}
	for scope := scopes[index-1]; scope != nil; scope = scope.Parent {
		if scope.Keys[physical] {
			intercepted, fallback, native = true, scope.Default, scope.Native
			if scope.Handle(physical) {
				taken = true
				return
			}
		}
	}
	return
}

// nativeKeyDefault prepares the focused control for the fallback carried
// by a KeyScope. The surface does not link its editing implementation.
func NativeKeyDefault(e *KeyEnvironment, key string, fallback func(KeyDefault) KeyResult) bool {
	n := e.Node
	if n == nil || hasAttr(n, "disabled") || fallback == nil {
		return false
	}
	kind := attr(n, "data-hotty-keykind")
	name := key
	if i := strings.LastIndexByte(key, '+'); i >= 0 {
		name = key[i+1:]
	}
	plain := key == name || key == "shift+"+name
	h := e.On[e.Focused]
	if kind == "button" || kind == "link" {
		if !plain || name != "enter" && name != "space" {
			return false
		}
		if h != nil && h.Click != nil {
			h.Click()
		}
		if kind == "button" && (attr(n, "type") == "submit" || !hasAttr(n, "type")) {
			e.submitKey(n)
		}
		return true
	}
	in := KeyDefault{Kind: kind, Key: key, Value: e.Values[e.Focused], Checked: e.Checked[e.Focused],
		Type: attr(n, "type"), Readonly: hasAttr(n, "readonly"), Platform: e.Platform}
	if n.Data == "textarea" {
		in.Type = "textarea"
	}
	in.Selection, in.HasSelection = e.Selections[e.Focused]
	if kind == "select" {
		var visit func(*html.Node)
		visit = func(p *html.Node) {
			for c := p.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				if c.Data == "option" {
					value, ok := get(c, "value")
					if !ok {
						value = textOf(c)
					}
					label, ok := get(c, "label")
					if !ok {
						label = textOf(c)
					}
					in.Options = append(in.Options, KeyOption{Value: value, Label: label, Disabled: hasAttr(c, "disabled")})
				} else {
					visit(c)
				}
			}
		}
		visit(n)
	}
	if kind == "text" || kind == "compound-text" {
		// A scope's own keymap, not its merged data-keys: a field's key
		// a binding declines stays the field's. surf's keymapsOf
		// (OwnKeys) reads data-keys as written, as a host does; keep the
		// two apart.
		for p := n; p != nil; p = p.Parent {
			v := attr(p, "data-keys")
			if hasAttr(p, "data-hotty-keyscope") {
				v = attr(p, "data-hotty-default-keys")
			}
			if v != "" {
				in.Keymaps = append(in.Keymaps, v)
			}
		}
		slices.Reverse(in.Keymaps)
	}
	out := fallback(in)
	if !out.Taken {
		return false
	}
	if out.Submit {
		e.submitKey(n)
		return true
	}
	if out.Changed {
		if kind == "checkbox" || kind == "radio" || kind == "switch" {
			if h != nil && h.Check != nil {
				h.Check(out.Checked)
			} else {
				v := out.Checked
				e.PendingNative[e.Focused] = hotty.FieldChange{Rev: e.Revisions[e.Focused], Checked: &v}
			}
		} else if h != nil && h.Input != nil {
			h.Input(out.Value)
		} else {
			v := out.Value
			change := hotty.FieldChange{Rev: e.Revisions[e.Focused], Value: &v}
			if out.HasSelection {
				sel := out.Selection
				change.Sel = &sel
			}
			e.PendingNative[e.Focused] = change
		}
	}
	if out.HasSelection {
		e.Selections[e.Focused] = out.Selection
		if !out.Changed || h != nil && h.Input != nil {
			e.PendingSelections[e.Focused] = out.Selection
		}
	}
	return true
}

func (e *KeyEnvironment) submitKey(n *html.Node) {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Data == "form" {
			if x := e.On[attr(p, "id")]; x != nil && x.Submit != nil {
				x.Submit()
			}
			return
		}
	}
}

func get(n *html.Node, key string) (string, bool) {
	if n == nil {
		return "", false
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}
func attr(n *html.Node, key string) string  { v, _ := get(n, key); return v }
func hasAttr(n *html.Node, key string) bool { _, ok := get(n, key); return ok }
func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type == html.ElementNode && n.Data == "svg" {
		return ""
	}
	if n.Type == html.ElementNode && n.Data == "img" {
		return attr(n, "alt")
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}
