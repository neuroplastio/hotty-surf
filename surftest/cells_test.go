package surftest_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// picker is a Select and a count of its picks.
type picker struct {
	level string
	picks int
}

func (m *picker) Render() *surfui.Node {
	return surfui.Surface(surfui.Row(surfui.Text("Level"), surfui.Select(m.level, "debug", "info", "warn").OnChange("pick")).Gap(1))
}

func (m *picker) Do(act surfui.Action, ev surfrun.Event) {
	m.level = ev.Value
	m.picks++
}

// In a terminal that is no host the tree is drawn in cells, and a
// Select's option is picked there as the user picks it.
func TestStartCells(t *testing.T) {
	m := &picker{level: "info"}
	a := surftest.StartCells(t, m, surfrun.Options{})
	if s := a.Cells().Text(); !strings.Contains(s, "Level  info") {
		t.Fatalf("the cells:\n%s", s)
	}
	a.Choose("r.0.1", "warn")
	a.Sync()
	var level string
	a.Call(func() { level = m.level })
	if level != "warn" {
		t.Errorf("picked %q", level)
	}
	if s := a.Cells().Text(); !strings.Contains(s, "Level  warn") {
		t.Errorf("the cells after the pick:\n%s", s)
	}
}
