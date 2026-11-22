package surftui

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

// load is a Screen cols x rows showing markup.
func load(t testing.TB, markup string, cols, rows int) *Screen {
	t.Helper()
	c := New(cols, rows, nil)
	if _, err := c.Write([]byte(hotty.Doc("s", markup))); err != nil {
		t.Fatal(err)
	}
	return c
}

// grid is a grid cols x rows of x.
func grid(cols, rows int, x string) [][]Cell {
	g := make([][]Cell, rows)
	for y := range g {
		g[y] = make([]Cell, cols)
		for i := range g[y] {
			g[y][i] = Cell{Text: x}
		}
	}
	return g
}

// Draw writes what Cells shows into g at col, row, clipped row by row to
// g; a wide character cut by g's edge is a blank; the rest of g is as it
// was.
func TestDrawClips(t *testing.T) {
	c := load(t, `<div class="flex flex-col"><span>ab日本cd</span><span class="text-primary">efgh</span></div>`, 8, 2)
	want := c.Cells()
	for _, at := range [][2]int{{0, 0}, {3, 1}, {-2, 0}, {-3, -1}, {6, 0}, {10, 0}, {0, 3}, {-9, 0}} {
		col, row := at[0], at[1]
		g := grid(10, 3, ".")
		c.Draw(g, col, row)
		for y := range g {
			for x := range g[y] {
				sx, sy := x-col, y-row
				exp := Cell{Text: "."}
				if sy >= 0 && sy < len(want) && sx >= 0 && sx < len(want[sy]) {
					exp = want[sy][sx]
					// A wide character's halves either side of g's edge.
					if x == 0 && sx > 0 && exp.Text == "" {
						exp.Text = " "
					}
					if x == len(g[y])-1 && sx+1 < len(want[sy]) && want[sy][sx+1].Text == "" {
						exp.Text = " "
					}
				}
				if g[y][x] != exp {
					t.Fatalf("at %d,%d: cell %d,%d is %+v, not %+v", col, row, x, y, g[y][x], exp)
				}
			}
		}
	}
}

// Draw counts as Cells for Dirty, and with nothing changed draws the
// same cells.
func TestDrawIsCellsForDirty(t *testing.T) {
	c := load(t, `<div class="flex"><span id="a">one</span><span id="b">two</span></div>`, 12, 1)
	g := grid(12, 1, "")
	c.Draw(g, 0, 0)
	if c.Dirty() {
		t.Fatal("dirty after a Draw")
	}
	first := fmt.Sprint(g)
	c.Draw(g, 0, 0)
	if fmt.Sprint(g) != first {
		t.Fatal("a Draw with nothing changed drew other cells")
	}
	// The same cells by another delta: not dirty.
	c.Write([]byte(hotty.Delta("s", []hotty.Op{hotty.OpAttr("a", "data-x", "1")})))
	if c.Dirty() {
		t.Fatal("dirty after a delta that draws the same")
	}
	c.Write([]byte(hotty.Delta("s", []hotty.Op{hotty.OpText("b", "TWO")})))
	if !c.Dirty() {
		t.Fatal("not dirty after a delta that draws other cells")
	}
	c.Draw(g, 0, 0)
	if c.Dirty() {
		t.Fatal("dirty after a Draw of the change")
	}
	if got := strings.TrimSpace(textOfRow(g[0])); got != "oneTWO" {
		t.Fatalf("drew %q", got)
	}
	if fmt.Sprint(c.Cells()) != fmt.Sprint(g) {
		t.Fatal("Draw and Cells differ")
	}
}

func textOfRow(r []Cell) string {
	var b strings.Builder
	for _, k := range r {
		b.WriteString(k.Text)
	}
	return b.String()
}

// Chrome a program draws every frame, through the deltas a surface sends
// and at new sizes, draws as a Screen given its markup afresh: a dialog's
// frame resized and retitled, a tab strip whose hover moves and whose
// label changes.
func TestDrawScenes(t *testing.T) {
	type step struct {
		markup     string
		ops        []hotty.Op
		cols, rows int
	}
	pills := func(hot int, label string) []hotty.Op {
		var ops []hotty.Op
		for i := range 6 {
			cls := "flex shrink-0 text-(--pill)"
			if i == hot {
				cls = "flex shrink-0 text-(--pill-hot) bg-(--pill-hover)"
			}
			ops = append(ops, hotty.OpAttr(fmt.Sprintf("pill-%d", i), "class", cls))
		}
		return append(ops, hotty.OpText("pill-3-label", label))
	}
	for _, sc := range []struct {
		name  string
		steps []step
	}{
		{"frame", []step{
			{markup: frameScene("Title", 30, 6), cols: 30, rows: 6},
			{markup: frameScene("Title", 34, 8), cols: 34, rows: 8},
			{markup: frameScene("Another title", 34, 8), cols: 34, rows: 8},
			{markup: frameScene("x", 12, 3), cols: 12, rows: 3},
		}},
		{"strip", []step{
			{markup: stripScene(6), cols: 60, rows: 1},
			{ops: pills(2, "tab 3"), cols: 60, rows: 1},
			{ops: pills(3, "a much longer label"), cols: 60, rows: 1},
			{ops: pills(-1, ""), cols: 40, rows: 1},
			{ops: pills(5, "tab 3"), cols: 60, rows: 1},
		}},
	} {
		t.Run(sc.name, func(t *testing.T) {
			c := New(1, 1, nil)
			var markup string
			g := grid(70, 10, "")
			for i, st := range sc.steps {
				c.Resize(st.cols, st.rows)
				if st.markup != "" {
					markup = st.markup
					c.Write([]byte(hotty.Doc("s", markup)))
				} else {
					c.Write([]byte(hotty.Delta("s", st.ops)))
				}
				c.Draw(g, 1, 1)
				fresh := New(st.cols, st.rows, nil)
				fresh.Write([]byte(hotty.Doc("s", render(t, c.body))))
				want := fresh.Cells()
				got := make([][]Cell, st.rows)
				for y := range got {
					got[y] = g[1+y][1 : 1+st.cols]
				}
				sameCells(t, fmt.Sprintf("step %d", i), got, want)
			}
		})
	}
}

const nbsp = " "

// frameScene is a dialog's frame, cols x rows: a rounded edge with a
// title in it, a side each side of a blank body, and its bottom edge.
func frameScene(title string, cols, rows int) string {
	body := rows - 2
	lines := func(s string) string {
		return strings.TrimSuffix(strings.Repeat(s+"\n", max(body, 0)), "\n")
	}
	run := func(n int) string { return strings.Repeat("─", max(n, 0)) }
	return `<div class="flex"><div class="flex flex-col text-(--frame-line) bg-(--frame-ground)">` +
		`<div class="flex"><span class="whitespace-pre">╭` + nbsp + `</span>` +
		`<span class="truncate font-bold text-(--frame-title)">` + title + `</span>` +
		`<span class="whitespace-pre">` + nbsp + run(cols-5-len(title)) + `╮</span></div>` +
		`<div class="flex"><div class="shrink-0 whitespace-pre">` + lines("│"+nbsp) + `</div>` +
		`<div class="flex flex-col grow"><div class="whitespace-pre">` + lines(strings.Repeat(nbsp, cols-4)) + `</div></div>` +
		`<div class="shrink-0 whitespace-pre">` + lines(nbsp+"│") + `</div></div>` +
		`<div class="flex"><span class="whitespace-pre">╰` + run(cols-2) + `╯</span></div></div></div>`
}

// stripScene is a tab strip of n pills in one row.
func stripScene(n int) string {
	var b strings.Builder
	b.WriteString(`<div class="flex bg-(--strip-ground)">`)
	for i := range n {
		fmt.Fprintf(&b, `<span id="pill-%d" class="flex shrink-0 text-(--pill)">`, i)
		b.WriteString(`<span class="whitespace-pre">` + nbsp + `</span><span class="text-(--ok)">●</span><span class="whitespace-pre">` + nbsp + `</span>`)
		fmt.Fprintf(&b, `<span id="pill-%d-label" class="truncate font-bold">tab %d</span>`, i, i)
		b.WriteString(`<span class="whitespace-pre">` + nbsp + `</span><span class="text-(--pill)">×</span><span class="whitespace-pre">` + nbsp + `</span></span>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}
