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
	h := t.f.Get(id)
	if h != nil {
		own = h.Keys
	}
	var binds []string
	text, multiline := nativeKind(n)
	pushed := 0
	// The element's bindings as laid out where no edit-shortcut group is
	// in effect, or again for the edit shortcuts in effect here.
	var lb, lbase, ledits []rt.Bind
	if h != nil {
		lb, lbase, ledits = h.Binds, h.Base, h.Edits
		if t.f.EditKeys != nil && (t.editN > 0 || slices.ContainsFunc(h.Flat, func(b rt.Bind) bool { return b.Edit })) &&
			slices.ContainsFunc(h.Flat, func(b rt.Bind) bool { return b.Base && !b.Edit && !b.Direct }) {
			lb, lbase, ledits = layout(nil, id, h.Keys, h.Flat, func(b rt.Bind) bool { return t.heldAt(h.Flat, b) })
		}
		if len(lb) > 0 || t.f.EditKeys != nil && text == "text" {
			h.Written = h.Written[:0]
		}
	}
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
	var unbound []rt.Bind
	faulted := false
	if len(lb) > 0 {
		var n2 int
		binds, bound, unbound, n2, faulted = s.writeBinds(id, h, lb, lbase, text, multiline, binds, t)
		pushed += n2
	}
	// The edit-shortcut groups' bindings here, and the Unbinds that went
	// back to one, which only fields are written: pushed for the fields
	// inside, and resolved on this one if it is a field.
	edits := 0
	if len(ledits) > 0 && text != "text" && text != "" {
		if !faulted {
			s.fault("surf.Bind: the id " + id + " binds a group of a text field's actions on a control that isn't one")
		}
		ledits = nil
	}
	for _, b := range slices.Concat(ledits, unbound) {
		t.edit = append(t.edit, keyAt{key: b.Key, action: b.Action, edit: true, yield: b.Yield})
		edits++
	}
	t.editN += edits
	if text == "text" && t.editN > 0 {
		binds, bound = s.writeEdits(h, binds, bound, t)
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
		// A program binding writeBinds wrote is a give-back (only CUAKeys' and
		// CocoaKeys' bindings are program there): not a key given.
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
	// What the element writes of the edit shortcuts' keys, for the fields
	// inside to resolve them against.
	written := 0
	if t.f.EditKeys != nil {
		for _, b := range binds {
			i := strings.LastIndexByte(b, '=')
			if k, a := b[:i], b[i+1:]; slices.Contains(t.f.EditKeys, k) {
				given := a == string(hotty.Prefix) || a == string(hotty.Program) && !contains(bound, k)
				t.edit = append(t.edit, keyAt{key: k, action: a, given: given})
				written++
			}
		}
	}
	base := 0
	if h != nil {
		t.base = append(t.base, lbase...)
		base = len(lbase)
	}
	return keyPush{printable: pushed, given: given, base: base, edit: edits + written, editN: edits}
}

// keyPush is what writeKeys pushed on a tree's stacks for an element's
// insides.
type keyPush struct{ printable, given, base, edit, editN int }

// A keyAt is one of the edit shortcuts' keys on an element around the
// one being numbered (tree.edit): an edit-shortcut group's binding there
// (edit), which only fields are written, or what the element wrote of
// the key, given saying it gives the program the key (a Key handler's, a
// KeyScope's, a prefix).
type keyAt struct {
	key, action        string
	edit, yield, given bool
}

// heldAt is layout's held for an element under an edit-shortcut group
// (its own, among flat, or one around it): the nearest edit-shortcut
// binding of b's key holds it if it is a real one, not a give-back; a
// key none of them binds is held as the host's edit shortcuts hold it.
func (t *tree) heldAt(flat []rt.Bind, b rt.Bind) bool {
	var e *rt.Bind
	for i := range flat {
		if x := &flat[i]; x.Edit && x.Key == b.Key && (e == nil || !giveBack(*x) || giveBack(*e)) {
			e = x
		}
	}
	if e != nil {
		return !giveBack(*e)
	}
	for i := len(t.edit) - 1; i >= 0; i-- {
		if x := t.edit[i]; x.edit && x.key == b.Key {
			return x.action != string(hotty.Program)
		}
	}
	return b.Held
}

// writeEdits writes on a text field the edit-shortcut groups' bindings in
// effect on it, each the nearest: of each of the edit shortcuts' keys
// that the field doesn't bind itself (binds), the edit-shortcut binding
// nearest it (t.edit), unless an element between writes the key (a field
// leaves out a scroll action), or gives it the program where the binding
// yields. A binding the field would do anyway is left out: the action
// its default keymap (Platform.Edit as the host reports it) and the
// data-keys above it give the key. So is a give-back of a key they give
// no edit action: a give-back takes a key only from an edit shortcut. It
// returns binds and the keys written (bound), and records them in
// h.Written.
func (s *Surface) writeEdits(h *rt.Handlers, binds, bound []string, t *tree) ([]string, []string) {
	defs := editDefaults(t.f)
	for _, k := range t.f.EditKeys {
		if slices.ContainsFunc(binds, func(b string) bool { return b[:strings.LastIndexByte(b, '=')] == k }) {
			continue
		}
		at := -1
		for i := len(t.edit) - 1; i >= 0; i-- {
			if x := t.edit[i]; x.edit && x.key == k {
				at = i
				break
			}
		}
		if at < 0 {
			continue
		}
		e, between := t.edit[at], false
		for _, x := range t.edit[at+1:] {
			if x.key == k && !x.edit && !hotty.Action(x.action).Scrolls() && (!x.given || e.yield) {
				between = true
				break
			}
		}
		if between || e.yield && slices.Contains(t.given, k) {
			continue
		}
		have := defs[k]
		for i := len(t.edit) - 1; i >= 0; i-- {
			x := t.edit[i]
			if x.key != k || x.edit || hotty.Action(x.action).Scrolls() {
				continue
			}
			switch hotty.Action(x.action) {
			case hotty.Default:
			case hotty.Program, hotty.Prefix:
				have = ""
			default:
				have = x.action
			}
			break
		}
		want := e.action
		if want == string(hotty.Program) {
			want = ""
		}
		if have == want || want == "" && !editAction(have) {
			// It changes nothing, or it is a give-back of a key the field
			// doesn't use as an edit shortcut (macOS's ctrl+a at the
			// line's start), which it only fills.
			continue
		}
		binds = append(binds, k+"="+e.action)
		bound = append(bound, k)
		if h != nil {
			h.Written = append(h.Written, rt.Bind{Key: k, Action: e.action, Yield: e.yield, Base: true, Edit: true})
		}
	}
	return binds, bound
}

// editDefaults is what a text field's default keymap does with each of
// the frame's edit shortcuts' keys (rt.Editing.Defaults), made once a
// platform.
func editDefaults(f *rt.Frame) map[string]string {
	if f.Editing == nil {
		f.Editing = new(rt.Editing)
	}
	if f.Editing.Defaults == nil {
		m := hotty.Resolve(f.Platform, true)
		f.Editing.Defaults = make(map[string]string, len(f.EditKeys))
		for _, k := range f.EditKeys {
			f.Editing.Defaults[k] = string(m.Lookup(k))
		}
	}
	return f.Editing.Defaults
}

// baseOf is CUAKeys', CocoaKeys' or an edit-shortcut group's binding of
// key in effect on the element whose base (rt.Handlers.Base) is base: the
// element's own, else the nearest one's around it (t.base).
func baseOf(base []rt.Bind, t *tree, key string) (rt.Bind, bool) {
	for _, c := range base {
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
// (lb, h.Binds or as laid out again under the edit shortcuts in effect,
// one per key already): it records in h.Written those that
// hold on the element, and adds to binds those the wire carries, each
// key=action. Of lb it drops:
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
// fields inside type it (K27). An Unbind that goes back to an
// edit-shortcut group's binding isn't written here: it is returned
// (unbound), for the fields' resolver (writeEdits) to write on the
// fields inside, and on this one if it is one. It returns
// binds, the keys it wrote, those Unbinds, how many it pushed, and
// whether it faulted a group.
func (s *Surface) writeBinds(id string, h *rt.Handlers, lb, base []rt.Bind, kind string, multiline bool, binds []string, t *tree) ([]string, []string, []rt.Bind, int, bool) {
	var bound []string
	var unbound []rt.Bind
	pushed, groupFault := 0, false
	for _, b := range lb {
		resolved := false
		if b.Action == string(hotty.Default) {
			if c, ok := baseOf(base, t, b.Key); ok {
				b.Action, b.Base, b.Edit, resolved = c.Action, true, c.Edit, true
			}
		}
		if b.Edit {
			// An Unbind that goes back to an edit-shortcut group's
			// binding: only fields are written it, and the fields'
			// resolver (writeEdits) writes it, this one's too.
			unbound = append(unbound, b)
			continue
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
	return binds, bound, unbound, pushed, groupFault
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
	// By slot, not by Order: the same handlers make the same data-keys
	// in any order, and a kept instance's are in their slots already.
	f.Each(func(id string, h *rt.Handlers) {
		if len(h.Keys)+len(h.Flat) > 0 {
			b = append(b, id...)
			for _, k := range h.Keys {
				b = append(b, ' ')
				b = append(b, k...)
			}
			// Every input of the resolver (writeKeys), which, with the
			// same HTML, makes the same data-keys: a Yield or Override
			// toggled alone repaints, and so does an edit-shortcut group,
			// which the fields inside are written. Flat is what Binds,
			// Base and Edits are laid out from, here and again under the
			// edit shortcuts in effect.
			for _, x := range h.Flat {
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
				if x.Edit {
					b = append(b, '%')
				}
				if x.Held {
					b = append(b, '#')
				}
			}
			b = append(b, '\n')
		}
	})
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
