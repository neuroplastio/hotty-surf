// Package markup reads a .surf file: an element tree, as HTML writes
// one, whose attributes and text may hold expressions. It knows nothing
// of what a tag means; package surf does (docs/spec.md, "Templates").
//
// What it reads:
//
//   - an element, <tag attr="value" …>children</tag>, or <tag … />. A tag
//     is lower case, letters, digits and dashes, starting with a letter.
//     Every element is closed, by its end tag or by />.
//   - an attribute, name="value", name='value' or a bare name. A name may
//     start with : (a bound attribute) or @ (an event), and holds
//     letters, digits, -, _ and .; it is case-sensitive.
//   - text, with {{ expression }} in it. What is between {{ and }} is
//     the expression's, as it is, so that `{{ a < b }}` is no tag; a }}
//     inside one of its strings doesn't end it.
//   - the entities &amp; &lt; &gt; &quot; and &apos;, and a character
//     by its number, &#39; or &#xF08BB; (an icon's MDI glyph), in text
//     and in attribute values, outside {{ }}. An & that starts no entity
//     is the character, as HTML reads an ambiguous ampersand, so that
//     :if="a && b" needs no &amp;; but &name; that names no entity this
//     reads (&nbsp;), or a number that is no character, is an error.
//   - <!-- comments -->. A comment in a text is dropped, and the text goes
//     on: a <!-- c --> b reads "a b".
//
// An attribute given twice, the same name or once bare and once bound
// (key and :key), is an error at the second, naming the first's place.
//
// Whitespace: a text that is all whitespace is dropped; any other is
// trimmed at both ends, and each run of whitespace in it outside {{ }}
// is one space.
package markup

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"neuroplast.io/hotty-surf/internal/stackprobe"
)

// MaxDepth is how deep a file nests its elements: an element inside
// MaxDepth others is an error. It is as deep as a render may go
// (surf.MaxDepth), and keeps the parser's recursion, and the compiler's
// on what it reads, within a goroutine's stack under TinyGo.
const MaxDepth = 32

// Pos is a place in a file: its line and column, from 1, the column in
// bytes.
type Pos struct{ Line, Col int }

// Kind is what a Node is.
type Kind uint8

// The kinds.
const (
	Element Kind = iota + 1
	Text
)

// Node is an element or a text.
type Node struct {
	Kind Kind
	Pos  Pos
	// Tag and Attrs are an element's, Kids its children in order.
	Tag   string
	Attrs []Attr
	Kids  []*Node
	// Parts are a text's: its literal runs and its expressions, in order.
	Parts []Part
	// Lead and Trail are a text's: whether white space at its start or
	// its end was trimmed, which a run of a paragraph keeps as a space.
	Lead, Trail bool
}

// Attr is one attribute of an element.
type Attr struct {
	Name string
	// Value is the attribute's value, its entities read; "" for a bare
	// attribute.
	Value string
	Bare  bool
	Pos   Pos
	// ValuePos is where the value starts, after its quote.
	ValuePos Pos
}

// Part is a literal run of text, or an expression in {{ }}.
type Part struct {
	Expr bool
	// Src is the run's text, or the expression's source, trimmed.
	Src string
	Pos Pos
}

// Attr is n's attribute name, if it has one.
func (n *Node) Attr(name string) (Attr, bool) {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a, true
		}
	}
	return Attr{}, false
}

// Error is a place in a file and what is wrong there.
type Error struct {
	File string
	Pos  Pos
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Pos.Line, e.Pos.Col, e.Msg)
}

// Parse reads src, the file name, as a list of top-level nodes.
func Parse(name, src string) ([]*Node, error) {
	p := &parser{name: name, src: src, line: 1, col: 1}
	kids, err := p.nodes("")
	if err != nil {
		return nil, err
	}
	return kids, nil
}

type parser struct {
	name      string
	src       string
	i         int
	line, col int
	depth     int // the elements open
}

func (p *parser) pos() Pos { return Pos{p.line, p.col} }

func (p *parser) fail(at Pos, format string, args ...any) error {
	return &Error{File: p.name, Pos: at, Msg: fmt.Sprintf(format, args...)}
}

// advance moves past n bytes, counting lines.
func (p *parser) advance(n int) {
	for _, c := range []byte(p.src[p.i : p.i+n]) {
		if c == '\n' {
			p.line, p.col = p.line+1, 1
		} else {
			p.col++
		}
	}
	p.i += n
}

func (p *parser) has(s string) bool { return strings.HasPrefix(p.src[p.i:], s) }

func (p *parser) eof() bool { return p.i >= len(p.src) }

// The parser recurses on a file's elements, nodes and element in turn,
// so that the two keep small frames: what doesn't recurse (an element's
// tag and attributes, a comment, an end tag, a text) is in helpers that
// return first (go:noinline, which keeps their locals off the frames).

// nodes reads nodes up to the end tag of parent ("" for the file's end),
// and past that end tag.
func (p *parser) nodes(parent string) ([]*Node, error) {
	var out []*Node
	for {
		if p.has("<") && !p.has("</") && !p.has("<!--") {
			n, err := p.element()
			if err != nil {
				return nil, err
			}
			out = append(out, n)
			continue
		}
		n, end, err := p.other(parent)
		if err != nil {
			return nil, err
		}
		if end {
			return out, nil
		}
		if n != nil {
			out = append(out, n)
		}
	}
}

// other reads what is no element's start: the file's end, a comment,
// parent's end tag, or a text (nil where it is only whitespace). end
// says it was parent's end.
//
//go:noinline
func (p *parser) other(parent string) (n *Node, end bool, err error) {
	switch {
	case p.eof():
		if parent != "" {
			return nil, false, p.fail(p.pos(), "<%s> is not closed", parent)
		}
		return nil, true, nil
	case p.has("<!--"):
		at := p.pos()
		end := strings.Index(p.src[p.i+4:], "-->")
		if end < 0 {
			return nil, false, p.fail(at, "a comment is not closed")
		}
		p.advance(4 + end + 3)
		return nil, false, nil
	case p.has("</"):
		at := p.pos()
		p.advance(2)
		tag := p.name0()
		p.space()
		if !p.has(">") {
			return nil, false, p.fail(p.pos(), "an end tag ends with >")
		}
		p.advance(1)
		if tag != parent {
			if parent == "" {
				return nil, false, p.fail(at, "</%s> closes nothing", tag)
			}
			return nil, false, p.fail(at, "</%s> where </%s> was due", tag, parent)
		}
		return nil, true, nil
	}
	n, err = p.text()
	return n, false, err
}

// name0 reads a tag's name: lower case, digits and dashes.
func (p *parser) name0() string {
	j := p.i
	for j < len(p.src) {
		c := p.src[j]
		if c >= 'a' && c <= 'z' || j > p.i && (c >= '0' && c <= '9' || c == '-') {
			j++
			continue
		}
		break
	}
	s := p.src[p.i:j]
	p.advance(j - p.i)
	return s
}

func (p *parser) space() {
	for !p.eof() {
		switch p.src[p.i] {
		case ' ', '\t', '\n', '\r':
			p.advance(1)
		default:
			return
		}
	}
}

// element reads an element, and its children.
//
//go:noinline
func (p *parser) element() (*Node, error) {
	stackprobe.Probe()
	n, open, err := p.start()
	if err != nil {
		return nil, err
	}
	if !open {
		return n, nil
	}
	p.depth++
	kids, err := p.nodes(n.Tag)
	p.depth--
	if err != nil {
		return nil, err
	}
	n.Kids = kids
	return n, nil
}

// start reads an element's start tag: open where it has children, not
// where it ends with />.
//
//go:noinline
func (p *parser) start() (n *Node, open bool, err error) {
	at := p.pos()
	p.advance(1)
	tag := p.name0()
	if tag == "" {
		return nil, false, p.fail(at, "a < starts no tag: write &lt; for the character")
	}
	if p.depth >= MaxDepth {
		return nil, false, p.fail(at, "<%s> is nested past %d elements (markup.MaxDepth)", tag, MaxDepth)
	}
	n = &Node{Kind: Element, Pos: at, Tag: tag}
	for {
		p.space()
		switch {
		case p.eof():
			return nil, false, p.fail(at, "<%s is not ended by > or />", tag)
		case p.has("/>"):
			p.advance(2)
			return n, false, nil
		case p.has(">"):
			p.advance(1)
			return n, true, nil
		}
		a, err := p.attr()
		if err != nil {
			return nil, false, err
		}
		for _, b := range n.Attrs {
			if strings.TrimPrefix(b.Name, ":") == strings.TrimPrefix(a.Name, ":") {
				return nil, false, p.fail(a.Pos, "<%s> has %s twice: %s at %d:%d, %s at %d:%d",
					tag, strings.TrimPrefix(a.Name, ":"), b.Name, b.Pos.Line, b.Pos.Col, a.Name, a.Pos.Line, a.Pos.Col)
			}
		}
		n.Attrs = append(n.Attrs, a)
	}
}

func attrChar(c byte, first bool) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case first:
		return c == ':' || c == '@'
	}
	return c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.'
}

func (p *parser) attr() (Attr, error) {
	at := p.pos()
	j := p.i
	for j < len(p.src) && attrChar(p.src[j], j == p.i) {
		j++
	}
	if j == p.i || (j == p.i+1 && (p.src[p.i] == ':' || p.src[p.i] == '@')) {
		return Attr{}, p.fail(at, "%q starts no attribute", p.src[p.i:min(len(p.src), p.i+1)])
	}
	a := Attr{Name: p.src[p.i:j], Pos: at}
	p.advance(j - p.i)
	p.space()
	if !p.has("=") {
		a.Bare = true
		return a, nil
	}
	p.advance(1)
	p.space()
	if p.eof() || p.src[p.i] != '"' && p.src[p.i] != '\'' {
		return Attr{}, p.fail(p.pos(), "%s's value is quoted", a.Name)
	}
	q := p.src[p.i]
	p.advance(1)
	a.ValuePos = p.pos()
	end := strings.IndexByte(p.src[p.i:], q)
	if end < 0 {
		return Attr{}, p.fail(a.ValuePos, "%s's value is not closed", a.Name)
	}
	v, err := entities(p.src[p.i:p.i+end], func(off int) Pos { return offset(a.ValuePos, p.src[p.i:p.i+off]) })
	if err != nil {
		return Attr{}, &Error{File: p.name, Pos: err.(posError).pos, Msg: err.(posError).msg}
	}
	a.Value = v
	p.advance(end + 1)
	return a, nil
}

// offset is where s ends, if it starts at at.
func offset(at Pos, s string) Pos {
	for _, c := range []byte(s) {
		if c == '\n' {
			at.Line, at.Col = at.Line+1, 1
		} else {
			at.Col++
		}
	}
	return at
}

type posError struct {
	pos Pos
	msg string
}

func (e posError) Error() string { return e.msg }

var entityText = map[string]string{"amp": "&", "lt": "<", "gt": ">", "quot": `"`, "apos": "'"}

// charRef is the character a numeric reference's name is, #39 or
// #xF08BB, and whether it is one: a scalar value, neither NUL nor a
// surrogate.
func charRef(name string) (rune, bool) {
	digits, base := name[1:], 10
	if len(digits) > 0 && (digits[0] == 'x' || digits[0] == 'X') {
		digits, base = digits[1:], 16
	}
	v, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return 0, false
	}
	r := rune(v)
	return r, r > 0 && r <= unicode.MaxRune && (r < 0xD800 || r > 0xDFFF)
}

// entities reads s's entities; at is where s's offset off is, for an error.
func entities(s string, at func(off int) Pos) (string, error) {
	if !strings.Contains(s, "&") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := refEnd(s[i+1:])
		if end < 0 { // no entity: the character, as HTML reads it
			b.WriteByte('&')
			i++
			continue
		}
		name := s[i+1 : i+1+end]
		if name[0] == '#' {
			r, ok := charRef(name)
			if !ok {
				return "", posError{at(i), fmt.Sprintf("&%s; is no character", name)}
			}
			b.WriteRune(r)
			i += end + 2
			continue
		}
		t, ok := entityText[name]
		if !ok {
			return "", posError{at(i), fmt.Sprintf("&%s; is no entity this reads: write &amp; for the character", name)}
		}
		b.WriteString(t)
		i += end + 2
	}
	return b.String(), nil
}

// refEnd is where the reference s starts ends, at its ;, if s starts one:
// letters and digits, or # and digits, then ;. Else -1.
func refEnd(s string) int {
	for j := 0; j < len(s); j++ {
		switch c := s[j]; {
		case c == ';':
			if j == 0 {
				return -1
			}
			return j
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '#' && j == 0:
		default:
			return -1
		}
	}
	return -1
}

// text reads a text up to the next < outside {{ }}, a comment in it
// dropped: nil if it is all whitespace.
func (p *parser) text() (*Node, error) {
	at := p.pos()
	n := &Node{Kind: Text, Pos: at}
	var lit strings.Builder
	litAt := at
	flush := func() error {
		if lit.Len() == 0 {
			return nil
		}
		s, err := entities(lit.String(), func(int) Pos { return litAt })
		if err != nil {
			return &Error{File: p.name, Pos: litAt, Msg: err.(posError).msg}
		}
		n.Parts = append(n.Parts, Part{Src: s, Pos: litAt})
		lit.Reset()
		return nil
	}
	for !p.eof() && (!p.has("<") || p.has("<!--")) {
		if p.has("<!--") {
			end := strings.Index(p.src[p.i+4:], "-->")
			if end < 0 {
				return nil, p.fail(p.pos(), "a comment is not closed")
			}
			p.advance(4 + end + 3)
			continue
		}
		if !p.has("{{") {
			if lit.Len() == 0 {
				litAt = p.pos()
			}
			lit.WriteByte(p.src[p.i])
			p.advance(1)
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
		open := p.pos()
		p.advance(2)
		end, err := exprEnd(p.src[p.i:])
		if err != nil {
			return nil, p.fail(open, "%s", err.Error())
		}
		src := strings.TrimSpace(p.src[p.i : p.i+end])
		if src == "" {
			return nil, p.fail(open, "{{ }} holds no expression")
		}
		n.Parts = append(n.Parts, Part{Expr: true, Src: src, Pos: offset(open, "{{"+leading(p.src[p.i:p.i+end]))})
		p.advance(end + 2)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	n.Parts, n.Lead, n.Trail = condense(n.Parts)
	if len(n.Parts) == 0 {
		return nil, nil
	}
	return n, nil
}

// leading is s's leading whitespace.
func leading(s string) string { return s[:len(s)-len(strings.TrimLeft(s, " \t\r\n"))] }

// exprEnd is the offset of the }} that ends the expression s starts,
// past any string literal in it.
func exprEnd(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"' || c == '\'':
			j := i + 1
			for ; j < len(s) && s[j] != c; j++ {
				if s[j] == '\\' {
					j++
				}
			}
			if j >= len(s) {
				return 0, fmt.Errorf("a string in {{ }} is not closed")
			}
			i = j
		case c == '}' && i+1 < len(s) && s[i+1] == '}':
			return i, nil
		}
	}
	return 0, fmt.Errorf("{{ is not closed by }}")
}

// condense makes each run of whitespace in a text's literal parts one
// space, and trims the text's ends, saying whether it trimmed either:
// an all-whitespace text is left with no parts.
func condense(parts []Part) (out []Part, lead, trail bool) {
	out = parts[:0]
	for _, pt := range parts {
		if !pt.Expr {
			pt.Src = collapse(pt.Src)
		}
		out = append(out, pt)
	}
	for len(out) > 0 && !out[0].Expr {
		t := strings.TrimLeft(out[0].Src, " ")
		lead = lead || t != out[0].Src
		if out[0].Src = t; t != "" {
			break
		}
		out = out[1:]
	}
	for len(out) > 0 && !out[len(out)-1].Expr {
		l := &out[len(out)-1]
		t := strings.TrimRight(l.Src, " ")
		trail = trail || t != l.Src
		if l.Src = t; t != "" {
			break
		}
		out = out[:len(out)-1]
	}
	return out, lead, trail
}

// collapse is s with each run of whitespace one space.
func collapse(s string) string {
	var b strings.Builder
	space := false
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			space = true
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteByte(s[i])
	}
	if space {
		b.WriteByte(' ')
	}
	return b.String()
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
