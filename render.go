package surf

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

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
	buf   bytes.Buffer
	p     parser
	last  []byte // the HTML the tree the host has was parsed from
	valid bool   // last is that tree's: false before one, and after a snapshot is adopted
	// at is when the view last rendered, in Unix nanoseconds (Peek's
	// Rendered reads it from another goroutine).
	at atomic.Int64
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
	rt.Peek.CSS = func(s any) string { return s.(*Surface).css }
	rt.Peek.Rendered = func(s any) time.Time {
		if n := s.(*Surface).r.at.Load(); n != 0 {
			return time.Unix(0, n)
		}
		return time.Time{}
	}
}

func (s *Surface) reset() {
	s.r.valid = false
	s.fields = nil
	s.ids = map[string]string{}
	s.byID = map[string]*html.Node{}
	s.vals, s.checked, s.revs = map[string]string{}, map[string]bool{}, map[string]int{}
}

var bodyCtx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

// paint renders the view into the renderer's buffer, with the frame of
// handlers it registers, and says whether the HTML is the last tree's, byte
// for byte: then the tree is the last one, ids and all, and needs no parse
// and no diff.
func (s *Surface) paint() (f *rt.Frame, same bool, err error) {
	f = &rt.Frame{On: map[string]*rt.Handlers{}, Platform: s.platform, Surface: s, Focused: s.focused, Memo: s.memo}
	s.r.buf.Reset()
	err = s.view.Render(rt.With(context.Background(), f), &s.r.buf)
	// When it ended: templ's dev mode read its strings before then.
	s.r.at.Store(time.Now().UnixNano())
	if err != nil {
		return nil, false, err
	}
	s.faults = append(s.faults, f.Faults...)
	return f, s.r.valid && bytes.Equal(s.r.buf.Bytes(), s.r.last), nil
}

// tree parses what paint rendered into a fresh tree under a root, with an
// id on every element. It doesn't touch what the surface last sent: the
// caller swaps the tree in with keep.
func (s *Surface) tree() (*tree, error) {
	t := &tree{
		root: newRoot(),
		ids:  make(map[string]string, len(s.ids)),
		byID: make(map[string]*html.Node, len(s.byID)),
	}
	src := s.r.buf.Bytes()
	if !s.r.p.parse(t.root, src) {
		t.root = newRoot()
		if err := parseSlow(t.root, src); err != nil {
			return nil, err
		}
	} else if parseCheck {
		checkParse(t.root, src)
	}
	t.byID["root"] = t.root
	s.number(t.root, "root", t)
	return t, nil
}

func newRoot() *html.Node {
	return &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []html.Attribute{{Key: "id", Val: "root"}}}
}

// parseSlow parses a render as x/net/html does, under root.
func parseSlow(root *html.Node, src []byte) error {
	kids, err := html.ParseFragment(bytes.NewReader(src), bodyCtx)
	if err != nil {
		return err
	}
	for _, k := range kids {
		root.AppendChild(k)
	}
	return nil
}

// A tree is a render's document under its #root, with its ids.
type tree struct {
	root   *html.Node
	ids    map[string]string     // an element's place → its id, for those without their own
	byID   map[string]*html.Node // every element by id
	fields []string              // the live fields' ids, in order
}

// keep takes t as what the host has, and the HTML paint rendered as its
// source.
func (s *Surface) keep(t *tree) {
	s.prev, s.ids, s.byID, s.fields = t.root, t.ids, t.byID, t.fields
	s.r.last = append(s.r.last[:0], s.r.buf.Bytes()...)
	s.r.valid = true
}

// number gives every element under p an id: its own, or the one its place
// had in the last frame (the parent's id, its tag, and its index among the
// siblings without an id of their own), or a new one.
//
// Whitespace-only text beside elements is dropped, as JSX drops it: the
// template's indentation is not the page's, and a parent whose children
// are all elements can be told as narrow ops.
func (s *Surface) number(p *html.Node, pid string, t *tree) {
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
			t.ids[key] = id
			c.Attr = withID(c.Attr, id)
		}
		t.byID[id] = c
		if isField(c) {
			t.fields = append(t.fields, id)
		}
		s.number(c, id, t)
		c = next
	}
}

// withID is as with the id attribute put first: in place, when the fast
// parser left room for it.
func withID(as []html.Attribute, id string) []html.Attribute {
	if len(as) < cap(as) {
		as = as[:len(as)+1]
		copy(as[1:], as)
		as[0] = html.Attribute{Key: "id", Val: id}
		return as
	}
	return append([]html.Attribute{{Key: "id", Val: id}}, as...)
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
