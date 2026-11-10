package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestProgressBounds(t *testing.T) {
	for _, c := range []struct{ percent, filled int }{{-10, 0}, {0, 0}, {50, 5}, {100, 10}, {110, 10}} {
		m := &model{render: func() *surfui.Node { return surfui.Surface(surfui.Progress(c.percent).Width(10)) }}
		r := show(m)
		screen := r.Draw(12, 2)
		if got := screen.Text(); strings.Count(got, "█") != c.filled || strings.Count(got, "░") != 10-c.filled {
			t.Errorf("%d: %q", c.percent, got)
		}
		if len(screen.Hits) != 0 {
			t.Fatal("progress is not an input")
		}
	}
}

func TestProgressGoldens(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(
			surfui.Progress(0).Label("Starting"),
			surfui.Progress(45).Label("Download"),
			surfui.Progress(100).Label("Complete"),
		))
	}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	goldenText(t, "progress.html", surfui.HTML(s.Tree())+"\n")
	golden(t, "progress", show(m).Draw(20, 4))
}
