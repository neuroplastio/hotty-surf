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
//
// Each builds its nodes from surfui.El, which hosts draw by the
// stylesheet below, and the cells draw by what they know: the text.
package gauge

import (
	"fmt"
	"strconv"

	surf "neuroplast.io/hotty-surf"
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
}

// Gauge is an empty gauge: a track with its fill, and its number.
func Gauge() *surfui.Node {
	return surfui.El("div",
		surfui.El("div", surfui.El("div").Class("x-gauge-fill")).Class("x-gauge-track"),
	).Class("x-gauge").Var("x-level", "0")
}

// SetLevel fills the gauge to v percent, 0 to 100, and writes it.
func SetLevel(n *surfui.Node, v int64) {
	v = max(0, min(v, 100))
	n.Var("x-level", strconv.FormatFloat(float64(v)/100, 'f', -1, 64))
	n.Add(surfui.El("span", surfui.Str(fmt.Sprintf("%d%%", v))).Class("x-gauge-text"))
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
