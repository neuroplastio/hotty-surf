package surf

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The keys a surface gives the program. A key is declared in Go, by the
// code that takes it: surf.Key's keys, and a ui.KeyScope's bindings
// around a kit control. SPEC §10.2's data-keys is how a host learns of
// them, so the surface writes it, here and nowhere else, after the view
// has rendered: a template carries no key binds. A data-keys the view
// wrote is a fault and is dropped, as a runtime id is; so are the kit's
// private markers, when they aren't the ones the kit gave.

// keysAttr is SPEC §10.2's attribute, which only writeKeys writes.
const keysAttr = "data-keys"

// scopeAttr is the kit's private marker on a control under a
// ui.KeyScope: the frame's secret (rt.Frame.KeyMark), then which of the
// frame's KeyScopes it is in (one-based); its kind is the frame's
// (KeyKinds). writeKeys takes it off the document into the tree, by id,
// so no host sees it. One without the secret, or naming a scope a
// marker named already, is a template's: a fault, and dropped. kindAttr
// and data-hotty-default-keys, which v0.7's first kit wrote, are always
// a template's.
const (
	scopeAttr = "data-hotty-keyscope"
	kindAttr  = "data-hotty-keykind"
)

// writeKeys takes what the view wrote of keys off n, an element of t with
// the id id, and writes its data-keys, sorted by key:
//   - its Key handlers' keys, each the program's; escape, which a host
//     ignores (it reaches the program anyway), only OwnKeys lists; tab
//     only on a text field, the one place SPEC lets it go;
//   - the keys the KeyScopes around a kit control forward;
//   - its surf.Bind bindings, key=action, but for a key
//     the two above give the program;
//   - on an element that types, each printable key an element around it
//     declares, given back (key=default), so a field types it (K27).
//
// It says how many printable keys n declares, and how many keys it gives
// the program, which it pushed on t's stacks for n's insides; number pops
// them after.
func (s *Surface) writeKeys(n *html.Node, id string, t *tree) keyPush {
	scope, kind := 0, ""
	for i := 0; i < len(n.Attr); i++ {
		a := n.Attr[i]
		switch a.Key {
		case keysAttr, "data-hotty-default-keys":
			s.fault("surf: " + a.Key + " on " + strconv.Quote(id) + " is the runtime's: declare keys in Go, with surf.Key(fn, keys...) or a ui.KeyScope")
		case kindAttr:
			s.fault("surf: " + kindAttr + " on " + strconv.Quote(id) + " is the kit's")
		case scopeAttr:
			k := -1
			if t.f != nil && t.f.KeyMark != "" {
				if rest, ok := strings.CutPrefix(a.Val, t.f.KeyMark); ok {
					if v, err := strconv.Atoi(rest); err == nil {
						k = v
					}
				}
			}
			if k <= 0 || k > len(t.f.KeyScopes) || t.marks[k] {
				s.fault("surf: " + scopeAttr + " on " + strconv.Quote(id) + " is the kit's")
				break
			}
			if t.marks == nil {
				t.marks = map[int]bool{}
			}
			t.marks[k], scope, kind = true, k, t.f.KeyKinds[k-1]
		default:
			continue
		}
		n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
		i--
	}
	if scope > 0 {
		if t.scopes == nil {
			t.scopes = map[string]keyMark{}
		}
		t.scopes[id] = keyMark{scope: scope, kind: kind}
	}
	if t.f == nil {
		return keyPush{}
	}
	var own []string
	h := t.f.On[id]
	if h != nil {
		own = h.Keys
	}
	var binds []string
	text, multiline := nativeKind(n)
	pushed := 0
	for _, k := range own {
		switch {
		case k == "escape":
			continue
		case k == "tab" && text != "text":
			s.fault("surf.Key: the id " + id + " declares tab, which only a text field gives the program (SPEC §10.2)")
			continue
		}
		binds = append(binds, k+"=program")
		if rt.Printable(k) {
			t.printable = append(t.printable, k)
			pushed++
		}
	}
	if scope > 0 {
		for _, k := range rt.Forwarded(t.f.KeyScopes[scope-1], kind) {
			if !contains(own, k) {
				binds = append(binds, k+"=program")
			}
		}
	}
	var bound []string
	if h != nil && len(h.Binds) > 0 {
		var n2 int
		binds, bound, n2 = s.writeBinds(id, h.Binds, text, multiline, binds, t)
		pushed += n2
	}
	if rt.Types(n) {
		for _, k := range t.printable[:len(t.printable)-pushed] {
			if !contains(own, k) && !contains(bound, k) && !slices.Contains(binds, k+"=default") {
				binds = append(binds, k+"=default")
			}
		}
	}
	// One binding per key, the first: a host reads a key's last binding,
	// and sorting orders two of one key by action.
	seen := make(map[string]bool, len(binds))
	binds = slices.DeleteFunc(binds, func(b string) bool {
		k := b[:strings.LastIndexByte(b, '=')]
		if seen[k] {
			return true
		}
		seen[k] = true
		return false
	})
	given := 0
	for _, b := range binds {
		if k, ok := strings.CutSuffix(b, "=program"); ok {
			t.given = append(t.given, k)
			given++
		} else if k, ok := strings.CutSuffix(b, "=prefix"); ok {
			t.given = append(t.given, k)
			given++
		}
	}
	if len(binds) > 0 {
		slices.Sort(binds)
		n.Attr = append(n.Attr, html.Attribute{Key: keysAttr, Val: strings.Join(binds, " ")})
	}
	return keyPush{printable: pushed, given: given}
}

// keyPush is what writeKeys pushed on a tree's stacks for an element's
// insides.
type keyPush struct{ printable, given int }

// writeBinds adds an element's surf.Bind bindings to
// binds, one per key, each key=action, but where binds gives the key to
// the program already (a Key handler's, a KeyScope's): that one wins, but
// a prefix, which is the program's too, is written in its place. A
// Readline binding also leaves alone a key an element around gives the
// program (t.given): a shortcut or a leader there still reaches it from
// the field (vault/knowledge/field-keys.md, 2026-11-04). It drops, as
// faults, a binding the element can't have (SPEC §10.2): an edit action
// on a control that isn't a text field (kind: a button's, a select's, a
// date field's), a multi-line one an input binds, a scroll action a text
// field binds; a group's multi-line action on an input is harmless. A printable key bound to prefix is pushed on t's stack, so
// the fields inside type it (K27). It returns binds, the keys it wrote,
// and how many it pushed.
func (s *Surface) writeBinds(id string, bs []rt.Bind, kind string, multiline bool, binds []string, t *tree) ([]string, []string, int) {
	var bound []string
	pushed, groupFault := 0, false
	for _, b := range bs {
		a := hotty.Action(b.Action)
		if i := slices.Index(binds, b.Key+"=program"); i >= 0 {
			if a == hotty.Prefix {
				binds[i] = b.Key + "=prefix"
			}
			continue
		}
		if b.Yield && slices.Contains(t.given, b.Key) {
			continue
		}
		switch {
		case a == hotty.Prefix:
			if rt.Printable(b.Key) {
				t.printable = append(t.printable, b.Key)
				pushed++
			}
		case a.Scrolls():
			if kind == "text" {
				s.fault("surf.Bind: the id " + id + " is a text field, whose keymap leaves out " + b.Action + " (SPEC §10.2): bind it on what scrolls")
				continue
			}
		case kind != "text" && kind != "":
			if !b.Group {
				s.fault("surf.Bind: the id " + id + " binds " + strconv.Quote(b.Key) + " to " + b.Action + ", a text field's action, on a control that isn't one")
			} else if !groupFault {
				groupFault = true
				s.fault("surf.Bind: the id " + id + " binds a group of a text field's actions (Readline's) on a control that isn't one")
			}
			continue
		case !b.Group && kind == "text" && !multiline && a.Multiline():
			s.fault("surf.Bind: the id " + id + " is an input, which has no " + b.Action + " (SPEC §10.2)")
			continue
		}
		binds = append(binds, b.Key+"="+b.Action)
		bound = append(bound, b.Key)
	}
	return binds, bound, pushed
}

// A keyMark is a kit control's place under the KeyScopes: the frame's
// scope it is in (one-based) and its kind, as ui marked it.
type keyMark struct {
	scope int
	kind  string
}

// keysig is what a render declares in keys, which its HTML no longer
// says: two renders with the same HTML and different keys aren't the same
// (paint).
func keysig(b []byte, f *rt.Frame) []byte {
	b = b[:0]
	for _, id := range f.Order {
		if h := f.On[id]; h != nil && len(h.Keys)+len(h.Binds) > 0 {
			b = append(b, id...)
			for _, k := range h.Keys {
				b = append(b, ' ')
				b = append(b, k...)
			}
			for _, x := range h.Binds {
				b = append(b, ' ')
				b = append(b, x.Key...)
				b = append(b, '=')
				b = append(b, x.Action...)
			}
			b = append(b, '\n')
		}
	}
	for _, sc := range f.KeyScopes {
		b = append(b, '|')
		for _, k := range rt.Forwarded(sc, "") {
			b = append(b, ' ')
			b = append(b, k...)
		}
	}
	return b
}
