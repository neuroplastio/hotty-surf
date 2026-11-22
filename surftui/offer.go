package surftui

import (
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// A surface in the same process offers the Screen its tree (rt.Host), so
// that the Screen need not parse what it was just rendered from. The
// commands still come on the wire, whole: an offer only stands in for the
// parse of the one document whose bytes it names. A Screen shows one
// surface, so it holds one offer at most.

func init() {
	rt.HostOf = hostOf
}

// hostOf is the Screen w is, or that the wrappers it is (rt.Wrapper)
// write to, nil for none: a tee or any other writer isn't one, and gets
// no offer.
func hostOf(w io.Writer) rt.Host {
	for w != nil {
		switch x := w.(type) {
		case *Screen:
			if x == nil {
				return nil
			}
			return screenHost{x}
		case rt.Wrapper:
			w = x.Wrapped(rt.Seal{})
		default:
			return nil
		}
	}
	return nil
}

// screenHost is a Screen as a surface's rt.Host.
type screenHost struct{ c *Screen }

// offered is a document's tree as its surface offered it: the surface's
// name, the document's bytes, and a body of the Screen's own whose
// children are the offered nodes' copies.
type offered struct {
	name, doc string
	body      *html.Node
}

// OfferDocument takes a copy of nodes, under a body, as the tree of the
// next a=doc for name whose payload is doc (load). The nodes are lent for
// the call: the copy shares nothing with them that the surface may change
// or that would hold its render, as a parse shares nothing.
func (h screenHost) OfferDocument(name, doc string, nodes []*html.Node) {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	for _, n := range copyNodes(nodes) {
		body.AppendChild(n)
	}
	h.c.offer = &offered{name: name, doc: doc, body: body}
}

// offeredDelta is a delta's fragments as its surface offered them: the
// surface's name, and each fragment's markup and its nodes' copies, of
// which the ops before next took theirs (fragment).
type offeredDelta struct {
	name  string
	frags []offeredFragment
	next  int
}

type offeredFragment struct {
	markup string
	nodes  []*html.Node
}

// OfferDelta takes a copy of each fragment's nodes, a copy a fragment,
// for the ops of the next a=delta for name whose markup is the
// fragment's (fragment). The nodes are lent for the call, as
// OfferDocument's are. A copy a fragment, not one for the delta: a node
// a copy holds keeps its array and its strings, and a row a log keeps
// would keep every other fragment of its delta (a status written again)
// as long as it stays.
func (h screenHost) OfferDelta(name string, frags []rt.Fragment) {
	o := &offeredDelta{name: name, frags: make([]offeredFragment, len(frags))}
	for i, f := range frags {
		c := copyNodes(f.Nodes)
		o.frags[i] = offeredFragment{markup: f.Markup, nodes: c[:len(c):len(c)]}
	}
	h.c.frags = o
}

// copyNodes is a deep copy of ns: its nodes in one array, its attributes
// in another, each node's list capped at its length (a set that adds one
// moves the list rather than write over the next node's), and its strings
// in one buffer of their own. A string of the surface's is a slice of its
// render, and a copy that shared it would keep the whole render as long
// as the Screen kept the node; an atom's name is static, and is shared.
func copyNodes(ns []*html.Node) []*html.Node {
	count, attrs, size := 0, 0, 0
	var measure func(*html.Node)
	measure = func(n *html.Node) {
		count++
		attrs += len(n.Attr)
		if !static(n.Data, n.DataAtom) {
			size += len(n.Data)
		}
		for _, a := range n.Attr {
			if !static(a.Key, atom.Lookup([]byte(a.Key))) {
				size += len(a.Key)
			}
			size += len(a.Val)
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			measure(k)
		}
	}
	for _, n := range ns {
		measure(n)
	}
	nodes := make([]html.Node, count)
	as := make([]html.Attribute, 0, attrs)
	var b strings.Builder
	b.Grow(size)
	// own is s in the buffer: what was written there stays as it is, as
	// the buffer never grows past the room made for it.
	own := func(s string) string {
		if s == "" {
			return ""
		}
		i := b.Len()
		b.WriteString(s)
		return b.String()[i:]
	}
	i := 0
	var cp func(*html.Node) *html.Node
	cp = func(n *html.Node) *html.Node {
		d := &nodes[i]
		i++
		d.Type, d.DataAtom, d.Namespace = n.Type, n.DataAtom, namespace(n.Namespace)
		if static(n.Data, n.DataAtom) {
			d.Data = n.DataAtom.String()
		} else {
			d.Data = own(n.Data)
		}
		if len(n.Attr) > 0 {
			j := len(as)
			for _, a := range n.Attr {
				c := html.Attribute{Namespace: namespace(a.Namespace), Val: own(a.Val)}
				if k := atom.Lookup([]byte(a.Key)); static(a.Key, k) {
					c.Key = k.String()
				} else {
					c.Key = own(a.Key)
				}
				as = append(as, c)
			}
			d.Attr = as[j:len(as):len(as)]
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			d.AppendChild(cp(k))
		}
		return d
	}
	out := make([]*html.Node, len(ns))
	for j, n := range ns {
		out[j] = cp(n)
	}
	return out
}

// static says whether s is a's name, which is a static string.
func static(s string, a atom.Atom) bool {
	return a != 0 && a.String() == s
}

// namespace is ns as a static string: the ones a tree has are few.
func namespace(ns string) string {
	switch ns {
	case "":
		return ""
	case "svg":
		return "svg"
	case "math":
		return "math"
	case "xlink":
		return "xlink"
	case "xml":
		return "xml"
	case "xmlns":
		return "xmlns"
	}
	return strings.Clone(ns)
}
