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
		ui.Chart(ui.ChartProps{Type: "area"}),
		ui.Chart(ui.ChartProps{Type: "pie", Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{-1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a", "b"}, Series: []ui.ChartSeries{{Label: "s", Values: []float64{1}}}}),
		ui.Chart(ui.ChartProps{Labels: []string{"a"}, Series: []ui.ChartSeries{{Label: "n", Values: []float64{math.NaN()}}}}),
		ui.Progress(ui.ProgressProps{Attrs: templ.Attributes{"aria-valuenow": "3"}}),
		ui.Breadcrumb(ui.BreadcrumbProps{Items: []ui.BreadcrumbItem{{Label: "a", Attrs: templ.Attributes{"data-on": "click", "role": "button"}}, {Label: "b"}}}),
		ui.Chart(ui.ChartProps{Type: "pie", Labels: []string{"a"}}),
	), ui.Stylesheet()))
	all := strings.Join(f.got, "\n")
	for _, w := range []string{
		"ui.Item: Muted with Variant Outline", `ui.Item: no size "lg"`, `ui.Alert: no variant "ghost"`,
		`ui.Card: no size "lg"`, "ui.Progress: Value 120 is outside 0 to 100", `ui.Chart: no Type "area"`,
		"ui.Chart: a pie's slice is -1", `ui.Chart: series "s" has 1 values for 2 labels`, `ui.Chart: series "n" has NaN`,
		"ui.Progress: Attrs has aria-valuenow, which the component writes itself (use Value)",
		"ui.BreadcrumbItem: Attrs has role", "ui.Chart: a pie has one series, not 0",
	} {
		if !strings.Contains(all, w) {
			t.Errorf("no fault %q in %q", w, all)
		}
	}
}

// What B2's components write: Item's Muted as its variant, a Progress'
// fill and value, a breadcrumb whose last step is the page, and an Item
// or a step that is a link by its click handler or href, a stop unless
// its Attrs give a tabindex or it has an href (kit rule K15).
func TestDisplayMarkup(t *testing.T) {
	r := surftest.Start(t, surf.New("t", templ.Join(
		ui.Item(ui.ItemProps{Muted: true, Size: ui.XSmall, Attrs: templ.Attributes{"id": "it"}}),
		ui.Progress(ui.ProgressProps{Value: 37.5, Attrs: templ.Attributes{"id": "pr"}}),
		ui.Breadcrumb(ui.BreadcrumbProps{Items: []ui.BreadcrumbItem{
			{Label: "Home", Attrs: templ.Attributes{"id": "home"}},
			{Label: "Docs"},
			{Label: "Here", Attrs: templ.Attributes{"id": "here"}},
		}}),
		ui.Item(ui.ItemProps{Attrs: templ.Attributes{"id": "click", "data-on": "click"}}),
		ui.Item(ui.ItemProps{Attrs: templ.Attributes{"id": "own", "data-on": "click", "tabindex": "-1"}}),
		ui.Item(ui.ItemProps{Attrs: templ.Attributes{"id": "web", "href": "https://example.com", "target": "_blank"}}),
		ui.Breadcrumb(ui.BreadcrumbProps{Items: []ui.BreadcrumbItem{
			{Label: "Only an id", Attrs: templ.Attributes{"id": "idonly"}},
			{Label: "Clicks", Attrs: templ.Attributes{"id": "step", "data-on": "click"}}, {Label: "Last"},
		}}),
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
	if h := r.One("#home"); h.Data != "span" {
		t.Errorf("a step with only an id is <%s>, want text", h.Data)
	}
	if h := r.One("#here"); h.Data != "span" || attr(h, "aria-current") != "page" {
		t.Errorf("the last step is <%s aria-current=%q>, want the page", h.Data, attr(h, "aria-current"))
	}
	for id, want := range map[string]string{"it": "div", "click": "a", "own": "a", "web": "a", "idonly": "span", "step": "a"} {
		if n := r.One("#" + id); n.Data != want {
			t.Errorf("#%s is <%s>, want <%s> (kit rule K15)", id, n.Data, want)
		}
	}
	for id, want := range map[string]string{"click": "0", "own": "-1", "web": "", "step": "0"} {
		if got := attr(r.One("#"+id), "tabindex"); got != want {
			t.Errorf("#%s has tabindex %q, want %q", id, got, want)
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
