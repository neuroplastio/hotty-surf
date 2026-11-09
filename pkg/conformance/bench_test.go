package conformance

import (
	"strconv"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
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

// page is a long synthetic page of n paragraphs, each a wrapping line of
// words and a chip, which a model gives as its tree.
type page struct{ n int }

func (p page) Render() *surfui.Node {
	var kids []*surfui.Node
	for i := range p.n {
		kids = append(kids, surfui.Stack(
			surfui.Heading("Section "+strconv.Itoa(i)),
			surfui.Line(
				surfui.Text(strings.Repeat("a paragraph of prose that wraps over rows, ", 6)),
				surfui.Chip("tag"),
				surfui.Text(strings.Repeat("and goes on after the chip. ", 4)),
			).Class("u-wrap"),
		).Key("s"+strconv.Itoa(i)))
	}
	return surfui.Surface(surfui.Stack(kids...))
}

func (page) Do(surfui.Action, surfrun.Event) {}

// pageWidths are what a site's prerender draws a page at.
var pageWidths = []int{40, 60, 80, 100, 120, 160}

// BenchmarkFullDraw is a long page drawn to its full height (Draw at
// rows 0) at six widths: cold, a renderer new for each page as a
// prerender has it, and cached, the same tree again, which is a lookup
// (make bench; not in check).
func BenchmarkFullDraw(b *testing.B) {
	show := func() *cells.Renderer {
		r := cells.New(surfrun.NewSurface(page{n: 200}, nil).Cells())
		r.Show()
		return r
	}
	b.Run("cold", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r := show()
			for _, w := range pageWidths {
				r.Draw(w, 0)
			}
		}
	})
	b.Run("cached", func(b *testing.B) {
		r := show()
		for _, w := range pageWidths {
			r.Draw(w, 0)
		}
		b.ReportAllocs()
		for b.Loop() {
			for _, w := range pageWidths {
				r.Draw(w, 0)
			}
		}
	})
	b.Run("measure", func(b *testing.B) {
		r := show()
		for _, w := range pageWidths {
			r.Measure(w)
		}
		b.ReportAllocs()
		for b.Loop() {
			for _, w := range pageWidths {
				r.Measure(w)
			}
		}
	})
}
