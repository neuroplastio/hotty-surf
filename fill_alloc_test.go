package surf_test

import (
	"io"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/net/html"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// No step of a compiled surface takes a whole slab of nodes: a fill
// parses into slabs sized to its own markup, the first fills after the
// document as the later ones. With the kit's stylesheet, the case the
// slabs once cost 20 KB more in. Its first step warms more than the
// slabs, as on any surface.
func TestFirstFillsAfterADocument(t *testing.T) {
	for _, sc := range scenes() {
		if sc.name != "hover" && sc.name != "label" {
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			v, step := sc.view(true)
			s := surf.New("z", v, ui.Stylesheet())
			s.Attach(io.Discard, hotty.Caps{})
			if _, err := s.Document(); err != nil {
				t.Fatal(err)
			}
			var steps []uint64
			for i := range 12 {
				var a, b runtime.MemStats
				runtime.ReadMemStats(&a)
				step(i)
				if err := s.Update(); err != nil {
					t.Fatal(err)
				}
				runtime.ReadMemStats(&b)
				steps = append(steps, b.TotalAlloc-a.TotalAlloc)
			}
			later := max(steps[9], steps[10], steps[11])
			t.Logf("bytes a step: %v", steps)
			whole := uint64(512 * unsafe.Sizeof(html.Node{}))
			for i, n := range steps {
				if n >= whole {
					t.Errorf("step %d takes %d B, a whole slab of nodes (%d B) or more", i, n, whole)
				}
			}
			if first := steps[1] + steps[2] + steps[3]; first > 3*later+24<<10 {
				t.Errorf("the first three steps take %d B, against at most %d for a later one", first, later)
			}
		})
	}
}
