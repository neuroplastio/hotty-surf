// Command hotty-surf-book is the storybook as an app of its own, which
// `hotty-surf book` runs under the reloader in a checkout, so that a saved
// story shows at once.
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf/internal/book"
)

func main() {
	if err := book.Main(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "hotty-surf book:", err)
		os.Exit(1)
	}
}
