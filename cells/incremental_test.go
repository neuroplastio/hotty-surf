package cells

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/internal/stories"
)

// A Screen lays out again only what changed, and paints again only the
// rows that did (laid.go, paint.go): after any steps, its layout and its
// grid are what a pass that keeps nothing makes of the same state
// (CheckFresh). Random walks over the kit's stories and random markup,
// every kind of step among them.
func TestIncrementalIsFresh(t *testing.T) {
	var docs []string
	for i, st := range stories.All() {
		if i%2 == 0 {
			docs = append(docs, withIDs(storyMarkup(t, st)))
		}
	}
	for seed := range 30 {
		docs = append(docs, randomDoc(rand.New(rand.NewPCG(uint64(seed), 7))))
	}
	for i, doc := range docs {
		r := rand.New(rand.NewPCG(uint64(i), 11))
		w := &walker{r: r, cols: 20 + r.IntN(60), rows: 4 + r.IntN(20)}
		w.c = New(w.cols, w.rows, func(hotty.Event) {})
		if _, err := w.c.Write([]byte(hotty.Doc("v", doc))); err != nil {
			t.Fatal(err)
		}
		for step := range 40 {
			act := w.step()
			switch r.IntN(3) {
			case 0:
				w.c.Cells()
			case 1:
				w.c.Dirty()
			}
			if err := CheckFresh(w.c); err != nil {
				t.Fatalf("doc %d, step %d (%s): %v", i, step, act, err)
			}
		}
	}
}

// withIDs gives every element of markup an id, so that every one is a
// delta's target.
func withIDs(markup string) string {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(markup), body)
	if err != nil {
		panic(err)
	}
	var b strings.Builder
	i := 0
	for _, n := range nodes {
		walk(n, func(m *html.Node) {
			if m.Type == html.ElementNode && !has(m, "id") {
				i++
				m.Attr = append(m.Attr, html.Attribute{Key: "id", Val: fmt.Sprintf("t%d", i)})
			}
		})
		html.Render(&b, n)
	}
	return b.String()
}

var (
	incTags    = []string{"div", "span", "p", "b", "button", "a", "li", "ul", "details", "summary", "label", "pre", "section"}
	incClasses = []string{"text-primary", "bg-muted", "text-(--pill)", "dark:text-primary", "text-muted-foreground", "font-bold",
		"hidden", "flex", "flex-col", "grow", "flex-1", "truncate", "shrink-0", "whitespace-pre", "p-1", "gap-1", "dark",
		"overflow-y-auto", "max-h-[2rem]", "h-full", "justify-between", "list-disc", "mt-1", "w-full", "btn"}
	incTexts = []string{"", " ", "x", "Machines", "a longer label that wraps", "日本語", "👍🏽 ok", "a\nb"}
	incAttrs = [][2]string{{"aria-selected", "true"}, {"aria-hidden", "true"}, {"open", ""}, {"hidden", ""},
		{"data-variant", "outline"}, {"disabled", ""}, {"data-tooltip", "tip"}, {"data-popover", ""},
		{"aria-activedescendant", "o2"}, {"style", "color:var(--x)"}, {"tabindex", "0"}, {"role", "option"}}
	incKeys = []string{"tab", "shift+tab", "enter", "space", "a", "backspace", "arrow_down", "arrow_up", "end", "home"}
)

// randomDoc is random nested markup, every element with an id, and the
// parts a pass keeps beyond the flow: a scrolled box, a column whose item
// grows, an open popover, a table, fields, an active descendant.
func randomDoc(r *rand.Rand) string {
	n := 0
	var el func(depth int) string
	el = func(depth int) string {
		if depth > 3 || r.IntN(4) == 0 {
			return html.EscapeString(incTexts[1+r.IntN(len(incTexts)-1)])
		}
		n++
		tag := incTags[r.IntN(len(incTags))]
		var cls []string
		for range r.IntN(3) {
			cls = append(cls, incClasses[r.IntN(len(incClasses))])
		}
		var b strings.Builder
		fmt.Fprintf(&b, `<%s id="r%d" class="%s">`, tag, n, strings.Join(cls, " "))
		for range r.IntN(4) {
			b.WriteString(el(depth + 1))
		}
		fmt.Fprintf(&b, "</%s>", tag)
		return b.String()
	}
	var b strings.Builder
	b.WriteString(`<div id="root" class="flex flex-col">`)
	for range 1 + r.IntN(4) {
		b.WriteString(el(0))
	}
	b.WriteString(`<div id="sc" class="overflow-y-auto max-h-[3rem]"><p id="s1">one</p><p id="s2">two</p><p id="s3">three</p><p id="s4">four</p></div>`)
	b.WriteString(`<div id="col" class="flex flex-col h-[6rem]"><div id="cg" class="flex-1 overflow-y-auto"><p id="g1">a</p><p id="g2">b</p><p id="g3">c</p></div><p id="cf">footer</p></div>`)
	b.WriteString(`<div id="pv" class="popover"><button id="pvb">Open</button><div id="pvp" data-popover aria-hidden="false"><p id="pv1">inside</p></div></div>`)
	b.WriteString(`<table id="tb"><thead id="th"><tr id="thr"><th id="th1">name</th></tr></thead><tbody id="tbb"><tr id="tr1"><td id="td1">alpha</td></tr></tbody></table>`)
	b.WriteString(`<div id="fm" class="flex"><select id="fs"><option value="a">alpha</option><option value="b">a long beta</option></select><input id="fc" type="checkbox"><input id="ft" class="input" value="typed"><textarea id="fta" rows="2" cols="8">one two three</textarea></div>`)
	b.WriteString(`<div id="lb" role="listbox" tabindex="0" aria-activedescendant="o1"><div id="o1" role="option">one</div><div id="o2" role="option">two</div></div>`)
	b.WriteString(`</div>`)
	return b.String()
}

// walker takes random steps on a Screen.
type walker struct {
	r          *rand.Rand
	c          *Screen
	cols, rows int
}

// target is a random id of the Screen's, or one it hasn't.
func (w *walker) target() string {
	var ids []string
	walk(w.c.body, func(n *html.Node) {
		if id := idOf(n); id != "" && n.Type == html.ElementNode {
			ids = append(ids, id)
		}
	})
	if len(ids) == 0 || w.r.IntN(30) == 0 {
		return "nope"
	}
	return ids[w.r.IntN(len(ids))]
}

// step takes a step, and says what it was.
func (w *walker) step() string {
	r, c := w.r, w.c
	pick := func(s []string) string { return s[r.IntN(len(s))] }
	switch k := r.IntN(16); {
	case k < 7:
		var ops []hotty.Op
		for range 1 + r.IntN(3) {
			id := w.target()
			switch r.IntN(8) {
			case 0, 1:
				ops = append(ops, hotty.OpAttr(id, "class", pick(incClasses)+" "+pick(incClasses)))
			case 2:
				a := incAttrs[r.IntN(len(incAttrs))]
				ops = append(ops, hotty.OpAttr(id, a[0], a[1]))
			case 3:
				ops = append(ops, hotty.OpUnattr(id, pick([]string{"class", "hidden", "open", "aria-selected"})))
			case 4, 5:
				ops = append(ops, hotty.OpText(id, pick(incTexts)))
			case 6:
				ops = append(ops, hotty.OpAppend(id, `<span id="new">more</span>`))
			default:
				v := hotty.RevealNearest
				ops = append(ops, hotty.Op{Op: "reveal", Target: id, Value: &v})
			}
		}
		c.Write([]byte(hotty.Delta("v", ops)))
		return fmt.Sprintf("delta %v", ops)
	case k < 9:
		key := pick(incKeys)
		c.Key(key)
		return "key " + key
	case k == 9:
		x, y := r.IntN(w.cols), r.IntN(w.rows)
		c.Move(x, y, 0)
		return fmt.Sprintf("move %d,%d", x, y)
	case k == 10:
		x, y := r.IntN(w.cols), r.IntN(w.rows)
		c.Click(x, y)
		return fmt.Sprintf("click %d,%d", x, y)
	case k == 11:
		x, y := r.IntN(w.cols), r.IntN(w.rows)
		c.Wheel(x, y, r.IntN(5)-2)
		return fmt.Sprintf("wheel %d,%d", x, y)
	case k == 12:
		w.cols, w.rows = 10+r.IntN(70), 2+r.IntN(25)
		c.Resize(w.cols, w.rows)
		return fmt.Sprintf("resize %d,%d", w.cols, w.rows)
	case k == 13:
		v := pick(incTexts)
		c.Write([]byte(hotty.SetField("v", pick([]string{"ft", "fta", "fs"}), hotty.FieldChange{Value: &v})))
		return "field"
	case k == 14:
		in := r.IntN(2) == 0
		c.Focus(in)
		return fmt.Sprintf("focus %v", in)
	default:
		c.Paste(pick(incTexts))
		return "paste"
	}
}
