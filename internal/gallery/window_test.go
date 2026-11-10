package gallery

import (
	"strconv"
	"testing"
)

// A window Window(cols) wide draws the canvas cols wide with both side
// columns whole and nothing left out, and a cell narrower narrows a side
// column: what internal/cmd/stories -gallery opens each story in.
func TestWindow(t *testing.T) {
	g := New()
	g.Pick("w-button") // a story with controls: a story with none has no column of them
	for _, cols := range []int{20, 30, 50, 80, 120} {
		g.SetWidth(strconv.Itoa(cols))
		g.Resize(Window(cols), 54)
		if g.Fit() != "" || g.SideWidth() != sideCols || g.AsideWidth() != asideCols {
			t.Errorf("a window %d wide round a %d-cell canvas: the columns %d and %d (%q), not %d and %d",
				Window(cols), cols, g.SideWidth(), g.AsideWidth(), g.Fit(), sideCols, asideCols)
		}
		g.Resize(Window(cols)-1, 54)
		if g.SideWidth()+g.AsideWidth() != sideCols+asideCols-1 {
			t.Errorf("a window %d wide round a %d-cell canvas: the columns %d and %d, not a cell narrower",
				Window(cols)-1, cols, g.SideWidth(), g.AsideWidth())
		}
	}
}
