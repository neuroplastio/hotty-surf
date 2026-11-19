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
	"neuroplast.io/hotty-go"

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
	// sig and lastSig are the keys this render and the last tree's
	// declare (keysig): the HTML doesn't carry them.
	sig, lastSig []byte
	modal        string // modal whose guards and background restrictions are in last
	// at is when the view last rendered, in Unix nanoseconds (Peek's
	// Rendered reads it from another goroutine).
	at atomic.Int64
}

func init() {
	rt.Peek.Tree = func(s any) *html.Node { return s.(*Surface).prev }
	rt.Peek.Focused = func(s any) string { return s.(*Surface).focused }
	rt.Peek.Frame = func(s any) *rt.Frame { return s.(*Surface).frame }
	rt.Peek.Scopes = func(s any) []*rt.KeyScope { return s.(*Surface).reached() }
	rt.Peek.FieldTakes = func(s any, key string) bool { return s.(*Surface).fieldTakes(key) }
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
	s.sels, s.pendingSel = map[string][2]int{}, map[string][2]int{}
	s.pendingNative = map[string]hotty.FieldChange{}
}

var bodyCtx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

// paint renders the view into the renderer's buffer, with the frame of
// handlers it registers, and says whether the HTML is the last tree's, byte
// for byte: then the tree is the last one, ids and all, and needs no parse
// and no diff.
func (s *Surface) paint() (f *rt.Frame, same bool, err error) {
	s.n++
	f = &rt.Frame{On: map[string]*rt.Handlers{}, Platform: s.platform, Surface: s, Focused: s.focused, Memo: s.memo, N: s.n, KeyMark: s.keyMark}
	s.r.buf.Reset()
	err = s.view.Render(rt.With(context.Background(), f), &s.r.buf)
	// When it ended: templ's dev mode read its strings before then.
	s.r.at.Store(time.Now().UnixNano())
	if err != nil {
		return nil, false, err
	}
	for i := 0; i < len(f.After); i++ {
		f.After[i]()
	}
	s.fault(f.Faults...)
	s.r.sig = keysig(s.r.sig, f)
	return f, s.r.valid && activeModal(f) == s.r.modal && bytes.Equal(s.r.buf.Bytes(), s.r.last) && bytes.Equal(s.r.sig, s.r.lastSig), nil
}

// tree parses what paint rendered into a fresh tree under a root, with an
// id on every element. It doesn't touch what the surface last sent: the
// caller swaps the tree in with keep.
func (s *Surface) tree(f *rt.Frame) (*tree, error) {
	t := &tree{
		f:    f,
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
	if f.ModalOps != nil {
		f.ModalOps.Prepare(t.root, activeModal(f))
	}
	t.byID["root"] = t.root
	s.number(t.root, "root", t, false)
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
	f      *rt.Frame             // the render's frame, whose keys writeKeys writes
	marks  map[int]bool          // the KeyScope markers seen (writeKeys)
	// scopes are the kit controls under a KeyScope, by id, as their
	// markers said before writeKeys took them off.
	scopes map[string]keyMark
	// printable are the printable keys the elements around the one being
	// numbered declare (writeKeys).
	printable []string
	// given are the keys the elements around the one being numbered give
	// the program (key=program, key=prefix), which Readline's bindings
	// leave alone (writeKeys).
	given []string
}

// keep takes t as what the host has, and the HTML paint rendered as its
// source.
func (s *Surface) keep(t *tree, f *rt.Frame) {
	s.prev, s.ids, s.byID, s.fields, s.keyMarks = t.root, t.ids, t.byID, t.fields, t.scopes
	s.r.last = append(s.r.last[:0], s.r.buf.Bytes()...)
	s.r.lastSig = append(s.r.lastSig[:0], s.r.sig...)
	s.r.valid = true
	s.r.modal = activeModal(f)
}

// number gives every element under p an id: its own, or the one its place
// had in the last frame (the parent's id, its tag, and its index among the
// siblings without an id of their own), or a new one.
//
// Whitespace-only text beside elements is dropped where it holds a line
// break, or isn't between two inline neighbours (text, or an element that
// flows in a line), as JSX drops it: the template's indentation is not the
// page's, and a parent whose children are all elements can be told as
// narrow ops. A space between two inline elements is the text's ("<b>a</b>
// <i>b</i>"), and stays. Inside a pre (pre is true) all of it stays.
func (s *Surface) number(p *html.Node, pid string, t *tree, pre bool) {
	n := 0
	for c := p.FirstChild; c != nil; {
		next := c.NextSibling
		if !pre && c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" && (c.PrevSibling != nil || next != nil) && (strings.IndexByte(c.Data, '\n') >= 0 || !(inline(c.PrevSibling) && inline(next))) {
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
		if id != "" && rt.RuntimeID(id) {
			s.fault(fmt.Sprintf("surf: an element's own id %q is the runtime's: root and hotty-css are the document's, and %s followed by base 36 is a generated id", id, rt.IDPrefix))
		}
		if id == "" {
			key := pid + "/" + c.Data + "#" + strconv.Itoa(n)
			n++
			id = s.ids[key]
			if id == "" {
				s.nextID++
				id = rt.IDPrefix + strconv.FormatInt(int64(s.nextID), 36)
			}
			t.ids[key] = id
			c.Attr = withID(c.Attr, id)
		}
		t.byID[id] = c
		pushed := s.writeKeys(c, id, t)
		if s.lifted && id == s.drag {
			addClass(c, "dragging")
		}
		if isField(c) {
			t.fields = append(t.fields, id)
		}
		s.number(c, id, t, pre || c.Data == "pre" || c.Data == "textarea" || c.Data == "listing")
		t.printable = t.printable[:len(t.printable)-pushed.printable]
		t.given = t.given[:len(t.given)-pushed.given]
		c = next
	}
}

// addClass adds class to n's class attribute.
func addClass(n *html.Node, class string) {
	for i, a := range n.Attr {
		if a.Key == "class" {
			n.Attr[i].Val = a.Val + " " + class
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: "class", Val: class})
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
	for _, sh := range s.sheets {
		var n *html.Node
		if limit, _ := s.overLimit(sh); limit != "" {
			// No room for it as a resource: the document carries it.
			if strings.Contains(strings.ToLower(sh.CSS), "</style") {
				return "", fmt.Errorf("surf: the sheet %q holds </style, which would end it early", sh.ID)
			}
			n = &html.Node{Type: html.ElementNode, Data: "style", DataAtom: atom.Style}
			n.AppendChild(&html.Node{Type: html.TextNode, Data: sh.CSS})
		} else {
			n = &html.Node{Type: html.ElementNode, Data: "link", DataAtom: atom.Link, Attr: []html.Attribute{{Key: "rel", Val: "stylesheet"}, {Key: "href", Val: "cid:" + sh.ID}}}
		}
		if err := html.Render(&b, n); err != nil {
			return "", err
		}
	}
	// With sheets linked, the surface's own sheet is there even empty, for
	// a dev reload to restyle.
	if s.css != "" || len(s.sheets) > 0 {
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
	case "select":
		// One value, as a=field sets it (SPEC §6.4); a multiple select's
		// options stay attributes.
		return !hasAttr(n, "multiple")
	case "input":
		switch strings.ToLower(attr(n, "type")) {
		case "hidden", "button", "submit", "reset", "image", "file":
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

// fieldValue is the value a field's markup gives it: a textarea's text, a
// select's chosen option's value, an input's value attribute.
func fieldValue(n *html.Node) string {
	if n.Data == "select" {
		return selectValue(n)
	}
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

// inline says whether n is text or an element that flows in a line, so a
// space beside it is the text's.
func inline(n *html.Node) bool {
	if n == nil {
		return false
	}
	if n.Type == html.TextNode {
		return true
	}
	return n.Type == html.ElementNode && (phrasing(n.DataAtom) || n.DataAtom == atom.Svg || n.DataAtom == atom.Select || n.DataAtom == atom.Textarea || n.DataAtom == 0)
}

// fresh forgets the revs of the fields under n that are new nodes on the
// host: a control entering the document starts at rev 0 (SPEC §6.4). With
// kept set, a field the host morphs (an inner, SPEC §6.2: the same id under
// a parent with the same id, as the last tree had it) keeps its rev.
func (s *Surface) fresh(n *html.Node, kept bool) {
	if n.Type == html.ElementNode && isField(n) {
		id := attr(n, "id")
		if o := s.byID[id]; !kept || o == nil || o.Data != n.Data || o.Parent == nil || n.Parent == nil || attr(o.Parent, "id") != attr(n.Parent, "id") {
			delete(s.revs, id)
			delete(s.sels, id)
			delete(s.pendingSel, id)
			delete(s.pendingNative, id)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.fresh(c, kept)
	}
}

// submits faults a Submit on an element that isn't a form: it can never
// run.
func (s *Surface) submits(f *rt.Frame) {
	for id, h := range f.On {
		if h.Submit == nil {
			continue
		}
		if n := s.byID[id]; n != nil && n.DataAtom != atom.Form {
			s.fault("surf.Submit: the element " + id + " is a <" + n.Data + ">, not a form: only a form submits (put its fields and its submit button in one)")
		}
	}
}

// unlabelled faults a button with no name, as HTML names one: no text (an
// image's alt counts), no aria-label, aria-labelledby or title, and no
// <label> for it or around it. The cells fallback draws such a button as
// "[ · ]", and a screen reader has nothing to say.
func (s *Surface) unlabelled(root *html.Node) {
	named := map[string]bool{}
	var labels func(n *html.Node)
	labels = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Label && attr(n, "for") != "" {
			named[attr(n, "for")] = true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			labels(c)
		}
	}
	labels(root)
	// near is where the author finds the button: the nearest app id.
	var walk func(n *html.Node, inLabel bool, near string)
	walk = func(n *html.Node, inLabel bool, near string) {
		if n.Type == html.ElementNode {
			if id := attr(n, "id"); id != "" && !rt.RuntimeID(id) {
				near = "#" + id
			}
			switch n.DataAtom {
			case atom.Label:
				inLabel = true
			case atom.Button:
				if !inLabel && !named[attr(n, "id")] && !hasLabel(n) && strings.TrimSpace(textOf(n)) == "" {
					where := near
					if id := attr(n, "id"); id == "" || rt.RuntimeID(id) {
						where = "in " + near
						if near == "" {
							where = "class " + strings.Fields(attr(n, "class") + " none")[0]
						}
					}
					s.fault("a button with no text and no label (" + where + "): give it text (an icon's beside a <span class=\"sr-only\">), an aria-label in its Attrs, or a <label for> it; a ui.Select with nothing chosen takes a Placeholder")
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inLabel, near)
		}
	}
	walk(root, false, "")
}

func hasLabel(n *html.Node) bool {
	for _, a := range n.Attr {
		if (a.Key == "aria-label" || a.Key == "aria-labelledby" || a.Key == "title") && strings.TrimSpace(a.Val) != "" {
			return true
		}
	}
	return false
}

// textOf is n's text, outside an svg.
func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type == html.ElementNode && n.Data == "svg" {
		return ""
	}
	if n.Type == html.ElementNode && n.DataAtom == atom.Img {
		return attr(n, "alt")
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}

// maxFaults is as many faults as a surface holds: surftest drains them
// after each act, and outside a test they would otherwise pile up, one
// per render.
const maxFaults = 64

// fault records faults while the surface holds fewer than maxFaults.
func (s *Surface) fault(fs ...string) {
	for _, f := range fs {
		if len(s.faults) >= maxFaults {
			return
		}
		s.faults = append(s.faults, f)
	}
}

// chooses says whether a select's markup chooses an option (one of its
// options has selected). One that doesn't is the host's, as an input
// without a value attribute is.
func chooses(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "option" && hasAttr(c, "selected") || c.Data == "optgroup" && chooses(c)) {
			return true
		}
	}
	return false
}

// selectValue is the value of the option a select's markup chooses, as
// HTML chooses it: the last with selected, or else the first that isn't
// disabled; "" for none.
func selectValue(n *html.Node) string {
	var chosen, first *html.Node
	var walk func(*html.Node)
	walk = func(p *html.Node) {
		for c := p.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type != html.ElementNode:
			case c.Data == "option":
				if hasAttr(c, "selected") {
					chosen = c
				}
				if first == nil && !hasAttr(c, "disabled") {
					first = c
				}
			case c.Data == "optgroup":
				walk(c)
			}
		}
	}
	walk(n)
	if chosen == nil {
		chosen = first
	}
	if chosen == nil {
		return ""
	}
	if v, ok := get(chosen, "value"); ok {
		return v
	}
	var b strings.Builder
	for c := chosen.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
