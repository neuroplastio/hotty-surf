//go:build !js

package run

import (
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// TestCellsHostIsSeenThrough: the surface on a cellsHost offers the
// Screen its tree, through the host's writer, which still notes a refused
// delta (TestCellsRefusedDeltaResends).
func TestCellsHostIsSeenThrough(t *testing.T) {
	c := newCellsHost(surf.New("t", templ.Raw("<p>x</p>"), ""), 10, 2)
	if rt.HostOf(c) == nil {
		t.Error("no host through the cellsHost")
	}
}
