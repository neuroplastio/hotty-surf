package jinja

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// TestJinja renders every case in testdata/cases.json with the translated
// component and compares it with what jinja2 rendered from Basecoat's macro
// (testdata/<name>.html, written by gen/jinja2templ.py --testdata),
// after normalising both: parsed as HTML, attributes sorted, whitespace
// collapsed, whitespace-only text dropped, random ids (name-NNNNNN) equal.
func TestJinja(t *testing.T) {
	var cases []struct {
		Name, Macro, Caller string
		Args                map[string]any
		SourceBug           string `json:"source_bug"`
	}
	b, err := os.ReadFile("testdata/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			want, err := os.ReadFile("testdata/" + c.Name + ".html")
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			var caller templ.Component
			if c.Caller != "" {
				caller = templ.Raw(c.Caller)
				ctx = templ.WithChildren(ctx, caller)
			}
			comp := Call(c.Macro, c.Args, caller)
			if comp == nil {
				t.Fatalf("no component for %s", c.Macro)
			}
			var got strings.Builder
			if err := comp.Render(ctx, &got); err != nil {
				t.Fatal(err)
			}
			w, g := normal(t, string(want), c.SourceBug != ""), normal(t, got.String(), c.SourceBug != "")
			if c.SourceBug != "" {
				t.Logf("source bug (%s); compared without the hidden input's attributes; templ renders:\n%s", c.SourceBug, hiddenInput(got.String()))
			}
			if w != g {
				i := 0
				for i < len(w) && i < len(g) && w[i] == g[i] {
					i++
				}
				lo := max(0, i-120)
				t.Errorf("differs at %d\njinja: %s\ntempl: %s", i, w[lo:min(len(w), i+200)], g[lo:min(len(g), i+200)])
			}
		})
	}
}

var (
	randID = regexp.MustCompile(`\b([a-z-]+)-\d{6}\b`)
	spaces = regexp.MustCompile(`\s+`)
)

func clean(s string) string {
	return randID.ReplaceAllString(strings.TrimSpace(spaces.ReplaceAllString(s, " ")), "$1-RAND")
}

func hiddenInput(s string) string {
	i := strings.Index(s, `<input type="hidden"`)
	if i < 0 {
		return ""
	}
	return s[i : i+strings.Index(s[i:], ">")+1]
}

// normal is s parsed and printed canonically; dropHidden drops the
// attributes of <input type=hidden> (for a macro whose own output there is
// malformed).
func normal(t *testing.T, s string, dropHidden bool) string {
	ns, err := html.ParseFragment(strings.NewReader(s), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			if s := clean(n.Data); s != "" {
				b.WriteString(s)
				b.WriteByte('\n')
			}
		case html.ElementNode:
			attrs := make([]string, 0, len(n.Attr))
			for _, a := range n.Attr {
				attrs = append(attrs, a.Key+`="`+clean(a.Val)+`"`)
			}
			if dropHidden && n.Data == "input" && len(attrs) > 0 {
				attrs = []string{"(hidden input: attributes dropped)"}
			}
			sort.Strings(attrs)
			b.WriteString("<" + n.Data)
			for _, a := range attrs {
				b.WriteString(" " + a)
			}
			b.WriteString(">\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			b.WriteString("</" + n.Data + ">\n")
		}
	}
	for _, n := range ns {
		walk(n)
	}
	return b.String()
}

// TestNormalDiscriminates checks the comparison isn't vacuous: every pair of
// distinct expected outputs stays distinct after normalising, and a one-
// attribute change is seen.
func TestNormalDiscriminates(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.html")
	seen := map[string]string{}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		n := normal(t, string(b), false)
		if g, ok := seen[n]; ok {
			t.Errorf("%s and %s normalise the same", f, g)
		}
		seen[n] = f
	}
	a := normal(t, `<div id="x" class="a b"><p>t</p></div>`, false)
	for _, b := range []string{`<div id="x" class="a c"><p>t</p></div>`, `<div id="x" class="a b" hidden><p>t</p></div>`, `<div id="x" class="a b"><p>u</p></div>`} {
		if normal(t, b, false) == a {
			t.Errorf("%s normalises like the original", b)
		}
	}
	if normal(t, `<div  class=" a  b " id="x">`+"\n  <p>t</p>\n</div>", false) != a {
		t.Error("whitespace and attribute order should not matter")
	}
}
