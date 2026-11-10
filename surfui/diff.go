package surfui

import (
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/stackprobe"
)

// Diff is what turns the document old shows into the one next shows, as
// a delta's ops (SPEC §6.1): `text`, `attr`, `unattr` and `var` where a
// leaf changed (the host's cheap path), `remove`, `after` and `prepend`
// where a keyed list gained or lost rows, and `inner` where children
// changed in a way those can't say (a reorder, mixed text). Both trees
// are resolved under the same prefix. A subtree that is the same node in
// both, as a cached render's is, is skipped.
//
// A field's value is not in it: the program sets one with a=field
// (SPEC §6.4), which keeps what the user typed meanwhile.
func Diff(old, next *Node) []hotty.Op {
	var ops []hotty.Op
	diffEl(old, next, &ops)
	return ops
}

// Coarse is the other way to send a change: the whole of next's root,
// morphed (SPEC §6.2), and nothing when nothing changed. The host keeps
// what the user is doing; the program sends more, and the host does the
// diffing. It is what F10's numbers compare Diff with.
func Coarse(old, next *Node) []hotty.Op {
	h := HTML(next)
	if HTML(old) == h {
		return nil
	}
	return []hotty.Op{hotty.OpMorph(next.id, h)}
}

// Diff recurses on the tree, diffEl and diffKids in turn, on a
// goroutine's stack: under TinyGo, a fixed 64 KB. So the two keep small
// frames, and what doesn't recurse (an element's own ops, the children's
// text and keyed rows) is in helpers that return first (go:noinline,
// which keeps their locals off the frames).

func diffEl(o, n *Node, ops *[]hotty.Op) {
	stackprobe.Probe()
	if o == n {
		return
	}
	if diffOwn(o, n, ops) {
		diffKids(o, n, ops)
	}
}

// diffOwn writes the ops of o's and n's own: a replace where they are
// not alike, their attributes, a raw node's HTML. It reports whether
// their children are left to diff.
//
//go:noinline
func diffOwn(o, n *Node, ops *[]hotty.Op) bool {
	if o.tag != n.tag || o.isRaw != n.isRaw {
		*ops = append(*ops, hotty.OpReplace(o.id, HTML(n)))
		return false
	}
	diffAttrs(o, n, ops)
	if o.isRaw {
		if o.raw != n.raw {
			*ops = append(*ops, hotty.OpInner(n.id, n.raw))
		}
		return false
	}
	return true
}

// fieldValue reports whether attribute k of n is a control's value,
// which goes as a=field and not as an attribute.
func fieldValue(n *Node, k string) bool {
	return k == "value" && (n.tag == "input" || n.tag == "textarea" || n.tag == "select")
}

func diffAttrs(o, n *Node, ops *[]hotty.Op) {
	oa, na := attrsOf(o), attrsOf(n)
	// Vars go as `var` ops while every one the old style had is still
	// there; otherwise the style goes whole.
	// The app's own style is in that style too: if it changed, whole.
	os, _ := AttrOf(o, "style")
	ns, _ := AttrOf(n, "style")
	varsOnly := (len(o.vars) > 0 || len(n.vars) > 0) && os == ns
	if varsOnly {
		for _, v := range o.vars {
			if !hasAttr(n.vars, v.k) {
				varsOnly = false
			}
		}
	}
	for _, a := range na {
		if fieldValue(n, a.k) {
			continue
		}
		if a.k == "style" && varsOnly {
			continue
		}
		if v, ok := find(oa, a.k); !ok || v != a.v {
			*ops = append(*ops, hotty.OpAttr(n.id, a.k, a.v))
		}
	}
	for _, a := range oa {
		if fieldValue(n, a.k) {
			continue
		}
		if _, ok := find(na, a.k); !ok {
			*ops = append(*ops, hotty.OpUnattr(n.id, a.k))
		}
	}
	if varsOnly {
		for _, v := range n.vars {
			if w, ok := find(o.vars, v.k); !ok || w != v.v {
				*ops = append(*ops, hotty.OpVar(n.id, v.k, v.v))
			}
		}
	}
}

func hasAttr(as []attr, k string) bool { _, ok := find(as, k); return ok }

func find(as []attr, k string) (string, bool) {
	for _, a := range as {
		if a.k == k {
			return a.v, true
		}
	}
	return "", false
}

func textOnly(n *Node) bool {
	for _, k := range n.kids {
		if k.tag != "" {
			return false
		}
	}
	return true
}

//go:noinline
func diffKids(o, n *Node, ops *[]hotty.Op) {
	stackprobe.Probe()
	olds := diffRows(o, n, ops)
	for i, k := range n.kids {
		if olds != nil && olds[i] != nil && k.tag != "" {
			diffEl(olds[i], k, ops)
		}
	}
}

// diffRows writes the ops for o's and n's children as a list: their text,
// or the rows that went and came. It returns, for each of n's children,
// the old one it is diffed with: nil for none, and a nil list where none
// is.
//
//go:noinline
func diffRows(o, n *Node, ops *[]hotty.Op) []*Node {
	if textOnly(o) && textOnly(n) {
		if t := TextOf(n); TextOf(o) != t {
			*ops = append(*ops, hotty.OpText(n.id, t))
		}
		return nil
	}
	if sameShape(o, n) {
		return o.kids
	}
	if hasText(o) || hasText(n) {
		*ops = append(*ops, hotty.OpInner(n.id, KidsHTML(n)))
		return nil
	}
	// Elements only, by id: what went is removed, what came is put after
	// the sibling before it, and what stayed is diffed, provided what
	// stayed kept its order. A reorder morphs the children, which the
	// host moves by id (SPEC §6.2).
	was := map[string]*Node{}
	for _, k := range o.kids {
		was[k.id] = k
	}
	now := map[string]bool{}
	for _, k := range n.kids {
		now[k.id] = true
	}
	var stayedOld, stayedNew []string
	for _, k := range o.kids {
		if now[k.id] {
			stayedOld = append(stayedOld, k.id)
		}
	}
	for _, k := range n.kids {
		if was[k.id] != nil {
			stayedNew = append(stayedNew, k.id)
		}
	}
	if !equal(stayedOld, stayedNew) {
		*ops = append(*ops, hotty.OpInner(n.id, KidsHTML(n)))
		return nil
	}
	for _, k := range o.kids {
		if !now[k.id] {
			*ops = append(*ops, hotty.OpRemove(k.id))
		}
	}
	prev := ""
	for _, k := range n.kids {
		if was[k.id] == nil {
			if prev == "" {
				*ops = append(*ops, hotty.OpPrepend(n.id, HTML(k)))
			} else {
				*ops = append(*ops, hotty.OpAfter(prev, HTML(k)))
			}
		}
		prev = k.id
	}
	olds := make([]*Node, len(n.kids))
	for i, k := range n.kids {
		olds[i] = was[k.id]
	}
	return olds
}

func hasText(n *Node) bool {
	for _, k := range n.kids {
		if k.tag == "" {
			return true
		}
	}
	return false
}

// sameShape reports whether o's and n's children are the same sequence:
// the same text, and elements of the same ids.
func sameShape(o, n *Node) bool {
	if len(o.kids) != len(n.kids) {
		return false
	}
	for i, k := range n.kids {
		ok := o.kids[i]
		if (k.tag == "") != (ok.tag == "") {
			return false
		}
		if k.tag == "" {
			if k.text != ok.text {
				return false
			}
			continue
		}
		if k.id != ok.id {
			return false
		}
	}
	return true
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// IsSelected reports whether n is selected, as Selected marks an Item.
func IsSelected(n *Node) bool {
	v, _ := AttrOf(n, "aria-selected")
	return v == "true"
}

// Reveals are the ops that bring into view (SPEC §6.1, reveal) each
// Item of next that is selected and was not in old, the index of the
// last tree: an Item the selection moved to, by as little as brings it
// into view, as a browser's list box keeps its selection in view. A
// selected button (a segment, a chip, a tab) is a value, not a place:
// it is not revealed, or a page with chips at its foot would open
// scrolled down to them. They
// are in document order, and go after the ops that make next. A new
// document's are Reveals(nil, root). The walk keeps a stack of its own:
// no recursion, for TinyGo's stacks.
func Reveals(old Index, next *Node) []hotty.Op {
	var ops []hotty.Op
	todo := []*Node{next}
	for len(todo) > 0 {
		n := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if n.tag == "" {
			continue
		}
		if IsSelected(n) && has(n, "u-item") {
			if o := old[n.id]; o == nil || !IsSelected(o) {
				ops = append(ops, hotty.OpReveal(n.id, hotty.RevealNearest))
			}
		}
		for i := len(n.kids) - 1; i >= 0; i-- {
			todo = append(todo, n.kids[i])
		}
	}
	return ops
}
