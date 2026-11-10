// Package gauge is three widgets registered from outside hotty-surf
// (surf.Register): what a package that isn't the framework's adds to the
// vocabulary, and the example docs/surf.md's "Widgets from outside"
// shows. Import it (a blank import will do) and its tags are in every
// template of the program, drawn alike by the interpreter, by the
// compiled mode and by surflua:
//
//	<x-gauge level="40"/>                 a level in a bar
//	<x-callout kind="warn">…</x-callout>  a panel round its children
//	<x-tagline loud>Heads up</x-tagline>  a line of text
//	<x-spark>3 1 4 1 5 9 2 6</x-spark>    a sparkline
//
// Each builds its nodes from surfui.El, which hosts draw by the
// stylesheet below. The cells draw a gauge's bar and a sparkline's
// strokes by blocks of their own (Node.Cells, surfui.Graphic), and the
// rest by what they know: the text.
package gauge

import (
	"fmt"
	"strconv"
	"strings"

	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/pkg/cellblock"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// Pkg is this package's import path, which the compiled mode imports.
const Pkg = "neuroplast.io/hotty-surf/examples/widgets/gauge"

// CSS is the widgets' stylesheet, sent once after surf.css: it reads
// surf.css's tokens (`--u-*`), so the widgets take the theme and the
// density the surface is in, and uses only what hotty-blitz is shown to
// draw (the fill is the meter's idiom: a width of calc() over a
// custom property).
const CSS = `
.x-gauge { display: flex; align-items: center; gap: var(--u-gx); height: var(--u-row); }
.x-gauge-track { display: block; flex: 0 0 auto; width: 120px; height: 4px; border-radius: 2px; overflow: hidden; background: var(--u-input); }
.x-gauge-fill { display: block; width: calc(var(--x-level) * 100%); height: 100%; border-radius: 2px; background: var(--u-accent); }
.x-gauge-text { color: var(--u-muted); }
.x-callout { display: flex; flex-direction: column; box-sizing: border-box; padding: var(--u-iy) var(--u-px); border-radius: var(--u-r-md); background: var(--u-raised); }
.x-callout-warn { background: var(--u-warn-soft); }
.x-spark { display: inline-block; height: 16px; color: var(--u-accent); }
.x-tagline { color: var(--u-ink); }
.x-loud { font-weight: 600; color: var(--u-accent); }
`

func init() {
	surf.MustRegister(surf.Widget{
		Tag: "x-gauge", Doc: "a level, 0 to 100, in a bar and as a number",
		Form: surf.FormNone, Pkg: Pkg, Func: "Gauge", New: Gauge,
		Attrs: []surf.Attr{
			{Name: "level", Type: surf.TypeInt, Doc: "how full it is, 0 to 100", Func: "SetLevel", Apply: SetLevel},
		},
		CSS: CSS,
	})
	surf.MustRegister(surf.Widget{
		Tag: "x-callout", Doc: "a panel round its children, in a kind's look",
		Form: surf.FormKids, Pkg: Pkg, Func: "Callout", New: Callout,
		Events: []string{"click"},
		Attrs: []surf.Attr{
			{Name: "kind", Type: surf.TypeString, Doc: "info (the default) or warn", Func: "SetKind", Apply: SetKind, Fails: true, Values: []string{"info", "warn"}},
		},
	})
	surf.MustRegister(surf.Widget{
		Tag: "x-tagline", Doc: "a line of text, loud where it asks",
		Form: surf.FormText, Pkg: Pkg, Func: "Tagline", New: Tagline,
		Attrs: []surf.Attr{
			{Name: "loud", Type: surf.TypeBool, Doc: "in the accent, semibold", Func: "SetLoud", Apply: SetLoud},
		},
	})
	surf.MustRegister(surf.Widget{
		Tag: "x-spark", Doc: "numbers as a sparkline: a graphic, strokes in the cells",
		Form: surf.FormText, Pkg: Pkg, Func: "Spark", New: Spark,
	})
}

// Gauge is an empty gauge: a track with its fill, and its number.
func Gauge() *surfui.Node {
	return surfui.El("div",
		surfui.El("div", surfui.El("div").Class("x-gauge-fill")).Class("x-gauge-track"),
	).Class("x-gauge").Var("x-level", "0").Cells(bar(0))
}

// SetLevel fills the gauge to v percent, 0 to 100, and writes it.
func SetLevel(n *surfui.Node, v int64) {
	v = max(0, min(v, 100))
	n.Var("x-level", strconv.FormatFloat(float64(v)/100, 'f', -1, 64))
	n.Add(surfui.El("span", surfui.Str(fmt.Sprintf("%d%%", v))).Class("x-gauge-text"))
	n.Cells(bar(int(v)))
}

// trackCells is the track's width in the cells, where there is room.
const trackCells = 20

// bar is a gauge's cells: its track, filled to level percent in the
// accent and the rest in a lighter stroke, then its number in muted: the
// row the host's flex has. The block depends on level and the width, and
// no more. It asks for as much as it wants (the track and the number),
// whatever it is given, and takes less where there is less.
func bar(level int) cells.BlockFunc {
	return func(w int, _ cells.Style) cells.Block {
		label := fmt.Sprintf(" %d%%", level)
		track := min(w-len(label), trackCells)
		if track < 1 {
			return cells.Block{Rows: [][]cells.Run{{cells.NewRun(0, strings.TrimLeft(label, " "), cells.Muted, "", 0)}}}
		}
		fill := (track*level + 50) / 100
		return cells.Block{Rows: [][]cells.Run{{
			cells.NewRun(0, strings.Repeat("█", fill), cells.Accent, "", 0),
			cells.NewRun(fill, strings.Repeat("░", track-fill), cells.Muted, "", 0),
			cells.NewRun(track, label, cells.Muted, "", 0),
		}}}
	}
}

// Spark is a sparkline of the numbers in text, split by white space and
// commas: on a host an SVG polyline in the text's colour, in the cells
// a stroke of eighths a cell each, as tall as the largest number. A
// text with no number is an empty line.
func Spark(text string) *surfui.Node {
	var vs []float64
	for _, f := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' }) {
		if v, err := strconv.ParseFloat(f, 64); err == nil {
			vs = append(vs, v)
		}
	}
	return surfui.Graphic(sparkSVG(vs), func(w int) cellblock.Block { return sparkCells(vs, w) }).Class("x-spark")
}

// sparkSVG draws vs as a polyline in a box 4 a number wide and 16 high.
func sparkSVG(vs []float64) string {
	lo, hi := span(vs)
	var pts []string
	for i, v := range vs {
		y := 14.0
		if hi > lo {
			y = 14 - 12*(v-lo)/(hi-lo)
		}
		pts = append(pts, fmt.Sprintf("%d,%s", i*4+2, strconv.FormatFloat(y, 'f', 1, 64)))
	}
	return fmt.Sprintf(`<svg viewBox="0 0 %d 16" width="%d" height="16" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><polyline points="%s"></polyline></svg>`,
		len(vs)*4, len(vs)*4, strings.Join(pts, " "))
}

// sparkCells is vs in eighths of a cell's height, one cell a number, as
// many as w holds, the first of them.
func sparkCells(vs []float64, w int) cellblock.Block {
	vs = vs[:min(len(vs), w)]
	lo, hi := span(vs)
	var b strings.Builder
	for _, v := range vs {
		i := 0
		if hi > lo {
			i = int((v - lo) / (hi - lo) * 7)
		}
		b.WriteRune([]rune("▁▂▃▄▅▆▇█")[i])
	}
	if b.Len() == 0 {
		return cellblock.Block{}
	}
	return cellblock.Block{Rows: [][]cellblock.Run{{cells.NewRun(0, b.String(), cells.Accent, "", 0)}}}
}

// span is the least and the largest of vs.
func span(vs []float64) (lo, hi float64) {
	for i, v := range vs {
		if i == 0 || v < lo {
			lo = v
		}
		if i == 0 || v > hi {
			hi = v
		}
	}
	return lo, hi
}

// Callout is a panel round kids.
func Callout(kids ...*surfui.Node) *surfui.Node {
	return surfui.El("div", kids...).Class("x-callout")
}

// SetKind gives a callout its kind's look: info or warn.
func SetKind(n *surfui.Node, v string) error {
	switch v {
	case "info":
	case "warn":
		n.Class("x-callout-warn")
	default:
		return fmt.Errorf("a callout's kind is info or warn, not %q", v)
	}
	return nil
}

// Tagline is a line of text.
func Tagline(text string) *surfui.Node {
	return surfui.El("span", surfui.Str(text)).Class("x-tagline")
}

// SetLoud makes a tagline loud where v is set.
func SetLoud(n *surfui.Node, v bool) {
	if v {
		n.Class("x-loud")
	}
}
