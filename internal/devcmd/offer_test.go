package devcmd

import (
	"testing"

	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/rt"
)

// TestDevWriterIsSeenThrough: the report surface offers its Screen its
// tree through devWriter.
func TestDevWriterIsSeenThrough(t *testing.T) {
	w := devWriter{&cellsView{dev: cells.New(10, 1, nil)}}
	if rt.HostOf(w) == nil {
		t.Error("no host through devWriter")
	}
}
