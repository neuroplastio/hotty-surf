package devcmd

import (
	"testing"

	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// TestDevWriterIsSeenThrough: the report surface offers its Screen its
// tree through devWriter.
func TestDevWriterIsSeenThrough(t *testing.T) {
	w := devWriter{&cellsView{dev: surftui.New(10, 1, nil)}}
	if rt.HostOf(w) == nil {
		t.Error("no host through devWriter")
	}
}
