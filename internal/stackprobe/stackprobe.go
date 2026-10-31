//go:build !(surfstack && tinygo)

// Package stackprobe marks the deepest points of hotty-surf's recursion:
// the parser's, a render's in either mode, and the tree's walks (HTML,
// Diff, Resolve, the keys' home). Probe does nothing, and costs nothing,
// unless the program is built with the surfstack tag under TinyGo, whose
// goroutine stacks are a fixed 64 KB; then it records how deep the stack
// went, for the test that measures it (`make stack`,
// pkg/conformance/stack_test.go).
package stackprobe

// Probe marks a point of a recursion.
func Probe() {}
