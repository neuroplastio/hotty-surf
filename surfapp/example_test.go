package surfapp_test

import (
	"context"
	"log"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfapp"
	"neuroplast.io/hotty-surf/surfui"
)

var page templ.Component // the app's view, from its .templ files

// An app's main: the same in a terminal and, built for js/wasm, in a
// browser.
func ExampleRun() {
	s := surf.New("tasks", page, surfui.Stylesheet())
	if err := surfapp.Run(context.Background(), s); err != nil {
		log.Fatal(err)
	}
}
