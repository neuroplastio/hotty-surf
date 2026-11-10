package surfrun_test

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// grower is margin's composer in a surface a program places itself: a
// title, a field of 3 to 8 rows whose @height it hears, and a button.
type grower struct {
	text    string
	heights []int
}

func (m *grower) Render() *surfui.Node {
	return surfui.Surface(surfui.Stack(
		surfui.Text("Reply"),
		surfui.Field().Key("text").Rows(3, 8).Value(m.text).
			OnHeight(func(e surfrun.Event) { m.heights = append(m.heights, e.Rows) }),
		surfui.Button("Send"),
	).Gap(1)).Density(surfui.Compact)
}

func (m *grower) Do(act surfui.Action, ev surfrun.Event) {
	if f, ok := act.(func(surfrun.Event)); ok {
		f(ev)
	}
}

var cellMetrics = surfui.Metrics{CellW: 10, CellH: 21.25, Scale: 1.6}

// Show asks for fit events, and a placement made from Placement says so
// and carries the rows: estimated first, then the host's.
func TestFitPlacement(t *testing.T) {
	m := &grower{}
	s := surfrun.NewSurface(m, nil)
	s.Size(50, cellMetrics)
	if f := s.Show(); f.Fit != hotty.On {
		t.Errorf("Show's Fit = %v, want On", f.Fit)
	}
	rows, measured := s.Rows()
	if measured || rows != surfui.Rows(s.Tree(), 50, cellMetrics) {
		t.Errorf("Rows = %d, %v; want the estimate %d, unmeasured", rows, measured, surfui.Rows(s.Tree(), 50, cellMetrics))
	}
	p := s.Placement()
	if p.Cols != 50 || p.Rows != rows || p.Fit != hotty.On {
		t.Errorf("Placement = %+v, want %d columns, %d rows, Fit on", p, 50, rows)
	}
	s.Event(hotty.Event{Kind: hotty.EventFit, FitRows: rows + 3})
	if got, measured := s.Rows(); got != rows+3 || !measured {
		t.Errorf("after a fit of %d, Rows = %d, %v", rows+3, got, measured)
	}
	if p := s.Placement(); p.Rows != rows+3 {
		t.Errorf("Placement after a fit has %d rows, want %d", p.Rows, rows+3)
	}
	// A new document is the host's to measure again.
	s.Show()
	if got, measured := s.Rows(); got != rows || measured {
		t.Errorf("after Show, Rows = %d, %v; want the estimate %d", got, measured, rows)
	}
}

// A host's @height is its measure: the field is as tall as the rows the
// document grew by, heard once for each height, and OnFit hears every
// measure.
func TestFitHeight(t *testing.T) {
	m := &grower{}
	s := surfrun.NewSurface(m, nil)
	s.Size(50, cellMetrics)
	var fits []int
	s.OnFit = func(rows int) { fits = append(fits, rows) }
	s.Show()
	base, _ := s.Rows() // the document with the field at its 3 rows
	s.Event(hotty.Event{Kind: hotty.EventFit, FitRows: base + 2})
	s.Event(hotty.Event{Kind: hotty.EventFit, FitRows: base + 2}) // the same measure again
	s.Event(hotty.Event{Kind: hotty.EventFit, FitRows: base + 5})
	s.Event(hotty.Event{Kind: hotty.EventFit, FitRows: base - 1}) // under its least: 3
	if want := []int{5, 8, 3}; !slices.Equal(m.heights, want) {
		t.Errorf("@height heard %v, want %v", m.heights, want)
	}
	if want := []int{base + 2, base + 2, base + 5, base - 1}; !slices.Equal(fits, want) {
		t.Errorf("OnFit heard %v, want %v", fits, want)
	}
	if !s.Fit(base + 3) { // an r=auto reply, which no fit event follows
		t.Errorf("Fit(%d) reports no Update due after a new height", base+3)
	}
	if got := m.heights[len(m.heights)-1]; got != 6 {
		t.Errorf("@height after Fit(%d) = %d, want 6 (%v)", base+3, got, m.heights)
	}
}

// Hold keeps the top still, and scrolls the screen where the surface
// would run off it.
func TestHold(t *testing.T) {
	for _, c := range []struct{ top, now, screen, wantTop, wantScroll int }{
		{5, 4, 24, 5, 0},    // room: the top stays
		{5, 19, 24, 5, 0},   // to the last row
		{5, 22, 24, 2, 3},   // three rows over: the screen scrolls three
		{20, 10, 24, 14, 6}, // a surface placed low grows over the edge
		{2, 40, 24, 0, 2},   // taller than the screen: as far as the top goes
		{3, 0, 24, 3, 0},    // no rows: nothing to hold
	} {
		top, scroll := surfrun.Hold(c.top, c.now, c.screen)
		if top != c.wantTop || scroll != c.wantScroll {
			t.Errorf("Hold(%d, %d, %d) = %d, %d; want %d, %d", c.top, c.now, c.screen, top, scroll, c.wantTop, c.wantScroll)
		}
	}
}
