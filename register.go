//go:build !surfdev

package surf

// register does nothing in a release build: only a dev build (the surfdev
// tag) hands its surfaces to package reload.
func register(*Surface) {}
