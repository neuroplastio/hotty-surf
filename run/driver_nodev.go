//go:build !surfdev && !js

package run

import "neuroplast.io/hotty-surf"

func driverStarted(*surf.Surface) func() { return func() {} }
