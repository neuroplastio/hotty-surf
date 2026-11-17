// Package devhook is where, in a dev build (the surfdev tag), surf.New
// hands each surface to package reload, and package run says where it
// placed one: a registry they import, so that none imports another's
// insides. A release build never adds to it.
package devhook

import "sync"

var (
	mu       sync.Mutex
	surfaces []any
	added    func(s any)
	areas    = map[any][2]int{}
	drivers  = map[any]<-chan struct{}{}
	placed   func(s any, cols, rows int)
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

// Placed records the cells a surface was placed at, and tells the listener
// if one is set and the area changed.
func Placed(s any, cols, rows int) {
	mu.Lock()
	was, ok := areas[s]
	areas[s] = [2]int{cols, rows}
	f := placed
	mu.Unlock()
	if f != nil && (!ok || was != [2]int{cols, rows}) {
		f(s, cols, rows)
	}
}

// ListenPlaced sets what hears each surface placed at another size from
// now on.
func ListenPlaced(f func(s any, cols, rows int)) {
	mu.Lock()
	defer mu.Unlock()
	placed = f
}

// Area is the cells s was last placed at, 0 by 0 before it was.
func Area(s any) (cols, rows int) {
	mu.Lock()
	defer mu.Unlock()
	a := areas[s]
	return a[0], a[1]
}

// Driver records a framework-owned surface driver's completion signal.
// A missing signal means the driver's lifetime is unknown.
func Driver(s any, done <-chan struct{}) {
	mu.Lock()
	drivers[s] = done
	mu.Unlock()
}

// DriverDone is closed only when the registered driver has returned.
func DriverDone(s any) <-chan struct{} {
	mu.Lock()
	defer mu.Unlock()
	return drivers[s]
}

// Reset forgets every surface and the listeners, for a test that serves
// more than once in one process.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	surfaces, added, placed = nil, nil, nil
	clear(areas)
	clear(drivers)
}
