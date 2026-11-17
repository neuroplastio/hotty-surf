package cells

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"
)

// The document: the surface's <style> and its #root, kept as an HTML tree
// under a body of the Screen's own, changed by the deltas as a host changes
// it (SPEC §5.1, §6.1).

// load takes a document as the surface's whole tree.
func (c *Screen) load(name, doc string) error {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(doc), body)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		body.AppendChild(n)
	}
	c.name, c.body = name, body
	c.fields = map[*html.Node]*field{}
	c.focused, c.remembered = nil, nil
	c.scroll = 0
	c.changed()
	return nil
}

// changed says the tree changed: its ids and its layout are made again
// when next asked for.
func (c *Screen) changed() {
	c.ids = nil
	c.lay = nil
}

// byID is the element with the id, or nil.
func (c *Screen) byID(id string) *html.Node {
	if c.body == nil {
		return nil
	}
	if c.ids == nil {
		c.ids = map[string]*html.Node{}
		walk(c.body, func(n *html.Node) {
			if n.Type == html.ElementNode {
				if v, ok := get(n, "id"); ok {
					if _, dup := c.ids[v]; !dup {
						c.ids[v] = n
					}
				}
			}
		})
	}
	return c.ids[id]
}

// attached says whether n is still in the document.
func (c *Screen) attached(n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == c.body {
			return true
		}
	}
	return false
}

// delta applies a delta's ops in order. The first it can't apply is the
// error, and the ops after it are not applied (the program sends the
// document again).
func (c *Screen) delta(ops []hotty.Op) error {
	defer c.settle()
	for i, o := range ops {
		if err := c.op(o); err != nil {
			return fmt.Errorf("cells: op %d (%s %s): %w", i, o.Op, o.Target, err)
		}
	}
	return nil
}

// settle forgets what left the document: the fields' state, and the
// focus of an element that is gone.
func (c *Screen) settle() {
	c.changed()
	for n, f := range c.fields {
		if !c.attached(n) {
			delete(c.fields, n)
			continue
		}
		// A default the markup changed sets the control, unless it has
		// the keyboard (SPEC §6.2); its rev stays.
		if d := defaultOf(n); d != f.def {
			f.def = d
			if n == c.focused {
				continue
			}
			if isCheck(n) {
				c.check(n, d == "true")
			} else {
				f.ed.Value, f.committed = d, d
				f.ed.Select(len(graphemesOf(d)), len(graphemesOf(d)))
			}
		}
	}
	if c.focused != nil && (!c.attached(c.focused) || !focusable(c.focused) || hidden(c.focused)) {
		// An element that stops taking focus loses it, as a browser's
		// does; the surface keeps the keyboard with nothing focused.
		c.focused = nil
	}
	if c.remembered != nil && !c.attached(c.remembered) {
		c.remembered = nil
	}
}

var errNoTarget = fmt.Errorf("no such element (%s)", hotty.ENOTARGET)

func (c *Screen) op(o hotty.Op) error {
	v := ""
	if o.Value != nil {
		v = *o.Value
	}
	switch o.Op {
	case "var", "scroll":
		// No custom property is read, and only the root scrolls.
		return nil
	case "morph":
		return fmt.Errorf("morph is not supported yet (%s)", hotty.EINVAL)
	}
	t := c.byID(o.Target)
	if t == nil {
		return errNoTarget
	}
	switch o.Op {
	case "attr":
		set(t, o.Key, v)
		if o.Key == "id" {
			c.ids = nil
		}
	case "unattr":
		unset(t, o.Key)
		if o.Key == "id" {
			c.ids = nil
		}
	case "text":
		clear(t)
		t.AppendChild(&html.Node{Type: html.TextNode, Data: v})
		c.ids = nil
	case "inner":
		old := map[string]*html.Node{}
		walk(t, func(n *html.Node) {
			if id, ok := get(n, "id"); ok && n != t && n.Type == html.ElementNode {
				old[id] = n
			}
		})
		clear(t)
		for _, n := range fragment(v, t) {
			t.AppendChild(n)
		}
		c.adopt(t, old)
		c.ids = nil
	case "replace", "before", "after":
		p := t.Parent
		if p == nil {
			return fmt.Errorf("no parent (%s)", hotty.EINVAL)
		}
		ref := t
		if o.Op == "after" {
			ref = t.NextSibling
		}
		for _, n := range fragment(v, p) {
			p.InsertBefore(n, ref)
		}
		if o.Op == "replace" {
			p.RemoveChild(t)
		}
		c.ids = nil
	case "append", "prepend":
		first := t.FirstChild
		for _, n := range fragment(v, t) {
			if o.Op == "append" || first == nil {
				t.AppendChild(n)
			} else {
				t.InsertBefore(n, first)
			}
		}
		c.ids = nil
	case "remove":
		if t.Parent == nil {
			return fmt.Errorf("no parent (%s)", hotty.EINVAL)
		}
		t.Parent.RemoveChild(t)
		c.ids = nil
	case "reveal":
		c.revealAfter = t
	default:
		return fmt.Errorf("unknown op %q (%s)", o.Op, hotty.EINVAL)
	}
	return nil
}

// adopt gives each element under t that has an old one's id and tag the
// old one's field and focus, as inner's morph keeps the element (SPEC
// §6.1, §6.2).
func (c *Screen) adopt(t *html.Node, old map[string]*html.Node) {
	walk(t, func(n *html.Node) {
		if n == t || n.Type != html.ElementNode {
			return
		}
		id, _ := get(n, "id")
		o := old[id]
		if o == nil || o.DataAtom != n.DataAtom {
			return
		}
		if f := c.fields[o]; f != nil {
			c.fields[n] = f
			delete(c.fields, o)
		}
		if c.focused == o {
			c.focused = n
		}
		if c.remembered == o {
			c.remembered = n
		}
	})
}

// hidden says whether n or an element it is in draws nothing
// (display:none, hidden, …).
func hidden(n *html.Node) bool {
	return closest(n, func(m *html.Node) bool { return styleOf(m).display == dNone }) != nil
}

// fragment parses HTML as the children of ctx, so that rows and items
// parse as they would in place.
func fragment(s string, ctx *html.Node) []*html.Node {
	if ctx == nil || ctx.Type != html.ElementNode {
		ctx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	}
	nodes, err := html.ParseFragment(strings.NewReader(s), ctx)
	if err != nil {
		return nil
	}
	return nodes
}

func clear(n *html.Node) {
	for k := n.FirstChild; k != nil; {
		next := k.NextSibling
		n.RemoveChild(k)
		k = next
	}
}

func walk(n *html.Node, f func(*html.Node)) {
	f(n)
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		walk(k, f)
	}
}

func get(n *html.Node, k string) (string, bool) {
	for _, a := range n.Attr {
		if a.Namespace == "" && a.Key == k {
			return a.Val, true
		}
	}
	return "", false
}

func attr(n *html.Node, k string) string {
	v, _ := get(n, k)
	return v
}

func has(n *html.Node, k string) bool {
	_, ok := get(n, k)
	return ok
}

func set(n *html.Node, k, v string) {
	for i, a := range n.Attr {
		if a.Namespace == "" && a.Key == k {
			n.Attr[i].Val = v
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: k, Val: v})
}

func unset(n *html.Node, k string) {
	for i, a := range n.Attr {
		if a.Namespace == "" && a.Key == k {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			return
		}
	}
}

// tokens says whether the space-separated list in n's attribute k holds
// tok (class~=, data-on~=).
func tokens(n *html.Node, k, tok string) bool {
	for _, t := range strings.Fields(attr(n, k)) {
		if t == tok {
			return true
		}
	}
	return false
}

// closest is n or its nearest ancestor for which f holds, or nil.
func closest(n *html.Node, f func(*html.Node) bool) *html.Node {
	for ; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && f(n) {
			return n
		}
	}
	return nil
}

// withID is n or its nearest ancestor that has an id: what an event names.
func withID(n *html.Node) *html.Node {
	return closest(n, func(n *html.Node) bool { return has(n, "id") })
}

// textOf is the text under n, its white space collapsed.
func textOf(n *html.Node) string {
	var b strings.Builder
	walk(n, func(k *html.Node) {
		if k.Type == html.TextNode && !inSVG(k) {
			b.WriteString(k.Data)
			b.WriteByte(' ')
		}
	})
	return strings.Join(strings.Fields(b.String()), " ")
}

func inSVG(n *html.Node) bool {
	return closest(n, func(n *html.Node) bool { return n.DataAtom == atom.Svg }) != nil
}
