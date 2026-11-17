package cells_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/ui"
)

var page templ.Component // the app's view

// A surface in a plain terminal: the Screen is its host, and the program
// draws the Screen's lines where the surface goes.
func ExampleNew() {
	s := surf.New("files", page, ui.Stylesheet())
	c := cells.New(80, 24, func(e hotty.Event) { s.Event(e) })
	s.Attach(c, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		return
	}
	c.Write([]byte(hotty.Doc(s.Name(), doc)))

	if !c.Key("arrow_down") {
		if taken, _ := s.Key("arrow_down"); !taken {
			// the program's own key
		}
	}
	for _, l := range c.Lines() {
		fmt.Println(l)
	}
}

// The terminal's mouse in cells: the program hands the Screen each
// press, motion and release at its own cells, and the surface's Click,
// Drag, Drop, Secondary and Hover handlers hear them as on a host.
func ExampleScreen_Press() {
	s := surf.New("list", page, ui.Stylesheet())
	c := cells.New(80, 24, func(e hotty.Event) { s.Event(e) })
	s.Attach(c, hotty.Caps{})

	c.Press(4, 2, 0, 0)              // a row's drag handle: dragstart
	c.Move(4, 7, 0)                  // over another row: drag
	c.Release(4, 7, 0)               // dragend, and that row's Drop
	c.Press(4, 3, 0, 2)              // a secondary press: surf.Secondary
	c.Move(10, 5, 0)                 // no button down: a hover
	c.Press(4, 3, hotty.ModShift, 0) // a shift-click …
	c.Release(4, 3, hotty.ModShift)

	// A carry that began outside the surface: which row, and which half.
	if id, a := c.At(4, 6); id != "" {
		lower := 6 >= a.Row+a.H/2
		_ = lower
	}
}

// A program with a theme of its own paints the cells itself: each role is
// a colour of the kit's, named as its CSS variable, which the program
// already maps to its theme when it writes the kit's variables.
func ExampleScreen_Cells() {
	c := cells.New(80, 24, nil)
	theme := map[cells.Role]string{"primary": "#7aa2f7", "destructive": "#f7768e"}
	paint := func(r cells.Role) (colour string, pct int) {
		name, p, tinted := strings.Cut(string(r), "/")
		pct = 100
		if tinted {
			pct, _ = strconv.Atoi(p)
		}
		return theme[cells.Role(name)], pct // "" for the terminal's own
	}
	for y, row := range c.Cells() {
		for x, cell := range row {
			if cell.Text == "" {
				continue // a wide character's second cell
			}
			fg, _ := paint(cell.Fg)
			bg, alpha := paint(cell.Bg) // a tint: alpha percent over the ground
			_, _, _, _, _ = x, y, fg, bg, alpha
		}
	}
}
