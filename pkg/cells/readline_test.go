package cells_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// lineModel is a surface with one field that commits, readline on the
// surface or off, and a Bind on ctrl+t.
type lineModel struct {
	readline bool
	bound    int
	commits  []string
}

func (m *lineModel) Attach(surfrun.Runner) {}
func (m *lineModel) Render() *surfui.Node {
	f := surfui.Field().Key("q").Autofocus().
		OnChange(func(e surfrun.Event) { m.commits = append(m.commits, e.Value) }).
		Bind("ctrl+t", "T", func(surfrun.Event) { m.bound++ })
	s := surfui.Surface(f)
	if m.readline {
		s.Readline()
	}
	return s
}
func (m *lineModel) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// typed shows m on platform mod, types text, sends keys and returns the
// text Enter commits, and the keys not taken (the program's).
func typed(t *testing.T, m *lineModel, mod, text string, program []string, keys ...string) (string, []string) {
	t.Helper()
	rn := &focuser{}
	s := surfrun.NewSurface(m, rn)
	rn.s = s
	s.Program = program
	r := cells.New(s.Cells())
	r.Platform = hotty.Platform{Mod: mod}
	r.Show()
	for _, c := range text {
		name := string(c)
		if c == ' ' {
			name = "space"
		}
		if taken, _ := r.Key(name); !taken {
			t.Fatalf("%q: not taken", name)
		}
	}
	var gave []string
	for _, k := range keys {
		if taken, _ := r.Key(k); !taken {
			gave = append(gave, k)
		}
	}
	m.commits = nil
	r.Key("enter")
	if len(m.commits) != 1 {
		t.Fatalf("commits %q", m.commits)
	}
	return m.commits[0], gave
}

// The readline keys in the cells, per platform and mode, on "one two"
// with the caret at its end; a key's effect is seen in what Enter commits
// after "x" is typed at the caret.
func TestReadlineCells(t *testing.T) {
	for _, mod := range []string{"super", "ctrl"} {
		for _, c := range []struct{ key, want string }{
			{"ctrl+backspace", "one x"}, {"ctrl+w", "one x"}, {"alt+backspace", "one x"},
			{"ctrl+u", "x"}, {"ctrl+k", "one twox"},
			{"ctrl+e", "one twox"}, {"alt+f", "one twox"},
			{"alt+b", "one xtwo"},
		} {
			got, _ := typed(t, &lineModel{readline: true}, mod, "one two", nil, c.key, "x")
			if got != c.want {
				t.Errorf("%s readline, %s: %q, want %q", mod, c.key, got, c.want)
			}
		}
		// ctrl+a to the start (ctrl platforms select all: a type replaces).
		got, _ := typed(t, &lineModel{readline: true}, mod, "one two", nil, "ctrl+a", "x")
		want := "xone two"
		if mod == "ctrl" {
			want = "x"
		}
		if got != want {
			t.Errorf("%s readline, ctrl+a: %q, want %q", mod, got, want)
		}
		got, _ = typed(t, &lineModel{readline: true}, mod, "one two", nil, "alt+b", "alt+d", "x")
		if got != "one x" {
			t.Errorf("%s readline, alt+b alt+d: %q", mod, got)
		}
	}
}

// The maintainer's case: a Mac (super), readline on, "one two" with the
// caret at its end, ctrl+backspace gives "one ".
func TestReadlineCtrlBackspaceOnAMac(t *testing.T) {
	got, _ := typed(t, &lineModel{readline: true}, "super", "one two", nil, "ctrl+backspace")
	if got != "one " {
		t.Errorf("committed %q, want %q", got, "one ")
	}
}

// Without readline, on a Mac, ctrl+backspace deletes no word: today's
// default. On a ctrl platform it does.
func TestReadlineOffCells(t *testing.T) {
	got, _ := typed(t, &lineModel{}, "super", "one two", nil, "ctrl+backspace")
	if got != "one two" {
		t.Errorf("super: committed %q, want the text as typed", got)
	}
	got, _ = typed(t, &lineModel{}, "ctrl", "one two", nil, "ctrl+backspace")
	if got != "one " {
		t.Errorf("ctrl: committed %q", got)
	}
}

// A Bind on ctrl+t wins; a key the program reserves reaches it from the
// field, with readline on and off, where the platform's default (super:
// ctrl+b is char-backward) would edit; unreserved, ctrl+b moves the caret.
func TestReservedKeyCells(t *testing.T) {
	for _, readline := range []bool{false, true} {
		name := map[bool]string{false: "off", true: "on"}[readline]
		m := &lineModel{readline: readline}
		got, gave := typed(t, m, "super", "one two", []string{"ctrl+b"}, "ctrl+b", "x")
		if got != "one twox" || strings.Join(gave, " ") != "ctrl+b" {
			t.Errorf("readline %s, reserved: committed %q, program's %q", name, got, gave)
		}
		got, gave = typed(t, &lineModel{readline: readline}, "super", "one two", nil, "ctrl+b", "x")
		if got != "one twxo" || len(gave) != 0 {
			t.Errorf("readline %s, not reserved: committed %q, program's %q", name, got, gave)
		}
		m = &lineModel{readline: readline}
		typed(t, m, "super", "one two", nil, "ctrl+t")
		if m.bound != 1 {
			t.Errorf("readline %s: Bind on ctrl+t ran %d times", name, m.bound)
		}
	}
}

// A Bind on ctrl+w beats readline's.
func TestReadlineBindWinsCells(t *testing.T) {
	m := &bindW{}
	got, _ := typed2(t, m)
	if got != "one two" || m.ran != 1 {
		t.Errorf("committed %q, bind ran %d", got, m.ran)
	}
}

type bindW struct {
	ran     int
	commits []string
}

func (m *bindW) Attach(surfrun.Runner) {}
func (m *bindW) Render() *surfui.Node {
	return surfui.Surface(surfui.Field().Key("q").Autofocus().
		OnChange(func(e surfrun.Event) { m.commits = append(m.commits, e.Value) }).
		Bind("ctrl+w", "W", func(surfrun.Event) { m.ran++ })).Readline()
}
func (m *bindW) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

func typed2(t *testing.T, m *bindW) (string, []string) {
	rn := &focuser{}
	s := surfrun.NewSurface(m, rn)
	rn.s = s
	r := cells.New(s.Cells())
	r.Platform = hotty.Platform{Mod: "super"}
	r.Show()
	for _, k := range []string{"o", "n", "e", "space", "t", "w", "o", "ctrl+w", "enter"} {
		r.Key(k)
	}
	if len(m.commits) != 1 {
		t.Fatalf("commits %q", m.commits)
	}
	return m.commits[0], nil
}
