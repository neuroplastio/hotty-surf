package surfrun

import (
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
)

// In the terminal's sixteen colours, tonal is no fill, so that a pane of
// it (a Compact box) is no slab of grey; a field keeps the grey, and a
// control with the keyboard its reverse video; a focused list's
// selection is the grey, quiet, with its text's colours and a focused
// entry's ▎ in the accent; a tone's fill is the tone in reverse video.
func TestSixteenColours(t *testing.T) {
	for _, c := range []struct {
		run  cells.Run
		want string
	}{
		{cells.Run{Fg: cells.Fg, Bg: cells.Tonal}, "\x1b[0m"},
		{cells.Run{Fg: cells.Muted, Bg: cells.Tonal}, "\x1b[0;2m"},
		{cells.Run{Fg: cells.Fg, Bg: cells.Surface}, "\x1b[0m"},
		{cells.Run{Fg: cells.Fg, Bg: cells.Field}, "\x1b[0;100m"},
		{cells.Run{Fg: cells.Fg, Bg: cells.Sel}, "\x1b[0;7m"},
		{cells.Run{Fg: cells.Fg, Bg: cells.Selected}, "\x1b[0;100m"},
		{cells.Run{Fg: cells.Accent, Bg: cells.Selected}, "\x1b[0;100;35m"},
		{cells.Run{Fg: cells.OK, Bg: cells.Selected}, "\x1b[0;100;92m"},
		// A tone's fill: the tone, reversed, its text the ground's colour.
		{cells.Run{Fg: cells.Fg, Bg: cells.Info}, "\x1b[0;96;7m"},
		{cells.Run{Fg: cells.Fg, Bg: cells.Accent, Attr: cells.Bold}, "\x1b[0;35;7;1m"},
	} {
		if got := cellsSGR(c.run); got != c.want {
			t.Errorf("%s on %s: %q, want %q", c.run.Fg, c.run.Bg, got, c.want)
		}
	}
}
