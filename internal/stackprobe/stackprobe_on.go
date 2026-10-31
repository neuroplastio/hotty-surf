//go:build surfstack && tinygo

package stackprobe

import "unsafe" // for go:linkname, and the paint

// low is the lowest stack pointer Probe has seen since Reset.
var low uintptr

// Probe records how deep the goroutine's stack is here.
//
//go:noinline
func Probe() {
	if sp := Here(); low == 0 || sp < low {
		low = sp
	}
}

// Reset forgets what Probe saw.
func Reset() { low = 0 }

// Used is how far below base the stack went at a Probe since Reset: 0
// where none ran.
func Used(base uintptr) int {
	if low == 0 || low > base {
		return 0
	}
	return int(base - low)
}

// Here is TinyGo's stack pointer: on wasm, the goroutine's shadow stack,
// where its frames keep what doesn't fit in wasm's own locals.
//
//go:linkname Here tinygo_getCurrentStackPointer
func Here() uintptr

// paint is what Paint fills the stack with: no pointer into the heap, so
// that a conservative scan of the stack's block keeps nothing alive.
const paint = 0xa5a5a5a5

// Paint fills the n bytes of the stack below the caller's frame with a
// pattern, so that Deepest can tell how far down a call went: the
// probes see their own frames only, the paint every frame, the leaves'
// (cel-lite's, the allocator's, the collector's) too. n must be well
// within the goroutine's stack (make stack runs on 1 MB stacks).
//
//go:noinline
func Paint(n uintptr) {
	sp := Here()
	for p := (sp - n) &^ 3; p < sp; p += 4 {
		*(*uint32)(unsafe.Pointer(p)) = paint
	}
}

// Deepest is how far below base the stack has gone since Paint(n) was
// called from base: the lowest word whose pattern is gone.
//
//go:noinline
func Deepest(base, n uintptr) int {
	for p := (base - n) &^ 3; p < base; p += 4 {
		if *(*uint32)(unsafe.Pointer(p)) != paint {
			return int(base - p)
		}
	}
	return 0
}
