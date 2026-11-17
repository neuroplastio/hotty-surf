// Command logs watches the log of parcel-relay, a made-up service: a new
// line every second or so, filtered by level and search, following the
// newest line until it is paused or scrolled back.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := NewApp()
	s := surf.New("logs", Page(a), CSS)
	reload.Keep(a.Save, a.Restore)
	go Feed(ctx, s, a, NewService(time.Now().UnixNano()))
	if err := run.App(ctx, s); err != nil {
		fmt.Fprintln(os.Stderr, "logs:", err)
		os.Exit(1)
	}
}
