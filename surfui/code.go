package surfui

import (
	"fmt"
	"strconv"
	"strings"
)

// Span is a run of source text and its syntax role: what a caller's own
// highlighter (Chroma, a language server) hands CodeBlock, so that the
// framework tokenizes nothing for it. It is plain data, a template's
// :spans and a Lua table's {text, role} alike.
//
// The roles are the lexer's own: keyword, tag, string, number, comment,
// type, attribute and plain. The empty role is plain.
type Span struct {
	Text string
	Role string
}

// SpanRoles are the roles of a Span, the syntax roles of surf.css
// (--u-syn-*) and of the cells (cells.SynKeyword …).
var SpanRoles = []string{"keyword", "tag", "string", "number", "comment", "type", "attribute", "plain"}

func knownRole(role string) bool {
	if role == "" {
		return true
	}
	for _, r := range SpanRoles {
		if r == role {
			return true
		}
	}
	return false
}

// CheckSpans says whether every role of spans is one of SpanRoles (or
// empty): the error a template's :spans fails with. CodeBlock draws an
// unknown role plain, so that a caller's lexer never breaks a screen;
// a template is told.
func CheckSpans(spans []Span) error {
	for i, s := range spans {
		if !knownRole(s.Role) {
			return fmt.Errorf("span %d: role is keyword, tag, string, number, comment, type, attribute or plain, not %q", i, s.Role)
		}
	}
	return nil
}

// codeSpec is what a CodeBlock node is built from. Modifiers set it in any
// order, and each rebuilds the node's lines.
type codeSpec struct {
	source   string
	language string
	spans    []Span
	spanned  bool
	numbers  bool
	from     int
}

// CodeBlock is read-only source: lines of coloured text in the terminal's
// face, on the card ground. Language selects its lexer (plain, go or
// surf). Source is escaped text, never HTML, and each line keeps its
// whitespace; a tab is four spaces.
//
// A line wider than the box is cut at its edge and the block scrolls
// across; with Wrap it breaks at a space instead, the rest hanging
// under the line's own indent. Numbers puts a muted gutter of line
// numbers before the lines, counting from From (1 by default).
func CodeBlock(source string) *Node {
	n := Box().Card().Class("u-codeblock")
	n.code = &codeSpec{source: source, language: "plain", from: 1}
	return n.codeBuild()
}

// CodeSpans is CodeBlock of a caller's own spans: source already
// highlighted, which the framework doesn't tokenize. Language does
// nothing on it. The spans are copied.
func CodeSpans(spans []Span) *Node {
	n := Box().Card().Class("u-codeblock")
	n.code = &codeSpec{spans: append([]Span(nil), spans...), spanned: true, from: 1}
	return n.codeBuild()
}

// Language highlights a CodeBlock node with language's lexer. Unknown names
// use plain text; templates reject them at load or render. It does
// nothing on another widget, or on one built from spans.
func (n *Node) Language(language string) *Node {
	if n.code == nil {
		return n
	}
	n.code.language = language
	return n.codeBuild()
}

// Spans sets a CodeBlock node's spans, which the lexer then doesn't read.
func (n *Node) Spans(spans []Span) *Node {
	if n.code == nil {
		return n
	}
	n.code.spans, n.code.spanned = append([]Span(nil), spans...), true
	return n.codeBuild()
}

// Numbers shows or hides a CodeBlock node's line numbers.
func (n *Node) Numbers(on bool) *Node {
	if n.code == nil {
		return n
	}
	n.code.numbers = on
	return n.codeBuild()
}

// From is the number of a CodeBlock node's first line.
func (n *Node) From(first int) *Node {
	if n.code == nil {
		return n
	}
	n.code.from = first
	return n.codeBuild()
}

// codeBuild makes n's lines of its spec. The source is retained, so
// that changing a modifier in place never tokenizes rendered markup.
func (n *Node) codeBuild() *Node {
	c := n.code
	n.kids = nil
	wrap := hasClass(n, "u-wrap")
	lines := codeLines(c)
	last := len(lines) - 1
	// A source that ends in a newline has a last line that isn't one: it
	// is drawn, and has no number.
	numbered := len(lines)
	if len(lines) > 1 && len(lines[last]) == 0 {
		numbered--
	}
	digits := 0
	if c.numbers {
		digits = max(len(strconv.Itoa(c.from)), len(strconv.Itoa(c.from+numbered-1)))
	}
	for i, line := range lines {
		row := Row().Class("u-code-line")
		if c.numbers {
			num := strings.Repeat(" ", digits+1)
			if i < numbered {
				num = fmt.Sprintf("%*d ", digits, c.from+i)
			}
			row.Add(Text(num).Keep().Muted().Class("u-code-num"))
		}
		empty := len(line) == 0
		if !wrap {
			row.Height(1)
			for _, tok := range line {
				row.Add(codeRun(tok, true))
			}
			n.Add(row)
			continue
		}
		if empty {
			n.Add(row.Height(1))
			continue
		}
		// The paragraph hangs under the line's indent: the indent is a
		// text of its own, and the words wrap beside it.
		row.Top()
		indent := 0
		for len(line) > 0 {
			t := strings.TrimLeft(line[0].text, " ")
			indent += len(line[0].text) - len(t)
			if t != "" {
				line[0].text = t
				break
			}
			line = line[1:]
		}
		if indent > 0 {
			row.Add(Text(strings.Repeat(" ", indent)).Keep())
		}
		flow := Line().Wrap().Grow().Class("u-code-flow")
		for _, tok := range line {
			flow.Add(codeRun(tok, false))
		}
		n.Add(row.Add(flow))
	}
	return n
}

func codeRun(tok codeToken, keep bool) *Node {
	run := Text(tok.text).Class("u-code-run", "u-syn-"+tok.kind)
	if keep {
		run.Keep()
	}
	return run
}

// codeLines are c's tokens by line, tabs and line ends as the lines
// keep them: no token holds a newline or is empty.
func codeLines(c *codeSpec) [][]codeToken {
	var toks []codeToken
	if c.spanned {
		for _, s := range c.spans {
			kind := s.Role
			if kind == "" || !knownRole(kind) {
				kind = "plain"
			}
			if text := codeSpace(s.Text); text != "" {
				toks = append(toks, codeToken{text, kind})
			}
		}
	} else {
		toks = codeTokens(codeSpace(c.source), c.language)
	}
	lines := [][]codeToken{nil}
	for _, tok := range toks {
		for i, part := range strings.Split(tok.text, "\n") {
			if i > 0 {
				lines = append(lines, nil)
			}
			if part != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], codeToken{part, tok.kind})
			}
		}
	}
	return lines
}

var codeSpacer = strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", "    ")

// codeSpace is source with its line ends as \n and its tabs four spaces.
func codeSpace(source string) string { return codeSpacer.Replace(source) }

func hasClass(n *Node, c string) bool {
	for _, x := range n.class {
		if x == c {
			return true
		}
	}
	return false
}
