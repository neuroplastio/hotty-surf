package cells_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// barBlock is a custom element's block: a bar of w cells filled to a third,
// its own colours on the first run and the element's on the rest, and a
// button the block places on a row of its own, at the right.
func barBlock(calls *int) cells.BlockFunc {
	return func(w int, in cells.Style) cells.Block {
		*calls++
		w = min(w, 12)
		fill := w / 3
		row := []cells.Run{
			cells.NewRun(0, strings.Repeat("█", fill), cells.Accent, "", 0),
			cells.NewRun(fill, strings.Repeat("░", w-fill), "", "", 0),
		}
		return cells.Block{
			Rows: [][]cells.Run{row, nil},
			Kids: []cells.Place{{Kid: 0, X: w - 4, Y: 1, W: 4, H: 1}},
		}
	}
}

// A custom element takes the width the layout gives it, and its block's
// height; the child its block places is the tree's own node, which takes
// a press and the keyboard where the block put it, and one it leaves out
// isn't drawn.
func TestCellsBlock(t *testing.T) {
	calls := 0
	clicked := false
	m := &model{render: func() *surfui.Node {
		custom := surfui.El("div",
			surfui.Button("Go").OnClick(func() { clicked = true }),
			surfui.Button("Unplaced"),
		).Cells(barBlock(&calls))
		return surfui.Surface(surfui.Stack(surfui.Text("head"), custom, surfui.Text("tail")))
	}}
	r := show(m)
	s := r.Draw(20, 6)
	golden(t, "block", s)
	if strings.Contains(s.Text(), "Unplaced") {
		t.Errorf("an unplaced child is drawn:\n%s", s.Text())
	}
	if n := calls; n == 0 {
		t.Fatal("the block func never ran")
	}
	// The same width again, on the same nodes, asks nothing of the func.
	n := calls
	r.Draw(20, 6)
	if calls != n {
		t.Errorf("a second draw at the same width made %d more blocks", calls-n)
	}
	// A press on the placed child is a press on it.
	x, y, ok := r.Where("r.0.1.0")
	if !ok || y != 2 || x != 8 {
		t.Fatalf("the placed button is at %d,%d (%v), want 8,2", x, y, ok)
	}
	r.Press(x, y, 0)
	r.Release(x, y, 0)
	if r.Focused() != "r.0.1.0" || !clicked {
		t.Errorf("a press on the placed button: focus %q, clicked %v", r.Focused(), clicked)
	}
	if hits := r.Draw(20, 6).Hits; !slices.ContainsFunc(hits, func(h cells.Hit) bool { return h.ID == "r.0.1.0" }) {
		t.Errorf("hits %v", hits)
	}
}

// A block's size is the layout's: at a narrower width it is asked again,
// and its rows are what the Stack's height is.
func TestCellsBlockWidth(t *testing.T) {
	var widths []int
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(surfui.El("div").Cells(func(w int, _ cells.Style) cells.Block {
			widths = append(widths, w)
			return cells.Block{Rows: [][]cells.Run{{cells.NewRun(0, strings.Repeat("x", w), "", "", 0)}}}
		}), surfui.Text("below")))
	}}
	s := show(m).Draw(7, 3)
	if got := strings.TrimRight(s.Text(), "\n "); got != "xxxxxxx\nbelow" {
		t.Errorf("%q", got)
	}
	if !slices.Contains(widths, 7) {
		t.Errorf("asked for widths %v, never 7", widths)
	}
}

// A block's height is the tree's: Measure and Draw(cols, 0) agree on it
// at each width (a narrower one wraps the block into more rows), Box
// covers the custom element by those rows, and a layout the renderer
// keeps (the same tree at the same width) asks the func nothing more.
func TestCellsBlockMeasured(t *testing.T) {
	calls := 0
	m := &model{render: func() *surfui.Node {
		wrap := surfui.El("div").Key("wrapped").Cells(func(w int, _ cells.Style) cells.Block {
			calls++
			var rows [][]cells.Run
			for left := 24; left > 0; left -= w {
				rows = append(rows, []cells.Run{cells.NewRun(0, strings.Repeat("=", min(w, left)), "", "", 0)})
			}
			return cells.Block{Rows: rows}
		})
		return surfui.Surface(surfui.Stack(surfui.Text("head"), wrap, surfui.Text("tail")))
	}}
	r := show(m)
	for _, c := range []struct{ cols, rows int }{{24, 1}, {10, 3}, {6, 4}} {
		h := r.Measure(c.cols)
		s := r.Draw(c.cols, 0)
		if h != s.Rows || h != c.rows+2 {
			t.Errorf("at %d cells: Measure %d, Draw %d rows, want %d", c.cols, h, s.Rows, c.rows+2)
		}
		x, y, w, bh, ok := r.BoxOf("wrapped")
		if !ok || x != 0 || y != 1 || w != c.cols || bh != c.rows {
			t.Errorf("at %d cells: BoxOf %d,%d %dx%d (%v), want 0,1 %dx%d", c.cols, x, y, w, bh, ok, c.cols, c.rows)
		}
		if first, last, ok := r.RowsOf("wrapped"); !ok || first != 1 || last != c.rows {
			t.Errorf("at %d cells: rows %d to %d", c.cols, first, last)
		}
		if got := strings.Count(s.Text(), "="); got != 24 {
			t.Errorf("at %d cells: %d strokes drawn, want 24:\n%s", c.cols, got, s.Text())
		}
	}
	n := calls
	r.Measure(10)
	r.Draw(10, 0)
	r.Draw(10, 0)
	if calls != n {
		t.Errorf("kept layouts made %d more blocks", calls-n)
	}
}
