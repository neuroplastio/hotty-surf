package cells_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"

	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

var (
	updateWalk = flag.Bool("update-walk", false, "write testdata/walk.golden from every run of TestWalkGolden")
	walkAll    = flag.Bool("walk-all", false, "make every run of TestWalkGolden, not an eightieth of them")
	walkDump   = flag.String("walk-dump", "", "write each step of TestWalkGolden's runs to this file, to compare two commits'")
)

// gwSteps is how many steps a run of TestWalkGolden takes.
const gwSteps = 150

// TestWalkGolden holds what a Screen shows under long random walks to
// what it showed when the golden was written: every story of the kit, in
// a plain, a dark and a muted wrapper, and random markup with the parts a
// layout keeps beyond the flow (a scrolled box, a column whose item
// grows, a popover, a table, fields, an active descendant, tooltips, a
// chart and its legend), each through 150 steps of every kind: deltas of
// every op, keys, the pointer, resizes, fields, focus, paste and cut.
// After each step it hashes Lines, Cells, Draw into a program's grid at
// an offset, Dirty before and after, the cursor, the selection, At and
// the events sent.
//
// A layout that keeps what it laid out before (laid.go) is held by
// CheckFresh to a pass that keeps nothing; this holds both to what the
// Screen drew before either. The golden was written by a commit that
// kept nothing between passes.
//
// It makes an eightieth of its 1,400 runs; -walk-all makes all of them.
// A change that draws differently writes the golden again
// (-update-walk). To find the step that differs, run -walk-all with
// -walk-dump at two commits and compare the files.
func TestWalkGolden(t *testing.T) {
	stride := 80
	if testing.Short() {
		stride = 200
	}
	if *walkAll || *updateWalk {
		stride = 1
	}
	const path = "testdata/walk.golden"
	want := map[string]string{}
	if !*updateWalk {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for l := range strings.Lines(string(b)) {
			if name, sum, ok := strings.Cut(strings.TrimSpace(l), " "); ok {
				want[name] = sum
			}
		}
	}
	dump := io.Discard
	if *walkDump != "" {
		f, err := os.Create(*walkDump)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		bw := bufio.NewWriter(f)
		defer bw.Flush()
		dump = bw
	}
	var out strings.Builder
	for i, r := range gwRuns(t) {
		if i%stride != 0 {
			continue
		}
		h := sha256.New()
		gwRunOne(t, io.MultiWriter(h, dump), r.name, r.markup, r.seed, gwSteps)
		sum := fmt.Sprintf("%x", h.Sum(nil)[:8])
		switch w, ok := want[r.name]; {
		case *updateWalk:
			fmt.Fprintf(&out, "%s %s\n", r.name, sum)
		case !ok:
			t.Errorf("%s: no such run in %s", r.name, path)
		case w != sum:
			t.Errorf("%s: the Screen shows what it didn't when %s was written", r.name, path)
		}
	}
	if *updateWalk {
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// gwRunAt is a run of TestWalkGolden: its name, its markup and its seed.
type gwRunAt struct {
	name, markup string
	seed         uint64
}

// gwRuns are TestWalkGolden's runs, two rounds of: every story in each
// wrapper, and 400 random documents.
func gwRuns(t *testing.T) []gwRunAt {
	var docs []string
	for _, st := range stories.All() {
		doc, err := surf.New("story", st.View(), ui.Stylesheet()).Document()
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, gwWithIDs(doc, "s"))
	}
	var runs []gwRunAt
	for round := range 2 {
		for si, st := range stories.All() {
			for k, wrap := range []string{"", "dark", "text-muted-foreground"} {
				m := docs[si]
				if wrap != "" {
					m = `<div id="wrap" class="` + wrap + `">` + m + `</div>`
				}
				runs = append(runs, gwRunAt{fmt.Sprintf("%s/%d/%d", st.ID, k, round), m, uint64(round*100000 + si*10 + k)})
			}
		}
		for seed := range 400 {
			g := &gwGen{r: rand.New(rand.NewPCG(uint64(round*100000+seed), 77)), cls: map[string]string{}}
			runs = append(runs, gwRunAt{fmt.Sprintf("rand%d/%d", seed, round), g.doc(), uint64(round*100000 + seed)})
		}
	}
	return runs
}

// sum is h's sum, short.
func sum(h io.Writer) string {
	return fmt.Sprintf("%x", h.(interface{ Sum([]byte) []byte }).Sum(nil)[:8])
}

func newH() io.Writer { return sha256.New() }

var (
	gwColours = []string{"text-primary", "text-foreground", "bg-accent", "bg-muted", "text-destructive", "bg-primary/10",
		"text-(--pill)", "text-(--pill-hot)", "bg-(--pill-hover)", "bg-[var(--x)]", "dark:bg-muted", "dark:text-primary",
		"text-muted-foreground/70", "text-accent-foreground", "bg-secondary", "text-primary-foreground", "bg-card",
		"text-(--muted-foreground)", "text-[var(--muted-foreground)]", "dark:text-(--muted-foreground)", "bg-background",
		"text-sidebar-foreground", "bg-sidebar-accent"}
	gwOther = []string{"text-muted-foreground", "dark:text-muted-foreground", "text-sm", "text-center", "bg-transparent",
		"font-bold", "font-normal", "hidden", "p-1", "px-2", "truncate", "flex", "flex-col", "shrink-0", "whitespace-pre",
		"grow", "opacity-50", "dark", "light", "btn", "badge", "card", "nav-item", "tabs", "skeleton", "hover:bg-accent",
		"line-through", "w-4", "gap-1", "input", "flex-1", "overflow-y-auto", "max-h-[3rem]", "h-full", "h-screen", "justify-between",
		"flex-wrap", "list-disc", "mt-1", "mb-1", "py-1", "w-full", "max-w-[10rem]", "flex-col-reverse", "popover",
		"max-h-[50vh]", "h-[40vh]"}
	gwRoleNames = map[string]bool{"foreground": true, "background": true, "primary": true, "primary-foreground": true,
		"secondary": true, "secondary-foreground": true, "muted": true, "accent": true, "accent-foreground": true,
		"destructive": true, "card": true, "card-foreground": true, "popover": true, "popover-foreground": true,
		"border": true, "input": true, "ring": true, "sidebar": true, "sidebar-foreground": true, "sidebar-accent": true,
		"sidebar-accent-foreground": true}
	gwTexts = []string{"", " ", "x", "Machines", "a longer label that wraps around the row", "日本語", "👍🏽 ok", "  ", "\n", "a\nb", "\tt"}
	gwAttrs = []struct{ k, v string }{
		{"data-variant", "outline"}, {"data-variant", "line"}, {"data-variant", "secondary"}, {"aria-hidden", "true"},
		{"aria-selected", "true"}, {"aria-selected", "false"}, {"open", ""}, {"data-side", "top"},
		{"data-align", "inline-end"}, {"data-orientation", "vertical"}, {"disabled", ""}, {"aria-disabled", "true"},
		{"aria-checked", "true"}, {"checked", ""}, {"value", "42"}, {"role", "separator"}, {"role", "tab"}, {"role", "tablist"},
		{"role", "heading"}, {"role", "group"}, {"role", "option"}, {"role", "row"}, {"role", "progressbar"},
		{"data-empty", "nothing"}, {"data-popover", ""}, {"aria-current", "true"}, {"style", "color:var(--x)"},
		{"style", "background:var(--muted-foreground)"}, {"style", "color:var(--muted-foreground)"}, {"hidden", ""},
		{"tabindex", "0"}, {"title", "tip"}, {"aria-activedescendant", "s3"}, {"selected", ""}, {"data-hotty-modal-inert", ""},
		{"aria-activedescendant", "o2"}, {"aria-activedescendant", "o3"}, {"data-tooltip", "a tip"}, {"data-side", "bottom"},
		{"style", "max-height:2rem;overflow-y:auto"}, {"style", "height:3rem"}, {"data-value", "7"}, {"size", "5"},
		{"rows", "3"}, {"aria-hidden", "false"}, {"data-side", "inline-end"},
	}
	gwUnattr = []string{"class", "hidden", "aria-selected", "data-variant", "style", "id", "open", "disabled", "role", "checked", "value"}
	gwKeys   = []string{"tab", "shift+tab", "enter", "space", "a", "backspace", "arrow_down", "arrow_up", "arrow_right", "arrow_left", "escape", "end", "home", "x", "page_down"}
	gwTags   = []string{"div", "span", "p", "b", "button", "a", "li", "ul", "details", "summary", "input", "label", "pre", "kbd", "section", "nav", "select", "textarea", "div", "span"}
)

type gwGen struct {
	r   *rand.Rand
	ids []string
	cls map[string]string
	n   int
}

func (g *gwGen) pick(s []string) string { return s[g.r.IntN(len(s))] }

func (g *gwGen) classes() string {
	var ts []string
	for range g.r.IntN(4) {
		if g.r.IntN(2) == 0 {
			ts = append(ts, g.pick(gwColours))
		} else {
			ts = append(ts, g.pick(gwOther[:24]))
		}
	}
	return strings.Join(ts, " ")
}

// el is a random element with an id, classes from both lists, texts.
func (g *gwGen) el(depth int) string {
	if depth > 4 || g.r.IntN(4) == 0 {
		return html.EscapeString(g.pick(gwTexts[1:]))
	}
	tag := g.pick(gwTags)
	g.n++
	id := fmt.Sprintf("g%d", g.n)
	var b strings.Builder
	fmt.Fprintf(&b, `<%s id="%s" class="%s"`, tag, id, g.classes())
	if g.r.IntN(5) == 0 {
		a := gwAttrs[g.r.IntN(len(gwAttrs))]
		fmt.Fprintf(&b, ` %s="%s"`, a.k, html.EscapeString(a.v))
	}
	switch tag {
	case "select":
		b.WriteString(`><option value="a">alpha</option><option value="b" selected>beta</option><option value="c">a long gamma</option></select>`)
		return b.String()
	case "textarea":
		fmt.Fprintf(&b, `>%s</textarea>`, html.EscapeString(g.pick(gwTexts)))
		return b.String()
	case "input":
		fmt.Fprintf(&b, ` type="%s" value="%s">`, g.pick([]string{"text", "checkbox", "radio"}), g.pick([]string{"v", "", "abc"}))
		return b.String()
	case "a":
		b.WriteString(` href="#x"`)
	}
	b.WriteString(">")
	for range g.r.IntN(4) {
		b.WriteString(g.el(depth + 1))
	}
	fmt.Fprintf(&b, "</%s>", tag)
	return b.String()
}

func (g *gwGen) doc() string {
	var b strings.Builder
	wrap := g.pick([]string{"", "dark", "text-muted-foreground", "light", "flex flex-col", "text-muted-foreground font-bold"})
	g.n++
	fmt.Fprintf(&b, `<div id="g%d" class="%s">`, g.n, wrap)
	for range 1 + g.r.IntN(4) {
		b.WriteString(g.el(0))
	}
	b.WriteString(`<div id="m1" class="text-muted-foreground"><span id="m2" class="text-(--pill)">muted <b id="m3">deep</b></span> rest <span id="m4">plain</span></div>`)
	b.WriteString(`<details id="d1"><summary id="d2" class="text-primary">more</summary><p id="d3" class="bg-muted">inside</p></details>`)
	// What a pass keeps beyond the flow: a scrolled box, a column whose
	// item grows and scrolls, an open popover, a table, fields, a list
	// with an active descendant, a tooltip, a chart and its legend.
	if g.r.IntN(2) == 0 {
		b.WriteString(`<div id="sc" class="overflow-y-auto max-h-[3rem]"><p id="sc1">one</p><p id="sc2">two</p><p id="sc3">three</p><p id="sc4" class="text-primary">four</p><p id="sc5">five</p></div>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<div id="col" class="flex flex-col h-[6rem]"><div id="cg" class="flex-1 overflow-y-auto"><p id="cg1">a</p><p id="cg2">b</p><p id="cg3">c</p><p id="cg4">d</p><p id="cg5">e</p><p id="cg6">f</p><p id="cg7">g</p></div><div id="cf" class="text-muted-foreground">footer</div></div>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<div id="pv" class="popover"><button id="pvb" aria-haspopup="menu">Open</button><div id="pvp" data-popover aria-hidden="false"><div id="pvm" role="menu"><div id="pv1" role="menuitem">first</div><div id="pv2" role="menuitem" class="text-destructive">second</div></div></div></div><p id="pva">after the popover</p>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<table id="tb"><thead id="th"><tr id="thr"><th id="th1">name</th><th id="th2">value</th></tr></thead><tbody id="tbb"><tr id="tr1" aria-selected="true"><td id="td1">alpha</td><td id="td2">1</td></tr><tr id="tr2"><td id="td3">a longer beta</td><td id="td4" class="text-primary">22</td></tr></tbody></table>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<form id="fm" class="flex gap-1"><select id="fs"><option value="a">alpha</option><option value="b">beta</option><option value="c">a long gamma</option></select><input id="fc" type="checkbox"><input id="fr1" type="radio" name="r"><input id="fr2" type="radio" name="r" checked><input id="ft" class="input" value="typed text" placeholder="ph"><textarea id="fta" rows="2" cols="8">line one line two line three</textarea></form>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<div id="lb" role="listbox" tabindex="0" aria-activedescendant="o2"><div id="o1" role="option">one</div><div id="o2" role="option" aria-selected="true">two</div><div id="o3" role="option">three</div></div>`)
	}
	if g.r.IntN(2) == 0 {
		b.WriteString(`<span id="tt" data-tooltip="a tooltip here" class="underline">hover me</span> <button id="ttb" data-tooltip="below" data-side="bottom">tip</button>`)
	}
	if g.r.IntN(3) == 0 {
		b.WriteString(`<div id="cht" data-slot="chart"><div id="chp" class="chart"><svg id="chs"><rect id="cr1" data-value="3"></rect><rect id="cr2" data-value="5"></rect><rect id="cr3" data-value="2"></rect><rect id="cr4" data-value="4"></rect></svg></div><div id="chl" aria-hidden="true" class="text-xs"><span id="chl1">jan</span><span id="chl2">feb</span></div><div id="chg" class="chart-legend"><div id="chg1"><span id="chi1" class="chart-legend-indicator"></span>one</div><div id="chg2"><span id="chi2" class="chart-legend-indicator"></span>two</div></div></div>`)
	}
	b.WriteString("</div>")
	return b.String()
}

func gwWalk(n *html.Node, f func(*html.Node)) {
	f(n)
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		gwWalk(k, f)
	}
}

func gwParse(markup string) []*html.Node {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	ns, _ := html.ParseFragment(strings.NewReader(markup), body)
	for _, n := range ns {
		body.AppendChild(n)
	}
	var out []*html.Node
	for k := body.FirstChild; k != nil; k = k.NextSibling {
		out = append(out, k)
	}
	return out
}

func gwRender(ns []*html.Node) string {
	var b strings.Builder
	for _, n := range ns {
		html.Render(&b, n)
	}
	return b.String()
}

func gwGetAttr(n *html.Node, k string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == k && a.Namespace == "" {
			return a.Val, true
		}
	}
	return "", false
}

func gwSetAttr(n *html.Node, k, v string) {
	for i := range n.Attr {
		if n.Attr[i].Key == k && n.Attr[i].Namespace == "" {
			n.Attr[i].Val = v
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: k, Val: v})
}

// gwWithIDs gives every element without an id one, so that every element
// is a delta's target.
func gwWithIDs(markup, prefix string) string {
	ns := gwParse(markup)
	i := 0
	for _, n := range ns {
		gwWalk(n, func(m *html.Node) {
			if m.Type == html.ElementNode {
				if _, ok := gwGetAttr(m, "id"); !ok {
					i++
					m.Attr = append(m.Attr, html.Attribute{Key: "id", Val: fmt.Sprintf("%s%d", prefix, i)})
				}
			}
		})
	}
	return gwRender(ns)
}

// learn adds markup's ids and classes to what the generator targets.
func (g *gwGen) learn(markup string) {
	for _, n := range gwParse(markup) {
		gwWalk(n, func(m *html.Node) {
			if m.Type != html.ElementNode {
				return
			}
			if id, ok := gwGetAttr(m, "id"); ok && id != "" {
				if _, seen := g.cls[id]; !seen {
					g.ids = append(g.ids, id)
				}
				c, _ := gwGetAttr(m, "class")
				g.cls[id] = c
			}
		})
	}
}

func gwIsColour(t string) bool {
	t = strings.TrimPrefix(t, "dark:")
	for _, p := range []string{"text-", "bg-"} {
		s, ok := strings.CutPrefix(t, p)
		if !ok {
			continue
		}
		if strings.HasPrefix(s, "(") || strings.HasPrefix(s, "[") {
			return true
		}
		name, _, _ := strings.Cut(s, "/")
		return gwRoleNames[name] || name == "muted-foreground" && strings.Contains(s, "/")
	}
	return false
}

// recolour is cls with its colour tokens swapped, the rest as it was; a
// sixth of the time with a token of another kind too.
func (g *gwGen) recolour(cls string) string {
	var ts []string
	for _, t := range strings.Fields(cls) {
		if !gwIsColour(t) {
			ts = append(ts, t)
		}
	}
	for range g.r.IntN(3) {
		i := g.r.IntN(len(ts) + 1)
		ts = append(ts[:i], append([]string{g.pick(gwColours)}, ts[i:]...)...)
	}
	if g.r.IntN(6) == 0 {
		i := g.r.IntN(len(ts) + 1)
		ts = append(ts[:i], append([]string{g.pick(gwOther)}, ts[i:]...)...)
	}
	return strings.Join(ts, " ")
}

func (g *gwGen) target() string {
	if len(g.ids) == 0 || g.r.IntN(40) == 0 {
		return "nope"
	}
	return g.ids[g.r.IntN(len(g.ids))]
}

func (g *gwGen) fragment() string {
	var b strings.Builder
	for range 1 + g.r.IntN(2) {
		b.WriteString(g.el(2))
	}
	s := b.String()
	g.learn(s)
	return s
}

// op is a random delta op and a short name for it.
func (g *gwGen) op() (hotty.Op, string) {
	id := g.target()
	switch k := g.r.IntN(20); {
	case k < 7:
		c := g.recolour(g.cls[id])
		g.cls[id] = c
		return hotty.OpAttr(id, "class", c), "recolour " + id + " " + strconv.Quote(c)
	case k == 7:
		a := gwAttrs[g.r.IntN(len(gwAttrs))]
		return hotty.OpAttr(id, a.k, a.v), "attr " + id + " " + a.k
	case k == 8:
		c := strings.TrimSpace(g.cls[id] + " " + g.pick(gwOther))
		g.cls[id] = c
		return hotty.OpAttr(id, "class", c), "class " + id + " " + strconv.Quote(c)
	case k == 9:
		u := g.pick(gwUnattr)
		if u == "class" {
			g.cls[id] = ""
		}
		return hotty.OpUnattr(id, u), "unattr " + id + " " + u
	case k == 10:
		g.n++
		nid := fmt.Sprintf("r%d", g.n)
		g.ids = append(g.ids, nid)
		g.cls[nid] = g.cls[id]
		return hotty.OpAttr(id, "id", nid), "reid " + id
	case k == 11, k == 12, k == 13:
		return hotty.OpText(id, g.pick(gwTexts)), "text " + id
	case k == 14:
		f := g.fragment()
		switch g.r.IntN(3) {
		case 0:
			return hotty.OpInner(id, f), "inner " + id
		case 1:
			return hotty.OpAppend(id, f), "append " + id
		default:
			return hotty.OpPrepend(id, f), "prepend " + id
		}
	case k == 15:
		f := g.fragment()
		switch g.r.IntN(3) {
		case 0:
			return hotty.OpBefore(id, f), "before " + id
		case 1:
			return hotty.OpAfter(id, f), "after " + id
		default:
			return hotty.OpReplace(id, f), "replace " + id
		}
	case k == 16:
		return hotty.OpRemove(id), "remove " + id
	case k == 17:
		w := g.pick([]string{hotty.RevealNearest, hotty.RevealStart, hotty.RevealEnd, "middle"})
		return hotty.Op{Op: "reveal", Target: id, Value: &w}, "reveal " + id
	case k == 18:
		switch g.r.IntN(3) {
		case 0:
			return hotty.OpVar(id, "--x", "red"), "var"
		case 1:
			return hotty.OpScroll(id, hotty.Action("down")), "scroll"
		default:
			return hotty.OpMorph(id, "<p>m</p>"), "morph"
		}
	default:
		s := g.pick([]string{"color:var(--primary)", "background-color:var(--accent)", "color:var(--muted-foreground)", ""})
		return hotty.OpAttr(id, "style", s), "style " + id
	}
}

// mutate gives markup random colour swaps, texts and attributes, in the
// same shape: a refill.
func (g *gwGen) mutate(markup string) string {
	ns := gwParse(markup)
	for _, n := range ns {
		gwWalk(n, func(m *html.Node) {
			switch m.Type {
			case html.TextNode:
				if g.r.IntN(5) == 0 {
					t := g.pick(gwTexts[1:])
					m.Data = t
				}
			case html.ElementNode:
				switch g.r.IntN(6) {
				case 0:
					c, _ := gwGetAttr(m, "class")
					gwSetAttr(m, "class", g.recolour(c))
				case 1:
					a := gwAttrs[g.r.IntN(len(gwAttrs))]
					gwSetAttr(m, a.k, a.v)
				}
			}
		})
	}
	return gwRender(ns)
}

type gwRun struct {
	t      *testing.T
	w      io.Writer
	c      *cells.Screen
	events []string
}

func (s *gwRun) dump(h io.Writer, g *gwGen, cols, rows int) {
	if g.r.IntN(4) != 0 {
		fmt.Fprintf(h, "dirty %v;", s.c.Dirty())
	}
	switch g.r.IntN(7) {
	case 0:
		return
	case 5, 6:
		// Draw into a program's grid, at an offset, cut by its edges.
		gw, gh := cols+g.r.IntN(5)-2, rows+g.r.IntN(5)-2
		grid := make([][]cells.Cell, max(gh, 0))
		for y := range grid {
			grid[y] = make([]cells.Cell, max(gw, 0))
			for x := range grid[y] {
				grid[y][x].Text = "~"
			}
		}
		s.c.Draw(grid, g.r.IntN(3)-1, g.r.IntN(3)-1)
		for _, r := range grid {
			for _, k := range r {
				fmt.Fprintf(h, "%s/%s/%s/%d ", k.Text, k.Fg, k.Bg, k.Attr)
			}
			h.Write([]byte{2})
		}
	case 1:
		for _, l := range s.c.Lines() {
			h.Write([]byte(l))
			h.Write([]byte{3})
		}
	case 2:
		for _, r := range s.c.Cells() {
			for _, k := range r {
				fmt.Fprintf(h, "%s/%s/%s/%d ", k.Text, k.Fg, k.Bg, k.Attr)
			}
		}
	default:
		for _, r := range s.c.Cells() {
			for _, k := range r {
				fmt.Fprintf(h, "%s/%s/%s/%d ", k.Text, k.Fg, k.Bg, k.Attr)
			}
			h.Write([]byte{2})
		}
		for _, l := range s.c.Lines() {
			h.Write([]byte(l))
			h.Write([]byte{3})
		}
		x, y, ok := s.c.Cursor()
		fmt.Fprintf(h, "cur %d %d %v;sel %q;", x, y, ok, s.c.Selection())
		for yy := 0; yy < rows; yy += 2 {
			for xx := 0; xx < cols; xx += 3 {
				id, a := s.c.At(xx, yy)
				fmt.Fprintf(h, "%s %+v;", id, a)
			}
		}
	}
	fmt.Fprintf(h, "dirty after %v;", s.c.Dirty())
	if g.r.IntN(3) == 0 {
		for _, r := range s.c.Cells() {
			for _, k := range r {
				fmt.Fprintf(h, "%s/%s/%s/%d ", k.Text, k.Fg, k.Bg, k.Attr)
			}
		}
	}
}

func gwRunOne(t *testing.T, w io.Writer, name, markup string, seed uint64, steps int) {
	g := &gwGen{r: rand.New(rand.NewPCG(seed, 1234)), cls: map[string]string{}, n: 100000}
	cols, rows := 20+g.r.IntN(70), 4+g.r.IntN(25)
	s := &gwRun{t: t, w: w}
	s.c = cells.New(cols, rows, func(e hotty.Event) {
		j, _ := json.Marshal(e)
		s.events = append(s.events, string(j))
	})
	cur := markup
	g.learn(cur)
	_, err := s.c.Write([]byte(hotty.Doc("v", cur)))
	h := newH()
	fmt.Fprintf(h, "%v", err)
	s.dump(h, g, cols, rows)
	fmt.Fprintf(w, "%s 0 load %s\n", name, sum(h))
	for i := range steps {
		var act string
		h := newH()
		func() {
			defer func() {
				if p := recover(); p != nil {
					act += fmt.Sprintf(" PANIC %v", p)
				}
			}()
			switch k := g.r.IntN(22); {
			case k < 2:
				key := g.pick(gwKeys)
				act = "key " + key
				fmt.Fprintf(h, "%v", s.c.Key(key))
			case k < 4:
				x, y := g.r.IntN(cols), g.r.IntN(rows)
				act = fmt.Sprintf("move %d,%d", x, y)
				fmt.Fprintf(h, "%v", s.c.Move(x, y, 0))
			case k == 4:
				x, y := g.r.IntN(cols), g.r.IntN(rows)
				act = fmt.Sprintf("click %d,%d", x, y)
				fmt.Fprintf(h, "%v", s.c.Click(x, y))
			case k == 5:
				x, y, x2, y2 := g.r.IntN(cols), g.r.IntN(rows), g.r.IntN(cols), g.r.IntN(rows)
				act = fmt.Sprintf("drag %d,%d-%d,%d", x, y, x2, y2)
				fmt.Fprintf(h, "%v %v %v", s.c.Press(x, y, 0, 0), s.c.Move(x2, y2, 0), s.c.Release(x2, y2, 0))
			case k == 6:
				cols, rows = 10+g.r.IntN(80), 2+g.r.IntN(30)
				act = fmt.Sprintf("resize %d,%d", cols, rows)
				s.c.Resize(cols, rows)
			case k == 7:
				cur = g.mutate(cur)
				g.learn(cur)
				act = "refill"
				_, err := s.c.Write([]byte(hotty.Doc("v", cur)))
				fmt.Fprintf(h, "%v", err)
			case k < 16:
				var ops []hotty.Op
				var names []string
				for range 1 + g.r.IntN(3) {
					o, n := g.op()
					ops = append(ops, o)
					names = append(names, n)
				}
				act = "delta " + strings.Join(names, "; ")
				_, err := s.c.Write([]byte(hotty.Delta("v", ops)))
				fmt.Fprintf(h, "%v", err)
			case k == 16:
				id := g.target()
				v := g.pick(gwTexts)
				act = "field " + id
				_, err := s.c.Write([]byte(hotty.SetField("v", id, hotty.FieldChange{Rev: g.r.IntN(3), Value: &v})))
				fmt.Fprintf(h, "%v", err)
			case k == 17:
				act = "wheel"
				fmt.Fprintf(h, "%v", s.c.Wheel(g.r.IntN(cols), g.r.IntN(rows), g.r.IntN(5)-2))
			case k == 18:
				p := g.pick(gwTexts)
				act = "paste"
				fmt.Fprintf(h, "%v", s.c.Paste(p))
			case k == 19:
				in := g.r.IntN(2) == 0
				act = fmt.Sprintf("focus %v", in)
				s.c.Focus(in)
			case k == 20:
				act = "cut"
				fmt.Fprintf(h, "%q", s.c.Cut())
			default:
				act = "same doc"
				_, err := s.c.Write([]byte(hotty.Doc("v", cur)))
				fmt.Fprintf(h, "%v", err)
			}
			s.dump(h, g, cols, rows)
		}()
		for _, e := range s.events {
			h.Write([]byte(e))
		}
		s.events = s.events[:0]
		fmt.Fprintf(w, "%s %d %s %s\n", name, i+1, sum(h), act)
	}
}
