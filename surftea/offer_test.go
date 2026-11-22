package surftea

import (
	"testing"

	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// TestCellsOutIsSeenThrough: a surface in cells offers its Screen its tree
// through cellsOut.
func TestCellsOutIsSeenThrough(t *testing.T) {
	h := &Host{scr: surftui.New(10, 2, nil)}
	if rt.HostOf(cellsOut{h}) == nil {
		t.Error("no host through cellsOut")
	}
}
