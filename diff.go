package surf

import (
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// differ says how the tree changed between two renders, as the ops of one
// hotty delta (SPEC §6.1) and the fields the surface sets (§6.4). Every
// element has an id (number), so each op names its target: an attribute
// set or removed, a text replaced, an element removed, inserted before a
// sibling or appended. Where a parent's children changed in a way those
// can't say (a move among them, text among elements), its children go
// whole as inner. An element moved to another parent is taken out first
// (moves) and inserted as new.
type differ struct {
	s      *Surface
	ops    []hotty.Op
	fields []string
	b      strings.Builder
	moved  map[string]bool // taken out first (moves): their old parent doesn't remove them
	// frags are the ops' markup and the nodes it renders, for an
	// in-process host (rt.Host.OfferDelta); none where offer is false.
	frags []rt.Fragment
	offer bool
}

// moves takes out, first, each element the render moved to another
// parent, so that the delta never puts an id in the document while it is
// still elsewhere (the walk visits the new parent first when it comes
// earlier, or sends its children whole). Its new parent then inserts it
// as new. One op an element, outermost first: an element inside one taken
// out goes with it.
func (d *differ) moves(o, n *html.Node) {
	parents := func(root *html.Node) map[string]string {
		m := map[string]string{}
		var walk func(p *html.Node, pid string)
		walk = func(p *html.Node, pid string) {
			for c := p.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				id := attr(c, "id")
				if id != "" {
					m[id] = pid
				}
				walk(c, id)
			}
		}
		walk(root, attr(root, "id"))
		return m
	}
	now := parents(n)
	var walk func(p *html.Node, pid string)
	walk = func(p *html.Node, pid string) {
		for c := p.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			id := attr(c, "id")
			if q, ok := now[id]; ok && id != "" && q != pid {
				if d.moved == nil {
					d.moved = map[string]bool{}
				}
				d.moved[id] = true
				d.ops = append(d.ops, hotty.OpRemove(id))
				continue // what it holds goes with it
			}
			walk(c, id)
		}
	}
	walk(o, attr(o, "id"))
}

// el compares two elements with the same id.
func (d *differ) el(o, n *html.Node) {
	id := attr(n, "id")
	if o.Data != n.Data {
		d.ops = append(d.ops, hotty.OpReplace(id, d.render(n)))
		d.s.fresh(n, false)
		d.s.remember(n)
		return
	}
	d.attrs(id, o, n)
	if isField(n) {
		d.field(id, n)
		if n.Data == "textarea" {
			return // its text is its value: the field's, not the tree's
		}
	}
	d.children(id, o, n)
}

func (d *differ) attrs(id string, o, n *html.Node) {
	// A field's value or state is a=field's, not an attribute's: a check
	// box's or radio's checked (its value is an attribute like any other,
	// what it submits), another field's value. An option's selected is its
	// select's value.
	var live string
	switch {
	case isCheck(n):
		live = "checked"
	case isField(n):
		live = "value"
	case n.Data == "option" && inSelect(n):
		live = "selected"
	}
	for _, a := range n.Attr {
		if a.Key == "id" || a.Key == live {
			continue
		}
		if v, ok := get(o, a.Key); !ok || v != a.Val {
			d.ops = append(d.ops, hotty.OpAttr(id, a.Key, a.Val))
		}
	}
	for _, a := range o.Attr {
		if a.Key == "id" || a.Key == live {
			continue
		}
		if _, ok := get(n, a.Key); !ok {
			d.ops = append(d.ops, hotty.OpUnattr(id, a.Key))
		}
	}
}

func get(n *html.Node, k string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val, true
		}
	}
	return "", false
}

// field sets a live field's value or state where the render wants another
// than the host holds: a reset clearing a filter, a select-all ticking the
// rows' boxes. Where the user's own edit made the state, they agree, and
// nothing is sent.
func (d *differ) field(id string, n *html.Node) {
	s := d.s
	if isCheck(n) {
		want := hasAttr(n, "checked")
		if s.checked[id] != want {
			s.checked[id] = want
			d.fields = append(d.fields, hotty.SetField(s.name, id, hotty.FieldChange{Rev: s.revs[id], Checked: &want}))
		}
		return
	}
	if n.Data == "input" && !hasAttr(n, "value") || n.Data == "select" && !chooses(n) {
		return // the markup asks for no value: the field is the host's
	}
	want := fieldValue(n)
	sel, selection := s.pendingSel[id]
	if s.vals[id] != want || selection {
		change := hotty.FieldChange{Rev: s.revs[id]}
		if s.vals[id] != want {
			change.Value = &want
		}
		if selection {
			change.Sel = &sel
			delete(s.pendingSel, id)
		}
		s.vals[id] = want
		d.fields = append(d.fields, hotty.SetField(s.name, id, change))
	}
}

func kids(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode || c.Type == html.TextNode {
			out = append(out, c)
		}
	}
	return out
}

func (d *differ) children(id string, o, n *html.Node) {
	oc, nc := kids(o), kids(n)
	same := len(oc) == len(nc)
	text := false
	for i := 0; same && i < len(oc); i++ {
		a, b := oc[i], nc[i]
		switch {
		case a.Type != b.Type:
			same = false
		case a.Type == html.TextNode:
			text = true
		case attr(a, "id") != attr(b, "id"):
			same = false
		}
	}
	if same {
		changed := false
		for i := range oc {
			if oc[i].Type == html.TextNode && oc[i].Data != nc[i].Data {
				changed = true
			}
		}
		if changed {
			if len(nc) == 1 {
				d.ops = append(d.ops, hotty.OpText(id, nc[0].Data))
			} else {
				d.inner(id, n)
				return
			}
		}
		for i := range oc {
			if oc[i].Type == html.ElementNode {
				d.el(oc[i], nc[i])
			}
		}
		return
	}
	if text || hasText(oc) || hasText(nc) {
		d.inner(id, n)
		return
	}
	// Elements only: removals, insertions among the elements that stayed,
	// in order; anything else (a move) is inner.
	old := make(map[string]*html.Node, len(oc))
	for _, c := range oc {
		old[attr(c, "id")] = c
	}
	inNew := make(map[string]bool, len(nc))
	for _, c := range nc {
		inNew[attr(c, "id")] = true
	}
	var keptOld, keptNew []string
	for _, c := range oc {
		if inNew[attr(c, "id")] {
			keptOld = append(keptOld, attr(c, "id"))
		}
	}
	for _, c := range nc {
		if old[attr(c, "id")] != nil {
			keptNew = append(keptNew, attr(c, "id"))
		}
	}
	if !equal(keptOld, keptNew) {
		d.inner(id, n)
		return
	}
	for _, c := range oc {
		if cid := attr(c, "id"); !inNew[cid] && !d.moved[cid] {
			d.ops = append(d.ops, hotty.OpRemove(cid))
		}
	}
	for i, c := range nc {
		cid := attr(c, "id")
		if o := old[cid]; o != nil {
			d.el(o, c)
			continue
		}
		// Before the next sibling that stays, or at the end.
		next := ""
		for _, x := range nc[i+1:] {
			if old[attr(x, "id")] != nil {
				next = attr(x, "id")
				break
			}
		}
		if next != "" {
			d.ops = append(d.ops, hotty.OpBefore(next, d.render(c)))
		} else {
			d.ops = append(d.ops, hotty.OpAppend(id, d.render(c)))
		}
		d.s.fresh(c, false)
		d.s.remember(c)
	}
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

func hasText(ns []*html.Node) bool {
	for _, n := range ns {
		if n.Type == html.TextNode {
			return true
		}
	}
	return false
}

func (d *differ) inner(id string, n *html.Node) {
	d.b.Reset()
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&d.b, c)
	}
	markup := d.b.String()
	d.ops = append(d.ops, hotty.OpInner(id, markup))
	if d.offer {
		var nodes []*html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			nodes = append(nodes, c)
		}
		d.frags = append(d.frags, rt.Fragment{Markup: markup, Nodes: nodes})
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d.s.fresh(c, true)
	}
	d.s.remember(n)
}

func (d *differ) render(n *html.Node) string {
	d.b.Reset()
	html.Render(&d.b, n)
	markup := d.b.String()
	if d.offer {
		d.frags = append(d.frags, rt.Fragment{Markup: markup, Nodes: []*html.Node{n}})
	}
	return markup
}

// inSelect says whether an option is a live select's, its own or its
// optgroup's.
func inSelect(n *html.Node) bool {
	p := n.Parent
	if p != nil && p.Data == "optgroup" {
		p = p.Parent
	}
	return p != nil && p.Data == "select" && isField(p)
}
