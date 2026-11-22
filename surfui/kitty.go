package surfui

import (
	"context"
	"sync/atomic"
)

type kittyKey struct{}

var kittyDefault atomic.Bool

// SetKittyDefault records the process-level Kitty graphics capability probed
// from the terminal via hottyterm.Term.KittyGraphics (called by run when
// falling back to cells/TUI mode).
func SetKittyDefault(supported bool) {
	kittyDefault.Store(supported)
}

// WithKitty explicitly overrides Kitty graphics protocol support on ctx.
func WithKitty(ctx context.Context, supported bool) context.Context {
	return context.WithValue(ctx, kittyKey{}, supported)
}

// SupportsKitty reports whether the target terminal supports the Kitty graphics
// protocol. It checks an explicit WithKitty context value first, then falls
// back to the terminal probe recorded by SetKittyDefault.
func SupportsKitty(ctx context.Context) bool {
	if ctx != nil {
		if v, ok := ctx.Value(kittyKey{}).(bool); ok {
			return v
		}
	}
	return kittyDefault.Load()
}
