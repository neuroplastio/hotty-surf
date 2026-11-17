package devcmd

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/lint"
	"neuroplast.io/hotty-surf/reload/reloader"
)

// `hotty-surf lint` and reloader.Text print a finding the same way.
func TestTextFindingLine(t *testing.T) {
	f := lint.Finding{File: "app.templ", Line: 12, Col: 5, Rule: "px", Text: "14px: use rem"}
	text := reloader.Text(&reloader.Report{Tier: "go", Lint: []lint.Finding{f}})
	lines := strings.SplitAfter(text, "\n")
	if last := lines[len(lines)-2]; last != "  "+findingLine(f)+"\n" {
		t.Fatalf("Text's finding line %q, lint's %q", last, findingLine(f))
	}
}
