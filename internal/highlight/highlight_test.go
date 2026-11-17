package code

import (
	"strings"
	"testing"
)

func TestHighlight(t *testing.T) {
	for _, lang := range []string{"go", "templ", "html"} {
		code := map[string]string{
			"go":    "func f() {\n\treturn \"x\" // c\n}\n",
			"templ": "templ f(p *P) {\n\t<div class=\"a\">{ p.Name }</div>\n}\n",
			"html":  "<div class=\"a\">\n  <span>x</span>\n</div>\n",
		}[lang]
		lines := Highlight(code, lang)
		if len(lines) != 3 {
			t.Fatalf("%s: %d lines", lang, len(lines))
		}
		var b strings.Builder
		kinds := map[string]bool{}
		for i, l := range lines {
			if i > 0 {
				b.WriteByte('\n')
			}
			for _, tok := range l {
				if strings.TrimSpace(tok.Text) == "" {
					t.Errorf("%s: a token of only spaces: %q", lang, tok.Text)
				}
				kinds[tok.Kind] = true
				b.WriteString(tok.Text)
			}
		}
		want := strings.ReplaceAll(strings.TrimSuffix(code, "\n"), "\t", "    ")
		if b.String() != want {
			t.Errorf("%s: the tokens say\n%s\nnot\n%s", lang, b.String(), want)
		}
		if len(kinds) < 3 {
			t.Errorf("%s: only kinds %v: is the lexer there?", lang, kinds)
		}
	}
}
