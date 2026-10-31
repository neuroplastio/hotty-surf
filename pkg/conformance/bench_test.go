package conformance

import (
	"strconv"
	"testing"
)

// benchBoard is a board of n notes, with its panel, which renders with
// no errors.
func benchBoard(n int) *Board {
	b := &Board{Title: "Bench", Ratio: 0.5, Panel: &InfoPanel{Open: true, Title: "Details"}}
	for i := range n {
		id := "n" + strconv.Itoa(i)
		b.Notes = append(b.Notes, &Note{ID: id, Text: "note " + id, Level: "warn", Unread: i%3 == 0, Tags: []string{"a", "b"}})
	}
	return b
}

// BenchmarkRender is a render of a 50-note board in each mode: the cost
// of the tree, not of the diff or the write (make bench; not in check).
func BenchmarkRender(b *testing.B) {
	for _, m := range modes {
		b.Run(m.name, func(b *testing.B) {
			v := m.view(benchBoard(50))
			b.ReportAllocs()
			for b.Loop() {
				v.Render(nil)
			}
		})
	}
}
