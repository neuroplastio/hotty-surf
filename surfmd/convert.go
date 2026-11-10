package surfmd

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"neuroplast.io/hotty-surf/surfui"
)

// Option changes what Convert makes.
type Option func(*config)

// NodeFunc makes the surfui node of one goldmark node, in place of the
// default's: n is the node, src the source it was parsed from, and kids
// what its children converted to (a block's blocks, an inline's runs).
// It returns nil to leave the node to the default.
type NodeFunc func(n ast.Node, src []byte, kids []*surfui.Node) *surfui.Node

type config struct {
	hard, gfm, compact bool
	anchors            func(string) string
	highlight          func(lang, code string) []surfui.Span
	nodes              map[ast.NodeKind]NodeFunc
}

// HardWraps makes every newline inside a paragraph a break, as a person
// who typed it meant it (margin's comments); by default it is a space,
// as CommonMark has it.
func HardWraps() Option { return func(c *config) { c.hard = true } }

// GFM turns on GitHub's extras: tables, strikethrough, task lists and
// alerts (a quote that starts with [!NOTE], [!TIP], [!IMPORTANT],
// [!WARNING] or [!CAUTION]).
func GFM() Option { return func(c *config) { c.gfm = true } }

// Compact makes the prose a comment's (surfui.Node.Compact): no measure,
// the surface's text size.
func Compact() Option { return func(c *config) { c.compact = true } }

// Highlight has f tokenize a fenced code block: lang is its info string's
// first word ("go", "" for none) and code its source, and f returns the
// spans the block shows (surfui.Span, the roles of a code block). It may
// return nil to leave a language to the block's own lexer (go, surf) or
// plain text. The converter tokenizes nothing itself; chroma stays the
// caller's.
func Highlight(f func(lang, code string) []surfui.Span) Option {
	return func(c *config) { c.highlight = f }
}

// Anchors names each heading by f of its text, the id a #fragment link
// finds it by; the website's is GitHub's. Without it a heading has none.
func Anchors(f func(text string) string) Option { return func(c *config) { c.anchors = f } }

// Node has f make the nodes of goldmark's kind (ast.KindFencedCodeBlock,
// east.KindTable, …) in place of the default: a callout, a fence of
// mermaid. f returns nil for a node it leaves to the default.
func Node(kind ast.NodeKind, f NodeFunc) Option {
	return func(c *config) {
		if c.nodes == nil {
			c.nodes = map[ast.NodeKind]NodeFunc{}
		}
		c.nodes[kind] = f
	}
}

func settings(opts []Option) *config {
	c := &config{}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Convert is the Markdown in src as a Prose of surfui's nodes. Raw HTML in
// it is text, never markup.
func Convert(src []byte, opts ...Option) *surfui.Node {
	c := settings(opts)
	var exts []goldmark.Extender
	if c.gfm {
		exts = append(exts, extension.Table, extension.Strikethrough, extension.TaskList)
	}
	md := goldmark.New(goldmark.WithExtensions(exts...))
	doc := md.Parser().Parse(text.NewReader(src))
	return convertDoc(doc, src, c)
}

// ConvertAST is Convert of a document goldmark parsed from src already,
// by a parser of the caller's own: with its own extensions, its own
// settings. The extensions' node kinds that this package knows (tables,
// strikethrough, task lists) are converted whether or not GFM is given.
func ConvertAST(n ast.Node, src []byte, opts ...Option) *surfui.Node {
	return convertDoc(n, src, settings(opts))
}

func convertDoc(n ast.Node, src []byte, c *config) *surfui.Node {
	cv := &converter{c: c, src: src}
	p := surfui.Prose(cv.blocks(n)...)
	if c.compact {
		p.Compact()
	}
	return p
}

type converter struct {
	c   *config
	src []byte
}

// blocks are the blocks n's children make.
func (cv *converter) blocks(n ast.Node) []*surfui.Node {
	var out []*surfui.Node
	for k := n.FirstChild(); k != nil; k = k.NextSibling() {
		if b := cv.block(k); b != nil {
			out = append(out, b)
		}
	}
	return out
}

// override is what a Node option makes of n, nil for none.
func (cv *converter) override(n ast.Node, kids func() []*surfui.Node) *surfui.Node {
	f := cv.c.nodes[n.Kind()]
	if f == nil {
		return nil
	}
	return f(n, cv.src, kids())
}

func (cv *converter) block(n ast.Node) *surfui.Node {
	kids := func() []*surfui.Node {
		if _, leaf := n.(interface{ Lines() *text.Segments }); leaf && !n.HasChildren() {
			return nil
		}
		if isInlineParent(n) {
			return cv.inlines(n)
		}
		return cv.blocks(n)
	}
	if o := cv.override(n, kids); o != nil {
		return o
	}
	switch n := n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return surfui.Para(cv.inlines(n)...)
	case *ast.Heading:
		anchor := ""
		if cv.c.anchors != nil {
			anchor = cv.c.anchors(cv.plain(n))
		}
		return surfui.H(n.Level, anchor, cv.inlines(n)...)
	case *ast.Blockquote:
		return cv.quote(n)
	case *ast.List:
		return cv.list(n)
	case *ast.ThematicBreak:
		return surfui.Rule()
	case *ast.FencedCodeBlock:
		return cv.code(string(n.Language(cv.src)), linesOf(n, cv.src))
	case *ast.CodeBlock:
		return cv.code("", linesOf(n, cv.src))
	case *ast.HTMLBlock:
		// Raw HTML is text: it shows as written and is never injected.
		return surfui.Para(surfui.Str(strings.TrimRight(string(n.Lines().Value(cv.src)), "\n")))
	case *east.Table:
		return cv.table(n)
	}
	// A block this package doesn't know: its children, as blocks.
	return surfui.Stack(cv.blocks(n)...)
}

// isInlineParent reports whether n's children are inlines.
func isInlineParent(n ast.Node) bool {
	switch n.(type) {
	case *ast.Paragraph, *ast.TextBlock, *ast.Heading:
		return true
	}
	return false
}

// linesOf are a code block's lines, without their newlines.
func linesOf(n ast.Node, src []byte) []string {
	l := n.Lines()
	var out []string
	for i := 0; i < l.Len(); i++ {
		s := l.At(i)
		out = append(out, strings.TrimRight(string(s.Value(src)), "\r\n"))
	}
	return out
}

// code is a fenced or indented block of code: surfui's code block, its
// spans the caller's Highlight gives where it does, else the block's own
// lexer for the language (go and surf; any other is plain). Where the prose
// is compact, a box beside a comment's, a line wraps at a space.
func (cv *converter) code(lang string, lines []string) *surfui.Node {
	source := strings.Join(lines, "\n")
	var n *surfui.Node
	if cv.c.highlight != nil {
		if spans := cv.c.highlight(lang, source); spans != nil {
			n = surfui.CodeSpans(spans)
		}
	}
	if n == nil {
		n = surfui.CodeBlock(source)
		if lang == "go" || lang == "surf" {
			n.Language(lang)
		}
	}
	if cv.c.compact {
		n.Wrap()
	}
	return n
}

var alertKinds = map[string]string{"NOTE": "note", "TIP": "tip", "IMPORTANT": "important", "WARNING": "warning", "CAUTION": "caution"}

// quote is a block quote, or with GFM an alert: a quote whose first line
// is [!KIND].
func (cv *converter) quote(n *ast.Blockquote) *surfui.Node {
	if cv.c.gfm {
		if kind, rest, ok := cv.alert(n); ok {
			return surfui.Alert(kind, rest...)
		}
	}
	return surfui.Quote(cv.blocks(n)...)
}

// alert reads an alert's kind off n's first paragraph's first line, and
// gives the blocks after it: the paragraph's other lines, and the rest.
func (cv *converter) alert(n *ast.Blockquote) (kind string, blocks []*surfui.Node, ok bool) {
	p, isPara := n.FirstChild().(*ast.Paragraph)
	if !isPara || p.Lines().Len() == 0 {
		return "", nil, false
	}
	first := p.Lines().At(0)
	line := strings.TrimSpace(string(first.Value(cv.src)))
	if !strings.HasPrefix(line, "[!") || !strings.HasSuffix(line, "]") {
		return "", nil, false
	}
	kind, ok = alertKinds[strings.ToUpper(line[2:len(line)-1])]
	if !ok {
		return "", nil, false
	}
	// The paragraph's runs after the first line.
	var runs []*surfui.Node
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		if t, isText := c.(*ast.Text); isText && t.Segment.Stop <= first.Stop {
			continue
		}
		runs = append(runs, cv.inline(c)...)
	}
	if len(runs) > 0 {
		blocks = append(blocks, surfui.Para(runs...))
	}
	for k := p.NextSibling(); k != nil; k = k.NextSibling() {
		if b := cv.block(k); b != nil {
			blocks = append(blocks, b)
		}
	}
	return kind, blocks, true
}

func (cv *converter) list(n *ast.List) *surfui.Node {
	var items []*surfui.Node
	for it := n.FirstChild(); it != nil; it = it.NextSibling() {
		items = append(items, cv.item(it))
	}
	if n.IsOrdered() {
		return surfui.Numbered(n.Start, items...)
	}
	return surfui.Bullets(items...)
}

// item is a list item; a task's checkbox, which GFM parses as the first
// inline of its first block, makes it a task.
func (cv *converter) item(n ast.Node) *surfui.Node {
	task, checked := false, false
	var kids []*surfui.Node
	for b := n.FirstChild(); b != nil; b = b.NextSibling() {
		if first := b.FirstChild(); first != nil && kids == nil {
			if cb, ok := first.(*east.TaskCheckBox); ok {
				task, checked = true, cb.IsChecked
			}
		}
		if blk := cv.block(b); blk != nil {
			kids = append(kids, blk)
		}
	}
	li := surfui.Li(kids...)
	if task {
		li.Checked(checked)
	}
	return li
}

// table is a GFM table as surfui's Table: a Col a column, its heading the
// header cell's text and its alignment the table's, a row an Item whose
// cells are texts, or a line of runs where a cell holds markup.
func (cv *converter) table(n *east.Table) *surfui.Node {
	var cols, rows []*surfui.Node
	for r := n.FirstChild(); r != nil; r = r.NextSibling() {
		var cells []*surfui.Node
		for c := r.FirstChild(); c != nil; c = c.NextSibling() {
			cells = append(cells, cv.cell(c))
		}
		if _, head := r.(*east.TableHeader); head {
			for i, c := range cells {
				col := surfui.Col(surfui.TextOf(c))
				if i < len(n.Alignments) {
					switch n.Alignments[i] {
					case east.AlignRight:
						col.Align("end")
					case east.AlignCenter:
						col.Align("center")
					}
				}
				cols = append(cols, col)
			}
			continue
		}
		rows = append(rows, surfui.Item(cells...).Key(strconv.Itoa(len(rows))))
	}
	return surfui.Table(append(cols, rows...)...)
}

// cell is a table cell: a text, or a line of runs where it holds markup.
func (cv *converter) cell(n ast.Node) *surfui.Node {
	runs := cv.inlines(n)
	plain := true
	for _, r := range runs {
		if surfui.Tag(r) != "" {
			plain = false
		}
	}
	if plain {
		return surfui.Text(cv.plain(n))
	}
	return surfui.Line(runs...)
}

// plain is the text of n's inlines, without their markup.
func (cv *converter) plain(n ast.Node) string {
	var b strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(cv.src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
			return
		case *ast.String:
			b.Write(t.Value)
			return
		case *ast.CodeSpan:
			for c := t.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

// inlines are the runs n's children make.
func (cv *converter) inlines(n ast.Node) []*surfui.Node {
	var out []*surfui.Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, isBox := c.(*east.TaskCheckBox); isBox {
			continue // the item's
		}
		out = append(out, cv.inline(c)...)
	}
	return out
}

func (cv *converter) inline(n ast.Node) []*surfui.Node {
	kids := func() []*surfui.Node { return cv.inlines(n) }
	if o := cv.override(n, kids); o != nil {
		return []*surfui.Node{o}
	}
	switch t := n.(type) {
	case *ast.Text:
		out := []*surfui.Node{surfui.Str(string(t.Segment.Value(cv.src)))}
		switch {
		case t.HardLineBreak():
			out = append(out, surfui.Br())
		case t.SoftLineBreak() && cv.c.hard:
			out = append(out, surfui.Br())
		case t.SoftLineBreak():
			out = append(out, surfui.Str(" "))
		}
		return out
	case *ast.String:
		return []*surfui.Node{surfui.Str(string(t.Value))}
	case *ast.Emphasis:
		if t.Level >= 2 {
			return []*surfui.Node{surfui.Strong(kids()...)}
		}
		return []*surfui.Node{surfui.Em(kids()...)}
	case *east.Strikethrough:
		return []*surfui.Node{surfui.Del(kids()...)}
	case *ast.CodeSpan:
		return []*surfui.Node{surfui.Code(surfui.Str(strings.ReplaceAll(cv.plain(t), "\n", " ")))}
	case *ast.Link:
		return cv.link(string(t.Destination), kids())
	case *ast.AutoLink:
		label := string(t.Label(cv.src))
		url := string(t.URL(cv.src))
		return cv.link(url, []*surfui.Node{surfui.Str(label)})
	case *ast.Image:
		src := string(t.Destination)
		if !safeURL(src) {
			src = ""
		}
		return []*surfui.Node{surfui.Img(src, cv.plain(t))}
	case *ast.RawHTML:
		// Raw HTML is text.
		var b strings.Builder
		for i := 0; i < t.Segments.Len(); i++ {
			s := t.Segments.At(i)
			b.Write(s.Value(cv.src))
		}
		return []*surfui.Node{surfui.Str(b.String())}
	}
	return cv.inlines(n)
}

// link is a link to dest of kids: external where dest has a scheme. A
// destination that is a scheme of no use (javascript:, data:) is no link:
// its text alone.
func (cv *converter) link(dest string, kids []*surfui.Node) []*surfui.Node {
	if !safeURL(dest) {
		return kids
	}
	l := surfui.Link(dest, kids...)
	if hasScheme(dest) {
		l.External()
	}
	return []*surfui.Node{l}
}

// hasScheme reports whether u starts with a URL scheme and a colon.
func hasScheme(u string) bool {
	i := strings.IndexByte(u, ':')
	if i <= 0 {
		return false
	}
	for j := 0; j < i; j++ {
		c := u[j]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || j > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	return true
}

// safeURL reports whether u may be a link's or an image's address: a
// relative one, a fragment, or one of http, https, mailto and cid (a
// resource the program sent). Raw HTML never comes in, and nor does a
// script by an address.
func safeURL(u string) bool {
	if !hasScheme(u) {
		return true
	}
	switch strings.ToLower(u[:strings.IndexByte(u, ':')]) {
	case "http", "https", "mailto", "cid":
		return true
	}
	return false
}
