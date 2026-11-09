package cells_test

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// focuser is a host's runner, which gives Focus to its surface.
type focuser struct{ s *surfrun.Surface }

func (f *focuser) Post(func())      {}
func (f *focuser) Quit()            {}
func (f *focuser) Focus(key string) { f.s.Focus(key) }

// palette is plx's palette in the cells, as run's test has it on a
// host: a query field that commits, with a key of its own, and a stop
// after it whose focus gives the keyboard to the element keyed to.
type palette struct {
	rn        surfrun.Runner
	to        string
	commits   []string
	completes int
	focuses   int
	next      int
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
	)
}

func (m *palette) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// showPalette is m in a renderer, its surface giving the program keys.
func showPalette(m *palette, keys ...string) (*cells.Renderer, *surfrun.Surface) {
	rn := &focuser{}
	s := surfrun.NewSurface(m, rn)
	rn.s = s
	s.Program = keys
	r := cells.New(s.Cells())
	r.Show()
	return r, s
}

// The host program's keys in the cells, as on a host: the field gives
// the program those it can do without, an edit's among them, Enter once
// it has committed, and keeps a letter and the arrows across; its own
// binding is its; the renderer's Keys has none of the program's as its
// own.
func TestProgramKeys(t *testing.T) {
	m := &palette{}
	r, _ := showPalette(m, "enter", "arrow_up", "arrow_down", "ctrl+p", "ctrl+n", "ctrl+backspace", "a", "arrow_left")
	if r.Focused() != "r.kquery" {
		t.Fatalf("focused %q", r.Focused())
	}
	keys, _ := r.Keys()
	for _, u := range keys {
		if u.Own && slices.Contains([]string{"enter", "arrow_up", "arrow_down", "ctrl+p", "ctrl+backspace"}, u.Name) {
			t.Errorf("Keys: %s is the renderer's own", u.Name)
		}
	}
	for _, k := range []string{"ctrl+p", "arrow_up", "arrow_down", "ctrl+backspace"} {
		if taken, _ := r.Key(k); taken {
			t.Errorf("%s: taken, the program's", k)
		}
	}
	for _, k := range []string{"s", "a", "p", "arrow_left"} {
		if taken, _ := r.Key(k); !taken {
			t.Errorf("%s: the field's, not taken", k)
		}
	}
	if taken, _ := r.Key("ctrl+n"); !taken || m.next != 1 {
		t.Errorf("ctrl+n, the field's own: taken %v, ran %d", taken, m.next)
	}
	if taken, _ := r.Key("enter"); taken || !slices.Equal(m.commits, []string{"sap"}) {
		t.Errorf("enter: taken %v, commits %q; want the commit, and enter the program's", taken, m.commits)
	}

	// Without them the field deletes a word with ctrl+backspace.
	r, _ = showPalette(&palette{})
	if taken, _ := r.Key("ctrl+backspace"); !taken {
		t.Error("ctrl+backspace with no Program: not the field's")
	}
}

// Focus from a handler, as Tab lands on the stop: the renderer gives the
// element the keyboard, and sends no focus event for it, as a host
// sends none (SPEC §10.1). A key no element has does nothing; a field
// Focus moves the keyboard from commits.
func TestFocus(t *testing.T) {
	for _, c := range []struct {
		to, want string
	}{{"query", "r.kquery"}, {"gone", "r.kstop"}} {
		m := &palette{to: c.to}
		r, _ := showPalette(m)
		r.Key("tab")
		if m.completes != 1 || r.Focused() != c.want || m.focuses != 0 {
			t.Errorf("Focus(%q): the stop ran %d, focused %q, the field heard focus %d; want 1, %q, 0",
				c.to, m.completes, r.Focused(), m.focuses, c.want)
		}
	}

	m := &palette{}
	r, s := showPalette(m)
	r.Key("s")
	r.Key("p")
	s.Focus("stop")
	r.Update()
	if r.Focused() != "r.kstop" || !slices.Equal(m.commits, []string{"sp"}) || m.completes != 0 {
		t.Errorf("focused %q, commits %q, the stop heard focus %d; want the stop, sp, 0", r.Focused(), m.commits, m.completes)
	}
}
