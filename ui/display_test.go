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

// B2's faults: an Item both Muted and Outline (kit rule K5), a Progress
// outside 0 to 100, and a Chart whose data can't be drawn.
func TestDisplayFaults(t *testing.T) {
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", templ.Join(
		ui.Item(ui.ItemProps{Variant: ui.Outline, Muted: true}),
		ui.Item(ui.ItemProps{Size: ui.Large}),
		ui.Alert(ui.AlertProps{Variant: ui.Ghost}),
		ui.Card(ui.CardProps{Size: ui.Large}),
		ui.Progress(ui.ProgressProps{Value: 120}),
		ui.Chart(ui.ChartProps{Kind: "area"}),
		ui.Chart(ui.ChartProps{Kind: "pie", Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{-1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a", "b"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "n", Values: []float64{math.NaN()}}}}),
		ui.Progress(ui.ProgressProps{Attrs: templ.Attributes{"aria-valuenow": "3"}}),
	), ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{
		"ui.Item: Muted with Variant Outline", `ui.Item: no size "lg"`, `ui.Alert: no variant "ghost"`,
		`ui.Card: no size "lg"`, "ui.Progress: Value 120 is outside 0 to 100", `ui.Chart: no Kind "area"`,
		"ui.Chart: a pie's slice is -1", `ui.Chart: series "s" has 1 values for 2 labels`, `ui.Chart: series "n" has NaN`,
		"ui.Progress: Attrs has aria-valuenow, which the component writes itself (use Value)",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}

// What B2's components write: Item's Muted as its variant, a Progress'
// fill and value, a breadcrumb whose last step is the page and whose
// steps with a handler are stops.
func TestDisplayMarkup(t *testing.T) {
	r := surftest.Start(t, surf.New("t", templ.Join(
		ui.Item(ui.ItemProps{Muted: true, Size: ui.XSmall, Attrs: templ.Attributes{"id": "it"}}),
		ui.Progress(ui.ProgressProps{Value: 37.5, Attrs: templ.Attributes{"id": "pr"}}),
		ui.Breadcrumb(ui.BreadcrumbProps{Items: []ui.BreadcrumbItem{
			{Label: "Home", Attrs: templ.Attributes{"id": "home"}},
			{Label: "Docs"},
			{Label: "Here", Attrs: templ.Attributes{"id": "here"}},
		}}),
		ui.Scrollbar(ui.ScrollbarProps{Size: ui.Small, Attrs: templ.Attributes{"id": "sb"}}),
	), ui.Stylesheet()))
	it := r.One("#it")
	if attr(it, "data-variant") != "muted" || attr(it, "data-size") != "xs" {
		t.Errorf("a muted xs item writes data-variant=%q data-size=%q", attr(it, "data-variant"), attr(it, "data-size"))
	}
	pr := r.One("#pr")
	if attr(pr, "aria-valuenow") != "37.5" || attr(pr, "role") != "progressbar" {
		t.Errorf("progress writes aria-valuenow=%q role=%q", attr(pr, "aria-valuenow"), attr(pr, "role"))
	}
	if st := attr(pr.FirstChild, "style"); !strings.Contains(st, "width: 37.5%") {
		t.Errorf("progress fill style %q", st)
	}
	if h := r.One("#home"); h.Data != "a" || attr(h, "tabindex") != "0" {
		t.Errorf("a step with a handler is <%s tabindex=%q>, want a stop", h.Data, attr(h, "tabindex"))
	}
	if h := r.One("#here"); h.Data != "span" || attr(h, "aria-current") != "page" {
		t.Errorf("the last step is <%s aria-current=%q>, want the page", h.Data, attr(h, "aria-current"))
	}
	if c := attr(r.One("#sb"), "class"); c != "scrollbar-sm" {
		t.Errorf("a small scrollbar has class %q", c)
	}
}

// A Chart's plot: a bar for each value of each series, a polyline for
// each series of a line chart, and a slice for each value of a pie.
func TestChartPlot(t *testing.T) {
	labels := []string{"a", "b", "c"}
	two := []ui.ChartSeries{{Label: "x", Values: []float64{1, 2, 3}}, {Label: "y", Values: []float64{3, 2, 1}}}
	for _, c := range []struct {
		p    ui.ChartProps
		tag  string
		want int
	}{
		{ui.ChartProps{Labels: labels, Series: two}, "rect", 6},
		{ui.ChartProps{Kind: "line", Labels: labels, Series: two}, "polyline", 2},
		{ui.ChartProps{Kind: "pie", Labels: labels, Series: two[:1], Legend: true}, "path", 3},
	} {
		r := surftest.Start(t, surf.New("t", ui.Chart(c.p), ui.Stylesheet()))
		if n := len(r.Find(c.tag)); n != c.want {
			t.Errorf("%q chart: %d <%s>, want %d", c.p.Kind, n, c.tag, c.want)
		}
		if c.p.Legend {
			if n := len(r.Find("span[class=chart-legend-item]")); n != 3 {
				t.Errorf("pie legend: %d items, want 3", n)
			}
		}
	}
}
