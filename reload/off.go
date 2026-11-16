//go:build !surfdev

package reload

import (
	"context"
	"errors"
	"io"

	"github.com/tinylib/msgp/msgp"
)

// Outside a dev build, nothing of the dev wire is linked.

func keep(func(*msgp.Writer) error, func(*msgp.Reader) error) {}

func serve(context.Context, io.ReadWriter, string) error {
	return errors.New("reload: not a dev build (build with the surfdev tag)")
}
