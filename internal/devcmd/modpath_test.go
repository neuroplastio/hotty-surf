package devcmd

import "testing"

// new skips the requirement step inside hotty-surf's own module: it reads
// which module a go.mod declares.
func TestModulePath(t *testing.T) {
	for in, want := range map[string]string{
		"module neuroplast.io/hotty-surf\n\ngo 1.26.0\n": "neuroplast.io/hotty-surf",
		"// x\nmodule \"todo\"\n":                        "todo",
		"go 1.26.0\n":                                    "",
	} {
		if got := modulePath([]byte(in)); got != want {
			t.Errorf("modulePath(%q) = %q, want %q", in, got, want)
		}
	}
}
