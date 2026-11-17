// Package devhook is where, in a dev build (the surfdev tag), surf.New
// hands each surface to package reload: a registry both import, so that
// neither imports the other's insides. A release build never adds to it.
package devhook

import "sync"

var (
	mu       sync.Mutex
	surfaces []any
	added    func(s any)
)

// Add registers a surface (a *surf.Surface), and tells the listener if
// one is set.
func Add(s any) {
	mu.Lock()
	surfaces = append(surfaces, s)
	f := added
	mu.Unlock()
	if f != nil {
		f(s)
	}
}

// Listen sets what hears each surface added from now on, and returns those
// added so far.
func Listen(f func(s any)) []any {
	mu.Lock()
	defer mu.Unlock()
	added = f
	return append([]any(nil), surfaces...)
}

// Reset forgets every surface and the listener, for a test that serves
// more than once in one process.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	surfaces, added = nil, nil
}
