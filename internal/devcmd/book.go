package devcmd

import (
	"context"
	"path/filepath"

	"neuroplast.io/hotty-surf/internal/book"
)

// Book is `hotty-surf book`: the kit's storybook in this terminal. Built
// from a checkout whose sources are still there, it runs the checkout's
// book under a Reloader, as run runs an app, so a story saved shows at
// once; anywhere else (a published version) it shows the stories this
// command was built with.
func Book(ctx context.Context, sock string) error {
	if moduleVersion() == "" {
		if root := checkout(); root != "" {
			return Run(ctx, filepath.Join(root, "internal", "book", "hotty-surf-book"), sock)
		}
	}
	return book.Main(ctx)
}
