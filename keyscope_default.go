package surf

import (
	"slices"
	"strings"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyedit"
	"neuroplast.io/hotty-surf/internal/keydefault"
)

// nativeKeyDefault runs the same control decisions as cells after all
// scopes decline a forwarded key. Unforwarded keys stay with the host.
func (s *Surface) nativeKeyDefault(key string) bool {
	n := s.byID[s.focused]
	if n == nil || hasAttr(n, "disabled") {
		return false
	}
	kind := attr(n, "data-hotty-keykind")
	name := key
	if i := strings.LastIndexByte(key, '+'); i >= 0 {
		name = key[i+1:]
	}
	plain := key == name || key == "shift+"+name
	h := s.frame.On[s.focused]
	switch kind {
	case "button", "link":
		if !plain || name != "enter" && name != "space" {
			return false
		}
		if h != nil && h.Click != nil {
			h.Click()
		}
		if kind == "button" && (attr(n, "type") == "submit" || !hasAttr(n, "type")) {
			for p := n.Parent; p != nil; p = p.Parent {
				if p.Data == "form" {
					if x := s.frame.On[attr(p, "id")]; x != nil && x.Submit != nil {
						x.Submit()
					}
					break
				}
			}
		}
		return true
	case "checkbox", "radio", "switch":
		if !plain || name != "space" {
			return false
		}
		next, changed := keydefault.Check(s.checked[s.focused], kind == "radio")
		if changed {
			if h != nil && h.Check != nil {
				h.Check(next)
			} else {
				s.pendingNative[s.focused] = hotty.FieldChange{Rev: s.revs[s.focused], Checked: &next}
			}
		}
		return true
	case "select":
		if !plain {
			return false
		}
		var options []keydefault.Option
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
					options = append(options, keydefault.Option{Value: value, Label: label, Disabled: hasAttr(c, "disabled")})
				} else {
					visit(c)
				}
			}
		}
		visit(n)
		value, changed, taken := keydefault.Select(options, s.vals[s.focused], key, name)
		if changed {
			if h != nil && h.Input != nil {
				h.Input(value)
			} else {
				s.pendingNative[s.focused] = hotty.FieldChange{Rev: s.revs[s.focused], Value: &value}
			}
		}
		return taken
	case "text", "compound-text":
		return s.nativeTextKey(n, key)
	}
	return false
}

func (s *Surface) nativeTextKey(n *html.Node, key string) bool {
	// The original keymaps are used for fallback, without the forwarding
	// bindings added for this scope.
	var maps []string
	for p := n; p != nil; p = p.Parent {
		v := attr(p, "data-keys")
		if hasAttr(p, "data-hotty-keyscope") {
			v = attr(p, "data-hotty-default-keys")
		}
		if v != "" {
			maps = append(maps, v)
		}
	}
	slices.Reverse(maps)
	km := hotty.Resolve(s.platform, n.Data == "textarea", maps...)
	value := s.vals[s.focused]
	f := hottyedit.Field{Value: value, Multiline: n.Data == "textarea", Password: attr(n, "type") == "password"}
	positions := []int{0}
	for g := graphemes.FromString(value); g.Next(); {
		positions = append(positions, positions[len(positions)-1]+len(g.Value()))
	}
	selection, known := s.sels[s.focused]
	if !known {
		selection = [2]int{len(value), len(value)}
	}
	char := func(b int) int {
		for i, pos := range positions {
			if pos >= b {
				return i
			}
		}
		return len(positions) - 1
	}
	f.Select(char(selection[0]), char(selection[1]))
	action, changed := f.Key(km, key)
	if action == "" || action == hotty.Program || action == hotty.Copy || action == hotty.Cut || action == hotty.Paste {
		return false
	}
	if hasAttr(n, "readonly") && changed {
		return true
	}
	if action == hotty.Submit {
		for p := n.Parent; p != nil; p = p.Parent {
			if p.Data == "form" {
				if x := s.frame.On[attr(p, "id")]; x != nil && x.Submit != nil {
					x.Submit()
				}
				break
			}
		}
		return true
	}
	hasInput := false
	if changed {
		if handlers := s.frame.On[s.focused]; handlers != nil && handlers.Input != nil {
			hasInput = true
			handlers.Input(f.Value)
		}
	}
	anchor, _ := f.Anchor()
	bytesAt := func(i int) int {
		g := graphemes.FromString(f.Value)
		off := 0
		for j := 0; j < i && g.Next(); j++ {
			off += len(g.Value())
		}
		return off
	}
	sel := [2]int{bytesAt(anchor), bytesAt(f.Caret)}
	if changed && !hasInput {
		v := f.Value
		s.pendingNative[s.focused] = hotty.FieldChange{Rev: s.revs[s.focused], Value: &v, Sel: &sel}
	} else {
		s.pendingSel[s.focused] = sel
	}
	s.sels[s.focused] = sel
	return true
}
