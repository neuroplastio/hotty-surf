package devcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"neuroplast.io/hotty-surf/internal/book"
)

// Book is `hotty-surf book`: the kit's storybook in this terminal. Built
// from a checkout whose sources are still there, it runs the checkout's
// book under a Reloader, as run runs an app, so a story saved shows at
// once; anywhere else (a published version) it shows the stories this
// command was built with.
func Book(ctx context.Context, sock string, out io.Writer) error {
	if moduleVersion() == "" {
		if root := checkout(); root != "" {
			if os.Getenv("HOTTY_SURF_SOCK") == "" {
				sock = filepath.Join(filepath.Dir(sock), "book.sock")
			}
			fmt.Fprintf(out, "hotty-surf book: ctl socket %s\n", sock)
			return Run(ctx, filepath.Join(root, "internal", "book", "hotty-surf-book"), sock)
		}
	}
	return book.Main(ctx)
}
