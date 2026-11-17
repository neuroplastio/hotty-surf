//go:build surfdev && !js

package run

import (
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devhook"
)

func driverStarted(s *surf.Surface) func() {
	done := make(chan struct{})
	devhook.Driver(s, done)
	return func() { close(done) }
}
