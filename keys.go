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
//     the two above give the program, an Unbind under CUAKeys or CocoaKeys
//     written as that keymap's binding;
//   - on an element that types, each printable key an element around it
//     declares, given back (key=default), so a field types it (K27).
//
// It says how many printable keys n declares, how many keys it gives
// the program (CUAKeys' and CocoaKeys' give-backs aren't: a binding inside
// still takes them), and how many of CUAKeys' and CocoaKeys' bindings n holds
// (h.Base), which it pushed on t's stacks for n's insides; number pops
// them after.
func (s *Surface) writeKeys(n *html.Node, id string, t *tree) keyPush {
	scope, kind := 0, ""
	var mark keyMark
	for i := 0; i < len(n.Attr); i++ {
		a := n.Attr[i]
		switch a.Key {
		case keysAttr, "data-hotty-default-keys":
			s.fault("surf: " + a.Key + " on " + strconv.Quote(id) + " is the runtime's: declare keys in Go, with surf.Key(fn, keys...) or a ui.KeyScope")
		case kindAttr:
			s.fault("surf: " + kindAttr + " on " + strconv.Quote(id) + " is the kit's")
		case scopeAttr:
			// The frame's secret, then the scope: in a compiled instance,
			// its id, a dot, and the scope among the instance's own.
			k, local := -1, 0
			var rec *record
			if t.f != nil && t.f.KeyMark != "" {
				if rest, ok := strings.CutPrefix(a.Val, t.f.KeyMark); ok {
					if i := strings.IndexByte(rest, '.'); i >= 0 {
						rec = s.c.byID[rest[:i]]
						rest = rest[i+1:]
					}
					if v, err := strconv.Atoi(rest); err == nil && v > 0 && (rec != nil || !strings.Contains(a.Val[len(t.f.KeyMark):], ".")) {
						k, local = v+scopeBase(rec), v
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
			mark = keyMark{rec: rec, local: local, kind: kind}
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
		t.scopes[id] = mark
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
		binds, bound, n2 = s.writeBinds(id, h, text, multiline, binds, t)
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
		// A program binding writeBinds wrote is a give-back (only CUAKeys'
		// and CocoaKeys' bindings are program there): not a key given.
		if k, ok := strings.CutSuffix(b, "=program"); ok && !contains(bound, k) {
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
	base := 0
	if h != nil {
		t.base = append(t.base, h.Base...)
		base = len(h.Base)
	}
	return keyPush{printable: pushed, given: given, base: base}
}

// keyPush is what writeKeys pushed on a tree's stacks for an element's
// insides.
type keyPush struct{ printable, given, base int }

// baseOf is CUAKeys' or CocoaKeys' binding of key in effect on the element
// whose handlers are h: the element's own, else the nearest one's around
// it (t.base).
func baseOf(h *rt.Handlers, t *tree, key string) (rt.Bind, bool) {
	for _, c := range h.Base {
		if c.Key == key {
			return c, true
		}
	}
	for i := len(t.base) - 1; i >= 0; i-- {
		if t.base[i].Key == key {
			return t.base[i], true
		}
	}
	return rt.Bind{}, false
}

// writeBinds is the one resolver of an element's surf.Bind bindings
// (h.Binds, one per key already): it records in h.Written those that
// hold on the element, and adds to binds those the wire carries, each
// key=action. Of h.Binds it drops:
//   - a yielding one of a key an element around gives the program
//     (t.given): a shortcut or a leader there still reaches it from the
//     field (vault/knowledge/field-keys.md, 2026-11-04);
//   - as faults, one the element can't have (SPEC §10.2): an edit action
//     on a control that isn't a text field (a button's, a select's, a
//     date field's; one fault for a group's), a direct multi-line one on
//     an input (a group's is harmless there), a scroll action on a text
//     field.
//
// An Unbind of a key CUAKeys or CocoaKeys binds, on the element or around it
// (baseOf), is that keymap's binding instead of key=default: the keymap,
// not the host's platform, is what it gives the key back to. It is the
// user's Unbind, so none of the faults above is its.
//
// Where binds gives the key to the program already (a Key handler's, a
// KeyScope's) the binding is recorded but not written: the program's
// wins, but a prefix, which is the program's too, is written in its
// place. A printable key bound to prefix is pushed on t's stack, so the
// fields inside type it (K27). It returns binds, the keys it wrote, and
// how many it pushed.
func (s *Surface) writeBinds(id string, h *rt.Handlers, kind string, multiline bool, binds []string, t *tree) ([]string, []string, int) {
	var bound []string
	pushed, groupFault := 0, false
	h.Written = h.Written[:0]
	for _, b := range h.Binds {
		resolved := false
		if b.Action == string(hotty.Default) {
			if c, ok := baseOf(h, t, b.Key); ok {
				b.Action, b.Base, resolved = c.Action, true, true
			}
		}
		a := hotty.Action(b.Action)
		if b.Yield && slices.Contains(t.given, b.Key) {
			continue
		}
		switch {
		case resolved, a == hotty.Prefix, a == hotty.Default:
		case a.Scrolls():
			if kind == "text" {
				s.fault("surf.Bind: the id " + id + " is a text field, whose keymap leaves out " + b.Action + " (SPEC §10.2): bind it on what scrolls")
				continue
			}
		case kind != "text" && kind != "":
			if b.Direct {
				s.fault("surf.Bind: the id " + id + " binds " + strconv.Quote(b.Key) + " to " + b.Action + ", a text field's action, on a control that isn't one")
			} else if !groupFault {
				groupFault = true
				s.fault("surf.Bind: the id " + id + " binds a group of a text field's actions on a control that isn't one")
			}
			continue
		case b.Direct && kind == "text" && !multiline && a.Multiline():
			s.fault("surf.Bind: the id " + id + " is an input, which has no " + b.Action + " (SPEC §10.2)")
			continue
		}
		h.Written = append(h.Written, b)
		if i := slices.Index(binds, b.Key+"=program"); i >= 0 {
			if a == hotty.Prefix {
				binds[i] = b.Key + "=prefix"
			}
			continue
		}
		if a == hotty.Prefix && rt.Printable(b.Key) {
			t.printable = append(t.printable, b.Key)
			pushed++
		}
		binds = append(binds, b.Key+"="+b.Action)
		bound = append(bound, b.Key)
	}
	return binds, bound, pushed
}

// A keyMark is a kit control's place under the KeyScopes: the scope it
// is in (one-based), of the frame's or, in a compiled instance, of the
// instance's (rec), and its kind, as ui marked it.
type keyMark struct {
	rec   *record
	local int
	kind  string
}

// markScope is m's scope in the last frame's KeyScopes (one-based), 0 for
// none: an instance's own are where the frame took them this render.
func (s *Surface) markScope(m keyMark) int {
	if m.local == 0 {
		return 0
	}
	return m.local + scopeBase(m.rec)
}

// scopeBase is where r's KeyScopes start in its frame's.
func scopeBase(r *record) int {
	n := 0
	for ; r != nil; r = r.up {
		n += r.scopeOff
	}
	return n
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
			// Every input of the resolver (writeBinds), which, with the
			// same HTML, makes the same data-keys: a Yield or Override
			// toggled alone repaints.
			for _, x := range h.Binds {
				b = append(b, ' ')
				b = append(b, x.Key...)
				b = append(b, '=')
				b = append(b, x.Action...)
				if x.Yield {
					b = append(b, '~')
				}
				if x.Direct {
					b = append(b, '!')
				}
				if x.Base {
					b = append(b, '^')
				}
			}
			// What an Unbind here or inside goes back to (CUAKeys', CocoaKeys').
			for _, x := range h.Base {
				b = append(b, " ^"...)
				b = append(b, x.Key...)
				b = append(b, '=')
				b = append(b, x.Action...)
			}
			b = append(b, '\n')
		}
	}
	for i, sc := range f.KeyScopes {
		b = append(b, '|')
		if i > 0 && sc == f.KeyScopes[i-1] {
			// The scope before's again, which forwards what it did.
			b = append(b, '=')
			continue
		}
		for _, k := range rt.Forwarded(sc, "") {
			b = append(b, ' ')
			b = append(b, k...)
		}
	}
	return b
}
