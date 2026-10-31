package main

import (
	"time"

	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
)

// reloading is h drawn by the templates in dir, which it reads again
// when one changes: hot reload. Where the new ones don't compile, it
// keeps the old ones and says why on stderr.
func reloading(h *Hello, dir string) (surfrun.Model, error) {
	set, err := surf.LoadDir(dir, 500*time.Millisecond, surfShapes)
	if err != nil {
		return nil, err
	}
	return surflive.New(surf.Component(set, surfHello{h})), nil
}
