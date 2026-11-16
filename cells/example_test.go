package cells_test

import (
	"fmt"

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
