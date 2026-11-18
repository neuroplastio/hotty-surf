package surf

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"

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

// The kit's private markers on a control under a ui.KeyScope: which of
// the frame's KeyScopes it is in (one-based) and what kind of control it
// is. ui writes them; writeKeys takes them off the document into the
// tree, by id, so no host sees them, and checks each names a scope once.
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
		case scopeAttr:
			k, err := strconv.Atoi(a.Val)
			if err != nil || t.f == nil || k <= 0 || k > len(t.f.KeyScopes) || t.marks[k] {
				s.fault("surf: " + scopeAttr + " on " + strconv.Quote(id) + " is the kit's")
				break
			}
			if t.marks == nil {
				t.marks = map[int]bool{}
			}
			t.marks[k], scope = true, k
		case kindAttr:
			kind = a.Val
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
	} else if kind != "" {
		s.fault("surf: " + kindAttr + " on " + strconv.Quote(id) + " is the kit's")
	}
	if t.f == nil {
		return 0
	}
	var own []string
	if h := t.f.On[id]; h != nil {
		own = h.Keys
	}
	var binds []string
	text, _ := nativeKind(n)
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
	if rt.Types(n) {
		for _, k := range t.printable[:len(t.printable)-pushed] {
			if !contains(own, k) && !slices.Contains(binds, k+"=default") {
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
		if h := f.On[id]; h != nil && len(h.Keys) > 0 {
			b = append(b, id...)
			for _, k := range h.Keys {
				b = append(b, ' ')
				b = append(b, k...)
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
