package cellblock_test

import (
	"testing"

	"neuroplast.io/hotty-surf/pkg/cellblock"
)

// A block's size is its rows', and where its places and presses reach.
func TestSize(t *testing.T) {
	b := cellblock.Block{
		Rows: [][]cellblock.Run{{{Col: 2, Width: 3}}, nil},
		Kids: []cellblock.Place{{Kid: 0, X: 4, Y: 1, W: 4, H: 3}},
		Hits: []cellblock.Hit{{Kid: 1, X: 0, Y: 5, W: 2, H: 1}},
	}
	if b.Height() != 6 || b.Width() != 8 {
		t.Errorf("%dx%d, want 8x6", b.Width(), b.Height())
	}
	if (cellblock.Block{}).Height() != 0 || (cellblock.Block{}).Width() != 0 {
		t.Error("an empty block has a size")
	}
}

func TestRoleAndAttr(t *testing.T) {
	if n, ok := cellblock.AnsiRole(9).Ansi(); !ok || n != 9 {
		t.Errorf("ansi-9: %d, %v", n, ok)
	}
	if _, ok := cellblock.Accent.Ansi(); ok {
		t.Error("accent is not a terminal colour")
	}
	if got := (cellblock.Bold | cellblock.Underline).String(); got != "bold+underline" {
		t.Errorf("%q", got)
	}
}
