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
