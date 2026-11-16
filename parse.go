package surf

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The fast parser. A render is templ's output: well-formed HTML, every
// element closed where it opened, attributes quoted and escaped. Parsed as
// HTML5 says, such markup is just its tags' nesting, so parseFast builds
// the tree x/net/html would by following the tags, with none of the
// tokenizer's copies: every name and value is a slice of one string of the
// render, and the nodes come in slabs.
//
// What HTML5 parses as something other than its nesting (a <p> that a
// block closes, a table's implied tbody, raw text, a misnested or
// unclosed tag, an entity without its semicolon) makes it give up, and
// the render is parsed by x/net/html instead. An <svg> is parsed by
// x/net/html too (foreign content has its own rules), once per distinct
// markup: an icon's tree is cloned from the first.
//
// The surfparsecheck build tag parses every render both ways and panics
// where they differ (parse_check.go): the examples' tests run with it.

// parser is a surface's fast parser, kept between renders.
type parser struct {
	src   string // the render, which every name and value slices
	nodes []html.Node
	attrs []html.Attribute
	stack []*html.Node
	svg   map[string]*html.Node // an <svg>'s markup → its tree, to clone
}

// slab sizes: the nodes and attributes a render takes are allocated this
// many at a time.
const (
	nodeSlab = 512
	attrSlab = 1024
	svgCache = 256 // distinct <svg> markups kept
)

func (p *parser) node() *html.Node {
	if len(p.nodes) == cap(p.nodes) {
		p.nodes = make([]html.Node, 0, nodeSlab)
	}
	p.nodes = p.nodes[:len(p.nodes)+1]
	return &p.nodes[len(p.nodes)-1]
}

// attrList is room for n attributes and one more, for the id a render's
// numbering puts first (number), capped there so that a later append moves
// them rather than writing over the next element's.
func (p *parser) attrList(n int) []html.Attribute {
	n++
	if cap(p.attrs)-len(p.attrs) < n {
		p.attrs = make([]html.Attribute, 0, max(attrSlab, n))
	}
	i := len(p.attrs)
	p.attrs = p.attrs[:i+n]
	return p.attrs[i : i : i+n]
}

// parse appends the render's nodes to root, as html.ParseFragment in a
// body would give them, and says whether it could: when it couldn't,
// root's children are whatever it got to, and the caller starts again.
func (p *parser) parse(root *html.Node, b []byte) bool {
	p.src = string(b)
	// The slabs aren't reused: the last tree still points into them.
	p.nodes, p.attrs = nil, nil
	p.stack = append(p.stack[:0], root)
	s := p.src
	for i := 0; i < len(s); {
		top := p.stack[len(p.stack)-1]
		if s[i] != '<' {
			j := strings.IndexByte(s[i:], '<')
			if j < 0 {
				j = len(s)
			} else {
				j += i
			}
			if !p.text(top, s[i:j]) {
				return false
			}
			i = j
			continue
		}
		if i+1 >= len(s) {
			return false
		}
		if s[i+1] == '/' {
			n, ok := p.end(top, s, i)
			if !ok {
				return false
			}
			i = n
			continue
		}
		if !isLetter(s[i+1]) {
			return false // a comment, a doctype, or a < in text
		}
		n, ok := p.start(top, s, i)
		if !ok {
			return false
		}
		i = n
	}
	return len(p.stack) == 1
}

// text adds a text node to top: what isn't in a tag.
func (p *parser) text(top *html.Node, t string) bool {
	if strings.IndexByte(t, 0) >= 0 || strings.IndexByte(t, '\r') >= 0 {
		return false
	}
	switch top.DataAtom {
	case atom.Table, atom.Thead, atom.Tbody, atom.Tfoot, atom.Tr, atom.Colgroup:
		// Text in a table outside a cell is fostered out of it, unless it
		// is only spaces.
		if strings.TrimLeft(t, " \t\n\f") != "" {
			return false
		}
	}
	if strings.IndexByte(t, '&') >= 0 {
		if !entities(t) {
			return false
		}
		t = html.UnescapeString(t)
	}
	if last := top.LastChild; last != nil && last.Type == html.TextNode {
		last.Data += t
		return true
	}
	n := p.node()
	n.Type, n.Data = html.TextNode, t
	top.AppendChild(n)
	return true
}

// end closes the element on top: an end tag must name it.
func (p *parser) end(top *html.Node, s string, i int) (int, bool) {
	j := i + 2
	k := j
	for k < len(s) && isNameByte(s[k]) {
		k++
	}
	name := s[j:k]
	for k < len(s) && isSpace(s[k]) {
		k++
	}
	if k >= len(s) || s[k] != '>' || len(p.stack) == 1 || !strings.EqualFold(name, top.Data) {
		return 0, false
	}
	p.stack = p.stack[:len(p.stack)-1]
	return k + 1, true
}

// start reads a start tag at s[i] and opens its element, or adds it if it
// is void.
func (p *parser) start(top *html.Node, s string, i int) (int, bool) {
	j := i + 1
	k := j
	for k < len(s) && isNameByte(s[k]) {
		k++
	}
	name := s[j:k]
	for x := 0; x < len(name); x++ {
		if name[x] >= 0x80 {
			return 0, false // the tokenizer lowercases ASCII alone, strings.ToLower more
		}
	}
	a := atom.Lookup([]byte(strings.ToLower(name)))
	if a == atom.Svg {
		return p.svgTree(top, s, i)
	}
	if !p.allowed(top, a) {
		return 0, false
	}
	// The attributes: count them first, for a list of the right size.
	var buf [16][2]string
	kv := buf[:0]
	self := false
	for {
		for k < len(s) && isSpace(s[k]) {
			k++
		}
		if k >= len(s) {
			return 0, false
		}
		if s[k] == '>' {
			k++
			break
		}
		if s[k] == '/' {
			if k+1 < len(s) && s[k+1] == '>' {
				self = true
				k += 2
				break
			}
			return 0, false
		}
		ks := k
		for k < len(s) && !isSpace(s[k]) && s[k] != '=' && s[k] != '>' && s[k] != '/' {
			c := s[k]
			if c == '"' || c == '\'' || c == '<' || c == 0 || c == '\r' || c >= 0x80 {
				return 0, false
			}
			k++
		}
		key := s[ks:k]
		if key == "" {
			return 0, false
		}
		if hasUpper(key) {
			key = strings.ToLower(key)
		} else if ka := atom.Lookup([]byte(key)); ka != 0 {
			key = ka.String() // a static string, not a slice of the render
		}
		val := ""
		for k < len(s) && isSpace(s[k]) {
			k++
		}
		if k < len(s) && s[k] == '=' {
			k++
			for k < len(s) && isSpace(s[k]) {
				k++
			}
			if k >= len(s) {
				return 0, false
			}
			switch q := s[k]; q {
			case '"', '\'':
				e := strings.IndexByte(s[k+1:], q)
				if e < 0 {
					return 0, false
				}
				val = s[k+1 : k+1+e]
				k += e + 2
			default:
				vs := k
				for k < len(s) && !isSpace(s[k]) && s[k] != '>' {
					switch s[k] {
					case '"', '\'', '<', '=', '`':
						return 0, false
					}
					k++
				}
				val = s[vs:k]
				if val == "" {
					return 0, false
				}
			}
			if strings.IndexByte(val, 0) >= 0 || strings.IndexByte(val, '\r') >= 0 {
				return 0, false
			}
			if strings.IndexByte(val, '&') >= 0 {
				if !entities(val) {
					return 0, false
				}
				val = html.UnescapeString(val)
			}
		}
		for _, x := range kv {
			if x[0] == key {
				return 0, false // the tokenizer and the parser keep the first
			}
		}
		kv = append(kv, [2]string{key, val})
	}
	n := p.node()
	n.Type, n.DataAtom = html.ElementNode, a
	if a != 0 {
		n.Data = a.String()
	} else {
		n.Data = strings.ToLower(name)
	}
	n.Attr = p.attrList(len(kv))
	for _, x := range kv {
		n.Attr = append(n.Attr, html.Attribute{Key: x[0], Val: x[1]})
	}
	top.AppendChild(n)
	if void(a) {
		return k, true
	}
	if self {
		return 0, false // HTML ignores the slash: the element stays open
	}
	p.stack = append(p.stack, n)
	return k, true
}

// allowed says whether an element a opened under top, with the open
// elements p.stack, is just nested there, as HTML5's tree construction
// would have it.
func (p *parser) allowed(top *html.Node, a atom.Atom) bool {
	switch a {
	case 0:
		return !p.has(atom.P) // a custom element is no phrasing content to the parser
	case atom.Script, atom.Style, atom.Textarea, atom.Title, atom.Xmp, atom.Iframe,
		atom.Noembed, atom.Noframes, atom.Plaintext, atom.Template, atom.Noscript,
		atom.Pre, atom.Listing, atom.Select, atom.Option, atom.Optgroup, atom.Datalist,
		atom.Html, atom.Head, atom.Body, atom.Frameset, atom.Frame, atom.Base, atom.Link,
		atom.Meta, atom.Math, atom.Image, atom.Object, atom.Applet, atom.Marquee,
		atom.Nobr, atom.Keygen, atom.Menuitem, atom.Rb, atom.Rp, atom.Rt, atom.Rtc,
		atom.Basefont, atom.Bgsound, atom.Param, atom.Caption, atom.Col, atom.Colgroup:
		return false
	case atom.A, atom.Button, atom.Form:
		if p.has(a) {
			return false
		}
	case atom.Li:
		if !in(top.DataAtom, atom.Ul, atom.Ol, atom.Menu) {
			return false
		}
	case atom.Dd, atom.Dt:
		if top.DataAtom != atom.Dl {
			return false
		}
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		if in(top.DataAtom, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6) {
			return false
		}
	case atom.Thead, atom.Tbody, atom.Tfoot:
		return top.DataAtom == atom.Table
	case atom.Tr:
		return in(top.DataAtom, atom.Thead, atom.Tbody, atom.Tfoot)
	case atom.Td, atom.Th:
		return top.DataAtom == atom.Tr
	}
	// Under a table, only its own parts nest.
	if in(top.DataAtom, atom.Table, atom.Thead, atom.Tbody, atom.Tfoot, atom.Tr) {
		return false
	}
	// A <p> is closed by what isn't phrasing content.
	if !phrasing(a) && p.has(atom.P) {
		return false
	}
	return true
}

func (p *parser) has(a atom.Atom) bool {
	for _, n := range p.stack[1:] {
		if n.DataAtom == a {
			return true
		}
	}
	return false
}

// svgTree adds the <svg> at s[i] and what it holds, parsed by x/net/html
// the first time its markup is seen and cloned after.
func (p *parser) svgTree(top *html.Node, s string, i int) (int, bool) {
	// An <svg> closes no <p>, but a table fosters it out.
	if in(top.DataAtom, atom.Table, atom.Thead, atom.Tbody, atom.Tfoot, atom.Tr) {
		return 0, false
	}
	end := svgEnd(s, i)
	if end < 0 {
		return 0, false
	}
	src := s[i:end]
	t, ok := p.svg[src]
	if !ok {
		kids, err := html.ParseFragment(strings.NewReader(src), bodyCtx)
		if err != nil || len(kids) != 1 || kids[0].DataAtom != atom.Svg || kids[0].Namespace != "svg" {
			return 0, false
		}
		t = kids[0]
		if p.svg == nil || len(p.svg) >= svgCache {
			p.svg = map[string]*html.Node{}
		}
		p.svg[strings.Clone(src)] = t
	}
	top.AppendChild(clone(t))
	return end, true
}

// svgEnd is where the <svg> element at s[i] ends, after its </svg>: -1
// when it doesn't.
func svgEnd(s string, i int) int {
	depth := 0
	for k := i; k < len(s); {
		j := strings.IndexByte(s[k:], '<')
		if j < 0 {
			return -1
		}
		k += j
		switch {
		case strings.HasPrefix(s[k:], "<svg") && k+4 < len(s) && (isSpace(s[k+4]) || s[k+4] == '>' || s[k+4] == '/'):
			depth++
		case strings.HasPrefix(s[k:], "</svg>"):
			depth--
			if depth == 0 {
				return k + len("</svg>")
			}
		}
		k++
	}
	return -1
}

// clone is a deep copy of n, which a render may change (an id put first,
// an empty class taken out) without touching n.
func clone(n *html.Node) *html.Node {
	c := &html.Node{Type: n.Type, DataAtom: n.DataAtom, Data: n.Data, Namespace: n.Namespace}
	if len(n.Attr) > 0 {
		c.Attr = make([]html.Attribute, len(n.Attr))
		copy(c.Attr, n.Attr)
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		c.AppendChild(clone(k))
	}
	return c
}

// entities says whether every & in t starts a character reference that
// ends with its semicolon, which html.UnescapeString reads as the
// tokenizer would, in text and in an attribute alike. A bare & (one
// followed by no name) is text in both.
func entities(t string) bool {
	for i := 0; i < len(t); i++ {
		if t[i] != '&' {
			continue
		}
		j := i + 1
		if j < len(t) && t[j] == '#' {
			j++
			if j < len(t) && (t[j] == 'x' || t[j] == 'X') {
				j++
			}
			k := j
			for k < len(t) && isHex(t[k]) {
				k++
			}
			if k == j || k >= len(t) || t[k] != ';' {
				return false
			}
			i = k
			continue
		}
		k := j
		for k < len(t) && (isLetter(t[k]) || t[k] >= '0' && t[k] <= '9') {
			k++
		}
		if k == j {
			continue // a bare &
		}
		if k >= len(t) || t[k] != ';' {
			return false
		}
		i = k
	}
	return true
}

func void(a atom.Atom) bool {
	switch a {
	case atom.Area, atom.Br, atom.Embed, atom.Hr, atom.Img, atom.Input, atom.Source, atom.Track, atom.Wbr:
		return true
	}
	return false
}

// phrasing says whether a is an element that doesn't close an open <p>.
func phrasing(a atom.Atom) bool {
	switch a {
	case atom.A, atom.Abbr, atom.B, atom.Bdi, atom.Bdo, atom.Br, atom.Button, atom.Cite,
		atom.Code, atom.Data, atom.Dfn, atom.Em, atom.I, atom.Img, atom.Input, atom.Kbd,
		atom.Label, atom.Mark, atom.Q, atom.S, atom.Samp, atom.Small, atom.Span,
		atom.Strong, atom.Sub, atom.Sup, atom.Time, atom.U, atom.Var, atom.Wbr:
		return true
	}
	return false
}

func in(a atom.Atom, set ...atom.Atom) bool {
	for _, x := range set {
		if a == x {
			return true
		}
	}
	return false
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\f' }

// isNameByte is a byte of a tag's name, as the tokenizer reads one.
func isNameByte(c byte) bool {
	return !isSpace(c) && c != '/' && c != '>' && c != 0 && c != '\r'
}

func hasUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}
	return false
}
