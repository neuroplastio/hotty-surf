package surf

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// Keymap attributes are written after rendering from the Go handlers
// and scopes. Templates may not provide either attribute.
const (
	keysAttr      = "data-keys"
	keysInnerAttr = "data-keys-inner"
)

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

func (s *Surface) readScope(t *tree, value string) (int, keyMark) {
	if t.f == nil || t.f.KeyMark == "" {
		return 0, keyMark{}
	}
	rest, ok := strings.CutPrefix(value, t.f.KeyMark)
	if !ok {
		return 0, keyMark{}
	}
	var rec *record
	if id, local, ok := strings.Cut(rest, "."); ok {
		rec = s.c.byID[id]
		if rec == nil {
			return 0, keyMark{}
		}
		rest = local
	}
	local, err := strconv.Atoi(rest)
	if err != nil || local <= 0 {
		return 0, keyMark{}
	}
	index := local + scopeBase(rec)
	if index > len(t.f.KeyScopes) {
		return 0, keyMark{}
	}
	return index, keyMark{rec: rec, local: local, kind: t.f.KeyKinds[index-1]}
}

func focusableKeys(n *html.Node) bool {
	kind, _ := rt.NativeKind(n)
	return kind != "" || hasAttr(n, "tabindex") || n.Data == "input" && attr(n, "type") != "hidden" || n.Data == "select"
}

// placeScopes gives a shared scope the nearest available enclosing element
// whose inner map reaches only its controls. An interleaved scope uses the
// same rule on each covered branch, preserving the authored focus order.
func (s *Surface) placeScopes(root *html.Node, t *tree) {
	if s.legacy || len(t.f.KeyScopes) == 0 {
		return
	}
	controls := map[*rt.KeyScope][]*html.Node{}
	focusCount := map[*html.Node]int{}
	walkEls(root, func(n *html.Node) {
		if focusableKeys(n) {
			for p := n; p != nil; p = p.Parent {
				focusCount[p]++
			}
		}
		if i, _ := s.readScope(t, attr(n, scopeAttr)); i > 0 {
			for scope := t.f.KeyScopes[i-1]; scope != nil; scope = scope.Parent {
				controls[scope] = append(controls[scope], n)
			}
		}
	})
	t.scopeKeys = map[*html.Node][]string{}
	t.scopeInner = map[*html.Node][]string{}
	roots := map[*html.Node]bool{}
	t.scopeFocus = map[*html.Node]int{}
	for i := len(t.f.Scopes) - 1; i >= 0; i-- {
		scope := t.f.Scopes[i]
		nodes := controls[scope]
		if len(nodes) == 0 || !rt.HasKeyBindings(scope) {
			continue
		}
		keys := rt.ScopeKeys(scope)
		if len(nodes) == 1 {
			t.scopeKeys[nodes[0]] = append(t.scopeKeys[nodes[0]], keys...)
			continue
		}
		members := make(map[*html.Node]bool, len(nodes))
		covered := map[*html.Node]int{}
		scopedFocus := map[*html.Node]int{}
		anchor := nodes[0]
		for _, n := range nodes {
			if members[n] {
				continue
			}
			members[n] = true
			for !within(n, anchor) {
				anchor = anchor.Parent
			}
			focusable := focusableKeys(n)
			for p := n; p != nil; p = p.Parent {
				covered[p]++
				if focusable {
					scopedFocus[p]++
				}
			}
		}
		common := anchor
		safe := func(n *html.Node) bool {
			count := focusCount[n] - scopedFocus[n]
			// The inner map does not apply to the anchor itself.
			if !members[n] && focusableKeys(n) {
				count--
			}
			return count == 0
		}
		for p := anchor; p != nil && safe(p); p = p.Parent {
			if !roots[p] {
				anchor = p
				break
			}
			if p == root {
				// Only the runtime's root needs a synthetic boundary.
				wrapper := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []html.Attribute{
					{Key: rt.KeyScopeBoundary, Val: ""},
					{Key: "style", Val: "display:contents"},
				}}
				for root.FirstChild != nil {
					n := root.FirstChild
					root.RemoveChild(n)
					wrapper.AppendChild(n)
				}
				root.AppendChild(wrapper)
				focusCount[wrapper] = focusCount[root]
				roots[wrapper], t.scopeInner[wrapper] = roots[root], t.scopeInner[root]
				delete(roots, root)
				anchor = root
				break
			}
		}
		inherited := rt.ProgramKeys(keys)
		var place func(*html.Node)
		place = func(n *html.Node) {
			if covered[n] == 0 {
				return
			}
			if covered[n] > 1 && safe(n) && !roots[n] {
				roots[n], t.scopeInner[n] = true, keys
				// A focusable container can act as the scope's anchor.
				kind, _ := rt.NativeKind(n)
				if !members[n] && n == common && kind == "" && focusableKeys(n) {
					t.scopeFocus[n] = i + 1
					members[n] = true
					for parent := scope.Parent; parent != nil; parent = parent.Parent {
						controls[parent] = append(controls[parent], n)
					}
				}
				if members[n] {
					t.scopeKeys[n] = append(t.scopeKeys[n], keys...)
				}
				return
			}
			if members[n] {
				// A branch containing only one covered control still uses
				// the reader's inherited-key protections.
				km := hotty.Resolve(t.f.Platform, rt.KeyReader(n), "", inherited)
				for _, key := range keys {
					if km.Program(key) {
						t.scopeKeys[n] = append(t.scopeKeys[n], key)
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				place(c)
			}
		}
		place(anchor)
	}
}

func within(n, ancestor *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}

// writeKeys removes private markers and writes the element's keymaps.
// It reports what it pushed onto the inherited binding stacks for number.
func (s *Surface) writeKeys(n *html.Node, id string, t *tree) keyPush {
	scope := 0
	var mark keyMark
	for i := 0; i < len(n.Attr); i++ {
		a := n.Attr[i]
		switch a.Key {
		case keysAttr, keysInnerAttr, "data-hotty-default-keys":
			s.fault("surf: " + a.Key + " on " + strconv.Quote(id) + " is the runtime's: declare keys in Go, with surf.Key(fn, keys...) or a ui.KeyScope")
		case kindAttr:
			s.fault("surf: " + kindAttr + " on " + strconv.Quote(id) + " is the kit's")
		case scopeAttr:
			// The frame's secret, then the scope: in a compiled instance,
			// its id, a dot, and the scope among the instance's own.
			k, placement := s.readScope(t, a.Val)
			if k <= 0 || k > len(t.f.KeyScopes) || t.marks[k] {
				s.fault("surf: " + scopeAttr + " on " + strconv.Quote(id) + " is the kit's")
				break
			}
			if t.marks == nil {
				t.marks = map[int]bool{}
			}
			t.marks[k], scope = true, k
			mark = placement
		default:
			continue
		}
		n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
		i--
	}
	mark.anchor = t.scopeFocus[n]
	if scope > 0 || mark.anchor > 0 {
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
	text, multiline := rt.NativeKind(n)
	pushed := 0
	// The element's bindings, laid out under the edit shortcuts in effect
	// here, into the tree's scratch: nothing keeps them past this element.
	var lb, lbase, ledits []rt.Bind
	if h != nil {
		if len(h.Flat) > 0 {
			t.lb, t.lbase, t.ledits = layout(s, id, h.Keys, h.Flat, t, t.lb[:0], t.lbase[:0], t.ledits[:0])
			lb, lbase, ledits = t.lb, t.lbase, t.ledits
		}
		if len(h.Flat) > 0 || t.f.EditKeys != nil && text == "text" {
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
		if s.legacy && rt.Printable(k) {
			t.printable = append(t.printable, k)
			pushed++
		}
	}
	scopeKeys := t.scopeKeys[n]
	if s.legacy && scope > 0 {
		scopeKeys = rt.Forwarded(t.f.KeyScopes[scope-1])
	}
	if s.legacy {
		for _, k := range scopeKeys {
			if k != "escape" && (!rt.Types(n) || !rt.Printable(k)) {
				binds = append(binds, k+"=program")
			}
		}
	} else if text == "text" && len(scopeKeys) > 0 {
		// A scope directly around a field still leaves typing to the host.
		km := hotty.Resolve(t.f.Platform, rt.KeyReader(n), "", rt.ProgramKeys(scopeKeys))
		for _, k := range scopeKeys {
			if km.Program(k) {
				binds = append(binds, k+"=program")
			}
		}
	} else {
		for _, k := range scopeKeys {
			if k != "escape" {
				binds = append(binds, k+"=program")
			}
		}
	}
	var inner []string
	for _, k := range t.scopeInner[n] {
		if k != "escape" {
			inner = append(inner, k+"=program")
		}
	}
	scopeInnerLen := len(inner)
	givenBefore := len(t.given)
	editBefore := len(t.edit)
	// Yield is resolved on the editing element itself, including an
	// unmarked wrapper inside a shared scope.
	var inheritedScopeKeys []string
	for p := n; p != nil; p = p.Parent {
		keys := t.scopeInner[p]
		if p == n {
			keys = slices.Concat(keys, t.scopeKeys[n])
		} else {
			inheritedScopeKeys = append(inheritedScopeKeys, keys...)
		}
		for _, k := range keys {
			t.given = append(t.given, k)
			if slices.Contains(t.f.EditKeys, k) {
				t.edit = append(t.edit, keyAt{key: k, action: string(hotty.Program), given: true})
			}
		}
	}
	scopeEdits := len(t.edit) - editBefore
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
		binds, bound = s.writeEdits(h, binds, bound, t, false, edits > 0)
	}
	if !s.legacy && edits > 0 && text == "" {
		editInner, _ := s.writeEdits(h, nil, nil, t, true, true)
		inner = append(inner, editInner...)
	}
	if s.legacy && rt.Types(n) {
		for _, k := range t.printable[:len(t.printable)-pushed] {
			if !contains(own, k) && !contains(bound, k) && !slices.Contains(binds, k+"=default") {
				binds = append(binds, k+"=default")
			}
		}
	}
	if !s.legacy && text == "" && n.FirstChild != nil {
		focusable := focusableKeys(n)
		for _, binding := range binds {
			i := strings.LastIndexByte(binding, '=')
			key, action := binding[:i], binding[i+1:]
			if action != "default" && !(action == "program" && (focusable && contains(own, key) || contains(scopeKeys, key))) {
				inner = append(inner, binding)
			}
		}
		if !focusable {
			binds = nil
		}
	}
	writtenKeys := binds
	if !s.legacy {
		writtenKeys = uniqueKeys(slices.Clone(inner[scopeInnerLen:]))
	}
	// Carry a shared scope's winning key at the yielding edit root as
	// well. Its other branches still inherit the shared map. Resolve the
	// actual outer map so this cannot mask a nearer authored prefix.
	if len(inheritedScopeKeys) > 0 && (text == "text" || text == "" && n.FirstChild != nil) {
		var inherited *hotty.Keymap
		for _, b := range slices.Concat(lb, ledits, unbound) {
			if !b.Yield || !slices.Contains(inheritedScopeKeys, b.Key) {
				continue
			}
			if inherited == nil {
				reader := rt.KeyReader(n)
				if text == "" {
					reader = hotty.ReaderMultiline
				}
				inherited = rt.ResolveKeys(t.f.Platform, n, nil, reader, false)
			}
			if inherited.Program(b.Key) && !inherited.Prefix(b.Key) {
				binding := b.Key + "=program"
				if text == "text" {
					binds = append(binds, binding)
				} else {
					inner = append(inner, binding)
				}
			}
		}
	}
	inner = uniqueKeys(inner)
	binds = uniqueKeys(binds)
	t.given = t.given[:givenBefore]
	t.edit = slices.Delete(t.edit, editBefore, editBefore+scopeEdits)
	given := 0
	for _, b := range writtenKeys {
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
	if len(inner) > 0 {
		slices.Sort(inner)
		n.Attr = append(n.Attr, html.Attribute{Key: keysInnerAttr, Val: strings.Join(inner, " ")})
	}
	// What the element writes of the edit shortcuts' keys, for the fields
	// inside to resolve them against.
	written := 0
	if t.f.EditKeys != nil {
		for _, b := range writtenKeys {
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

// uniqueKeys keeps the first binding for each key, with a prefix winning
// over a program binding of the same key.
func uniqueKeys(binds []string) []string {
	if len(binds) < 2 {
		return binds
	}
	index := make(map[string]int, len(binds))
	out := binds[:0]
	for _, b := range binds {
		k := b[:strings.LastIndexByte(b, '=')]
		if i, seen := index[k]; seen {
			if strings.HasSuffix(b, "=prefix") && strings.HasSuffix(out[i], "=program") {
				out[i] = b
			}
			continue
		}
		index[k] = len(out)
		out = append(out, b)
	}
	return out
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

// heldAt says whether the edit shortcuts in effect on an element hold
// b's key, for layout: the nearest edit-shortcut binding of it (the
// element's own, among flat, or one around it) holds it if it is a real
// one, not a give-back; a key none of them binds, as where no group is in
// effect, is held as the host's edit shortcuts hold it (b.Held).
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

// editedAround says whether an edit-shortcut binding of key is in effect
// around the element being written (t.edit), real or a give-back: the
// edit shortcuts in effect then decide key's edit action, as heldAt finds
// them.
func (t *tree) editedAround(key string) bool {
	return slices.ContainsFunc(t.edit, func(x keyAt) bool { return x.edit && x.key == key })
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
func (s *Surface) writeEdits(h *rt.Handlers, binds, bound []string, t *tree, inner, local bool) ([]string, []string) {
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
		if !s.legacy {
			program := e.action == string(hotty.Program)
			if inner && program {
				continue
			}
			if !inner && !program && !local {
				// Ordinarily the inherited group already supplies this
				// action. An Override must also win over a nearer program key.
				if e.yield || !slices.ContainsFunc(t.edit[at+1:], func(x keyAt) bool {
					return x.key == k && x.given
				}) {
					continue
				}
			}
		}
		for _, x := range t.edit[at+1:] {
			if x.key == k && !x.edit && x.action != e.action && !hotty.Action(x.action).Scrolls() && (!x.given || e.yield) {
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
		m := hotty.Resolve(f.Platform, hotty.ReaderMultiline, "")
		f.Editing.Defaults = make(map[string]string, len(f.EditKeys))
		for _, k := range f.EditKeys {
			f.Editing.Defaults[k] = string(m.Lookup(k))
		}
	}
	return f.Editing.Defaults
}

// baseOf is CUAKeys', CocoaKeys' or an edit-shortcut group's binding of
// key in effect on the element whose base (layout's) is base: the
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
// (lb, as layout laid them out under the edit shortcuts in effect, one
// per key already): it records in h.Written those that hold on the
// element, and adds to binds those the wire carries, each key=action. Of
// lb it drops:
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
		if s.legacy && a == hotty.Prefix && rt.Printable(b.Key) {
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
	rec    *record
	local  int
	anchor int // position in Frame.Scopes of a focusable shared scope root
	kind   string
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
	for _, sc := range f.Scopes {
		b = appendScopeSignature(b, sc)
	}
	return b
}

// appendScopeSignature includes disabled declarations because they keep
// scope placement and outside-control exceptions stable across updates.
func appendScopeSignature(b []byte, scope *rt.KeyScope) []byte {
	b = append(b, '|')
	for _, binding := range scope.Bindings {
		if binding.Physical == "" {
			continue
		}
		b = append(b, ' ')
		b = append(b, binding.Physical...)
		if binding.Disabled {
			b = append(b, '-')
		}
	}
	return b
}
