package run_test

import (
	"context"
	"log"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/run"
)

var page templ.Component // the app's view, from its .templ files

// An app's main: the same in a terminal and, built for js/wasm, in a
// browser.
func ExampleApp() {
	s := surf.New("tasks", page, "")
	if err := run.App(context.Background(), s); err != nil {
		log.Fatal(err)
	}
}
