package cells

import (
	"flag"
	"os"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/internal/stories"
)

var updateLines = flag.Bool("update-lines", false, "write testdata/lines.golden")

// TestLinesGolden holds Lines to what it drew before the Screen read
// colours (G-cells-colour, K28): every story's lines, byte for byte.
// Colour is Cells'; Lines stays attributes only.
func TestLinesGolden(t *testing.T) {
	var b strings.Builder
	for _, st := range stories.All() {
		r := openStory(t, st, 80, 60)
		b.WriteString("== " + st.ID + "\n")
		for _, l := range r.c.Lines() {
			b.WriteString(strings.TrimRight(l, " ") + "\n")
		}
	}
	const path = "testdata/lines.golden"
	if *updateLines {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range min(len(gl), len(wl)) {
			if gl[i] != wl[i] {
				t.Fatalf("Lines changed at line %d:\n got %q\nwant %q", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("Lines changed: %d lines, want %d", len(gl), len(wl))
	}
}
