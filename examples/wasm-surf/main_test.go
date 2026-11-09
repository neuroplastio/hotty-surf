package main

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surflive"
)

// The example runs natively too, in each mode, with the race detector in
// make test: the clicks and ops of examples/wasm. Its document is
// examples/wasm's too, so it writes the same bytes.
func TestExample(t *testing.T) {
	for _, m := range []struct {
		name string
		draw func(*Counter) surflive.Component
	}{{"interpreted", surfInterpCounter}, {"compiled", surfCompiledCounter}} {
		t.Run(m.name, func(t *testing.T) {
			var out strings.Builder
			if err := example(&out, m.name, m.draw(&Counter{})); err != nil {
				t.Fatal(err)
			}
			if want := m.name + `: 3 clicks, 3 updates of 3 ops`; !strings.Contains(out.String(), want) {
				t.Errorf("the example printed %q, want it to say %s", out.String(), want)
			}
			if want := `the host shows "3 Add"`; !strings.Contains(out.String(), want) {
				t.Errorf("the example printed %q, want it to say %s", out.String(), want)
			}
		})
	}
}
