package surf_test

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// A made-up list, in the shapes compiled instances come in: rows of
// a block list, list items, table rows and inline spans, a group that
// holds rows, kit controls under a KeyScope, and a parent's keys.
type rowProps struct {
	Tag    string // div, li, tr or span
	Key    string
	Title  string
	Badge  string
	Tone   string
	Sel    bool
	Field  bool
	Kit    bool
	Clicks bool
	App    *listApp
}

type groupProps struct {
	Key  string
	Name string
	Rows [3]rowProps
}

type listApp struct {
	shape    string // div, ul, table or p
	rows     []rowProps
	group    groupProps
	grouped  bool
	keys     bool // the list declares j, which a row's field then types
	bind     bool // the KeyScope binds a key
	kitFirst bool // a kit control before the list
	typed    map[string]string
	clicked  []string
	counts   []int
}

var (
	rowC   = surf.Compile(rowView)
	groupC = surf.Compile(groupView)
)

func onAttrs(ctx context.Context, id string, hs ...surf.Handler) string {
	var b strings.Builder
	for _, kv := range surf.On(ctx, id, hs...).Items() {
		fmt.Fprintf(&b, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
	}
	return b.String()
}

func rowView(p rowProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := p.App
		click := surf.Click(func() { a.clicked = append(a.clicked, p.Key) })
		if p.Clicks {
			click = surf.Clicks(func(n int) { a.counts = append(a.counts, n) })
		}
		on := onAttrs(ctx, "row-"+p.Key, click)
		cls := strings.TrimSpace("row " + p.Tone)
		switch p.Tag {
		case "tr":
			fmt.Fprintf(w, `<tr class="%s"%s><td>%s</td><td>%s</td></tr>`, cls, on, p.Title, p.Badge)
			return nil
		case "span":
			fmt.Fprintf(w, `<span class="%s"%s>%s</span>`, cls, on, p.Title)
			return nil
		}
		fmt.Fprintf(w, `<%s class="%s"%s><span>%s</span>`, p.Tag, cls, on, p.Title)
		if p.Badge != "" {
			fmt.Fprintf(w, ` <b>%s</b>`, p.Badge)
		}
		if p.Field {
			fmt.Fprintf(w, `<input type="text"%s value="%s">`, onAttrs(ctx, "in-"+p.Key, surf.Input(func(v string) { a.typed[p.Key] = v })), p.Title)
		}
		if p.Kit {
			err := ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "kit-"+p.Key, surf.Click(func() {}))}).Render(templ.WithChildren(ctx, templ.Raw("Go")), w)
			if err != nil {
				return err
			}
		}
		if p.Sel {
			surf.Autofocus(ctx, "row-"+p.Key)
		}
		fmt.Fprintf(w, `</%s>`, p.Tag)
		return nil
	})
}

func groupView(p groupProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<section class="group"><h3>%s</h3>`, p.Name)
		for _, r := range p.Rows {
			if err := rowC.With(r.Key, r).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</section>`)
		return err
	})
}

func row(p rowProps, compiled bool) templ.Component {
	if compiled {
		return rowC.With(p.Key, p)
	}
	return rowView(p)
}

// view is a's view, its rows compiled or plain.
func (a *listApp) view(compiled bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var keys []string
		if a.keys {
			keys = []string{"j"}
		}
		fmt.Fprintf(w, `<div%s>`, onAttrs(ctx, "outer", surf.Key(func(string) bool { return false }, keys...)))
		var bindings []ui.KeyBinding
		if a.bind {
			bindings = append(bindings, ui.KeyBinding{Key: "mod+s", Label: "Save", Run: func() bool { return true }})
		}
		inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			if a.kitFirst {
				if err := ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "first", surf.Click(func() {}))}).Render(templ.WithChildren(ctx, templ.Raw("First")), w); err != nil {
					return err
				}
			}
			open, end := map[string][2]string{
				"div":   {`<div id="list">`, `</div>`},
				"ul":    {`<ul id="list">`, `</ul>`},
				"table": {`<table id="list"><tbody>`, `</tbody></table>`},
				"p":     {`<p id="list">`, `</p>`},
			}[a.shape][0], map[string]string{"div": `</div>`, "ul": `</ul>`, "table": `</tbody></table>`, "p": `</p>`}[a.shape]
			io.WriteString(w, open)
			for i, r := range a.rows {
				if a.shape == "p" && i > 0 {
					io.WriteString(w, " ")
				}
				if err := row(r, compiled).Render(ctx, w); err != nil {
					return err
				}
			}
			io.WriteString(w, end)
			if a.grouped {
				if compiled {
					return groupC.With(a.group.Key, a.group).Render(ctx, w)
				}
				fmt.Fprintf(w, `<section class="group"><h3>%s</h3>`, a.group.Name)
				for _, r := range a.group.Rows {
					if err := rowView(r).Render(ctx, w); err != nil {
						return err
					}
				}
				io.WriteString(w, `</section>`)
			}
			return nil
		})
		if err := ui.KeyScope(bindings).Render(templ.WithChildren(ctx, inner), w); err != nil {
			return err
		}
		io.WriteString(w, `</div>`)
		return nil
	})
}

var tags = map[string]string{"div": "div", "ul": "li", "table": "tr", "p": "span"}

func newListApp(r *rand.Rand, shape string) *listApp {
	a := &listApp{shape: shape, typed: map[string]string{}}
	for i := range 6 {
		a.rows = append(a.rows, a.randRow(r, fmt.Sprintf("r%d", i), tags[shape]))
	}
	a.group = groupProps{Key: "g", Name: "Group"}
	for i := range a.group.Rows {
		a.group.Rows[i] = a.randRow(r, fmt.Sprintf("g%d", i), "div")
	}
	return a
}

func (a *listApp) randRow(r *rand.Rand, key, tag string) rowProps {
	p := rowProps{Tag: tag, Key: key, Title: "Title " + key, App: a}
	a.vary(r, &p)
	return p
}

func (a *listApp) vary(r *rand.Rand, p *rowProps) {
	switch r.IntN(6) {
	case 0:
		p.Title = []string{"Tab", "A longer title", "日本語", "x"}[r.IntN(4)] + " " + p.Key
	case 1:
		p.Badge = []string{"", "3", "new"}[r.IntN(3)]
	case 2:
		p.Tone = []string{"", "text-primary", "bg-muted", "text-muted-foreground"}[r.IntN(4)]
	case 3:
		p.Field = p.Tag == "div" || p.Tag == "li"
		p.Field = p.Field && r.IntN(2) == 0
	case 4:
		p.Kit = (p.Tag == "div" || p.Tag == "li") && r.IntN(2) == 0
	case 5:
		p.Clicks = r.IntN(2) == 0
	}
}

// step changes a at random, the way an app's state moves.
func (a *listApp) step(r *rand.Rand) string {
	switch r.IntN(10) {
	case 0, 1, 2, 3:
		i := r.IntN(len(a.rows))
		a.vary(r, &a.rows[i])
		return "vary " + a.rows[i].Key
	case 4:
		for i := range a.rows {
			a.rows[i].Sel = false
		}
		i := r.IntN(len(a.rows))
		a.rows[i].Sel = true
		return "select " + a.rows[i].Key
	case 5:
		if len(a.rows) > 1 {
			i := r.IntN(len(a.rows))
			k := a.rows[i].Key
			a.rows = slices.Delete(a.rows, i, i+1)
			return "remove " + k
		}
		fallthrough
	case 6:
		k := fmt.Sprintf("n%d", r.IntN(1000))
		for _, x := range a.rows {
			if x.Key == k {
				return "none"
			}
		}
		i := r.IntN(len(a.rows) + 1)
		a.rows = slices.Insert(a.rows, i, a.randRow(r, k, tags[a.shape]))
		return "add " + k
	case 7:
		r.Shuffle(len(a.rows), func(i, j int) { a.rows[i], a.rows[j] = a.rows[j], a.rows[i] })
		return "shuffle"
	case 8:
		switch r.IntN(4) {
		case 0:
			a.keys = !a.keys
			return "keys"
		case 1:
			a.bind = !a.bind
			return "bind"
		case 2:
			a.kitFirst = !a.kitFirst
			return "kit first"
		}
		a.grouped = !a.grouped
		return "grouped"
	}
	i := r.IntN(len(a.group.Rows))
	a.vary(r, &a.group.Rows[i])
	return "vary group " + a.group.Rows[i].Key
}

var runtimeID = regexp.MustCompile(` id="` + regexp.QuoteMeta(rt.IDPrefix) + `[0-9a-z]+"`)

// plain is a document with the runtime's ids taken out: a compiled
// instance's elements are numbered by its key, the view's by place.
func plain(doc string) string { return runtimeID.ReplaceAllString(doc, "") }

// frameOf is what a frame registered, the runtime's ids aside: each id in
// order with its keys and resolved bindings, and its scopes.
func frameOf(s *surf.Surface) string {
	f := rt.Peek.Frame(s)
	var b strings.Builder
	for _, id := range f.Order {
		h := f.On[id]
		fmt.Fprintf(&b, "%s %v %v %v|", id, h.Keys, h.Written, h.Scope != nil)
	}
	fmt.Fprintf(&b, "scopes %d keyscopes %d focus %q", len(f.Scopes), len(f.KeyScopes), f.Focus)
	return b.String()
}

// lockstep is a's view on two surfaces, compiled and plain.
type lockstep struct {
	a      *listApp
	sc, sp *surf.Surface
	rc, rp *surftest.Rig
	t      *testing.T
	step   int
}

func startLockstep(t *testing.T, a *listApp) *lockstep {
	l := &lockstep{a: a, t: t}
	l.sc, l.sp = surf.New("t", a.view(true), ""), surf.New("t", a.view(false), "")
	l.rc, l.rp = surftest.Start(t, l.sc), surftest.Start(t, l.sp)
	l.same("start")
	return l
}

func (l *lockstep) update(what string) {
	l.rc.Update()
	l.rp.Update()
	l.same(what)
}

// same fails the test where the compiled surface's host isn't the plain
// one's: its document, the runtime's ids aside, its fields' values, and
// what the frame registered.
func (l *lockstep) same(what string) {
	l.t.Helper()
	l.step++
	if got, w := plain(l.rc.HTML()), plain(l.rp.HTML()); got != w {
		l.t.Fatalf("step %d (%s): the document isn't a full render's:\ngot  %s\nwant %s", l.step, what, got, w)
	}
	if got, w := frameOf(l.sc), frameOf(l.sp); got != w {
		l.t.Fatalf("step %d (%s): the frame isn't a full render's:\ngot  %s\nwant %s", l.step, what, got, w)
	}
	for _, x := range l.a.rows {
		if x.Field {
			if got, w := l.rc.Value("#in-"+x.Key), l.rp.Value("#in-"+x.Key); got != w {
				l.t.Fatalf("step %d (%s): field %s holds %q, a full render's %q", l.step, what, x.Key, got, w)
			}
		}
	}
}

// both does act on each rig, compiled then plain.
func (l *lockstep) both(what string, act func(r *surftest.Rig)) {
	act(l.rc)
	act(l.rp)
	l.same(what)
}

// A surface of compiled instances, through random steps, gives its host
// the document a full render does, and its frame: the host's document
// and fields, each handler's keys and bindings, and the scopes.
func TestCompiledIsAFullRender(t *testing.T) {
	for _, shape := range []string{"div", "ul", "table", "p"} {
		for seed := range uint64(8) {
			t.Run(fmt.Sprintf("%s/%d", shape, seed), func(t *testing.T) {
				r := rand.New(rand.NewPCG(seed, 7))
				a := newListApp(r, shape)
				l := startLockstep(t, a)
				var kept int
				for range 40 {
					l.update(a.step(r))
					for _, x := range rt.Peek.Instances(l.sc) {
						if strings.HasSuffix(x, ": kept") {
							kept++
						}
					}
					if r.IntN(5) == 0 {
						for _, x := range a.rows {
							if x.Field {
								l.both("type in "+x.Key, func(g *surftest.Rig) { g.Type("#in-"+x.Key, "zz") })
								break
							}
						}
					}
				}
				if kept == 0 {
					t.Errorf("no instance was kept: %v", rt.Peek.Instances(l.sc))
				}
			})
		}
	}
}

// A kept row's handler hears a double click's count: the click's, not
// the frame that made it.
func TestCompiledKeptClicks(t *testing.T) {
	a := newListApp(rand.New(rand.NewPCG(1, 1)), "div")
	for i := range a.rows {
		a.rows[i].Clicks, a.rows[i].Field, a.rows[i].Kit = true, false, false
	}
	l := startLockstep(t, a)
	a.rows[0].Title = "changed"
	l.update("title")
	a.counts = nil
	l.rc.DoubleClick("#row-r3")
	if fmt.Sprint(a.counts) != "[1 2]" {
		t.Fatalf("a kept row heard %v", a.counts)
	}
}

// A field whose user's edit its props don't take is set back to its
// props' value, with props that didn't change, as a full render does.
func TestCompiledFieldReset(t *testing.T) {
	a := newListApp(rand.New(rand.NewPCG(2, 1)), "div")
	a.rows[2].Field = true
	l := startLockstep(t, a)
	l.both("type", func(g *surftest.Rig) { g.Type("#in-r2", "typed") })
	if v := l.rc.Value("#in-r2"); v != a.rows[2].Title {
		t.Fatalf("the field holds %q, not its props' %q", v, a.rows[2].Title)
	}
}

// A kept row's kit control keeps its KeyScope where a control before it,
// outside, comes and goes, and where the scope's keys change.
func TestCompiledKitAfterAnotherComes(t *testing.T) {
	a := newListApp(rand.New(rand.NewPCG(3, 1)), "div")
	a.bind = true
	for i := range a.rows {
		a.rows[i].Kit = true
	}
	l := startLockstep(t, a)
	for _, x := range []string{"first", "first", "bind", "first", "bind", "keys"} {
		switch x {
		case "first":
			a.kitFirst = !a.kitFirst
		case "bind":
			a.bind = !a.bind
		case "keys":
			a.keys = !a.keys
		}
		l.update(x)
		if got, w := dataKeysOf(l.rc.HTML()), dataKeysOf(l.rp.HTML()); got != w {
			t.Fatalf("after %s: data-keys %s, a full render's %s", x, got, w)
		}
	}
}

func dataKeysOf(doc string) string {
	re := regexp.MustCompile(`id="([^"]+)"[^>]*data-keys="([^"]*)"`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(doc, -1) {
		if !strings.HasPrefix(m[1], rt.IDPrefix) {
			out = append(out, m[1]+"="+m[2])
		}
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

// A List of compiled rows: each row's item registers with the list where
// the row is kept, and the runtime writes the list's tab stop, so keys,
// clicks and the stop are a full render's.
type listRowProps struct {
	Key, Title string
	Sel, Off   bool
	App        *pickApp
}

type pickApp struct {
	sel    int
	titles []string
	off    int
}

var listRowC = surf.Compile(listRowView)

func listRowView(p listRowProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := templ.Attributes{"id": "it-" + p.Key}
		if p.Off {
			a["disabled"] = true
		}
		body := templ.Raw(templ.EscapeString(p.Title))
		if _, err := io.WriteString(w, `<div class="row">`); err != nil {
			return err
		}
		if err := ui.ListItem(ui.ListItemProps{Selected: p.Sel, Attrs: a}).Render(templ.WithChildren(ctx, body), w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

func (a *pickApp) view(compiled bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		rows := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for i, t := range a.titles {
				p := listRowProps{Key: fmt.Sprint(i), Title: t, Sel: i == a.sel, Off: i == a.off, App: a}
				c := listRowView(p)
				if compiled {
					c = listRowC.With(p.Key, p)
				}
				if err := c.Render(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})
		return ui.List(ui.ListProps{OnChange: func(i int) { a.sel = i }, Attrs: templ.Attributes{"id": "picks", "aria-label": "Picks"}}).Render(templ.WithChildren(ctx, rows), w)
	})
}

func TestCompiledListRows(t *testing.T) {
	a := &pickApp{sel: 1, off: 4, titles: []string{"a", "b", "c", "d", "e", "f"}}
	sc, sp := surf.New("t", a.view(true), ""), surf.New("t", a.view(false), "")
	rc, rp := surftest.Start(t, sc), surftest.Start(t, sp)
	// Each step acts on both surfaces from the same state.
	var sels [2]int
	act := func(what string, fn func(r *surftest.Rig)) {
		t.Helper()
		a.sel = sels[0]
		fn(rc)
		sels[0] = a.sel
		a.sel = sels[1]
		fn(rp)
		sels[1] = a.sel
		if sels[0] != sels[1] {
			t.Fatalf("%s: compiled selects %d, a full render %d", what, sels[0], sels[1])
		}
		if got, w := plain(rc.HTML()), plain(rp.HTML()); got != w {
			t.Fatalf("%s: the document isn't a full render's:\ngot  %s\nwant %s", what, got, w)
		}
		if got, w := frameOf(sc), frameOf(sp); got != w {
			t.Fatalf("%s: the frame isn't a full render's:\ngot  %s\nwant %s", what, got, w)
		}
	}
	sels = [2]int{1, 1}
	key := func(k string) func(r *surftest.Rig) { return func(r *surftest.Rig) { r.Key(k) } }
	act("start", func(*surftest.Rig) {})
	act("click c", func(r *surftest.Rig) { r.Click("#it-2") })
	if sels[0] != 2 {
		t.Fatalf("selects %d, not 2", sels[0])
	}
	act("down", key("arrow_down"))
	act("down past the disabled", key("arrow_down"))
	if sels[0] != 5 {
		t.Fatalf("selects %d, not 5", sels[0])
	}
	act("end", key("end"))
	act("home", key("home"))
	if sels[0] != 0 {
		t.Fatalf("selects %d, not 0", sels[0])
	}
	act("retitle", func(r *surftest.Rig) { a.titles[3] = "D"; r.Update() })
	act("down", key("arrow_down"))
	act("app selects", func(r *surftest.Rig) { a.sel = 5; r.Update() })
	// The keyboard stays on b, which the app's selection doesn't move.
	act("up", key("arrow_up"))
	if sels[0] != 0 {
		t.Fatalf("selects %d, not 0", sels[0])
	}
	act("a row goes", func(r *surftest.Rig) {
		if len(a.titles) == 6 {
			a.titles = a.titles[:5]
		}
		r.Update()
	})
	act("click a", func(r *surftest.Rig) { r.Click("#it-0") })
	act("down", key("arrow_down"))
	var kept int
	for _, x := range rt.Peek.Instances(sc) {
		if strings.HasSuffix(x, ": kept") {
			kept++
		}
	}
	if kept == 0 {
		t.Errorf("no row was kept: %v", rt.Peek.Instances(sc))
	}
}
