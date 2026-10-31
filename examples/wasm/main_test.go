package main

import (
	"strings"
	"testing"
)

// The example runs natively too, with the race detector in make test.
func TestExample(t *testing.T) {
	var out strings.Builder
	if err := example(&out); err != nil {
		t.Fatal(err)
	}
	if want := `the host shows "3 Add"`; !strings.Contains(out.String(), want) {
		t.Errorf("the example printed %q, want it to say %s", out.String(), want)
	}
}
