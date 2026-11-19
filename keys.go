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
//   - its surf.Bind and surf.Readline bindings, key=action, but for a key
//     the two above give the program;
//   - on an element that types, each printable key an element around it
//     declares, given back (key=default), so a field types it (K27).
//
// It says how many printable keys n declares, which it pushed on t's
// stack for n's insides; number pops them after.
func (s *Surface) writeKeys(n *html.Node, id string, t *tree) int {
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
		return 0
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
		binds, bound, n2 = s.writeBinds(n, id, h.Binds, text == "text", multiline, binds, t)
		pushed += n2
	}
	if rt.Types(n) {
		for _, k := range t.printable[:len(t.printable)-pushed] {
			if !contains(own, k) && !contains(bound, k) && !slices.Contains(binds, k+"=default") {
				binds = append(binds, k+"=default")
			}
		}
	}
	if len(binds) > 0 {
		slices.Sort(binds)
		n.Attr = append(n.Attr, html.Attribute{Key: keysAttr, Val: strings.Join(binds, " ")})
	}
	return pushed
}

// writeBinds adds n's surf.Bind and surf.Readline bindings to binds, each
// key=action, but where binds gives the key to the program already (a
// Key handler's, a KeyScope's): that one wins, but a prefix, which is
// the program's too, is written in its place. It drops, as faults, a
// binding the element can't have (SPEC §10.2): an edit action where no
// text field is (n or inside it), a multi-line one an input binds, a
// scroll action a text field binds. A printable key bound to prefix is
// pushed on t's stack, so the fields inside type it (K27). It returns
// binds, the keys it wrote, and how many it pushed.
func (s *Surface) writeBinds(n *html.Node, id string, bs []rt.Bind, field, multiline bool, binds []string, t *tree) ([]string, []string, int) {
	var bound []string
	pushed, holds, readlineFault := 0, -1, false
	for _, b := range bs {
		a := hotty.Action(b.Action)
		if i := slices.Index(binds, b.Key+"=program"); i >= 0 {
			// The program's already; a prefix is the program's too, and
			// holds the keys after it for it.
			if a == hotty.Prefix {
				binds[i] = b.Key + "=prefix"
			}
			continue
		}
		switch {
		case a == hotty.Prefix:
			if rt.Printable(b.Key) {
				t.printable = append(t.printable, b.Key)
				pushed++
			}
		case a.Scrolls():
			if field {
				s.fault("surf.Bind: the id " + id + " is a text field, whose keymap leaves out " + b.Action + " (SPEC §10.2): bind it on what scrolls")
				continue
			}
		default:
			if holds < 0 {
				holds = 0
				if field || holdsField(n) {
					holds = 1
				}
			}
			if holds == 0 {
				if !b.Readline {
					s.fault("surf.Bind: the id " + id + " binds " + strconv.Quote(b.Key) + " to " + b.Action + ", a text field's action, but is no text field and holds none")
				} else if !readlineFault {
					readlineFault = true
					s.fault("surf.Readline: the id " + id + " is no text field and holds none")
				}
				continue
			}
			if !b.Readline && field && !multiline && a.Multiline() {
				s.fault("surf.Bind: the id " + id + " is an input, which has no " + b.Action + " (SPEC §10.2)")
				continue
			}
		}
		binds = append(binds, b.Key+"="+b.Action)
		bound = append(bound, b.Key)
	}
	return binds, bound, pushed
}

// holdsField says whether a text field is inside n.
func holdsField(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if k, _ := nativeKind(c); k == "text" || holdsField(c) {
			return true
		}
	}
	return false
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
