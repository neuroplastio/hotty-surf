package surf

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf/internal/rt"
)

// The render: the view's HTML, parsed into a tree under a #root, every
// element given an id. An element keeps its id as long as it keeps its
// place, so the diff (diff.go) can say what changed on it.

// renderer is a surface's render buffer and parser, kept between renders.
type renderer struct {
	buf bytes.Buffer
}

func init() {
	rt.Peek.Tree = func(s any) *html.Node { return s.(*Surface).prev }
	rt.Peek.Faults = func(s any) []string {
		x := s.(*Surface)
		f := x.faults
		x.faults = nil
		return f
	}
	rt.Peek.Snapshot = func(s any, w *msgp.Writer) error { return s.(*Surface).snap(w) }
	rt.Peek.Adopt = func(s any, r *msgp.Reader) error { return s.(*Surface).adopt(r) }
	rt.Peek.Restyle = func(s any, css string) error { return s.(*Surface).restyle(css) }
	rt.Peek.Document = func(s any) (string, error) {
		x := s.(*Surface)
		if x.prev == nil {
			return "", nil
		}
		return x.documentOf(x.prev)
	}
}

func (s *Surface) reset() {
	s.ids = map[string]string{}
	s.byID = map[string]*html.Node{}
	s.vals, s.checked, s.revs = map[string]string{}, map[string]bool{}, map[string]int{}
}

var bodyCtx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

// render renders the view into a fresh tree under a root, with an id on
// every element, and the frame of handlers it registered. It doesn't touch
// what the surface last sent: the caller swaps them in.
func (s *Surface) render() (*html.Node, *rt.Frame, map[string]string, map[string]*html.Node, error) {
	f := &rt.Frame{On: map[string]*rt.Handlers{}, Platform: s.platform, Surface: s}
	s.r.buf.Reset()
	if err := s.view.Render(rt.With(context.Background(), f), &s.r.buf); err != nil {
		return nil, nil, nil, nil, err
	}
	kids, err := html.ParseFragment(&s.r.buf, bodyCtx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []html.Attribute{{Key: "id", Val: "root"}}}
	for _, k := range kids {
		root.AppendChild(k)
	}
	ids := make(map[string]string, len(s.ids))
	byID := make(map[string]*html.Node, len(s.byID))
	byID["root"] = root
	s.number(root, "root", ids, byID)
	s.faults = append(s.faults, f.Faults...)
	return root, f, ids, byID, nil
}

// number gives every element under p an id: its own, or the one its place
// had in the last frame (the parent's id, its tag, and its index among the
// siblings without an id of their own), or a new one.
//
// Whitespace-only text beside elements is dropped, as JSX drops it: the
// template's indentation is not the page's, and a parent whose children
// are all elements can be told as narrow ops.
func (s *Surface) number(p *html.Node, pid string, ids map[string]string, byID map[string]*html.Node) {
	n := 0
	for c := p.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" && (c.PrevSibling != nil || next != nil) {
			p.RemoveChild(c)
			c = next
			continue
		}
		if c.Type != html.ElementNode {
			c = next
			continue
		}
		// An empty class is no class, as Preact leaves it out.
		for i, a := range c.Attr {
			if a.Key == "class" && a.Val == "" {
				c.Attr = append(c.Attr[:i], c.Attr[i+1:]...)
				break
			}
		}
		id := attr(c, "id")
		if id == "" {
			key := pid + "/" + c.Data + "#" + strconv.Itoa(n)
			n++
			id = s.ids[key]
			if id == "" {
				s.nextID++
				id = "e" + strconv.FormatInt(int64(s.nextID), 36)
			}
			ids[key] = id
			c.Attr = append([]html.Attribute{{Key: "id", Val: id}}, c.Attr...)
		}
		byID[id] = c
		s.number(c, id, ids, byID)
		c = next
	}
}

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, k string) bool {
	for _, a := range n.Attr {
		if a.Key == k {
			return true
		}
	}
	return false
}

// cssID is the id of the document's <style>, which a reload restyles.
const cssID = "hotty-css"

// documentOf is the whole document: the stylesheet, then the tree.
func (s *Surface) documentOf(root *html.Node) (string, error) {
	var b strings.Builder
	if s.css != "" {
		if strings.Contains(strings.ToLower(s.css), "</style") {
			return "", fmt.Errorf("surf: the stylesheet holds </style, which would end it early")
		}
		st := &html.Node{Type: html.ElementNode, Data: "style", DataAtom: atom.Style, Attr: []html.Attribute{{Key: "id", Val: cssID}}}
		st.AppendChild(&html.Node{Type: html.TextNode, Data: s.css})
		if err := html.Render(&b, st); err != nil {
			return "", err
		}
	}
	if err := html.Render(&b, root); err != nil {
		return "", err
	}
	return b.String(), nil
}

// isField says whether n is a live field: a control whose value or state
// is the host's once shown (SPEC §6.4), set by a=field rather than by its
// attributes. A hidden input, and an input that is a button, are plain
// elements: their value is an attribute like any other.
func isField(n *html.Node) bool {
	switch n.Data {
	case "textarea":
		return true
	case "input":
		switch strings.ToLower(attr(n, "type")) {
		case "hidden", "button", "submit", "reset", "image":
			return false
		}
		return true
	}
	return false
}

func isCheck(n *html.Node) bool {
	t := strings.ToLower(attr(n, "type"))
	return n.Data == "input" && (t == "checkbox" || t == "radio")
}

// remember notes what the fields under n hold as the document gives them.
func (s *Surface) remember(n *html.Node) {
	if n.Type == html.ElementNode && isField(n) {
		id := attr(n, "id")
		if isCheck(n) {
			s.checked[id] = hasAttr(n, "checked")
		} else {
			s.vals[id] = fieldValue(n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.remember(c)
	}
}

// fieldValue is the value a field's markup gives it: a textarea's text, an
// input's value attribute.
func fieldValue(n *html.Node) string {
	if n.Data == "textarea" {
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				b.WriteString(c.Data)
			}
		}
		return b.String()
	}
	return attr(n, "value")
}
