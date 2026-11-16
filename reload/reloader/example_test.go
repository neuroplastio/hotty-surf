package reloader_test

import (
	"context"
	"io"

	"neuroplast.io/hotty-surf/reload/reloader"
)

// A dev parent starts each build the way it starts processes, and shows the
// reports.
func ExampleNew() {
	start := func(ctx context.Context, bin string) (io.ReadWriteCloser, error) {
		// start bin as the program starts its tools, with a pipe pair for
		// the dev wire, and return that pair
		return nil, nil
	}
	report := func(r reloader.Report) {
		// show reloader.View(r) over the app while it has a failed build
		// or findings
	}
	r := reloader.New("./examples/hello", start, report)
	r.Run(context.Background())
}
