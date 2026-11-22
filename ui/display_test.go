package ui_test

import (
	"math"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// B2's faults: a Chart whose data can't be drawn.
func TestDisplayFaults(t *testing.T) {
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", templ.Join(
		ui.Chart(ui.ChartProps{Type: "area"}),
		ui.Chart(ui.ChartProps{Type: "pie", Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{-1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a", "b"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "n", Values: []float64{math.NaN()}}}}),
		ui.Chart(ui.ChartProps{Type: "pie", Labels: []string{"a"}}),
	), ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{
		`ui.Chart: no Type "area"`,
		"ui.Chart: a pie's slice is -1", `ui.Chart: series "s" has 1 values for 2 labels`, `ui.Chart: series "n" has NaN`,
		"ui.Chart: a pie has one series, not 0",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}

// A Chart's plot: a bar for each value of each series, a polyline for
// each series of a line chart, and a slice for each value of a pie. Its
// root is a figure that Attrs' aria-label names, and the plot is hidden
// from who can't see it (kit rule K18).
func TestChartPlot(t *testing.T) {
	labels := []string{"a", "b", "c"}
	two := []ui.ChartSeries{{Label: "x", Values: []float64{1, 2, 3}}, {Label: "y", Values: []float64{3, 2, 1}}}
	for _, c := range []struct {
		p    ui.ChartProps
		tag  string
		want int
	}{
		{ui.ChartProps{Labels: labels, Series: two}, "rect", 6},
		{ui.ChartProps{Type: "line", Labels: labels, Series: two}, "polyline", 2},
		{ui.ChartProps{Type: "pie", Labels: labels, Series: two[:1]}, "path", 3},
	} {
		c.p.Attrs = templ.Attributes{"aria-label": "Visits"}
		r := surftest.Start(t, surf.New("t", ui.Chart(c.p), ui.Stylesheet()))
		root := r.One("div[data-slot=chart]")
		if attr(root, "role") != "figure" || attr(root, "aria-label") != "Visits" {
			t.Errorf("%q chart: root role=%q aria-label=%q, want a figure named Visits", c.p.Type, attr(root, "role"), attr(root, "aria-label"))
		}
		if a := attr(r.One("svg"), "aria-hidden"); a != "true" {
			t.Errorf("%q chart: svg aria-hidden=%q, want true", c.p.Type, a)
		}
		if n := len(r.Find(c.tag)); n != c.want {
			t.Errorf("%q chart: %d <%s>, want %d", c.p.Type, n, c.tag, c.want)
		}
		if c.p.Type == "pie" {
			if n := len(r.Find("span[class=chart-legend-item]")); n != 3 {
				t.Errorf("a pie without Legend writes %d labels, want 3", n)
			}
		}
	}
}

// A Chart's Aspect is its plot's width over its height: the svg's box and
// the .chart box's aspect-ratio follow it. 0 is Basecoat's 16:9, written
// with no aspect-ratio; below 0 is a fault. The box is min-width 0, so a
// narrow column cuts the labels and never widens past the svg's own width.
func TestChartAspect(t *testing.T) {
	p := ui.ChartProps{Labels: []string{"a", "b"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{1, 2}}},
		Attrs: templ.Attributes{"aria-label": "x"}}
	for _, c := range []struct {
		aspect     float64
		box, style string
	}{{0, "0 0 640 360", "min-width: 0;"}, {3, "0 0 640 213.33", "min-width: 0; aspect-ratio: 640 / 213.33;"}} {
		p.Aspect = c.aspect
		r := surftest.Start(t, surf.New("t", ui.Chart(p), ui.Stylesheet()))
		if b := attr(r.One("svg"), "viewBox"); b != c.box {
			t.Errorf("Aspect %v: viewBox %q, want %q", c.aspect, b, c.box)
		}
		if st := attr(r.One("div[class=chart]"), "style"); st != c.style {
			t.Errorf("Aspect %v: .chart style %q, want %q", c.aspect, st, c.style)
		}
	}
	f := &faults{T: t}
	p.Aspect = -1
	surftest.Start(f, surf.New("t", ui.Chart(p), ui.Stylesheet()))
	if all := strings.Join(f.got, "\n"); !strings.Contains(all, "ui.Chart: Aspect -1") {
		t.Errorf("no fault for Aspect -1 in %q", all)
	}
}
