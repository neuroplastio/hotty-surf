package cells_test

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/clipperhouse/displaywidth"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/cells"
)

// Random documents and acts: the Screen never panics, and its lines are
// always rows lines each cols cells wide, with no escape but SGR, and the
// caret inside.

var escRE = regexp.MustCompile("\x1b")

var (
	fTags    = []string{"div", "p", "span", "b", "button", "input", "textarea", "select", "label", "ul", "li", "table", "tr", "td", "th", "thead", "tbody", "details", "summary", "pre", "a", "br", "hr", "form", "dialog", "img", "h1", "code"}
	fClasses = []string{"flex", "flex-col", "grid", "gap-2", "gap-x-4", "p-4", "px-2", "py-4", "flex-1", "w-full", "justify-between",
		"justify-end", "justify-center", "hidden", "sr-only", "list-disc", "list-decimal", "whitespace-pre", "input", "btn", "font-bold", "md:flex"}
	fStyles = []string{"display:flex", "display:none", "padding:16px 8px", "gap:16px", "flex:1", "width:100%", "white-space:pre", "display:grid"}
	fTexts  = []string{"hello", "a long line of words that wraps", "漢字かな", "é", "👩‍👩‍👧", "\t", "\n", " ", "x‍y", "\x1b[31mred", "\u009b31m", "\r\n", "ab cd", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	fTypes  = []string{"text", "checkbox", "radio", "password", "submit", "button", "hidden", "range", "number"}
	fKeys   = []string{"tab", "shift+tab", "enter", "space", "a", "漢", "backspace", "delete", "arrow_left", "arrow_right", "arrow_up", "arrow_down",
		"home", "end", "page_up", "page_down", "shift+arrow_left", "ctrl+a", "alt+backspace", "shift+enter", "escape", "ctrl+z", "x", "ctrl+home"}
)

type gen struct {
	r   *rand.Rand
	ids int
}

func (g *gen) pick(s []string) string { return s[g.r.IntN(len(s))] }

func (g *gen) el(depth int) string {
	if depth > 4 || g.r.IntN(4) == 0 {
		return g.pick(fTexts)
	}
	tag := g.pick(fTags)
	var b strings.Builder
	g.ids++
	fmt.Fprintf(&b, `<%s id="e%d"`, tag, g.ids)
	if g.r.IntN(2) == 0 {
		fmt.Fprintf(&b, ` class="%s %s"`, g.pick(fClasses), g.pick(fClasses))
	}
	if g.r.IntN(4) == 0 {
		fmt.Fprintf(&b, ` style="%s"`, g.pick(fStyles))
	}
	switch tag {
	case "input":
		fmt.Fprintf(&b, ` type="%s" value="%s" name="n%d" data-on="input"`, g.pick(fTypes), strings.ReplaceAll(g.pick(fTexts), `"`, ""), g.r.IntN(3))
		if g.r.IntN(3) == 0 {
			fmt.Fprintf(&b, ` size="%d"`, g.r.IntN(5))
		}
		b.WriteString(">")
		return b.String()
	case "textarea":
		fmt.Fprintf(&b, ` rows="%d" cols="%d">%s</textarea>`, g.r.IntN(4), g.r.IntN(6), g.pick(fTexts))
		return b.String()
	case "select":
		b.WriteString(">")
		for i := range g.r.IntN(4) {
			fmt.Fprintf(&b, `<option value="v%d">%s</option>`, i, g.pick(fTexts))
		}
		b.WriteString("</select>")
		return b.String()
	case "img":
		fmt.Fprintf(&b, ` alt="%s">`, g.pick(fTexts))
		return b.String()
	case "br", "hr":
		b.WriteString(">")
		return b.String()
	case "details", "dialog":
		if g.r.IntN(2) == 0 {
			b.WriteString(" open")
		}
	case "a":
		b.WriteString(` href="#x"`)
	case "div", "span":
		if g.r.IntN(3) == 0 {
			b.WriteString(` tabindex="0" data-on="click"`)
		}
	}
	b.WriteString(">")
	for range g.r.IntN(5) {
		b.WriteString(g.el(depth + 1))
	}
	fmt.Fprintf(&b, "</%s>", tag)
	return b.String()
}

// whole says what is wrong with the Screen's lines, or "".
func whole(c *cells.Screen, cols, rows int) string {
	ls := c.Lines()
	if len(ls) != rows {
		return fmt.Sprintf("%d lines, want %d", len(ls), rows)
	}
	for i, l := range ls {
		plain := sgrRE.ReplaceAllString(l, "")
		if escRE.MatchString(plain) || strings.ContainsRune(plain, '\u009b') {
			return fmt.Sprintf("line %d holds an escape that isn't SGR: %q", i, l)
		}
		if w := displaywidth.String(plain); w != cols {
			return fmt.Sprintf("line %d is %d cells, want %d: %q", i, w, cols, plain)
		}
	}
	if col, row, ok := c.Cursor(); ok && (col < 0 || col >= cols || row < 0 || row >= rows) {
		return fmt.Sprintf("the caret is outside: %d,%d", col, row)
	}
	return ""
}

func TestRandomDocumentsDrawWhole(t *testing.T) {
	seeds := 3000
	if testing.Short() {
		seeds = 300
	}
	fails := 0
	for seed := range seeds {
		g := &gen{r: rand.New(rand.NewPCG(uint64(seed), 7))}
		cols, rows := 1+g.r.IntN(30), 1+g.r.IntN(8)
		doc := ""
		for range 1 + g.r.IntN(3) {
			doc += g.el(0)
		}
		var acts []string
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("seed %d: panic %v\ndoc %q\nacts %v", seed, p, doc, acts)
					fails++
				}
			}()
			c := cells.New(cols, rows, func(hotty.Event) {})
			if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			for range 40 {
				switch g.r.IntN(10) {
				case 0, 1, 2, 3:
					k := g.pick(fKeys)
					acts = append(acts, "key "+k)
					c.Key(k)
				case 4:
					x, y := g.r.IntN(cols+1), g.r.IntN(rows+1)
					acts = append(acts, fmt.Sprintf("click %d,%d", x, y))
					c.Click(x, y)
				case 5:
					acts = append(acts, "wheel")
					c.Wheel(0, 0, g.r.IntN(7)-3)
				case 6:
					p := g.pick(fTexts)
					acts = append(acts, "paste "+p)
					c.Paste(p)
				case 7:
					cols, rows = 1+g.r.IntN(30), 1+g.r.IntN(8)
					acts = append(acts, fmt.Sprintf("resize %d,%d", cols, rows))
					c.Resize(cols, rows)
				case 8:
					in := g.r.IntN(2) == 0
					acts = append(acts, fmt.Sprintf("focus %v", in))
					c.Focus(in)
				case 9:
					id := fmt.Sprintf("e%d", 1+g.r.IntN(max(g.ids, 1)))
					var cmd string
					switch g.r.IntN(5) {
					case 0:
						cmd = hotty.Delta("t", []hotty.Op{hotty.OpInner(id, g.el(2))})
					case 1:
						cmd = hotty.Delta("t", []hotty.Op{hotty.OpRemove(id)})
					case 2:
						cmd = hotty.Delta("t", []hotty.Op{hotty.OpAttr(id, "class", g.pick(fClasses))})
					case 3:
						v := g.pick(fTexts)
						cmd = hotty.SetField("t", id, hotty.FieldChange{Rev: 0, Value: &v})
					case 4:
						cmd = hotty.Focus("t", id)
					}
					acts = append(acts, "write "+id)
					c.Write([]byte(cmd))
				}
				if what := whole(c, cols, rows); what != "" {
					t.Errorf("seed %d after %v: %s\ndoc %q", seed, acts, what, doc)
					fails++
					return
				}
			}
		}()
		if fails > 5 {
			t.Fatal("stopping after 5 failures")
		}
	}
}
