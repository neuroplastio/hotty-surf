package surfrun_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// tipper draws a row with a tooltip while tips is set, and a button that
// takes it away.
type tipper struct {
	tips  bool
	count int
	heard []string
}

func (m *tipper) Render() *surfui.Node {
	row := surfui.Row(surfui.Text("agent").Key("name"), surfui.Chip("p-3").Key("pane")).Key("row")
	if m.tips {
		row.Tip(surfui.Stack(
			surfui.Row(surfui.Dot().Tone(surfui.OK), surfui.Text("claude-2").Bold()).Gap(1),
			surfui.Text("Waiting for you, "+strings.Repeat("and waiting, ", m.count)+"in p-3").Muted().Wrap(),
		))
	}
	return surfui.Surface(row, surfui.Button("Go").Key("go").OnClick(func() { m.tips = !m.tips }))
}

func (m *tipper) Do(act surfui.Action, ev surfrun.Event) {
	m.heard = append(m.heard, ev.Kind)
	if f, ok := act.(func()); ok {
		f()
	}
}

// A surface asks for hover while its view has a tooltip: Show's frame
// says so, and an Update's only when that changes. Tip finds the row's
// from its children, sized for a host and drawn in cells, at most
// surfui.TipCols wide and wrapped; a stale id finds none, and a hover
// reaches no handler.
func TestTipHoverAndSize(t *testing.T) {
	m := &tipper{}
	s := surfrun.NewSurface(m, nil)
	if f := s.Show(); f.Hover != hotty.Off {
		t.Errorf("no tooltip: Show's hover %v, want off", f.Hover)
	}
	if _, ok := s.Tip(surfrun.TipAt{ID: "r.krow.kname"}); ok {
		t.Error("a tooltip where there is none")
	}
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kgo"})
	if f := s.Update(); f.Hover != hotty.On {
		t.Errorf("a tooltip came: hover %v, want on", f.Hover)
	}
	if f := s.Update(); f.Hover != 0 {
		t.Errorf("nothing changed: hover %v, want none", f.Hover)
	}
	tip, ok := s.Tip(surfrun.TipAt{ID: "r.krow.kpane"})
	if !ok || tip.ID != "r.krow" {
		t.Fatalf("the row's tooltip from its chip: %+v, %v", tip, ok)
	}
	if tip.Cols != 25 || tip.Rows != 3 || tip.Cells.Rows != 4 {
		t.Errorf("one line: %d×%d on a host, %d rows in the cells\n%s", tip.Cols, tip.Rows, tip.Cells.Rows, tip.Cells.Text())
	}
	if took, _ := s.Event(hotty.Event{Kind: hotty.EventHover, Target: "r.krow"}); took || len(m.heard) != 1 {
		t.Errorf("a hover went to the model: took %v, heard %q", took, m.heard)
	}

	m.count = 9
	s.Update()
	tip, _ = s.Tip(surfrun.TipAt{ID: "r.krow"})
	if tip.Cols != surfui.TipCols || tip.Rows < 4 {
		t.Errorf("a long line wraps at %d: %d×%d\n%s", surfui.TipCols, tip.Cols, tip.Rows, tip.Cells.Text())
	}
	for _, l := range strings.Split(tip.Cells.Text(), "\n") {
		if w := len([]rune(l)); w > surfui.TipCols {
			t.Errorf("a line %d wide: %q", w, l)
		}
	}

	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kgo"})
	if f := s.Update(); f.Hover != hotty.Off {
		t.Errorf("the tooltip went: hover %v, want off", f.Hover)
	}
	if _, ok := s.Tip(surfrun.TipAt{ID: "r.krow"}); ok {
		t.Error("a tooltip that went is found")
	}
	if _, ok := s.Tip(surfrun.TipAt{ID: "r.gone"}); ok {
		t.Error("a stale id finds a tooltip")
	}
}
