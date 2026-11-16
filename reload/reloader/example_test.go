package reloader_test

import (
	"context"
	"io"
	"net"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/reload/reloader"
	"neuroplast.io/hotty-surf/ui"
)

// A dev parent: one Reloader for the app's directory, processes dialing in
// on a socket of the program's, each served by the Reloader.
func ExampleNew() {
	ctx := context.Background()
	var last reloader.Report
	dev := surf.New("dev", reloader.View(&last), ui.Stylesheet())

	restart := func(ctx context.Context, bin string) error {
		// start bin again in each place the app runs, every one of them
		// with the dial-in address and its instance key
		return nil
	}
	report := func(rep reloader.Report) {
		dev.Do(func() { last = rep }) // the dev surface shows it over the app
	}
	r := reloader.New("./examples/hello", restart, report,
		reloader.Output("/path/the/program/runs/hello"))
	go r.Run(ctx)

	var ln net.Listener // where the processes dial in
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go r.Serve(ctx, io.ReadWriteCloser(c))
	}
}
