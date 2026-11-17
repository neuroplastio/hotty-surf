// Command settings is a settings screen in sections: Account,
// Notifications and Appearance, checked before it saves.
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
)

func main() {
	a := NewApp()
	s := newSurface("settings", Page(a))
	reload.Keep(a.Keep, a.Restore)
	if err := run.App(context.Background(), s); err != nil {
		fmt.Fprintln(os.Stderr, "settings:", err)
		os.Exit(1)
	}
}
