package ui_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

// TestKeysCells is the keys-cells test of ui/coverage.tsv: every story with
// keys is used by keyboard alone in cells, the rendition of a terminal
// that is no hotty host, and its lines then show what it should. A key the
// Screen doesn't take goes to the surface, as run.App's does.
func TestKeysCells(t *testing.T) {
	for _, st := range stories.All() {
		if len(st.Keys) == 0 {
			continue
		}
		t.Run(st.ID, func(t *testing.T) {
			if st.Gap != "" {
				t.Skip(st.Gap)
			}
			lines := inCells(t, st, 80, 24)
			if text := strings.Join(lines, "\n"); !strings.Contains(text, st.Want) {
				t.Errorf("after %v the lines don't show %q:\n%s", st.Keys, st.Want, text)
			}
		})
	}
}

// inCells puts the story on a cells.Screen, presses its keys, and gives
// back the lines.
func inCells(t *testing.T, st stories.Story, cols, rows int) []string {
	t.Helper()
	s := surf.New("story", st.View(), ui.Stylesheet())
	c := cells.New(cols, rows, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Errorf("event %s %s: %v", e.Kind, e.Target, err)
		}
	})
	s.Attach(c, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte(hotty.Doc(s.Name(), doc))); err != nil {
		t.Fatal(err)
	}
	update := func() {
		for {
			select {
			case <-s.Pending():
				if err := s.Update(); err != nil {
					t.Fatal(err)
				}
			default:
				return
			}
		}
	}
	update()
	for _, k := range st.Keys {
		if !c.Key(k) {
			if _, err := s.Key(k); err != nil {
				t.Fatalf("key %s: %v", k, err)
			}
		}
		update()
	}
	return c.Lines()
}
