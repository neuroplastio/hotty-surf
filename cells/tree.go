package cells

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"
)

// The document: the surface's <style> and its #root, kept as an HTML tree
// under a body of the Screen's own, changed by the deltas as a host changes
// it (SPEC §5.1, §6.1).

// load takes a document as the surface's whole tree: the tree its
// surface offered for it (offer.go), or else the one it parses as.
func (c *Screen) load(name, doc string) error {
	if o := c.offer; o != nil {
		c.offer = nil
		if o.name == name && o.doc == doc {
			return c.loadBody(name, o.body)
		}
	}
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(doc), body)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		body.AppendChild(n)
	}
	return c.loadBody(name, body)
}

// loadBody takes body, a tree of the Screen's own, as the surface's whole
// tree.
func (c *Screen) loadBody(name string, body *html.Node) error {
	c.cutDrag()
	if c.body != nil && c.name == name && c.kept != nil && sameShape(c.body, body) {
		// The same elements with other text or attributes: the
		// kept tree takes the new one's, and keeps what a pass knows of
		// what didn't change. The rest is as for a new tree, where
		// what the pointer and a reveal held is detached.
		if c.refill(c.body, body) {
			// An element's style reads its parent's, children's,
			// descendants' and siblings' attributes too (kit, filtered,
			// barLabels): any change, and none of them holds.
			c.kept.forgetStyles()
		}
		c.fields = map[*html.Node]*field{}
		c.focused, c.remembered = nil, nil
		c.scroll = 0
		c.boxScroll = nil
		if c.down != nil {
			// Still a button down, as Move sees it, but on nothing a
			// release clicks.
			c.down = &pressed{}
		}
		c.clicks = clickRun{}
		// The chain and its tooltip are kept, as the elements are: the
		// next pass hits the pointer again (rehit).
		c.revealAfter, c.revealWhere = nil, ""
		c.ids, c.lay = nil, nil
		return nil
	}
	c.name, c.body = name, body
	c.fields = map[*html.Node]*field{}
	c.focused, c.remembered = nil, nil
	c.hovered, c.tip = nil, nil // until the next pass hits the pointer on the new tree
	c.scroll = 0
	c.boxScroll = nil
	c.changed()
	return nil
}

// attrsChanged forgets what an element's attributes, was before, reach:
// where only colour classes changed (colourOnly), its colours; else every
// style too, which reads its neighbours' attributes (kit, filtered).
func (c *Screen) attrsChanged(n *html.Node, was []html.Attribute) {
	k := c.kept
	if slices.Equal(was, n.Attr) {
		return
	}
	if k == nil {
		c.relayout = true
		return
	}
	if colourOnly(was, n.Attr) {
		// The layout reads a colour only where an element's own ends the
		// muted text around it (lookOf): where that holds, so do the
		// looks, and only the roles change.
		if m := k.nodes[n]; m == nil || !m.hasCol || m.inMuted && unmutes(m.fg) != unmutes(c.ownFg(n)) {
			// Its look may change: its layouts, and what holds them.
			c.relayout = true
			k.unlayAll(n)
		}
		k.forgetColours(n)
		return
	}
	c.relayout = true
	k.forgetStylesAround(n)
	k.forgetLooks(n)
}

// changed says the tree changed: its ids and its layout are made again
// when next asked for.
func (c *Screen) changed() {
	c.ids = nil
	c.lay = nil
	c.kept = nil
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
	c.reshaped, c.reIDed, c.relayout = false, false, false
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
	// What the ops reached is forgotten (op); where they changed the
	// tree's shape, all of it is. Where they changed only colours, which
	// left every look as it was (attrsChanged), the layout holds and its
	// cells are painted again: what was drawn is no longer this
	// layout's (Dirty).
	if c.reshaped || c.relayout || c.lay == nil {
		c.lay = nil
	} else {
		c.shownLay = nil
	}
	if c.reshaped || c.reIDed {
		c.ids = nil
	}
	if c.reshaped {
		c.kept = nil
		c.pruneHover()
	}
	for n := range c.boxScroll {
		if !c.attached(n) {
			delete(c.boxScroll, n)
		}
	}
	for n, f := range c.fields {
		if !c.attached(n) {
			delete(c.fields, n)
			continue
		}
		// A default the markup changed sets the control, unless it has
		// the keyboard (SPEC §6.2); its rev stays.
		if d := defaultOf(n); d != f.def {
			f.def = d
			c.lay = nil
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
		c.lay = nil
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
	case "attr", "unattr":
		was := slices.Clone(t.Attr)
		if o.Op == "attr" {
			set(t, o.Key, v)
		} else {
			unset(t, o.Key)
		}
		if o.Key == "id" {
			c.reIDed, c.ids = true, nil
		}
		c.attrsChanged(t, was)
	case "text":
		if k := t.FirstChild; k != nil && k.Type == html.TextNode && k.NextSibling == nil {
			// Its one text's: only that text's pieces are forgotten.
			if k.Data != v {
				k.Data = v
				c.relayout = true
				if c.kept != nil {
					c.kept.forgetText(k)
				}
			}
			return nil
		}
		clear(t)
		t.AppendChild(&html.Node{Type: html.TextNode, Data: v})
		c.reshaped, c.ids = true, nil
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
		c.reshaped, c.ids = true, nil
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
		c.reshaped, c.ids = true, nil
	case "append", "prepend":
		first := t.FirstChild
		for _, n := range fragment(v, t) {
			if o.Op == "append" || first == nil {
				t.AppendChild(n)
			} else {
				t.InsertBefore(n, first)
			}
		}
		c.reshaped, c.ids = true, nil
	case "remove":
		if t.Parent == nil {
			return fmt.Errorf("no parent (%s)", hotty.EINVAL)
		}
		t.Parent.RemoveChild(t)
		c.reshaped, c.ids = true, nil
	case "reveal":
		if v != hotty.RevealNearest && v != hotty.RevealStart && v != hotty.RevealEnd {
			return fmt.Errorf("unknown reveal alignment %q (%s)", v, hotty.EINVAL)
		}
		c.revealAfter, c.revealWhere = t, v
		c.relayout = true // a layout reveals it
	default:
		return fmt.Errorf("unknown op %q (%s)", o.Op, hotty.EINVAL)
	}
	return nil
}

// adopt gives each element under t that has an old one's id and tag, under
// a parent of the same id, the old one's field and focus, as inner's morph
// keeps the element (SPEC §6.1, §6.2).
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
		// Under a parent of the same id, as the surface keeps a field's
		// rev (fresh in package surf): one that moved elsewhere enters
		// again. clear detached t's own children, whose parent was t.
		from := o.Parent
		if from == nil {
			from = t
		}
		if idOf(from) != idOf(n.Parent) {
			return
		}
		if offset, ok := c.boxScroll[o]; ok {
			c.boxScroll[n] = offset
			delete(c.boxScroll, o)
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
	// Whatever the pointer is over: a hover: class showing or hiding an
	// element under it doesn't take its focus.
	return closest(n, func(m *html.Node) bool { return styleOf(m, false).display == dNone }) != nil
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
	return hasField(attr(n, k), tok)
}

// hasField says whether tok is one of strings.Fields(s), for a tok with
// no space in it, found where it stands rather than by splitting s.
func hasField(s, tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i+len(tok) <= len(s); {
		j := strings.Index(s[i:], tok)
		if j < 0 {
			return false
		}
		j += i
		end := j + len(tok)
		before, _ := utf8.DecodeLastRuneInString(s[:j])
		after, _ := utf8.DecodeRuneInString(s[end:])
		if (j == 0 || unicode.IsSpace(before)) && (end == len(s) || unicode.IsSpace(after)) {
			return true
		}
		i = j + 1
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

// idOf is n's id, "" for none.
func idOf(n *html.Node) string {
	id, _ := get(n, "id")
	return id
}

// sameShape says whether two trees have the same nodes in the same places:
// their types and tags, whatever their text and attributes.
func sameShape(a, b *html.Node) bool {
	if a.Type != b.Type || a.DataAtom != b.DataAtom || a.Type == html.ElementNode && a.Data != b.Data {
		return false
	}
	ka, kb := a.FirstChild, b.FirstChild
	for ; ka != nil && kb != nil; ka, kb = ka.NextSibling, kb.NextSibling {
		if !sameShape(ka, kb) {
			return false
		}
	}
	return ka == nil && kb == nil
}

// refill gives the kept tree a the text and attributes of b, its shape,
// and forgets what a pass knew of each text that changed and of the
// colours under each element whose attributes changed. It says whether
// any element's attributes changed.
func (c *Screen) refill(a, b *html.Node) (attrs bool) {
	k := c.kept
	switch a.Type {
	case html.TextNode:
		if a.Data != b.Data {
			a.Data = b.Data
			k.forgetText(a)
		}
	case html.ElementNode:
		if !slices.Equal(a.Attr, b.Attr) {
			a.Attr = b.Attr
			attrs = true
			// Its colours, and its descendants' (a scheme's class), and
			// the pieces of its texts.
			k.forgetLooks(a)
		}
	}
	for ka, kb := a.FirstChild, b.FirstChild; ka != nil; ka, kb = ka.NextSibling, kb.NextSibling {
		if c.refill(ka, kb) {
			attrs = true
		}
	}
	return attrs
}
