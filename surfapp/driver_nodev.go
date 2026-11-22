//go:build !surfdev && !js

package surfapp

import "neuroplast.io/hotty-surf"

func driverStarted(*surf.Surface) func() { return func() {} }
