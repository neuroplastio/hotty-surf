//go:build surfdev

package surf

import "neuroplast.io/hotty-surf/internal/devhook"

// register hands s to package reload, in a dev build, so that a live
// reload reaches it wherever it was made.
func register(s *Surface) { devhook.Add(s) }
