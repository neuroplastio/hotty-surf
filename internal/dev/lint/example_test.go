package lint_test

import (
	"fmt"

	"neuroplast.io/hotty-surf/internal/dev/lint"
)

// A dev tool lints the app after every save.
func ExampleDir() {
	fs, err := lint.Dir("examples/hello")
	if err != nil {
		return
	}
	for _, f := range fs {
		fmt.Printf("%s:%d:%d: %s: %s\n", f.File, f.Line, f.Col, f.Rule, f.Text)
	}
}
