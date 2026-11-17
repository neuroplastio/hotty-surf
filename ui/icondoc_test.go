package ui

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Icon's doc lists the kit's icons, which an app can't see otherwise:
// every one, and no other.
func TestIconDocListsTheIcons(t *testing.T) {
	src, err := os.ReadFile("ui.go")
	if err != nil {
		t.Fatal(err)
	}
	doc := regexp.MustCompile(`(?s)// Icon is the icon named name.*?\nfunc Icon`).Find(src)
	if doc == nil {
		t.Fatal("no doc on Icon")
	}
	text := strings.NewReplacer("//", " ", "\n", " ").Replace(string(doc))
	names := iconNames()
	listed := regexp.MustCompile(`[a-z0-9]+(?:-[a-z0-9]+)*`).FindAllString(text[strings.Index(text, "draw with:"):strings.Index(text, "Another name")], -1)
	listed = slices.DeleteFunc(listed, func(w string) bool { return w == "draw" || w == "with" || w == "and" })
	if !slices.Equal(listed, names) {
		t.Errorf("Icon's doc lists %q, the kit has %q", listed, names)
	}
}
