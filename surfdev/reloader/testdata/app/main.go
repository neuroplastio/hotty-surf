// Command app is the reloader's tests' app: a count, a button, a label.
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surfdev"
	"neuroplast.io/hotty-surf/surfapp"
)

func main() {
	a := &App{}
	s := surf.New("t", Page(a), CSS)
	surfdev.Keep(a.Save, a.Restore)
	if err := surfapp.Run(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "app:", err)
		os.Exit(1)
	}
}
