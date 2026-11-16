package cells_test

import (
	"fmt"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

var page templ.Component // the app's view

// A surface in a plain terminal: the Screen is its host, and the program
// draws the Screen's lines where the surface goes.
func ExampleNew() {
	s := surf.New("files", page, "")
	c, err := cells.New(s, 80, 24)
	if err != nil {
		return
	}
	if taken, _ := c.Key("arrow_down"); !taken {
		// the program's own key
	}
	for _, l := range c.Lines() {
		fmt.Println(l)
	}
}
