package surfui_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/internal/book/stories"
	"neuroplast.io/hotty-surf/surfui"
	"neuroplast.io/hotty-surf/surfui/dialog"
	"neuroplast.io/hotty-surf/surfui/input"
)

type dialogApp struct {
	open bool
	name string
}

func (a *dialogApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			button("open", "Open", func() { a.open = true }),
			with(dialog.View(dialog.Props{ID: "d", Open: a.open, Title: "Profile", OnClose: func() { a.open = false }}),
				input.View(input.Props{ID: "name", Value: a.name, OnInput: func(v string) { a.name = v }})),
		).Render(ctx, w)
	})
}

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
			lines := inCells(t, st, 80, 24, st.Keys)
			if text := strings.Join(lines, "\n"); !strings.Contains(text, st.Want) {
				// An open list is drawn over what is under it, as a host
				// draws it: the user closes it to read what it covers.
				t.Logf("%s: %q not shown with the keys alone; read again after escape", st.ID, st.Want)
				lines = inCells(t, st, 80, 24, append(st.Keys[:len(st.Keys):len(st.Keys)], "escape"))
			}
			if text := strings.Join(lines, "\n"); !strings.Contains(text, st.Want) {
				t.Errorf("after %v the lines don't show %q:\n%s", st.Keys, st.Want, text)
			}
		})
	}
}

func TestDialogModalInCells(t *testing.T) {
	a := &dialogApp{}
	s := surf.New("modal", a.view(), surfui.Stylesheet())
	c := surftui.New(80, 24, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Fatal(err)
		}
	})
	s.Attach(c, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Write([]byte(hotty.Doc(s.Name(), doc))); err != nil {
		t.Fatal(err)
	}
	key := func(k string) {
		if !c.Key(k) {
			if _, err := s.Key(k); err != nil {
				t.Fatal(err)
			}
		}
	}
	key("tab")
	key("enter")
	if got := rt.Peek.Focused(s); got != "name" {
		t.Fatalf("opened on %q", got)
	}
	key("shift+tab")
	if got := rt.Peek.Focused(s); got != "d-close" {
		t.Fatalf("reverse wrap focused %q", got)
	}
	key("tab")
	if got := rt.Peek.Focused(s); got != "name" {
		t.Fatalf("forward wrap focused %q", got)
	}
	key("escape")
	if a.open {
		t.Fatal("Escape did not close the dialog")
	}
}

// inCells puts the story on a surftui.Screen, presses its keys, and gives
// back the lines.
func inCells(t *testing.T, st stories.Story, cols, rows int, keys []string) []string {
	t.Helper()
	s := surf.New("story", st.View(), surfui.Stylesheet())
	c := surftui.New(cols, rows, func(e hotty.Event) {
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
	for _, k := range keys {
		if !c.Key(k) {
			if _, err := s.Key(k); err != nil {
				t.Fatalf("key %s: %v", k, err)
			}
		}
		update()
	}
	return c.Lines()
}
