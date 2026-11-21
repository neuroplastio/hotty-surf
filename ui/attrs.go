package ui

import (
	"maps"
	"strings"

	"github.com/a-h/templ"
)

// MergeAttrs combines any number of templ.Attributes into a fresh map,
// skipping nil or empty entries without mutating any input. Later maps
// overwrite earlier keys. Returns nil if all inputs are empty.
func MergeAttrs(sets ...templ.Attributes) templ.Attributes {
	n := 0
	for _, s := range sets {
		n += len(s)
	}
	if n == 0 {
		return nil
	}
	out := make(templ.Attributes, n)
	for _, s := range sets {
		maps.Copy(out, s)
	}
	return out
}

// JoinClass joins non-empty, whitespace-trimmed CSS class strings with spaces.
func JoinClass(classes ...string) string {
	var parts []string
	for _, c := range classes {
		if c = strings.TrimSpace(c); c != "" {
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, " ")
}
