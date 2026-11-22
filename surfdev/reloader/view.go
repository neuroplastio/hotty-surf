package reloader

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// shows says whether View shows anything for r.
func shows(r *Report) bool {
	return r != nil && (r.Build != "" || len(r.Failed) > 0 || len(r.Lint) > 0)
}

func failedKeys(r *Report) []string { return slices.Sorted(maps.Keys(r.Failed)) }

func failedTitle(key string) string { return fmt.Sprintf("The process %q failed", key) }

func lintTitle(r *Report) string {
	if len(r.Lint) == 1 {
		return "lint: 1 finding (the app runs; fix it, or allow it with a reason)"
	}
	return fmt.Sprintf("lint: %d findings (the app runs; fix them, or allow each with a reason)", len(r.Lint))
}

// lintLines is a line a finding, as `hotty-surf lint` prints them.
func lintLines(r *Report) string {
	var b strings.Builder
	for i, f := range r.Lint {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s:%d:%d: %s: %s", f.File, f.Line, f.Col, f.Rule, f.Text)
	}
	return b.String()
}
