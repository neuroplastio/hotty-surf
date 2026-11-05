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

// Lines paints each row's runs as the runtime does, and leaves out the
// rows at the end that show nothing: spaces on the ground, or on a fill
// the sixteen colours don't paint (tonal).
func TestLines(t *testing.T) {
	run := func(text string, fg, bg cells.Role) cells.Run {
		return cells.Run{Text: text, Width: len(text), Fg: fg, Bg: bg}
	}
	s := cells.Screen{Cols: 4, Rows: 4, Lines: [][]cells.Run{
		{run("ab", cells.Accent, cells.Bg), run("  ", cells.Fg, cells.Bg)},
		{run("    ", cells.Fg, cells.Field)},
		{run("    ", cells.Fg, cells.Bg)},
		{run("    ", cells.Fg, cells.Tonal)},
	}}
	got := Lines(s)
	want := []string{"\x1b[0;35mab\x1b[0m  \x1b[0m", "\x1b[0;100m    \x1b[0m"}
	if len(got) != len(want) {
		t.Fatalf("%d rows %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: %q, want %q", i, got[i], want[i])
		}
	}
}

// Paint keeps every row, the blank ones at the end too, and paints each
// as Lines does the rows it keeps. A zero screen gives no rows.
func TestPaint(t *testing.T) {
	run := func(text string, fg, bg cells.Role) cells.Run {
		return cells.Run{Text: text, Width: len(text), Fg: fg, Bg: bg}
	}
	s := cells.Screen{Cols: 4, Rows: 4, Lines: [][]cells.Run{
		{run("ab", cells.Accent, cells.Bg), run("  ", cells.Fg, cells.Bg)},
		{run("    ", cells.Fg, cells.Field)},
		{run("    ", cells.Fg, cells.Bg)},
		{run("    ", cells.Fg, cells.Tonal)},
	}}
	got, kept := Paint(s), Lines(s)
	if len(got) != len(s.Lines) {
		t.Fatalf("%d rows %q, want %d", len(got), got, len(s.Lines))
	}
	if len(kept) != 2 {
		t.Fatalf("Lines kept %d rows, want 2", len(kept))
	}
	for i := range kept {
		if got[i] != kept[i] {
			t.Errorf("row %d: %q, Lines has %q", i, got[i], kept[i])
		}
	}
	for i := len(kept); i < len(got); i++ {
		if got[i] != "\x1b[0m    \x1b[0m" {
			t.Errorf("blank row %d: %q", i, got[i])
		}
	}
	if got := Paint(cells.Screen{}); len(got) != 0 {
		t.Errorf("zero screen: %q, want no rows", got)
	}
}
