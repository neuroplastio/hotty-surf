// Command app is the reloader's tests' app: a count, a button, a label.
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
	a := &App{}
	s := surf.New("t", Page(a), CSS)
	reload.Keep(a.Save, a.Restore)
	if err := run.App(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "app:", err)
		os.Exit(1)
	}
}
