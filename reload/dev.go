//go:build surfdev

package reload

import (
	"context"
	"io"

	"github.com/tinylib/msgp/msgp"
	"neuroplast.io/hotty-surf"

	"neuroplast.io/hotty-surf/internal/devchild"
	"neuroplast.io/hotty-surf/internal/devhook"
)

// In a dev build, the process's end of the reload wire is devchild's.

func keep(get func(*msgp.Writer) error, set func(*msgp.Reader) error) {
	devchild.Keep(get, set)
}

func placed(s *surf.Surface, cols, rows int) {
	devhook.Placed(s, cols, rows)
}

func serve(ctx context.Context, rw io.ReadWriter, key string) error {
	_, err := devchild.Serve(ctx, rw, key)
	return err
}
