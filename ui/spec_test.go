package ui_test

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	x "golang.org/x/net/html"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

// TestSpec is the spec test of ui/coverage.tsv: in every story, each class
// is one the kit's stylesheet (Basecoat's CSS, built) defines, and each
// value of the attributes Basecoat styles by (data-variant, data-size, …)
// is one a selector of it names. A value Basecoat has no rule for would
// draw as nothing at all, which no other test sees.
func TestSpec(t *testing.T) {
	css := kitSelectors()
	for _, st := range stories.All() {
		t.Run(st.ID, func(t *testing.T) {
			doc, err := surf.New("story", st.View(), ui.Stylesheet()).Document()
			if err != nil {
				t.Fatal(err)
			}
			for _, bad := range css.check(doc) {
				t.Error(bad)
			}
		})
	}
}

// styledBy are the attributes Basecoat's CSS selects values of: a value
// one of its components writes is one a rule names.
var styledBy = []string{
	"data-variant", "data-size", "data-orientation", "data-align", "data-side",
	"data-invalid", "data-disabled", "aria-invalid",
}

type selectors struct {
	classes map[string]bool
	values  map[string]bool // attr=value, and attr alone for [attr]
}

var (
	kitOnce sync.Once
	kitSel  selectors
)

var (
	classRe = regexp.MustCompile(`\.((?:[A-Za-z_-]|\\.)(?:[\w-]|\\.)*)`)
	attrRe  = regexp.MustCompile(`\[([a-z-]+)(?:[~|^$*]?=(?:"([^"]*)"|'([^']*)'|([^\]\s]+)))?(?:\s+[is])?\]`)
)

// kitSelectors reads every class and attribute selector of the kit's
// stylesheet.
func kitSelectors() selectors {
	kitOnce.Do(func() {
		css := ui.Stylesheet()
		kitSel = selectors{classes: map[string]bool{}, values: map[string]bool{}}
		for _, m := range classRe.FindAllStringSubmatch(css, -1) {
			kitSel.classes[unescape(m[1])] = true
		}
		for _, m := range attrRe.FindAllStringSubmatch(css, -1) {
			kitSel.values[m[1]] = true
			if v := m[2] + m[3] + m[4]; v != "" {
				kitSel.values[m[1]+"="+v] = true
			}
		}
	})
	return kitSel
}

func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// check gives what doc's markup writes that the stylesheet doesn't know.
func (s selectors) check(doc string) []string {
	root, err := x.Parse(strings.NewReader(doc))
	if err != nil {
		return []string{err.Error()}
	}
	var bad []string
	var walk func(n *x.Node)
	walk = func(n *x.Node) {
		if n.Type == x.ElementNode && n.Data != "svg" && n.Data != "style" {
			for _, a := range n.Attr {
				switch {
				case a.Key == "class":
					for _, c := range strings.Fields(a.Val) {
						if !s.classes[c] && !strings.HasPrefix(c, "lucide") {
							bad = append(bad, "<"+n.Data+"> class "+c+": no rule has it")
						}
					}
				case slices.Contains(styledBy, a.Key):
					if !s.values[a.Key+"="+a.Val] {
						bad = append(bad, "<"+n.Data+"> "+a.Key+"="+a.Val+": no rule selects it")
					}
				}
			}
		}
		if n.Data == "svg" {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return bad
}

// The spec test sees a class and a value Basecoat has no rule for, and
// passes those it has.
func TestSpecBites(t *testing.T) {
	css := kitSelectors()
	bad := css.check(`<div class="btn input-group nope" data-variant="loud" data-align="inline-start" data-size="sm" aria-invalid="true"></div>`)
	if len(bad) != 2 || !strings.Contains(bad[0], "nope") || !strings.Contains(bad[1], "loud") {
		t.Fatalf("check gave %q, want nope and loud only", bad)
	}
}
