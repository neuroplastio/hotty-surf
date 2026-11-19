package cells

// What the Screen's tests share: a Screen's cells for markup, the kit's
// stories as markup, and random texts and attributes on a tree.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

// sameCells reports where got isn't want, a few cells of it.
func sameCells(t testing.TB, what string, got, want [][]Cell) bool {
	t.Helper()
	bad := 0
	for y := range want {
		for x := range want[y] {
			if y >= len(got) || x >= len(got[y]) || got[y][x] != want[y][x] {
				if bad < 3 {
					var g Cell
					if y < len(got) && x < len(got[y]) {
						g = got[y][x]
					}
					t.Errorf("%s: %d,%d = %+v, want %+v", what, x, y, g, want[y][x])
				}
				bad++
			}
		}
	}
	return bad == 0
}

// storyMarkup is a story's first document.
func storyMarkup(t testing.TB, st stories.Story) string {
	t.Helper()
	s := surf.New("story", st.View(), ui.Stylesheet())
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// mutate gives body's tree random texts and attributes, of the same shape
// as a delta can give it: texts between elements may go empty or come.
// With bySet, only what Set reaches changes: no id, and texts only of
// elements that hold nothing else.
func mutate(r *rand.Rand, body *html.Node, bySet bool) {
	texts := []string{"", " ", "x", "Machines", "a longer label that wraps", "日本語", "👍🏽 ok", "é", "tab\tand  spaces", "  "}
	colours := []string{"text-primary", "text-destructive", "bg-muted", "bg-(--pill-hover)", "text-(--ok)", "dark:text-primary", "text-muted-foreground"}
	others := []string{"font-bold", "flex", "grow", "truncate", "hidden", "dark", "p-1", "gap-2", "flex-col", "shrink-0", "whitespace-pre"}
	attrs := []struct{ key, val string }{
		{"data-variant", "outline"}, {"data-variant", "destructive"}, {"aria-hidden", "true"},
		{"aria-selected", "true"}, {"open", ""}, {"data-side", "inline-end"}, {"data-align", "block-end"},
		{"data-orientation", "horizontal"}, {"disabled", ""}, {"aria-checked", "true"}, {"value", "42"},
	}
	walk(body, func(n *html.Node) {
		if n == body {
			return
		}
		switch n.Type {
		case html.TextNode:
			if bySet && (n.Parent.FirstChild != n || n.NextSibling != nil) {
				return
			}
			if r.IntN(3) == 0 && textKept(n.Parent) {
				n.Data = texts[r.IntN(len(texts))]
			}
		case html.ElementNode:
			if r.IntN(4) != 0 {
				return
			}
			switch r.IntN(4) {
			case 0: // colours in its class
				cls := strings.Fields(attr(n, "class"))
				cls = slices.DeleteFunc(cls, func(c string) bool { return colourToken(c) && r.IntN(2) == 0 })
				cls = append(cls, colours[r.IntN(len(colours))])
				set(n, "class", strings.Join(cls, " "))
			case 1: // any class
				cls := strings.Fields(attr(n, "class"))
				if len(cls) > 0 && r.IntN(2) == 0 {
					cls = slices.Delete(cls, 0, 1)
				}
				cls = append(cls, others[r.IntN(len(others))])
				set(n, "class", strings.Join(cls, " "))
			case 2:
				a := attrs[r.IntN(len(attrs))]
				set(n, a.key, a.val)
			case 3:
				if len(n.Attr) > 0 {
					if k := n.Attr[r.IntN(len(n.Attr))].Key; k != "id" || !bySet {
						unset(n, k)
					}
				}
			}
			// A text that comes where an element had none.
			if n.FirstChild == nil && !voids[n.DataAtom] && textKept(n) && n.Namespace == "" && r.IntN(4) == 0 && !skipped[n.DataAtom] {
				n.AppendChild(&html.Node{Type: html.TextNode, Data: texts[r.IntN(len(texts))]})
			}
		}
	})
	// Drop empty texts, as the parser gives none.
	var empties []*html.Node
	walk(body, func(n *html.Node) {
		if n.Type == html.TextNode && n.Data == "" {
			empties = append(empties, n)
		}
	})
	for _, n := range empties {
		n.Parent.RemoveChild(n)
	}
}

// render is body's children as markup.
func render(t testing.TB, body *html.Node) string {
	t.Helper()
	var b strings.Builder
	for k := body.FirstChild; k != nil; k = k.NextSibling {
		if err := html.Render(&b, k); err != nil {
			t.Fatal(err)
		}
	}
	return b.String()
}

func parseBody(t testing.TB, markup string) *html.Node {
	t.Helper()
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	ns, err := html.ParseFragment(strings.NewReader(markup), body)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range ns {
		body.AppendChild(n)
	}
	return body
}

// textKept says whether the parser keeps text in n where markup puts it:
// not in a table's or a select's structure, which moves it.
func textKept(n *html.Node) bool {
	if n.Namespace != "" {
		return true
	}
	switch n.DataAtom {
	case atom.Table, atom.Tbody, atom.Thead, atom.Tfoot, atom.Tr, atom.Colgroup,
		atom.Frameset, atom.Html, atom.Head, atom.Template:
		return false
	}
	return true
}

// voids are the elements with no end tag.
var voids = map[atom.Atom]bool{
	atom.Area: true, atom.Base: true, atom.Br: true, atom.Col: true, atom.Embed: true,
	atom.Hr: true, atom.Img: true, atom.Input: true, atom.Link: true, atom.Meta: true,
	atom.Param: true, atom.Source: true, atom.Track: true, atom.Wbr: true,
	atom.Keygen: true,
}

// A colour class that a rule reads by name (besides coloursOf) changes
// more than a colour: namedColours must hold every one the package's
// source names, and no rule reads text- or bg- classes by their prefix.
func TestColourTokensNamed(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fs := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fs, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for c := range strings.FieldsSeq(s) {
				if colourToken(c) {
					t.Errorf("%s: a rule names colour class %q: add it to namedColours", fs.Position(lit.Pos()), c)
				}
			}
			if (s == "text-" || s == "bg-") && f != "colour.go" {
				t.Errorf("%s: a rule reads %q classes by prefix", fs.Position(lit.Pos()), s)
			}
			return true
		})
	}
	if colourToken("text-muted-foreground") || !colourToken("text-(--pill)") || !colourToken("dark:bg-muted") || colourToken("font-bold") {
		t.Error("colourToken")
	}
}

// sameSlots says whether a and b have one shape:
// the same elements and comments in the same places, the texts between
// them slots that either may leave empty.
func sameSlots(a, b *html.Node) bool {
	if a.Type != b.Type {
		return false
	}
	if a.Type == html.ElementNode && (a.DataAtom != b.DataAtom || a.Data != b.Data || a.Namespace != b.Namespace) {
		return false
	}
	ka, kb := a.FirstChild, b.FirstChild
	for {
		for ka != nil && ka.Type == html.TextNode {
			ka = ka.NextSibling
		}
		for kb != nil && kb.Type == html.TextNode {
			kb = kb.NextSibling
		}
		if ka == nil || kb == nil {
			return ka == nil && kb == nil
		}
		if !sameSlots(ka, kb) {
			return false
		}
		ka, kb = ka.NextSibling, kb.NextSibling
	}
}
