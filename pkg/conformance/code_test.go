package conformance

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestCodeModes(t *testing.T) {
	for _, lang := range []string{"plain", "go", "surf"} {
		state := &CodeExample{Source: "<script>\n\n\t日本語 & text\n", Language: lang}
		var wantHTML, wantCells string
		for _, mode := range []func(*CodeExample) surflive.Component{surfInterpCodeExample, surfCompiledCodeExample} {
			view := surflive.New(mode(state))
			s := surfrun.NewSurface(view, nil)
			r := cells.New(s.Cells())
			r.Show()
			host, plain := surfui.HTML(s.Tree()), r.Draw(40, 8).Text()
			if strings.Contains(host, "<script>") || !strings.Contains(host, "&amp;") {
				t.Fatalf("%s: source became markup: %s", lang, host)
			}
			if !strings.Contains(plain, "     日本語 & text") {
				t.Fatalf("%s: indentation lost:\n%s", lang, plain)
			}
			if wantHTML != "" && (host != wantHTML || plain != wantCells) {
				t.Fatalf("%s: template modes disagree", lang)
			}
			wantHTML, wantCells = host, plain
		}
	}
}

func TestCodeSpansModes(t *testing.T) {
	spans := []surfui.Span{
		{Text: "func", Role: "keyword"}, {Text: " run(", Role: "plain"}, {Text: "\"a b c d e f g h\"", Role: "string"},
		{Text: ")\n\t// <b>done</b>\n", Role: "comment"}, {Text: "\treturn ", Role: "keyword"}, {Text: "7", Role: "number"},
	}
	for _, wrap := range []bool{false, true} {
		state := &CodeSpansExample{Spans: spans, From: 99, Wrap: wrap}
		var wantHTML, wantCells string
		for _, mode := range []func(*CodeSpansExample) surflive.Component{surfInterpCodeSpansExample, surfCompiledCodeSpansExample} {
			view := surflive.New(mode(state))
			s := surfrun.NewSurface(view, nil)
			r := cells.New(s.Cells())
			r.Show()
			host, plain := surfui.HTML(s.Tree()), r.Draw(30, 8).Text()
			for _, want := range []string{"u-syn-keyword", "u-syn-string", "u-syn-comment", "u-syn-number", "&lt;b&gt;", "u-code-num"} {
				if !strings.Contains(host, want) {
					t.Fatalf("wrap %v: no %q in %s", wrap, want, host)
				}
			}
			if strings.Contains(host, "<b>") || strings.Contains(host, "u-code-flow") != wrap {
				t.Fatalf("wrap %v: %s", wrap, host)
			}
			for _, want := range []string{" 99 func run(", "100     // <b>done</b>", "101     return 7"} {
				if !strings.Contains(plain, want) && !wrap {
					t.Fatalf("wrap %v: no %q in\n%s", wrap, want, plain)
				}
			}
			if wantHTML != "" && (host != wantHTML || plain != wantCells) {
				t.Fatalf("wrap %v: template modes disagree:\n%s\n%s", wrap, plain, wantCells)
			}
			wantHTML, wantCells = host, plain
		}
	}
}
