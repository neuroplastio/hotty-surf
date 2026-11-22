package icon

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// View's doc lists the kit's icons, which an app can't see otherwise:
// every one, and no other.
func TestIconDocListsTheIcons(t *testing.T) {
	src, err := os.ReadFile("icon.templ")
	if err != nil {
		t.Fatal(err)
	}
	doc := regexp.MustCompile(`(?s)// View is the icon named name.*?\ntempl View`).Find(src)
	if doc == nil {
		t.Fatal("no doc on View")
	}
	text := strings.NewReplacer("//", " ", "\n", " ").Replace(string(doc))
	names := iconNames()
	listed := regexp.MustCompile(`[a-z0-9]+(?:-[a-z0-9]+)*`).FindAllString(text[strings.Index(text, "draw with:"):strings.Index(text, "Another name")], -1)
	listed = slices.DeleteFunc(listed, func(w string) bool { return w == "draw" || w == "with" || w == "and" })
	if !slices.Equal(listed, names) {
		t.Errorf("View's doc lists %q, the kit has %q", listed, names)
	}
}
