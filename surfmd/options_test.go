package surfmd_test

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"neuroplast.io/hotty-surf/surfmd"
	"neuroplast.io/hotty-surf/surfui"
)

func html(n *surfui.Node) string {
	surfui.Resolve(surfui.Surface(n), "r")
	return surfui.HTML(n)
}

// A newline in a paragraph is a space, or with HardWraps a break; a line
// ending in two spaces is a break either way.
func TestHardWraps(t *testing.T) {
	src := []byte("one\ntwo  \nthree\n")
	plain, hard := html(surfmd.Convert(src)), html(surfmd.Convert(src, surfmd.HardWraps()))
	if strings.Count(plain, "<br") != 1 || !strings.Contains(plain, "one two<br") {
		t.Errorf("plain: %s", plain)
	}
	if strings.Count(hard, "<br") != 2 || strings.Contains(hard, "one two") {
		t.Errorf("hard wraps: %s", hard)
	}
}

// Compact is the prose's, and Anchors names the headings.
func TestCompactAndAnchors(t *testing.T) {
	src := []byte("# A title\n\ntext\n")
	if h := html(surfmd.Convert(src)); strings.Contains(h, "u-prose-compact") || strings.Contains(h, `id="a-title"`) {
		t.Errorf("by default: %s", h)
	}
	h := html(surfmd.Convert(src, surfmd.Compact(), surfmd.Anchors(func(s string) string { return strings.ReplaceAll(strings.ToLower(s), " ", "-") })))
	if !strings.Contains(h, "u-prose-compact") || !strings.Contains(h, `<h1 id="a-title"`) {
		t.Errorf("compact and anchored: %s", h)
	}
}

// Raw HTML is text, a link to a script is no link, and an image with one
// has no source: nothing a document says can inject markup or a script.
func TestNothingInjected(t *testing.T) {
	src := []byte("<script>alert(1)</script>\n\nA <b onclick=x>b</b>, [x](javascript:alert(1)), [y](JaVaScRiPt:1), [z](data:text/html;base64,AAA) and ![i](javascript:1).\n\n[ok](/docs), [web](https://example.com), [mail](mailto:a@b.c), [frag](#f).\n")
	h := html(surfmd.Convert(src))
	for _, bad := range []string{"<script", "<b ", "javascript", "JaVaScRiPt", "data:text"} {
		if strings.Contains(h, bad) {
			t.Errorf("%q in %s", bad, h)
		}
	}
	for _, want := range []string{`href="/docs"`, `href="https://example.com"`, `href="mailto:a@b.c"`, `href="#f"`, "&lt;script&gt;"} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in %s", want, h)
		}
	}
	if strings.Contains(h, `href="/docs" target`) || !strings.Contains(h, `href="https://example.com" target="_blank"`) {
		t.Errorf("a link with a scheme is external, a path is not: %s", h)
	}
}

// Node has a function make a kind's nodes, with the source to read; nil
// leaves the node to the default.
func TestNodeOverride(t *testing.T) {
	src := []byte("```mermaid\ngraph TD\n```\n\n```go\nx := 1\n```\n")
	n := surfmd.Convert(src, surfmd.Node(ast.KindFencedCodeBlock, func(n ast.Node, src []byte, _ []*surfui.Node) *surfui.Node {
		f := n.(*ast.FencedCodeBlock)
		if string(f.Language(src)) != "mermaid" {
			return nil
		}
		return surfui.Text("diagram: " + strings.TrimSpace(string(f.Lines().Value(src))))
	}))
	h := html(n)
	if !strings.Contains(h, "diagram: graph TD") || !strings.Contains(h, "x := ") || strings.Contains(h, "mermaid") {
		t.Errorf("override: %s", h)
	}
}

// ConvertAST takes a document the caller parsed.
func TestConvertAST(t *testing.T) {
	src := []byte("*x*\n")
	doc := goldmark.New().Parser().Parse(text.NewReader(src))
	if h := html(surfmd.ConvertAST(doc, src)); !strings.Contains(h, `<em`) || !strings.Contains(h, "u-prose") {
		t.Errorf("%s", h)
	}
}

// A fenced block is surfui's code block: Highlight's spans where it gives
// them, the block's own lexer for go and surf where it doesn't, plain
// otherwise; the language is the info string's first word. In compact
// prose a block wraps.
func TestHighlight(t *testing.T) {
	src := []byte("```go title=x\nx := 1\n```\n\n```text\nplain\n```\n\n```\nnone\n```\n")
	var langs []string
	h := surfmd.Highlight(func(lang, code string) []surfui.Span {
		langs = append(langs, lang)
		if lang == "text" {
			return []surfui.Span{{Text: code, Role: "string"}}
		}
		return nil
	})
	out := html(surfmd.Convert(src, h, surfmd.Compact()))
	if strings.Join(langs, ",") != "go,text," {
		t.Errorf("languages asked: %q", langs)
	}
	if strings.Count(out, "u-codeblock") != 3 || !strings.Contains(out, "u-syn-string") || !strings.Contains(out, "u-wrap") {
		t.Errorf("code blocks: %s", out)
	}
	if !strings.Contains(html(surfmd.Convert([]byte("```go\nfunc f() {}\n```\n"))), "u-syn-keyword") {
		t.Error("the block's own lexer isn't used for go")
	}
}
