package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/internal/gallery"
)

// -list's ids are the sidebar's stories, each once, and -story with one
// of them opens that story, with the list whole; with other text, the
// first story whose component or name has it.
func TestPick(t *testing.T) {
	g := gallery.New()
	ids := storyIDs(g)
	if len(ids) < 2 {
		t.Fatalf("the gallery lists %d stories", len(ids))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("%s is listed twice", id)
		}
		seen[id] = true
		pick(g, id)
		if g.Sel != id || g.Filter != "" {
			t.Errorf("-story %s opened %s, the list narrowed to %q", id, g.Sel, g.Filter)
		}
	}
	pick(g, "button/")
	if !strings.Contains(g.Title(), "button /") || g.Filter != "button/" {
		t.Errorf("-story button/ opened %s (%s), the list narrowed to %q", g.Sel, g.Title(), g.Filter)
	}
	if !slices.Contains(storyIDs(g), g.Sel) {
		t.Errorf("-story button/ opened %s, which the list doesn't show", g.Sel)
	}
}
