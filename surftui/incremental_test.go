package surftui

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/internal/book/stories"
)

// A Screen lays out again only what changed, and paints again only the
// rows that did (laid.go, paint.go): after any steps, its layout and its
// grid are what a pass that keeps nothing makes of the same state
// (CheckFresh). Random walks over every story of the kit and random
// markup, every kind of step among them.
func TestIncrementalIsFresh(t *testing.T) {
	var docs []string
	for _, st := range stories.All() {
		docs = append(docs, withIDs(storyMarkup(t, st)))
	}
	seeds, steps := 60, 40
	if testing.Short() {
		seeds, steps = 30, 30
	}
	for seed := range seeds {
		docs = append(docs, randomDoc(rand.New(rand.NewPCG(uint64(seed), 7))))
	}
	for i, doc := range docs {
		r := rand.New(rand.NewPCG(uint64(i), 11))
		w := &walker{r: r, cols: 20 + r.IntN(60), rows: 4 + r.IntN(20)}
		w.c = New(w.cols, w.rows, func(hotty.Event) {})
		if _, err := w.c.Write([]byte(hotty.Doc("v", doc))); err != nil {
			t.Fatal(err)
		}
		for step := range steps {
			act := w.step()
			switch r.IntN(3) {
			case 0:
				w.c.Cells()
			case 1:
				w.c.Dirty()
			}
			untip(w.c)
			if err := CheckFresh(w.c); err != nil {
				t.Fatalf("doc %d, step %d (%s): %v", i, step, act, err)
			}
		}
	}
}

// untip takes a tooltip away before CheckFresh. A pass draws the tip at
// the document's scroll as it was before the pass bounded it, which a
// pass after, for the same state, doesn't: the tip and a fresh pass can
// differ, as they always have. Which scroll the tip should be drawn at
// is a change of what the Screen draws, apart from this test.
func untip(c *Screen) {
	if c.tip != nil {
		c.tip, c.lay = nil, nil
	}
}

// fresh is a Screen of doc, cols by rows, drawn, held to a fresh pass.
func fresh(t *testing.T, cols, rows int, doc string) *Screen {
	t.Helper()
	c := New(cols, rows, func(hotty.Event) {})
	if _, err := c.Write([]byte(hotty.Doc("v", doc))); err != nil {
		t.Fatal(err)
	}
	c.Cells()
	if err := CheckFresh(c); err != nil {
		t.Fatalf("drawn: %v", err)
	}
	return c
}

// stepFresh writes ops, draws, and holds the Screen to a fresh pass.
func stepFresh(t *testing.T, c *Screen, what string, ops ...hotty.Op) {
	t.Helper()
	if _, err := c.Write([]byte(hotty.Delta("v", ops))); err != nil {
		t.Fatal(err)
	}
	c.Cells()
	if err := CheckFresh(c); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// A box as high as half the Screen keeps no layout of a Screen of
// another height (holds, the rows a laid read).
func TestIncrementalVhResize(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<div id="root"><div id="sc" class="overflow-y-auto max-h-[50vh]">`)
	for i := range 12 {
		fmt.Fprintf(&b, `<p id="p%d">line %d</p>`, i, i)
	}
	b.WriteString(`</div><p id="after">after</p></div>`)
	c := fresh(t, 30, 10, b.String())
	c.Resize(30, 20)
	c.Cells()
	if err := CheckFresh(c); err != nil {
		t.Fatal(err)
	}
}

// A colour-only class change can end or start the muted text round an
// element, which changes its look and not only its colours: it is laid
// out again (tree.go's unlayAll).
func TestIncrementalColourOnlyUnmutes(t *testing.T) {
	c := fresh(t, 30, 6, `<div id="root"><div class="text-muted-foreground">`+
		`<p id="p" class="text-primary x">hello</p><p id="q" class="x">there</p></div></div>`)
	stepFresh(t, c, "muted again", hotty.OpAttr("p", "class", "x"))
	stepFresh(t, c, "muted no more", hotty.OpAttr("q", "class", "text-primary x"))
}

// A bar chart's legend draws its series' shades, read from the plot
// beside it: when the plot has no bars, the legend's marks are squares
// (dependOn).
func TestIncrementalLegendFollowsPlot(t *testing.T) {
	c := fresh(t, 50, 12, `<div id="root"><div id="fig" data-slot="chart">`+
		`<svg id="plot" class="chart"><rect id="r1" data-value="3"></rect><rect id="r2" data-value="5"></rect></svg>`+
		`<div id="lg" class="chart-legend">`+
		`<span class="chart-legend-item"><span class="chart-legend-indicator"></span>One</span>`+
		`<span class="chart-legend-item"><span class="chart-legend-indicator"></span>Two</span>`+
		`</div></div></div>`)
	if l := strings.Join(c.Lines(), "\n"); !strings.Contains(l, "▒") {
		t.Fatalf("no second series' shade in the legend:\n%s", l)
	}
	stepFresh(t, c, "no bars", hotty.OpUnattr("r1", "data-value"), hotty.OpUnattr("r2", "data-value"))
	if l := strings.Join(c.Lines(), "\n"); strings.Contains(l, "▒") || strings.Count(l, "■") != 2 {
		t.Fatalf("the legend's marks are not squares:\n%s", l)
	}
}

// A column's growing item lays out at its share, scrolled within it; the
// same element, as high and as wide but not grown, is as high as its
// content. Its laid for one isn't the other's (laidKey's grown). The
// column grows in its own column, and a text beside it takes all of
// that one's rows: no class changes, which would forget the item's
// style, and its laids with it.
func TestIncrementalGrownIsKept(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<div id="root"><div id="outer" class="flex flex-col h-[7rem]">`)
	b.WriteString(`<div id="col" class="flex-1 flex flex-col">`)
	b.WriteString(`<div id="g" class="flex-1 h-[5rem] overflow-y-auto">`)
	for i := range 8 {
		fmt.Fprintf(&b, `<p id="g%d">item %d</p>`, i, i)
	}
	b.WriteString(`</div><p id="f">footer</p></div><p id="sib">x</p></div></div>`)
	c := fresh(t, 30, 20, b.String())
	long := strings.Repeat("word ", 40)
	stepFresh(t, c, "no rows to grow in", hotty.OpText("sib", long))
	stepFresh(t, c, "rows again", hotty.OpText("sib", "x"))
}

// A column's growing item that is an open popover is laid out after the
// flow, at the share the column gave it. Where the column's laid is
// replayed, the replay gives the item its share again (oHeight).
func TestIncrementalGrowingPopover(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<div id="root"><div id="col" class="flex flex-col h-[6rem]">`)
	b.WriteString(`<div id="g" class="flex-1 overflow-y-auto" data-popover="">`)
	for i := range 8 {
		fmt.Fprintf(&b, `<p id="g%d">item %d</p>`, i, i)
	}
	b.WriteString(`</div><p id="f">footer</p></div></div>`)
	c := fresh(t, 29, 7, b.String())
	c.Resize(29, 7) // laid out again, the same
	c.Cells()
	if err := CheckFresh(c); err != nil {
		t.Fatal(err)
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
		"overflow-y-auto", "max-h-[2rem]", "h-full", "justify-between", "list-disc", "mt-1", "w-full", "btn",
		"max-h-[50vh]", "h-[40vh]", "h-screen", "max-h-[30vh]",
		"hover:bg-accent", "hover:underline", "hover:font-bold", "hover:text-muted-foreground", "dark:hover:bg-muted",
		"hover:hidden", "hover:p-1", "hover:text-primary"}
	// incColours are the classes a colour-only change swaps.
	incColours = []string{"text-primary", "bg-muted", "text-(--pill)", "dark:text-primary", "text-destructive", "bg-accent",
		"hover:bg-accent", "hover:text-primary"}
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
	b.WriteString(`<div id="hk" class="flex gap-1"><button id="hb1" class="btn" data-variant="outline">out</button><button id="hb2" class="btn" data-variant="ghost">ghost</button><a id="hb3" class="btn" data-variant="link" href="#x">link</a><a id="hb4" class="badge" href="#x">badge</a><button id="hb6" class="btn">go</button></div>`)
	b.WriteString(`<table id="ht" class="table"><tbody id="htb"><tr id="htr1"><td id="htd1">one</td></tr><tr id="htr2" role="row" aria-selected="true"><td id="htd2">two</td></tr></tbody></table>`)
	b.WriteString(`<p id="hp" class="hover:bg-accent hover:text-accent-foreground">lit <span id="hps" class="hover:underline text-primary">inside</span></p>`)
	b.WriteString(`<div id="hn" class="flex gap-1"><button id="hn1" class="btn" disabled>off</button><div class="tabs"><div role="tablist"><button id="hn2" role="tab" aria-disabled="true" data-tooltip="no">tab</button></div></div>` +
		`<div id="hn3" data-hotty-modal-inert style="pointer-events:none!important"><p id="hn4" class="hover:bg-accent pointer-events-auto">back</p><p id="hn5" class="hover:bg-accent">behind</p></div></div>`)
	b.WriteString(`<div id="mu" class="text-muted-foreground"><p id="mu1" class="text-primary x">coloured</p><p id="mu2" class="x">muted</p></div>`)
	b.WriteString(`<div id="vh" class="overflow-y-auto max-h-[40vh]"><p id="vh1">a</p><p id="vh2">b</p><p id="vh3">c</p><p id="vh4">d</p><p id="vh5">e</p></div>`)
	b.WriteString(`<div id="ch" data-slot="chart"><svg id="chs" class="chart"><rect id="cr1" data-value="3"></rect><rect id="cr2" data-value="5"></rect></svg>` +
		`<div id="chl" class="chart-legend"><span id="chl1" class="chart-legend-item"><span id="chi1" class="chart-legend-indicator"></span>one</span>` +
		`<span id="chl2" class="chart-legend-item"><span id="chi2" class="chart-legend-indicator"></span>two</span></div></div>`)
	b.WriteString(`</div>`)
	return b.String()
}

// walker takes random steps on a Screen.
type walker struct {
	r          *rand.Rand
	c          *Screen
	cols, rows int
}

// recolour is n's class with its colours swapped and nothing else: a
// colour-only change.
func (w *walker) recolour(n *html.Node) string {
	var ts []string
	for _, t := range strings.Fields(attr(n, "class")) {
		if !slices.Contains(incColours, t) && t != "text-muted-foreground" {
			ts = append(ts, t)
		}
	}
	for range w.r.IntN(3) {
		ts = append(ts, incColours[w.r.IntN(len(incColours))])
	}
	return strings.Join(ts, " ")
}

// off is a cell just off the Screen: the pointer leaving it.
func (w *walker) off() (x, y int) {
	switch w.r.IntN(4) {
	case 0:
		return -1, w.r.IntN(w.rows)
	case 1:
		return w.r.IntN(w.cols), -1
	case 2:
		return w.cols, w.r.IntN(w.rows)
	default:
		return w.r.IntN(w.cols), w.rows
	}
}

// byID is the Screen's element with the id, or nil.
func (w *walker) byID(id string) *html.Node {
	var found *html.Node
	walk(w.c.body, func(n *html.Node) {
		if found == nil && n.Type == html.ElementNode && idOf(n) == id {
			found = n
		}
	})
	return found
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
	switch k := r.IntN(19); {
	case k < 7:
		var ops []hotty.Op
		for range 1 + r.IntN(3) {
			id := w.target()
			switch r.IntN(10) {
			case 0, 1:
				ops = append(ops, hotty.OpAttr(id, "class", pick(incClasses)+" "+pick(incClasses)))
			case 8:
				if n := w.byID(id); n != nil {
					ops = append(ops, hotty.OpAttr(id, "class", w.recolour(n)))
				}
			case 9:
				// The chart's bars, which its legend reads.
				bar := pick([]string{"cr1", "cr2"})
				if r.IntN(2) == 0 {
					ops = append(ops, hotty.OpUnattr(bar, "data-value"))
				} else {
					ops = append(ops, hotty.OpAttr(bar, "data-value", fmt.Sprint(r.IntN(9))))
				}
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
	case k == 16:
		x, y := w.off()
		c.Move(x, y, 0)
		return fmt.Sprintf("move off %d,%d", x, y)
	case k == 17:
		// A button down, moved with, and let go or still down.
		x, y := r.IntN(w.cols), r.IntN(w.rows)
		c.Press(x, y, 0, r.IntN(3))
		act := fmt.Sprintf("press %d,%d", x, y)
		for range 1 + r.IntN(3) {
			if r.IntN(4) == 0 {
				x, y = w.off()
			} else {
				x, y = r.IntN(w.cols), r.IntN(w.rows)
			}
			c.Move(x, y, 0)
			act += fmt.Sprintf(", move %d,%d", x, y)
		}
		if r.IntN(3) != 0 {
			c.Release(x, y, 0)
			act += ", release"
		}
		return act
	case k == 18:
		act := "moves"
		for range 2 + r.IntN(3) {
			x, y := r.IntN(w.cols), r.IntN(w.rows)
			c.Move(x, y, 0)
			act += fmt.Sprintf(" %d,%d", x, y)
		}
		return act
	default:
		c.Paste(pick(incTexts))
		return "paste"
	}
}
