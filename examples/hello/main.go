// Command hello is the smallest hotty-surf app: a name, a greeting and a
// button, in a terminal or, built for js/wasm, in a browser.
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
)

func main() {
	a := NewApp()
	s := surf.New("hello", Page(a), CSS)
	reload.Keep(a.Save, a.Restore)
	if err := run.App(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "hello:", err)
		os.Exit(1)
	}
}
