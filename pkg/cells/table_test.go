package cells_test

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// threads is margin's threads view with made-up data: a state glyph, a
// document column capped at 40%, a thread that fills, and an age that
// sits at the end under a sort chevron.
func threads(sel int) *surfui.Node {
	row := func(i int, st, doc, sum, age string) *surfui.Node {
		return surfui.Item(surfui.Glyph(st), surfui.Text(doc), surfui.Text(sum), surfui.Text(age)).Key(doc).Selected(i == sel)
	}
	return surfui.Surface(surfui.Table(
		surfui.Col("").Width(2),
		surfui.Col("document").ColMax(0, 40),
		surfui.Col("thread").FillCol(),
		surfui.Col("age").Align("end").Muted().Sort("descending"),
		row(0, "●", "README.md", "Is this still true after the rename of everything?", "3h"),
		row(1, "✓", "docs/surf.md", "Typo", "2d"),
		row(2, "●", "a.go", "Short", "10m"),
	)).Density(surfui.Compact)
}

// A table's cells line up across its rows: an auto column is as wide as
// its widest cell, a fill column takes the room left and is cut with …,
// an end-aligned column sits at the end, and the header has one rule under
// it. With no row selected the rows are split by hairlines; with one,
// they are not, and it has the list's selected fill.
func TestTableThreads(t *testing.T) {
	for _, c := range []struct {
		name string
		sel  int
	}{{"table-ruled", -1}, {"table-threads", 1}} {
		m := &model{render: func() *surfui.Node { return threads(c.sel) }}
		r := show(m)
		s := r.Draw(60, 10)
		golden(t, c.name, s)
		text := s.Text()
		if !strings.Contains(text, "Is this still true after the renam…") {
			t.Errorf("%s: the fill column isn't cut with …:\n%s", c.name, text)
		}
		if got := strings.Count(text, "─") > 60; got != (c.sel < 0) {
			t.Errorf("%s: hairlines are %v:\n%s", c.name, got, text)
		}
	}
}

// A narrow table cuts the fill column first and keeps the others whole.
func TestTableNarrow(t *testing.T) {
	m := &model{render: func() *surfui.Node { return threads(1) }}
	r := show(m)
	s := r.Draw(34, 10)
	golden(t, "table-narrow", s)
	for _, want := range []string{"README.md", "docs/surf.md", "2d"} {
		if !strings.Contains(s.Text(), want) {
			t.Errorf("narrow lost %q:\n%s", want, s.Text())
		}
	}
}

// A table with no rows shows its empty, and keeps its header.
func TestTableEmpty(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Table(surfui.Col("document"), surfui.Col("thread").FillCol(),
			surfui.Empty(surfui.Text("No threads yet.").Muted()))).Density(surfui.Compact)
	}}
	s := show(m).Draw(40, 6)
	golden(t, "table-empty", s)
	if !strings.Contains(s.Text(), "document") || !strings.Contains(s.Text(), "No threads yet.") {
		t.Errorf("empty:\n%s", s.Text())
	}
}

func many(n, sel int) *surfui.Node {
	t := surfui.Table(surfui.Col("name"), surfui.Col("size").Align("end"))
	for i := range n {
		t.Add(surfui.Item(surfui.Text(fmt.Sprintf("file-%02d.go", i)), surfui.Text(fmt.Sprint(i*7))).Key(fmt.Sprint(i)).Selected(i == sel))
	}
	t.Add(surfui.Text(surfui.RangeText()).Muted().Key("range"))
	return surfui.Surface(t.Height(8).Scroll()).Density(surfui.Compact)
}

// A table with a Height and Scroll scrolls its rows and not its header,
// its bar in the last column (│ track, ┃ thumb), keeps the selected row in
// view where the document scrolls, and tells its range.
func TestTableScroll(t *testing.T) {
	sel := 0
	m := &model{render: func() *surfui.Node { return many(20, sel) }}
	sf := surfrun.NewSurface(m, nil)
	sf.Scroll = hotty.ScrollVertical
	r := cells.New(sf.Cells())
	r.Scroll = hotty.ScrollVertical
	r.Show()
	first := r.Draw(30, 12)
	golden(t, "table-scroll", first)
	text := first.Text()
	if !strings.Contains(text, "name") || !strings.Contains(text, "file-00.go") || !strings.Contains(text, "1–5 of 20") {
		t.Errorf("top:\n%s", text)
	}
	if !strings.Contains(text, "┃") {
		t.Errorf("no scrollbar:\n%s", text)
	}
	sel = 9
	r.Update()
	s := r.Draw(30, 12)
	golden(t, "table-scroll-selected", s)
	text = s.Text()
	if !strings.Contains(text, "name") || !strings.Contains(text, "file-09.go") || strings.Contains(text, "file-00.go") || !strings.Contains(text, "6–10 of 20") {
		t.Errorf("selected row 9:\n%s", text)
	}
	// Up again within the view: it stays where it is, and the range says
	// what is shown, which the tree (that puts the window by the selection
	// alone) can't know.
	sel = 7
	r.Update()
	text = r.Draw(30, 12).Text()
	if !strings.Contains(text, "file-05.go") || !strings.Contains(text, "6–10 of 20") {
		t.Errorf("selected row 7:\n%s", text)
	}
}

// surfui measures an auto column's cells without the cells' layout, which
// it can't import: what it says is what the layout gives the same cell,
// at every density, for the kinds a cell holds.
func TestTableMeasureIsTheLayouts(t *testing.T) {
	kinds := map[string]func() *surfui.Node{
		"text":    func() *surfui.Node { return surfui.Text("document.md") },
		"wide":    func() *surfui.Node { return surfui.Text("日本語のテキスト") },
		"glyph":   func() *surfui.Node { return surfui.Glyph("●") },
		"chip":    func() *surfui.Node { return surfui.Chip("block gone") },
		"key":     func() *surfui.Node { return surfui.Key("ctrl+k") },
		"line":    func() *surfui.Node { return surfui.Line(surfui.Text("Typo"), surfui.Chip("warn")) },
		"row":     func() *surfui.Node { return surfui.Row(surfui.Glyph("✓"), surfui.Text("done")).Gap(2) },
		"heading": func() *surfui.Node { return surfui.Heading("Title") },
	}
	for name, mk := range kinds {
		for _, d := range []surfui.Density{surfui.Dense, surfui.Compact} {
			n := mk()
			m := &model{render: func() *surfui.Node { return surfui.Surface(surfui.Row(n.Key("c"))).Density(d) }}
			r := show(m)
			r.Draw(80, 4)
			_, _, w, _, ok := r.BoxOf("c")
			if !ok {
				t.Fatalf("%s: no box", name)
			}
			if got := surfui.NaturalWidth(mk(), d); got != w {
				t.Errorf("%s at %s: surfui measures %d, the layout gives %d", name, d, got, w)
			}
		}
	}
}
