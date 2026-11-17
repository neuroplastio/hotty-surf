package ui

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/rt"
)

// ChartProps are a Chart's attributes.
type ChartProps struct {
	// Type is "" or "bar" (Basecoat's default), "line" or "pie". surftest
	// fails any other.
	Type string
	// Labels are the categories, one per value of each series, written
	// under the bars or the line's points; a pie's slices. A label may be
	// "": nothing is written under its point, as a month of days labels
	// every fifth. Each has an equal share of the width, and one wider is
	// cut with "…".
	Labels []string
	// Series are the data: one or more for bars and lines, one for a pie
	// (its slices, one per label). Each has as many values as Labels;
	// surftest fails a chart that doesn't, a negative slice, and a value
	// that isn't a number.
	Series []ChartSeries
	// Legend writes the series' names under bars or lines, each beside its
	// colour. A pie always writes its labels beside their colours, since
	// they are its categories, as bars write theirs under the plot. Legend
	// changes nothing there.
	Legend bool
	// Aspect is the plot's width over its height: 0 is Basecoat's 16:9
	// (aspect-video), 3 a short, wide plot for a dashboard's row. The plot
	// takes its width from its box and its height from that, so a chart
	// in a narrow column is short. surftest fails one below 0.
	Aspect float64
	// Attrs are more attributes for the div element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class, data-slot or role), which are a fault here. A chart is named
	// for who can't see it by aria-label here:
	// templ.Attributes{"aria-label": "Visitors by month"}.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// A ChartSeries is one series of a Chart: its name and its values.
type ChartSeries struct {
	Label  string
	Values []float64
}

// Chart is Basecoat's chart, drawn as an SVG plot in the theme's chart
// colours (--chart-1 to --chart-5, in turn): bars, a line or a pie, the
// categories under it and a legend. Basecoat's chart.js draws on a canvas
// with Chart.js, which a surface can't run; this is a still plot, without
// its tooltip.
//
// It shows no values: Basecoat's y axis has no ticks and gives values by
// its tooltip, which a still plot hasn't. A tooltip comes back as
// ChartState, Chart(st, p), a MINOR break as Tabs' would be. The values
// are in the markup, for a host that draws no SVG and for a test: each
// bar's <rect> and each slice has data-value, and each line's <polyline>
// data-values, its values in order, space-separated (surftest:
// "rect[data-value=676]").
//
//	@ui.Chart(ui.ChartProps{Labels: []string{"Jan", "Feb", "Mar"},
//		Series: []ui.ChartSeries{{Label: "Desktop", Values: []float64{186, 305, 237}}},
//		Attrs:  templ.Attributes{"aria-label": "Visitors by month"}})
func Chart(p ChartProps) templ.Component {
	return chart(p)
}

var chartTypes = []string{"", "bar", "line", "pie"}

// The plot's box is chartW wide and chartW/Aspect high: Basecoat's .chart
// is 16:9 (aspect-video). Its strokes scale with it, as Blitz applies no
// vector-effect (vault/knowledge/blitz-gaps.md), so none is a hairline.
const chartW = 640.0

// chartH is the plot's height in its box.
func chartH(p ChartProps) float64 {
	if p.Aspect > 0 && !math.IsInf(p.Aspect, 0) {
		return chartW / p.Aspect
	}
	return 360
}

// chartBox is the svg's viewBox.
func chartBox(p ChartProps) string { return "0 0 " + num(chartW) + " " + num(chartH(p)) }

// chartRatio is the .chart box's style: its aspect-ratio, when Aspect
// isn't the default's.
func chartRatio(p ChartProps) templ.SafeCSS {
	// min-width 0: the plot takes the width it is given, never the svg's
	// own, so a narrow column cuts the labels, not the card.
	if p.Aspect <= 0 || math.IsInf(p.Aspect, 0) || math.IsNaN(p.Aspect) {
		return "min-width: 0;"
	}
	return templ.SafeCSS("min-width: 0; aspect-ratio: " + num(chartW) + " / " + num(chartH(p)) + ";")
}

// checkChart faults data the plot can't draw.
func checkChart(ctx context.Context, p ChartProps) string {
	if p.Aspect < 0 || math.IsNaN(p.Aspect) || math.IsInf(p.Aspect, 0) {
		rt.Fault(ctx, fmt.Sprintf("ui.Chart: Aspect %v (a width over a height, or 0 for 16:9)", p.Aspect))
	}
	if p.Type == "pie" && len(p.Series) != 1 {
		rt.Fault(ctx, fmt.Sprintf("ui.Chart: a pie has one series, not %d", len(p.Series)))
	}
	for _, s := range p.Series {
		if len(s.Values) != len(p.Labels) {
			rt.Fault(ctx, fmt.Sprintf("ui.Chart: series %q has %d values for %d labels", s.Label, len(s.Values), len(p.Labels)))
		}
		for _, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				rt.Fault(ctx, fmt.Sprintf("ui.Chart: series %q has %v", s.Label, v))
			} else if p.Type == "pie" && v < 0 {
				rt.Fault(ctx, fmt.Sprintf("ui.Chart: a pie's slice is %v, below 0", v))
			}
		}
	}
	return ""
}

// chartColour is the i'th colour, in turn.
func chartColour(i int) string {
	return "var(--chart-" + strconv.Itoa(i%5+1) + ")"
}

// A chartKey is one entry of a legend.
type chartKey struct {
	Label string
	Style templ.SafeCSS
}

func chartKeys(p ChartProps) []chartKey {
	var ks []chartKey
	key := func(i int, l string) {
		ks = append(ks, chartKey{l, templ.SafeCSS("--chart-indicator-color: " + chartColour(i) + ";")})
	}
	if p.Type == "pie" {
		for i, l := range p.Labels {
			key(i, l)
		}
		return ks
	}
	for i, s := range p.Series {
		key(i, s.Label)
	}
	return ks
}

// chartPlot is the SVG's inner markup: numbers and the theme's colours
// only, never the app's text, which the templ writes escaped. Each bar
// and slice carries its value (data-value), and each line its values
// (data-values), for a host that draws no SVG (cells).
func chartPlot(p ChartProps) templ.Component {
	var b strings.Builder
	n := len(p.Labels)
	ok := n > 0
	for _, s := range p.Series {
		for _, v := range s.Values {
			ok = ok && !math.IsNaN(v) && !math.IsInf(v, 0)
		}
		ok = ok && len(s.Values) == n
	}
	if ok {
		switch p.Type {
		case "pie":
			if len(p.Series) > 0 {
				piePlot(&b, p.Series[0].Values, chartH(p))
			}
		default:
			barPlot(&b, p)
		}
	}
	return templ.Raw(b.String())
}

func num(v float64) string { return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) }

// barPlot draws bars or lines from 0 (or the lowest value, below it) to
// the highest, with a grid of four lines.
func barPlot(b *strings.Builder, p ChartProps) {
	lo, hi := 0.0, 0.0
	for _, s := range p.Series {
		for _, v := range s.Values {
			lo, hi = min(lo, v), max(hi, v)
		}
	}
	if hi == lo {
		hi = lo + 1
	}
	top, bottom := 8.0, chartH(p)-2
	y := func(v float64) float64 { return bottom - (v-lo)/(hi-lo)*(bottom-top) }
	for i := range 5 {
		gy := top + float64(i)*(bottom-top)/4
		fmt.Fprintf(b, `<line x1="0" x2="%s" y1="%s" y2="%s" style="stroke: var(--border); stroke-width: 2;"/>`, num(chartW), num(gy), num(gy))
	}
	n := len(p.Labels)
	band := chartW / float64(n)
	if p.Type == "line" {
		for si, s := range p.Series {
			var pts, vals []string
			for i, v := range s.Values {
				pts = append(pts, num((float64(i)+0.5)*band)+","+num(y(v)))
				vals = append(vals, num(v))
			}
			fmt.Fprintf(b, `<polyline points="%s" data-values="%s" style="fill: none; stroke: %s; stroke-width: 4; stroke-linejoin: round; stroke-linecap: round;"/>`, strings.Join(pts, " "), strings.Join(vals, " "), chartColour(si))
		}
		return
	}
	k := float64(len(p.Series))
	gap := band * 0.2
	w := (band - 2*gap) / k
	for si, s := range p.Series {
		for i, v := range s.Values {
			y0, y1 := y(max(v, 0)), y(min(v, 0))
			fmt.Fprintf(b, `<rect x="%s" y="%s" width="%s" height="%s" rx="6" data-value="%s" style="fill: %s;"/>`,
				num(float64(i)*band+gap+float64(si)*w+1), num(y0), num(max(w-2, 1)), num(max(y1-y0, 0)), num(v), chartColour(si))
		}
	}
}

// piePlot draws a pie's slices, from the top, clockwise.
func piePlot(b *strings.Builder, vs []float64, h float64) {
	sum := 0.0
	for _, v := range vs {
		sum += v
	}
	if sum <= 0 {
		return
	}
	cx, cy, r := chartW/2, h/2, min(chartW, h)/2-4
	a := -math.Pi / 2
	for i, v := range vs {
		if v <= 0 {
			continue
		}
		if v == sum {
			fmt.Fprintf(b, `<circle cx="%s" cy="%s" r="%s" data-value="%s" style="fill: %s;"/>`, num(cx), num(cy), num(r), num(v), chartColour(i))
			return
		}
		d := v / sum * 2 * math.Pi
		large := 0
		if d > math.Pi {
			large = 1
		}
		fmt.Fprintf(b, `<path d="M%s %sL%s %sA%s %s 0 %d 1 %s %sZ" data-value="%s" style="fill: %s; stroke: var(--background); stroke-width: 2;"/>`,
			num(cx), num(cy), num(cx+r*math.Cos(a)), num(cy+r*math.Sin(a)), num(r), num(r), large,
			num(cx+r*math.Cos(a+d)), num(cy+r*math.Sin(a+d)), num(v), chartColour(i))
		a += d
	}
}
