//go:build unix

package surfrun

import (
	"bytes"
	"os"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/neuroplastio/hotty-go/hottyterm"
)

// A SIGWINCH is the window's new size, which Sizes delivers to Loop; once
// the Term closes, a SIGWINCH is nothing to it.
func TestResizeOnSIGWINCH(t *testing.T) {
	var cols atomic.Int64 // the window's, as the signal's goroutine reads it
	cols.Store(90)
	term := hottyterm.New(bytes.NewReader(nil), &bytes.Buffer{}, "t", func() (int, int) { return int(cols.Load()), 40 }, nil)
	onResize(term)
	cols.Store(180)
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	if s := <-term.Sizes(); s != (hottyterm.Size{Cols: 180, Rows: 40}) {
		t.Errorf("after SIGWINCH: %+v, want 180×40", s)
	}
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}
}
