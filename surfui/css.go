package surfui

import (
	_ "embed"
	"strings"
	"sync"
)

// StyleID is the resource the stylesheet is sent as, once a session
// (SPEC §7.1), so that a document refers to it and doesn't carry it.
const StyleID = "surf-css"

// CSS is the one stylesheet, surfui/surf.css, embedded: a program sends it
// as it is, and a dev mode may read the file and send it again as it
// changes (surfrun.Runtime.SetCSS). It uses only what hotty-blitz is shown to
// draw (vault/knowledge/css-subset.md):
//   - flexbox, padding, margin, gap, widths in ch and cells; a Fit row's
//     flex-wrap in row-reverse under overflow: clip, which leaves out
//     whole what has no room;
//   - colours by custom property and color-mix() in srgb;
//   - fills with a radius; the kit's thin lines (a control's, a key
//     cap's), which Blitz drops at some positions (blitz-gaps.md);
//   - classes and attribute selectors for states, :focus, :focus-visible,
//     :focus-within (hotty-blitz's R-23 build) and :hover, not :invalid,
//     :placeholder-shown or the UA's :disabled looks, which Blitz lacks;
//   - text-overflow: ellipsis with overflow: hidden, on a text or on a
//     Line of runs, which is cut once;
//   - cursor, which draws nothing but reaches the pointer (SPEC §9: the
//     host shows the shape the document asks for): a pointer over what
//     acts on a click, the text caret over a field only, the arrow over
//     the rest, text included;
//   - one animation, a pulse dot's opacity (hotty-blitz runs them from
//     0.6.1), off under prefers-reduced-motion and in the gate's still
//     documents; no transitions, filters or text-shadow.
//
// Its tokens are custom properties under .u-surface, named for roles
// (gov R-24), never levels: the layers (canvas, popover, raised, float),
// the translucent states (hover, selected, border, input), a sunk field,
// a neutral ink with muted and dim, the accent and its soft and strong
// fills, solid and on-solid, the tones, and the hairlines a control
// shows under the pointer and with the keyboard (hover-line,
// focus-line). The ground, the tones and the
// accent come from the host's stylesheet (SPEC §8) and the theme; the
// units (a row, gaps, a control's height, radii, the type) from the
// density. Every rule of it reads tokens, never a colour or a size.
//
// Focus is never a ring: the sheet takes back the host's :focus-visible
// outline on everything a surface holds, and a control shows focus as its
// focus-line, inset on its own edge, so that nothing moves.
//
//go:embed surf.css
var CSS string

var (
	extraMu  sync.Mutex
	extraCSS []string
)

// AddCSS adds a widget's stylesheet (surf.Register) to what a run sends
// as the stylesheet resource (StyleID): Sheet is CSS, then each added
// sheet in the order they were added, once each (the same text added
// twice is one). It is for a program's start, before a run begins, and
// reads CSS's tokens (`--u-*`); a sheet that is blank is not kept.
func AddCSS(css string) {
	css = strings.TrimSpace(css)
	if css == "" {
		return
	}
	extraMu.Lock()
	defer extraMu.Unlock()
	for _, c := range extraCSS {
		if c == css {
			return
		}
	}
	extraCSS = append(extraCSS, css)
}

// Sheet is the stylesheet a run sends: CSS, then the sheets of the
// widgets registered from outside (AddCSS), each on lines of its own.
// The palette's rules (Palette.CSS) follow it.
func Sheet() string {
	extraMu.Lock()
	defer extraMu.Unlock()
	if len(extraCSS) == 0 {
		return CSS
	}
	return CSS + "\n" + strings.Join(extraCSS, "\n") + "\n"
}
