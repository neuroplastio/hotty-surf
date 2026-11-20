package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/ui"
)

// The cost targets' scenes (api/notes.md, "Compiled components"), with
// made-up data: a nav of 8 workspaces of 6 tabs in a program's row shape
// (a ui.List's items under a ui.KeyScope, each row a drag source with a
// press, a secondary press and an item), a strip of 20 pills, and a box.
// Each runs compiled and as a plain view, on a host and on a Screen.

const nbsp = " "

type navModel struct {
	cur      string
	compiled bool
	rows     []navRow // the app's rows, Sel aside
}

// newNav is a nav of 8 workspaces of 6 tabs.
func newNav(cur string, compiled bool) *navModel {
	m := &navModel{cur: cur, compiled: compiled}
	for ws := range 8 {
		m.rows = append(m.rows, navRow{M: m, Key: fmt.Sprintf("w%d", ws), Title: fmt.Sprintf("workspace %d", ws), Lead: "◆", Edge: "edge-ws"})
		for tab := 1; tab < 7; tab++ {
			p := navRow{M: m, Key: fmt.Sprintf("w%d-t%d", ws, tab), Title: fmt.Sprintf("tab %d of %d", tab, ws), Lead: "●", Indent: 1, Edge: "edge-tab", Dim: tab == 5}
			if tab == 2 {
				p.Origin = "~/made/up/path"
			}
			m.rows = append(m.rows, p)
		}
	}
	return m
}

type navRow struct {
	M      *navModel
	Key    string
	Title  string
	Lead   string
	Origin string
	Indent int
	Sel    bool
	Dim    bool
	Edge   string
}

var navRowC = surf.Compile(navRowView)

func navRowView(p navRow) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		m, key := p.M, p.Key
		id := "row-" + key
		root := surf.On(ctx, id, surf.Drag(func(surf.DragEvent) {}), surf.Drop(func(surf.DragEvent) {}),
			surf.Click(func() { m.cur = key }), surf.Secondary(func(hotty.Cell, hotty.Area) {}))
		fmt.Fprintf(w, `<div class="row %s" id="%s" data-on="%s" data-steps="2"><div class="row-main flex">`, p.Edge, root["id"], root["data-on"])
		item := surf.On(ctx, id+"-item", surf.Click(func() { m.cur = key }), surf.Secondary(func(hotty.Cell, hotty.Area) {}))
		body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintf(w, `<span class="row-head flex"><span class="whitespace-pre">%s</span><span class="row-lead text-muted-foreground">%s</span>`, strings.Repeat(nbsp, p.Indent*2), p.Lead)
			if err := ui.ItemTitle(ui.ItemTitleProps{Class: "truncate"}).Render(templ.WithChildren(ctx, templ.Raw(templ.EscapeString(p.Title))), w); err != nil {
				return err
			}
			io.WriteString(w, `</span>`)
			if p.Origin != "" {
				return ui.ItemSubtitle(ui.ItemSubtitleProps{Class: "text-muted-foreground"}).Render(templ.WithChildren(ctx, templ.Raw(templ.EscapeString(p.Origin))), w)
			}
			return nil
		})
		cls := ""
		if p.Dim {
			cls = "opacity-50"
		}
		if err := ui.ListItem(ui.ListItemProps{Selected: p.Sel, Attrs: item, Class: cls}).Render(templ.WithChildren(ctx, body), w); err != nil {
			return err
		}
		if p.Sel {
			surf.Autofocus(ctx, id+"-item")
		}
		io.WriteString(w, `</div></div>`)
		return nil
	})
}

func navView(m *navModel) templ.Component {
	bindings := []ui.KeyBinding{{Key: "mod+k", Label: "Find", Run: func() bool { return true }}}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		rows := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for _, p := range m.rows {
				p.Sel = p.Key == m.cur
				if m.compiled {
					if err := navRowC.With(p.Key, p).Render(ctx, w); err != nil {
						return err
					}
				} else if err := navRowView(p).Render(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})
		list := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			io.WriteString(w, `<div class="flex flex-col"><div class="overflow-y-auto">`)
			err := ui.List(ui.ListProps{OnChange: func(int) {}, Attrs: templ.Attributes{"id": "nav", "aria-label": "Workspaces"}}).Render(templ.WithChildren(ctx, rows), w)
			io.WriteString(w, `</div></div>`)
			return err
		})
		return ui.KeyScope(bindings).Render(templ.WithChildren(ctx, list), w)
	})
}

type pill struct {
	I     int
	Hot   bool
	Label string
}

var pillC = surf.Compile(pillView)

func pillView(p pill) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		cls := "flex shrink-0 text-(--pill)"
		if p.Hot {
			cls = "flex shrink-0 text-(--pill-hot) bg-(--pill-hover)"
		}
		a := surf.On(ctx, fmt.Sprintf("pill-%d", p.I), surf.Click(func() {}), surf.Hover(func(bool) {}))
		fmt.Fprintf(w, `<span id="%s" data-on="%s" class="%s"><span class="whitespace-pre">%s</span><span class="text-(--ok)">●</span><span class="whitespace-pre">%s</span><span class="truncate font-bold">%s</span><span class="whitespace-pre">%s</span><span class="text-(--pill)">×</span><span class="whitespace-pre">%s</span></span>`,
			a["id"], a["data-on"], cls, nbsp, nbsp, templ.EscapeString(p.Label), nbsp, nbsp)
		return nil
	})
}

type stripModel struct {
	hot      int
	label    string
	compiled bool
}

var pillLabels, pillKeys [20]string

func init() {
	for i := range 20 {
		pillLabels[i], pillKeys[i] = fmt.Sprintf("tab %d", i), fmt.Sprint(i)
	}
}

func stripView(m *stripModel) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<div class="flex bg-(--strip-ground)">`)
		for i := range 20 {
			p := pill{I: i, Hot: i == m.hot, Label: pillLabels[i]}
			if i == 7 {
				p.Label = m.label
			}
			if m.compiled {
				if err := pillC.With(pillKeys[i], p).Render(ctx, w); err != nil {
					return err
				}
			} else if err := pillView(p).Render(ctx, w); err != nil {
				return err
			}
		}
		io.WriteString(w, `</div>`)
		return nil
	})
}

type box struct {
	Title      string
	Cols, Rows int
}

var boxC = surf.Compile(boxView)

func boxView(p box) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		body := p.Rows - 2
		lines := func(s string) string {
			return strings.TrimSuffix(strings.Repeat(s+"\n", max(body, 0)), "\n")
		}
		run := func(n int) string { return strings.Repeat("─", max(n, 0)) }
		_, err := io.WriteString(w, `<div class="flex flex-col text-(--frame-line) bg-(--frame-ground)">`+
			`<div class="flex"><span class="whitespace-pre">╭`+nbsp+`</span>`+
			`<span class="truncate font-bold text-(--frame-title)">`+p.Title+`</span>`+
			`<span class="whitespace-pre">`+nbsp+run(p.Cols-5-len(p.Title))+`╮</span></div>`+
			`<div class="flex"><div class="shrink-0 whitespace-pre">`+lines("│"+nbsp)+`</div>`+
			`<div class="flex flex-col grow"><div class="whitespace-pre">`+lines(strings.Repeat(nbsp, p.Cols-4))+`</div></div>`+
			`<div class="shrink-0 whitespace-pre">`+lines(nbsp+"│")+`</div></div>`+
			`<div class="flex"><span class="whitespace-pre">╰`+run(p.Cols-2)+`╯</span></div></div>`)
		return err
	})
}

type boxModel struct {
	b        box
	compiled bool
}

func boxSceneView(m *boxModel) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<div class="flex">`)
		c := boxView(m.b)
		if m.compiled {
			c = boxC.With("box", m.b)
		}
		if err := c.Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

// scene is one benchmark: a view, what each update changes (step i),
// and the Screen's size.
type scene struct {
	name       string
	view       func(compiled bool) (templ.Component, func(i int))
	cols, rows int
	resize     func(i int) (int, int) // a size for each update, on a Screen
}

func scenes() []scene {
	return []scene{
		{name: "nav", cols: 40, rows: 60, view: func(c bool) (templ.Component, func(int)) {
			m := newNav("w3-t3", c)
			return navView(m), func(i int) { m.cur = [2]string{"w3-t3", "w4-t2"}[i%2] }
		}},
		{name: "hover", cols: 240, rows: 1, view: func(c bool) (templ.Component, func(int)) {
			m := &stripModel{hot: 3, label: "tab 7", compiled: c}
			return stripView(m), func(i int) { m.hot = [2]int{3, 4}[i%2] }
		}},
		{name: "label", cols: 240, rows: 1, view: func(c bool) (templ.Component, func(int)) {
			m := &stripModel{hot: 3, label: "tab 7", compiled: c}
			return stripView(m), func(i int) { m.label = [2]string{"tab 7", "a label"}[i%2] }
		}},
		{name: "box", cols: 40, rows: 8, view: func(c bool) (templ.Component, func(int)) {
			m := &boxModel{b: box{Title: "Title", Cols: 40, Rows: 8}, compiled: c}
			return boxSceneView(m), func(i int) { m.b.Cols, m.b.Rows = [2]int{40, 44}[i%2], [2]int{8, 10}[i%2] }
		}, resize: func(i int) (int, int) { return [2]int{40, 44}[i%2], [2]int{8, 10}[i%2] }},
		{name: "unchanged", cols: 40, rows: 60, view: func(c bool) (templ.Component, func(int)) {
			m := newNav("w3-t3", c)
			return navView(m), func(int) {}
		}},
	}
}

func BenchmarkCompiled(b *testing.B) {
	for _, sc := range scenes() {
		for _, mode := range []string{"plain", "compiled"} {
			b.Run(sc.name+"/host/"+mode, func(b *testing.B) {
				v, step := sc.view(mode == "compiled")
				s := surf.New("z", v, ui.Stylesheet())
				s.Attach(io.Discard, hotty.Caps{})
				if _, err := s.Document(); err != nil {
					b.Fatal(err)
				}
				for i := range 2 {
					step(i)
					s.Update()
				}
				b.ResetTimer()
				for i := range b.N {
					step(i)
					if err := s.Update(); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run(sc.name+"/cells/"+mode, func(b *testing.B) {
				v, step := sc.view(mode == "compiled")
				s := surf.New("z", v, ui.Stylesheet())
				c := cells.New(sc.cols, sc.rows, func(e hotty.Event) { s.Event(e) })
				s.Attach(c, hotty.Caps{})
				doc, err := s.Document()
				if err != nil {
					b.Fatal(err)
				}
				c.Write([]byte(hotty.Doc("z", doc)))
				// A program's grid, which Draw writes into.
				g := make([][]cells.Cell, sc.rows+2)
				for y := range g {
					g[y] = make([]cells.Cell, sc.cols+4)
				}
				c.Draw(g, 0, 0)
				for i := range 2 {
					step(i)
					s.Update()
					c.Draw(g, 0, 0)
				}
				b.ResetTimer()
				for i := range b.N {
					step(i)
					if sc.resize != nil {
						c.Resize(sc.resize(i))
					}
					if err := s.Update(); err != nil {
						b.Fatal(err)
					}
					c.Draw(g, 0, 0)
				}
			})
		}
	}
}

// Which instances each compiled scene keeps, and why the others render.
func TestCompiledScenesKeep(t *testing.T) {
	for _, sc := range scenes() {
		v, step := sc.view(true)
		s := surf.New("z", v, ui.Stylesheet())
		s.Attach(io.Discard, hotty.Caps{})
		if _, err := s.Document(); err != nil {
			t.Fatal(err)
		}
		for i := range 3 {
			step(i)
			if err := s.Update(); err != nil {
				t.Fatal(err)
			}
		}
		why := map[string]int{}
		for _, x := range rt.Peek.Instances(s) {
			why[x[strings.LastIndex(x, ": ")+2:]]++
		}
		t.Logf("%s: %v", sc.name, why)
	}
}
