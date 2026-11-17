package cells

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

// storyScreen shows st on a Screen cols × rows, its surface driven as a
// program drives one: the Screen's events to the surface, and every
// update the surface has pending sent before the next key.
type storyScreen struct {
	t      *testing.T
	s      *surf.Surface
	c      *Screen
	clicks int  // the clicks the Screen sent
	taken  bool // whether the surface's Key handlers took the last key
}

func openStory(t *testing.T, st stories.Story, cols, rows int) *storyScreen {
	t.Helper()
	r := &storyScreen{t: t, s: surf.New("story", st.View(), ui.Stylesheet())}
	r.c = New(cols, rows, func(e hotty.Event) {
		if e.Kind == hotty.EventClick {
			r.clicks++
		}
		if err := r.s.Event(e); err != nil {
			t.Errorf("event %s %s: %v", e.Kind, e.Target, err)
		}
	})
	r.s.Attach(r.c, hotty.Caps{})
	doc, err := r.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.c.Write([]byte(hotty.Doc(r.s.Name(), doc))); err != nil {
		t.Fatal(err)
	}
	r.c.Focus(true)
	r.update()
	return r
}

func (r *storyScreen) update() {
	for {
		select {
		case <-r.s.Pending():
			if err := r.s.Update(); err != nil {
				r.t.Fatal(err)
			}
		default:
			return
		}
	}
}

// key presses k as a program in cells does: the Screen first, then the
// surface's Key handlers.
func (r *storyScreen) key(k string) {
	r.taken = false
	if !r.c.Key(k) {
		taken, err := r.s.Key(k)
		if err != nil {
			r.t.Fatalf("key %s: %v", k, err)
		}
		r.taken = taken
	}
	r.update()
}

// name is how a failure names n: its id, or its tag and text.
func name(n *html.Node) string {
	if id := idOf(n); id != "" && !strings.HasPrefix(id, "_") {
		return "#" + id
	}
	return "<" + n.Data + "> " + textOf(n)
}

// The gate (L3): every story is usable by keyboard in cells.
//
//   - Tab goes round every control drawn, each in turn, and the one with
//     the keyboard is drawn reversed.
//   - Every control drawn that the app listens to (data-on) can take the
//     keyboard: Tab reaches it, or it is one of a group the arrows move
//     in (a tabindex of -1 beside one Tab reaches).
//   - A story's Keys do what its Want says (ui's TestKeysCells).
func TestEveryStoryByKeyboard(t *testing.T) {
	for _, st := range stories.All() {
		t.Run(st.ID, func(t *testing.T) {
			r := openStory(t, st, 80, 60)
			l := r.c.layout()
			var tabs []*html.Node
			walk(r.c.body, func(n *html.Node) {
				if _, shown := l.boxes[n]; shown && tabbable(n) {
					tabs = append(tabs, n)
				}
			})
			// Tab round, once.
			seen := map[*html.Node]bool{}
			for i := range tabs {
				r.key("tab")
				f := r.c.focused
				if f == nil {
					t.Fatalf("tab %d: nothing has the keyboard", i+1)
				}
				if !reversed.MatchString(strings.Join(r.c.Lines(), "\n")) &&
					!isTextField(f) {
					t.Errorf("tab %d: %s has the keyboard, and nothing is drawn reversed", i+1, name(f))
				}
				seen[f] = true
			}
			if len(seen) < len(tabs) {
				var missed []string
				for _, n := range tabs {
					if !seen[n] {
						missed = append(missed, name(n))
					}
				}
				t.Errorf("tab went round %d of %d controls; not to %s", len(seen), len(tabs), strings.Join(missed, ", "))
			}
			// What the app listens to takes the keyboard.
			r2 := openStory(t, st, 80, 60)
			l = r2.c.layout()
			walk(r2.c.body, func(n *html.Node) {
				on := attr(n, "data-on")
				if on == "" || n.Type != html.ElementNode {
					return
				}
				if _, shown := l.boxes[n]; !shown {
					return
				}
				if !listens(on) || disabled(n) {
					return
				}
				switch {
				case tabbable(n):
				case focusable(n) && roving(n):
					if !arrows(t, st, idOf(n)) {
						t.Errorf("%s listens for %q, and the arrows don't reach it from its group", name(n), on)
					}
				case tokens(n, "data-on", "submit") && n.DataAtom == atom.Form:
				case controller(r2.c, n) != "":
					// An option of a combobox: the arrows make it the
					// field's active one, and Enter picks it.
					if !picks(t, st, controller(r2.c, n), idOf(n)) {
						t.Errorf("%s listens for %q, and the arrows and Enter in #%s don't pick it", name(n), on, controller(r2.c, n))
					}
					return
				default:
					t.Errorf("%s listens for %q, and the keyboard can't reach it", name(n), on)
					return
				}
				if tokens(n, "data-on", "click") && !activatable(n) && !isField(n) {
					// Not a button: Enter or Space has to do what a click does.
					id := idOf(n)
					if !enters(t, st, id) {
						t.Errorf("%s listens for clicks, and neither Enter nor Space on it does anything", name(n))
					}
				}
			})
		})
	}
}

// enters says whether Enter or Space, on the element id of a fresh
// st, clicks it or is taken by the surface's Key handlers.
func enters(t *testing.T, st stories.Story, id string) bool {
	for _, k := range []string{"enter", "space"} {
		r := openStory(t, st, 80, 60)
		n := r.c.byID(id)
		if n == nil {
			return false
		}
		r.c.focusTo(n, true)
		r.c.flush()
		r.update()
		r.key(k)
		if r.clicks > 0 || r.taken {
			return true
		}
	}
	return false
}

// arrows says whether the element id of a fresh st, in a group the
// arrows move in, takes the keyboard by the arrows from the group's
// member Tab reaches.
func arrows(t *testing.T, st stories.Story, id string) bool {
	for _, k := range []string{"arrow_right", "arrow_down", "arrow_left", "arrow_up"} {
		r := openStory(t, st, 80, 60)
		n := r.c.byID(id)
		if n == nil {
			return false
		}
		// Tab to the group.
		for range 40 {
			r.key("tab")
			if f := r.c.focused; f != nil && f.Parent == n.Parent {
				break
			}
		}
		for range 20 {
			if focusedID(r.c) == id {
				return true
			}
			r.key(k)
		}
		if focusedID(r.c) == id {
			return true
		}
	}
	return false
}

// controller is the id of the field whose aria-controls holds n most
// closely (a combobox's list, inside a popover a trigger controls), ""
// for none.
func controller(c *Screen, n *html.Node) string {
	out, best := "", -1
	walk(c.body, func(k *html.Node) {
		if !tabbable(k) {
			return
		}
		for _, id := range strings.Fields(attr(k, "aria-controls")) {
			list := c.byID(id)
			if list == nil {
				continue
			}
			depth := 0
			for p := list; p != nil; p = p.Parent {
				depth++
			}
			if closest(n, func(a *html.Node) bool { return a == list }) != nil && depth > best {
				out, best = idOf(k), depth
			}
		}
	})
	return out
}

// picks says whether, on a fresh st, the arrows in the field make the
// option id its active one (aria-activedescendant), and Enter then is
// taken.
func picks(t *testing.T, st stories.Story, field, id string) bool {
	r := openStory(t, st, 80, 60)
	for range 40 {
		if focusedID(r.c) == field {
			break
		}
		r.key("tab")
	}
	if focusedID(r.c) != field {
		return false
	}
	for range 30 {
		if attr(r.c.byID(field), "aria-activedescendant") == id {
			r.key("enter")
			return r.taken || r.clicks > 0
		}
		r.key("arrow_down")
	}
	return false
}

// focusedID is the id of the element with the keyboard, "" for none.
func focusedID(c *Screen) string {
	if c.focused == nil {
		return ""
	}
	return idOf(c.focused)
}

// reversed is an SGR that draws reversed.
var reversed = regexp.MustCompile(`\x1b\[[0-9;]*;7[;m]`)

// listens says whether data-on names an event a user makes with a key:
// a click, an edit, a submit.
func listens(on string) bool {
	for _, e := range strings.Fields(on) {
		switch e {
		case "click", "input", "change", "submit":
			return true
		}
	}
	return false
}

// roving says whether n, focusable but not by Tab, is in a group the
// arrows move in: a sibling, or a sibling's, Tab reaches.
func roving(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		found := false
		walk(p, func(k *html.Node) {
			if k != n && tabbable(k) {
				found = true
			}
		})
		if found {
			return true
		}
		if attr(p, "role") != "" {
			return false
		}
	}
	return false
}
