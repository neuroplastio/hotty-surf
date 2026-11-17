package surf

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/internal/rt"
)

// nativeKeyDefault prepares the focused control for the fallback carried
// by a KeyScope. The surface does not link its editing implementation.
func (s *Surface) nativeKeyDefault(key string, fallback func(rt.KeyDefault) rt.KeyResult) bool {
	n := s.byID[s.focused]
	if n == nil || hasAttr(n, "disabled") || fallback == nil {
		return false
	}
	kind := attr(n, "data-hotty-keykind")
	name := key
	if i := strings.LastIndexByte(key, '+'); i >= 0 {
		name = key[i+1:]
	}
	plain := key == name || key == "shift+"+name
	h := s.frame.On[s.focused]
	if kind == "button" || kind == "link" {
		if !plain || name != "enter" && name != "space" {
			return false
		}
		if h != nil && h.Click != nil {
			h.Click()
		}
		if kind == "button" && (attr(n, "type") == "submit" || !hasAttr(n, "type")) {
			s.submitKey(n)
		}
		return true
	}
	in := rt.KeyDefault{Kind: kind, Key: key, Value: s.vals[s.focused], Checked: s.checked[s.focused],
		Type: attr(n, "type"), Readonly: hasAttr(n, "readonly"), Platform: s.platform}
	if n.Data == "textarea" {
		in.Type = "textarea"
	}
	in.Selection, in.HasSelection = s.sels[s.focused]
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
					in.Options = append(in.Options, rt.KeyOption{Value: value, Label: label, Disabled: hasAttr(c, "disabled")})
				} else {
					visit(c)
				}
			}
		}
		visit(n)
	}
	if kind == "text" || kind == "compound-text" {
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
		s.submitKey(n)
		return true
	}
	if out.Changed {
		if kind == "checkbox" || kind == "radio" || kind == "switch" {
			if h != nil && h.Check != nil {
				h.Check(out.Checked)
			} else {
				v := out.Checked
				s.pendingNative[s.focused] = hotty.FieldChange{Rev: s.revs[s.focused], Checked: &v}
			}
		} else if h != nil && h.Input != nil {
			h.Input(out.Value)
		} else {
			v := out.Value
			change := hotty.FieldChange{Rev: s.revs[s.focused], Value: &v}
			if out.HasSelection {
				sel := out.Selection
				change.Sel = &sel
			}
			s.pendingNative[s.focused] = change
		}
	}
	if out.HasSelection {
		s.sels[s.focused] = out.Selection
		if !out.Changed || h != nil && h.Input != nil {
			s.pendingSel[s.focused] = out.Selection
		}
	}
	return true
}

func (s *Surface) submitKey(n *html.Node) {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Data == "form" {
			if x := s.frame.On[attr(p, "id")]; x != nil && x.Submit != nil {
				x.Submit()
			}
			return
		}
	}
}
