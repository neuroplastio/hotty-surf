package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestCodeSource(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.CodeBlock("package demo\n\ntype App struct {\n\tCount int // items\n}\nconst count = 3\n").Language("go"))
	}}
	s := surfrun.NewSurface(m, nil)
	r := show(m)
	s.Show()
	goldenText(t, "code.html", surfui.HTML(s.Tree())+"\n")
	golden(t, "code", r.Draw(32, 9))

	// A long source line must not paint over its neighbour, even at a
	// token boundary. A blank source line still occupies one cell row.
	m.render = func() *surfui.Node {
		return surfui.Surface(surfui.Row(
			surfui.CodeBlock("0123456789abcdef\n\nend").Width(8),
			surfui.Text("next").Width(4),
		).Top())
	}
	r = show(m)
	got := r.Draw(20, 5).Text()
	if !strings.Contains(got, "012345 next") || !strings.Contains(got, "end") || strings.Contains(got, "abcdef") {
		t.Fatalf("code failed to clip within its own box:\n%s", got)
	}
}

// A caller's spans draw in their roles, in the cells as on a host.
func TestCodeSpans(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.CodeSpans([]surfui.Span{
			{Text: "func", Role: "keyword"}, {Text: " main() ", Role: "plain"},
			{Text: "{ // go\n", Role: "comment"}, {Text: "\treturn ", Role: "keyword"},
			{Text: "\"a\"", Role: "string"}, {Text: " + ", Role: ""}, {Text: "1", Role: "number"},
			{Text: "\n}", Role: "plain"},
		}))
	}}
	s := surfrun.NewSurface(m, nil)
	r := show(m)
	s.Show()
	goldenText(t, "code-spans.html", surfui.HTML(s.Tree())+"\n")
	golden(t, "code-spans", r.Draw(28, 5))
}

// Wrap breaks a line at a space, the rest hanging under the line's own
// indent; numbers are a muted gutter, and a wrapped line's number is on
// its first row only.
func TestCodeWrap(t *testing.T) {
	src := "func f() {\n\tprintln(\"aaaa bbbb cccc dddd eeee ffff\")\n\n\tx := 1\n}\n"
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.CodeBlock(src).Language("go").Wrap())
	}}
	r := show(m)
	got := r.Draw(24, 8)
	golden(t, "code-wrap", got)
	text := got.Text()
	want := " func f() {\n     println(\"aaaa\n     bbbb cccc dddd\n     eeee ffff\")"
	if !strings.Contains(text, want) {
		t.Errorf("wrap:\n%s\nwant\n%s", text, want)
	}

	m.render = func() *surfui.Node {
		return surfui.Surface(surfui.CodeBlock(src).Language("go").Wrap().Numbers(true).From(9))
	}
	golden(t, "code-wrap-numbers", show(m).Draw(24, 9))

	// Without wrap a long line is cut at the box's edge.
	m.render = func() *surfui.Node {
		return surfui.Surface(surfui.CodeBlock(src).Language("go").Numbers(true))
	}
	golden(t, "code-numbers", show(m).Draw(24, 6))
}

// Every space of a wrapped line is kept, and a word wider than the box
// breaks where the box ends, as overflow-wrap: anywhere does.
func TestCodeWrapKeepsSpaces(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.CodeBlock("a  b   c\n  0123456789abcdef\n").Wrap())
	}}
	text := show(m).Draw(11, 5).Text()
	for _, want := range []string{"a  b   c", "  012345", "  6789ab", "  cdef"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q:\n%s", want, text)
		}
	}
}
