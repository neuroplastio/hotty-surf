package surfui

import (
	"slices"
	"strings"
	"testing"
)

func TestCodeLines(t *testing.T) {
	n := CodeBlock("α\t界\tend\r\n\r<script>&\n")
	want := []string{"α    界    end", "", "<script>&", ""}
	for _, language := range []string{"plain", "go", "surf", "plain"} {
		n.Language(language)
		var got []string
		for _, line := range Kids(n) {
			got = append(got, TextOf(line))
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: lines %q, want %q", language, got, want)
		}
		if doc := HTML(n); strings.Contains(doc, "<script>") || !strings.Contains(doc, "&amp;") {
			t.Errorf("source became markup: %s", doc)
		}
	}
}

// Token boundaries must never change source, including unfinished source
// and template punctuation that would be active if emitted as HTML.
func TestCodeTokensPreserveSource(t *testing.T) {
	for _, language := range []string{"plain", "go", "surf"} {
		for _, source := range []string{
			"", "\t\n\r\n  ", "日本語 👩‍💻", "<script>&</script>",
			"package demo\n\ntype App struct {\n\tCount int // items\n}\n",
			"var s = `first\n\tsecond`\nx := \"a\\\"b\" /* end */\n",
			"// unfinished", "/* unfinished", "\"unfinished\\", "`unfinished",
			"<surface>\n  <text :if=\"count > 0\">{{ count }}</text>\n</surface>\n",
			"<!-- <text/> -->\n<field :value=\"'&'\"/>",
		} {
			var got strings.Builder
			for _, tok := range codeTokens(source, language) {
				if tok.text == "" {
					t.Fatal("empty token")
				}
				got.WriteString(tok.text)
			}
			if got.String() != source {
				t.Errorf("%s: got %q, want %q", language, got.String(), source)
			}
		}
	}
}

func TestCodeTokenKinds(t *testing.T) {
	for _, c := range []struct {
		language, source, text, kind string
	}{
		{"go", "type App struct { N int }", "type", "keyword"},
		{"go", "type App struct { N int }", "int", "type"},
		{"go", "var s = \"<tag>\"", "\"<tag>\"", "string"},
		{"go", "// return 42", "// return 42", "comment"},
		{"go", "x := 0x2a", "0x2a", "number"},
		{"go", "x := 1+2", "1", "number"},
		{"surf", "<text :if=\"ready\">hi</text>", "text", "tag"},
		{"surf", "<text :if=\"ready\">hi</text>", ":if", "attribute"},
		{"surf", "<text :if=\"ready\">hi</text>", "\"ready\"", "string"},
		{"surf", "<!-- <text> -->", "<!-- <text> -->", "comment"},
		{"surf", "<text>{{ true }}</text>", "true", "keyword"},
	} {
		found := false
		for _, tok := range codeTokens(c.source, c.language) {
			if tok.text == c.text && tok.kind == c.kind {
				found = true
			}
		}
		if !found {
			t.Errorf("%s %q: missing %s %q: %+v", c.language, c.source,
				c.kind, c.text, codeTokens(c.source, c.language))
		}
	}
}

func lineTexts(n *Node) []string {
	var got []string
	for _, line := range Kids(n) {
		got = append(got, TextOf(line))
	}
	return got
}

// A tab is four spaces, wherever it stands: the same in the highlighter's
// source and in a caller's spans.
func TestCodeTabsAreFourSpaces(t *testing.T) {
	got := lineTexts(CodeBlock("\tx\ty").Language("go"))
	if want := []string{"    x    y"}; !slices.Equal(got, want) {
		t.Errorf("language: %q, want %q", got, want)
	}
	got = lineTexts(CodeSpans([]Span{{"\tx", "keyword"}, {"\ty", "plain"}}))
	if want := []string{"    x    y"}; !slices.Equal(got, want) {
		t.Errorf("spans: %q, want %q", got, want)
	}
}

// Spans carry the caller's own highlighting: lines split at a newline
// inside a span, every role is a class both renderers read, and a role
// nobody knows is plain, never markup.
func TestCodeSpans(t *testing.T) {
	n := CodeSpans([]Span{
		{"func", "keyword"}, {" main() {\n", "plain"},
		{"\t// <b>", "comment"}, {"x", "nonsense"}, {"\n}", ""},
	})
	if got, want := lineTexts(n), []string{"func main() {", "    // <b>x", "}"}; !slices.Equal(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
	doc := HTML(n)
	for _, want := range []string{"u-syn-keyword", "u-syn-comment", "&lt;b&gt;"} {
		if !strings.Contains(doc, want) {
			t.Errorf("no %q in %s", want, doc)
		}
	}
	if strings.Contains(doc, "nonsense") || strings.Contains(doc, "<b>") {
		t.Errorf("a role or text became markup: %s", doc)
	}
	if err := CheckSpans([]Span{{"a", "keyword"}, {"b", "loud"}}); err == nil || !strings.Contains(err.Error(), "loud") {
		t.Errorf("CheckSpans of an unknown role: %v", err)
	}
	if err := CheckSpans([]Span{{"a", "keyword"}, {"b", ""}}); err != nil {
		t.Errorf("CheckSpans of the empty role (plain): %v", err)
	}
}

// Numbers: a muted gutter as wide as the last number, counting from
// From. The modifiers apply in any order.
func TestCodeNumbers(t *testing.T) {
	n := CodeBlock("a\nb\nc").Language("go").From(8).Numbers(true)
	got := lineTexts(n)
	if want := []string{" 8 a", " 9 b", "10 c"}; !slices.Equal(got, want) {
		t.Errorf("numbers %q, want %q", got, want)
	}
	m := CodeBlock("a\nb\nc").From(8).Numbers(true).Language("go")
	if got := lineTexts(m); !slices.Equal(got, lineTexts(n)) {
		t.Errorf("another order: %q", got)
	}
	if !strings.Contains(HTML(n), "u-code-num") {
		t.Error("the gutter has no class")
	}
	if got := lineTexts(CodeBlock("a\nb").Numbers(false)); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("no numbers: %q", got)
	}
}

// Wrap puts each source line in a paragraph that hangs under the line's
// own indent; the indent and the text are the same bytes.
func TestCodeWrapKeepsSource(t *testing.T) {
	n := CodeBlock("a\n\n    b  c").Wrap()
	doc := HTML(n)
	if !strings.Contains(doc, "u-code-flow") {
		t.Fatalf("no flow: %s", doc)
	}
	if got, want := lineTexts(n), []string{"a", "", "    b  c"}; !slices.Equal(got, want) {
		t.Errorf("wrap lines %q, want %q", got, want)
	}
}
