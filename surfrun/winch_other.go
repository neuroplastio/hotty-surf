//go:build !unix && !js

package surfrun

import "neuroplast.io/hotty-go/hottyterm"

// onResize does nothing where there is no SIGWINCH: the size stays the
// one Loop starts with.
func onResize(*hottyterm.Term) {}
