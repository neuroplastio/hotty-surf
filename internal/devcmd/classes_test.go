package devcmd

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

// classes lists the kit's classes an app can use, sorted, without the
// kit's own kit-* ones.
func TestClasses(t *testing.T) {
	var out bytes.Buffer
	if code := Classes(&out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	cs := strings.Fields(out.String())
	if !slices.IsSorted(cs) || !slices.Contains(cs, "flex") || !slices.Contains(cs, "text-muted-foreground") {
		t.Errorf("the list: %q", cs)
	}
	for _, c := range cs {
		if strings.HasPrefix(c, "kit-") {
			t.Errorf("a kit-only class listed: %s", c)
		}
	}
}
