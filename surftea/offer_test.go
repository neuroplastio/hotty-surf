package surftea

import (
	"testing"

	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/rt"
)

// TestCellsOutIsSeenThrough: a surface in cells offers its Screen its tree
// through cellsOut.
func TestCellsOutIsSeenThrough(t *testing.T) {
	h := &Host{scr: cells.New(10, 2, nil)}
	if rt.HostOf(cellsOut{h}) == nil {
		t.Error("no host through cellsOut")
	}
}
