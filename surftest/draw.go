package surftest

import (
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
)

// Screen is a screen drawn in cells (pkg/cells' Screen): Text is its
// characters, a row a line, and Golden its runs with their roles, as a
// golden file of a screen's cells holds them.
type Screen = cells.Screen

// Drawn is a surface drawn in a terminal's cells with no host and no
// terminal, as a host that paints its own cells shows a program where no
// hotty host draws it: the cells renderer (pkg/cells) on the surface,
// called directly. A call has done what it does when it returns, unlike
// StartCells' runtime, which reads what is typed in a terminal. For a
// test of what a screen's cells show, and of keys and clicks there.
type Drawn struct{ r *cells.Renderer }

// Draw draws s, a model's surface (surfrun.NewSurface), in cells as a new
// document, scrolling along s.Scroll as a host does. Set the surface's
// Theme, NoQuit and Scroll first. The keyboard is the terminal's until
// Focus says otherwise: an element that takes it as the document appears
// has it.
func Draw(s *surfrun.Surface) *Drawn {
	r := cells.New(s.Cells())
	r.Scroll = s.Scroll
	r.Show()
	return &Drawn{r: r}
}

// Screen is the surface drawn cols by rows now, after what changed the
// model off the renderer's input (a function posted to the surface's
// runner, a tick), with what took the keyboard or was revealed brought
// into view.
func (d *Drawn) Screen(cols, rows int) Screen {
	d.r.Update()
	return d.r.Draw(cols, rows)
}

// Key types the key name (myelin's names: "enter", "shift+tab",
// "ctrl+a"), and says whether something took it.
func (d *Drawn) Key(name string) bool {
	taken, _ := d.r.Key(name)
	return taken
}

// Focus is the terminal's focus coming (in) or going, as a host's pane
// gains or loses it: going, the surface loses the keyboard (blur, and a
// field commits); coming back, what had it has it again.
func (d *Drawn) Focus(in bool) { d.r.Focus(in) }

// Focused is the id of the element with the keyboard, or "".
func (d *Drawn) Focused() string { return d.r.Focused() }

// Click is a click of the main button at column x and row y of the last
// screen drawn, counted from 0: a press there and its release, with no
// modifier held.
func (d *Drawn) Click(x, y int) {
	d.r.Press(x, y, 0)
	d.r.Release(x, y, 0)
}
