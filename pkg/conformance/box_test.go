package conformance

import (
	"fmt"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

// In the cells, the full draw of a board lays every element out where
// Box says, and the two modes (interpreted and compiled) lay it out
// alike: each element's box is the same, and covers every cell a press
// on the element reaches.
func TestBoxInBothModes(t *testing.T) {
	var boxes [2]map[string]string
	for i, m := range modes {
		var heard []string
		s := surfrun.NewSurface(surflive.New(hear(m.view(benchBoard(12)), &heard)), nil)
		r := cells.New(s.Cells())
		r.Show()
		h := r.Measure(CellsCols)
		scr := r.Draw(CellsCols, 0)
		if scr.Rows != h || h <= CellsRows {
			t.Errorf("%s: Measure %d, full draw %d rows, a screen is %d", m.name, h, scr.Rows, CellsRows)
		}
		boxes[i] = map[string]string{}
		for id := range r.Index() {
			x, y, w, bh, ok := r.Box(id)
			if !ok {
				continue // not drawn: a hidden element
			}
			if x < 0 || y < 0 || x+w > CellsCols || y+bh > h {
				t.Errorf("%s: %s at %d,%d %dx%d is outside the %dx%d document", m.name, id, x, y, w, bh, CellsCols, h)
			}
			boxes[i][id] = fmt.Sprint(x, y, w, bh)
		}
		for _, hit := range scr.Hits {
			x, y, w, bh, ok := r.Box(hit.ID)
			if !ok || hit.X < x || hit.Y < y || hit.X+hit.W > x+w || hit.Y+hit.H > y+bh {
				t.Errorf("%s: hit %s %d,%d %dx%d is not in its box %d,%d %dx%d (%v)", m.name, hit.ID, hit.X, hit.Y, hit.W, hit.H, x, y, w, bh, ok)
			}
		}
		if len(boxes[i]) < 50 {
			t.Errorf("%s: only %d elements have a box", m.name, len(boxes[i]))
		}
	}
	for id, a := range boxes[0] {
		if b, ok := boxes[1][id]; ok && a != b {
			t.Errorf("%s: %s interpreted, %s compiled", id, a, b)
		}
	}
}
