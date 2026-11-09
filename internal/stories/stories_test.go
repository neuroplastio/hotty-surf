package stories_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surfui"
)

// Every story draws a document at every theme and density, as the gate
// renders them, and no two share a name, as the gate's files would. Under
// TinyGo's wasm (make wasm), it draws them all on a goroutine's fixed
// 64 KB stack.
func TestEveryStoryDraws(t *testing.T) {
	names := map[string]bool{}
	for _, s := range stories.All {
		if names[s.Name] {
			t.Errorf("two stories are named %s", s.Name)
		}
		names[s.Name] = true
		for _, theme := range []string{"graphite", "stone", "neuroplastio", "terminal", "paper"} {
			for _, d := range []surfui.Density{surfui.Dense, surfui.Roomy, surfui.Compact} {
				root := stories.Surface(s, theme, d)
				surfui.Resolve(root, "r")
				doc := stories.Document(root)
				if !strings.HasSuffix(doc, "</body></html>") || !strings.Contains(doc, `class="u-surface`) {
					t.Errorf("%s at %s %s: not a surface's document:\n%.200s", s.Name, theme, d, doc)
				}
			}
		}
	}
	if len(names) == 0 {
		t.Error("no stories")
	}
}

// Beside keeps every rule and adds a forced copy after each that reads
// :focus or :hover, so that a forced node and a real one draw alike.
func TestBeside(t *testing.T) {
	got := stories.Beside(".a { x: 1; }\n.a:hover, .b:focus-visible { y: 2; }\n@media (m) {\n  .c:focus-within { z: 3; }\n}\n")
	want := ".a { x: 1; }\n.a:hover, .b:focus-visible { y: 2; }\n.a[data-force~=hover], .b[data-force~=focus] { y: 2; }\n@media (m) {\n  .c:focus-within { z: 3; }\n  .c[data-force~=within] { z: 3; }\n}\n"
	if got != want {
		t.Errorf("Beside:\n%s\nwant:\n%s", got, want)
	}
	all := stories.Beside(surfui.CSS)
	if strings.Count(all, "[data-force~=") != strings.Count(stories.CSS(), "[data-force~=") {
		t.Errorf("Beside(surfui.CSS) has %d forced selectors, CSS() %d", strings.Count(all, "[data-force~="), strings.Count(stories.CSS(), "[data-force~="))
	}
	if strings.Count(all, ":hover") != strings.Count(surfui.CSS, ":hover") {
		t.Error("Beside(surfui.CSS) lost a real :hover rule")
	}
}
