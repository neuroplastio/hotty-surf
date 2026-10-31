package stackprobe

import "testing"

// TestProbe is the package's test, which tinygo test needs to build it:
// a Probe costs nothing and changes nothing in a normal build.
func TestProbe(t *testing.T) {
	for range 3 {
		Probe()
	}
}
