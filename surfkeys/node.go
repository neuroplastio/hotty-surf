package surfkeys

import (
	"neuroplast.io/hotty-surf/surfui"
)

// The classes and the attribute a tree's keys have.
const (
	// classKey is a cap, and a surfui.Key that holds a sequence, not yet
	// drawn in a Style.
	classKey = "u-key"
	// classKeys is a sequence drawn in a Style: its caps, and what
	// stands between them.
	classKeys = "u-keys"
	// classThen is what stands between two caps (Between).
	classThen = "u-then"
	// attrSeq is the sequence a drawn one holds, so that a renderer of
	// another style draws it again: one with none was drawn in a style of
	// its own (Node).
	attrSeq = "data-seq"
)

// Node is seq's caps drawn in st, as a host draws them (host) or cells
// do: a kbd.u-keys that keeps whole, a kbd.u-key cap a chord, with what
// Between says between them, and the caps as words for a screen reader.
// A renderer draws it as it is: a model shows keys in a style that is
// not the surface's so (plx's onboarding, which shows sample keys in each
// look as the person picks it). A surfui.Key is drawn in the renderer's
// style instead (Expand).
func Node(st Style, host bool, seq string) *surfui.Node {
	n := surfui.El("kbd")
	draw(n, st, host, seq)
	return n
}

// Expand draws each key in tree, in place, in st as a host draws keys
// (host) or cells do: a surfui.Key, a kbd.u-key that holds a sequence,
// becomes its caps as Node draws them, and holds the sequence
// (data-seq); one drawn so already is drawn again from it, so that a
// tree a model renders again from the same nodes draws in st too. A
// Node is left as it is.
func Expand(tree *surfui.Node, st Style, host bool) {
	if tree == nil {
		return
	}
	if seq, ok := SeqOf(tree); ok {
		draw(tree, st, host, seq)
		tree.Attr(attrSeq, seq)
		return
	}
	if hasClass(tree, classKeys) {
		return
	}
	for _, k := range surfui.Kids(tree) {
		Expand(k, st, host)
	}
}

// SeqOf is the sequence n holds where a renderer draws it in its own
// style: a surfui.Key's text, or what a key drawn by Expand holds; false for
// any other node, a Node among them.
func SeqOf(n *surfui.Node) (string, bool) {
	if hasClass(n, classKeys) {
		return surfui.AttrOf(n, attrSeq)
	}
	if hasClass(n, classKey) && surfui.Tag(n) == "kbd" {
		return surfui.TextOf(n), true
	}
	return "", false
}

// draw makes n seq's caps in st: kbd.u-keys, its kids the caps and what
// stands between them, its label the caps in words (Words).
func draw(n *surfui.Node, st Style, host bool, seq string) {
	caps := Caps(st, host, seq)
	then := Between(st)
	kids := make([]*surfui.Node, 0, 2*len(caps))
	for i, c := range caps {
		if i > 0 && then != "" {
			kids = append(kids, surfui.El("span", surfui.Str(then)).Class(classThen))
		}
		kids = append(kids, surfui.El("kbd", surfui.Str(c)).Class(classKey))
	}
	surfui.SetKids(n, kids...)
	n.Unclass(classKey)
	if !hasClass(n, classKeys) {
		n.Class(classKeys)
	}
	n.Attr("aria-label", Words(st, seq))
}

// hasClass is whether n has class c.
func hasClass(n *surfui.Node, c string) bool {
	for _, x := range surfui.Classes(n) {
		if x == c {
			return true
		}
	}
	return false
}
