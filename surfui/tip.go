package surfui

// A tooltip is an element's (Tip, a template's title= or <title>): a
// small view of its own, which the tree never draws, as a context menu
// isn't. Its host shows it after the pointer rests on the element
// (surfrun.Surface.Tip), in a surface of its own above the element's:
// TipCard is that surface's root.

// TipID is the id of a tooltip's root, in its own surface (TipCard).
const TipID = "t"

// TipCols is the most cells a tooltip is wide, its card's edges in it:
// its text wraps there.
const TipCols = 48

// Tip gives n a tooltip, content: any nodes, a row with a dot, bold text
// and a muted line under it, which the host shows over the surface
// while the pointer rests on n, and which n's tree never draws. It is a
// view, never a control: nothing in it is reached, since it goes when
// the pointer moves. nil takes it away. Given again, it replaces the
// last.
func (n *Node) Tip(content *Node) *Node {
	n.tip = content
	return n
}

// Title gives n a tooltip of text s, wrapped (Tip); "" takes it away.
// It writes no HTML title: the host's tooltip would show it a second
// time.
func (n *Node) Title(s string) *Node {
	if s == "" {
		return n.Tip(nil)
	}
	return n.Tip(Text(s).Wrap())
}

// TipOf is n's tooltip's content, nil for none.
func TipOf(n *Node) *Node { return n.tip }

// HasTip reports whether the tree under n has a tooltip anywhere: a
// surface whose view has one asks its host for hover (hotty SPEC §9.4).
func HasTip(n *Node) bool {
	if n == nil {
		return false
	}
	found := false
	walkEls(n, func(k *Node) bool {
		found = k.tip != nil
		return !found
	})
	return found
}

// walkEls calls f on each element under n, n first, in tree order, until
// f returns false; with no recursion, for TinyGo's stacks.
func walkEls(n *Node, f func(*Node) bool) {
	stack := []*Node{n}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if k.tag == "" {
			continue
		}
		if !f(k) {
			return
		}
		for i := len(k.kids) - 1; i >= 0; i-- {
			stack = append(stack, k.kids[i])
		}
	}
}

// TipCard is a tooltip as its host shows it, in a surface of its own:
// the root of that surface, in the theme and density nearest the element
// (from, or a node of its tree: the first of each from it up to its
// root), holding a card in a bar's tip's look (the float ground, the
// input's hairline), its text small, and in it a copy of content. A text directly
// in content (a template's <title>'s text) wraps. Resolve it under
// TipID.
func TipCard(from *Node, content *Node) *Node {
	c := Clone(content)
	for _, k := range c.kids {
		if k.tag != "" && hasClassOf(k, "u-text") && !hasClassOf(k, "u-wrap") {
			k.Wrap()
		}
	}
	if hasClassOf(c, "u-text") && !hasClassOf(c, "u-wrap") {
		c.Wrap()
	}
	s := El("div", El("div", c).Class("u-tip", "u-small")).Class("u-surface")
	density, theme := false, false
	for n := from; n != nil; n = n.parent {
		for _, cl := range n.class {
			switch {
			case !density && (cl == "u-dense" || cl == "u-roomy" || cl == "u-compact"):
				density = true
				s.Class(cl)
			case !theme && len(cl) > 8 && cl[:8] == "u-theme-":
				theme = true
				s.Class(cl)
			}
		}
	}
	return s
}

// hasClassOf reports whether n has class c.
func hasClassOf(n *Node, c string) bool {
	for _, x := range n.class {
		if x == c {
			return true
		}
	}
	return false
}
