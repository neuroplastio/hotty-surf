package surfrun_test

import (
	"slices"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// palette is plx's palette as plx draws it on a host: a query field
// the keys start in, which commits, with a key of its own; a stop after
// it that Tab lands on, which completes the line and gives the keyboard
// to the element keyed to (Runner.Focus); and its list.
type palette struct {
	rn        surfrun.Runner
	to        string
	commits   []string
	completes int
	focuses   int // the field's OnFocus
	next      int // the field's ctrl+n
}

func (m *palette) Attach(r surfrun.Runner) { m.rn = r }

func (m *palette) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Field().Key("query").Autofocus().
			OnFocus(func(surfrun.Event) { m.focuses++ }).
			OnChange(func(e surfrun.Event) { m.commits = append(m.commits, e.Value) }).
			Bind("ctrl+n", "Next", func(surfrun.Event) { m.next++ }),
		surfui.Text("Complete").Key("stop").Focusable().OnFocus(func(surfrun.Event) {
			m.completes++
			if m.to != "" {
				m.rn.Focus(m.to)
			}
		}),
		surfui.List(surfui.Item(surfui.Text("Pane: Split right"))).Key("list"),
	)
}

func (m *palette) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// plx's palette keymap context: its keys, a letter and an arrow across
// among them, which the field keeps.
var paletteKeys = []string{"enter", "arrow_up", "arrow_down", "ctrl+p", "ctrl+n", "ctrl+b", "a", "arrow_left"}

// The host program's keys (Surface.Program) are the root's to give: the
// field gives the program those it can do without and binds the arrow
// across back to its edit, the root gives the program the arrows it
// would scroll with; Key reports each not taken, Enter once it has
// committed the field, but one the field binds itself, and Keys lists
// none of them; Escape given the program blurs nothing.
func TestProgramKeys(t *testing.T) {
	m := &palette{}
	s := surfrun.NewSurface(m, nil)
	s.Program = paletteKeys
	s.Show()
	ix := surfui.Resolve(s.Tree(), "r")
	if got, want := surfui.DataKeys(ix["r.kquery"]), "enter=program ctrl+n=program arrow_up=program arrow_down=program ctrl+p=program ctrl+b=program arrow_left=char-backward"; got != want {
		t.Errorf("the field's data-keys:\n%s\nwant\n%s", got, want)
	}
	if got, want := surfui.DataKeys(ix["r"]), "arrow_up=program arrow_down=program arrow_left=program"; got != want {
		t.Errorf("the root's data-keys: %q, want %q", got, want)
	}
	for _, k := range []string{"ctrl+p", "arrow_down", "ctrl+b", "enter"} {
		if took, _ := s.Key(k); took {
			t.Errorf("%s: taken, the program's", k)
		}
	}
	if took, _ := s.Key("ctrl+n"); !took || m.next != 1 {
		t.Errorf("ctrl+n, the field's own: taken %v, ran %d", took, m.next)
	}
	keys, _ := s.Keys()
	var names []string
	for _, u := range keys {
		names = append(names, u.Name)
	}
	if !slices.Equal(names, []string{"ctrl+n", "escape"}) {
		t.Errorf("Keys: %q, want the field's ctrl+n and the Escape that blurs", names)
	}

	s.Event(hotty.Event{Kind: hotty.EventInput, Target: "r.kquery", Value: "sp", Rev: 1})
	if took, redraw := s.Key("enter"); took || !redraw || !slices.Equal(m.commits, []string{"sp"}) {
		t.Errorf("enter on an edit: taken %v, redraw %v, commits %q; want the commit, and enter the program's", took, redraw, m.commits)
	}

	// Escape given the program blurs nothing: the program closes the
	// palette.
	s = surfrun.NewSurface(&palette{}, nil)
	s.Program = []string{"escape"}
	s.Show()
	if took, _ := s.Key("escape"); took || s.Focused() != "r.kquery" {
		t.Errorf("escape given the program: taken %v, focused %q", took, s.Focused())
	}
}

// A handler gives the keyboard to an element by its key (Runner.Focus),
// mid-Event, as the palette's stop does when Tab lands on it: the next
// frame focuses it on the host, which sends no focus
// event for it (SPEC §10.1), so its OnFocus doesn't run. A key no
// element has does nothing.
func TestFocusFromAHandler(t *testing.T) {
	for _, c := range []struct {
		to, want string
	}{{"query", "r.kquery"}, {"gone", "r.kstop"}, {"list", "r.kstop"}} {
		m := &palette{to: c.to}
		r := surftest.NewRelay(t, m, surfrun.Options{})
		s, h := r.Surface(), r.Host
		r.Settle()
		h.Key("tab")
		r.Settle()
		if m.completes != 1 {
			t.Errorf("Focus(%q): the stop's focus ran %d times", c.to, m.completes)
		}
		if s.Focused() != c.want || h.Surface(r.Name).Focused() != c.want {
			t.Errorf("Focus(%q): the surface has %q, the host %q; want %q", c.to, s.Focused(), h.Surface(r.Name).Focused(), c.want)
		}
		if m.focuses != 0 {
			t.Errorf("Focus(%q): the field heard focus %d times, want none", c.to, m.focuses)
		}
	}
}

// A field Focus moves the keyboard from commits what the user typed, as
// leaving it does: the host sends change.
func TestFocusCommits(t *testing.T) {
	m := &palette{}
	r := surftest.NewRelay(t, m, surfrun.Options{})
	s, h := r.Surface(), r.Host
	r.Settle()
	h.Key("s")
	h.Key("p")
	r.Settle()
	s.Focus("stop")
	r.Update()
	r.Settle()
	if !slices.Equal(m.commits, []string{"sp"}) || s.Focused() != "r.kstop" {
		t.Errorf("commits %q, focused %q; want sp, and the stop", m.commits, s.Focused())
	}
	if m.completes != 0 {
		t.Errorf("the stop heard focus %d times, want none", m.completes)
	}
}

// While the host has the keyboard (Lend), Focus picks the element it is
// given back to; hidden, the surface drops it.
func TestFocusLentAndHidden(t *testing.T) {
	m := &palette{}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Lend(true)
	s.Focus("stop")
	if f := s.Update(); f.Focus != "" {
		t.Errorf("lent, the frame focuses %q", f.Focus)
	}
	if f := s.Lend(false); f.Focus != "r.kstop" {
		t.Errorf("given back to %q, want the stop", f.Focus)
	}

	s.Hide()
	s.Focus("stop")
	if f := s.Show(); f.Focus != "r.kquery" {
		t.Errorf("shown, the frame focuses %q, want the field", f.Focus)
	}
	if f := s.Update(); f.Focus != "" {
		t.Errorf("hidden, Focus waited: the frame focuses %q", f.Focus)
	}
}
