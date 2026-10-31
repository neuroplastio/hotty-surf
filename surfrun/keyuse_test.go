package surfrun_test

import (
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// bound is a field, a Select, a Cycle and a button in a list, with
// bindings on the field, the list and the root, some of the same keys:
// each binding's action is its title, which ran logs.
type bound struct {
	theme, mode string
	ran         []string
}

func (m *bound) Render() *surfui.Node {
	field := surfui.Field().Key("f").OnChange(func(e surfrun.Event) { m.ran = append(m.ran, "commit "+e.Value) }).
		Bind("ctrl+s", "Send", "Send").Bind("enter", "Submit", "Submit")
	list := surfui.List(
		surfui.Item(field),
		surfui.Item(surfui.Select(m.theme, themes...).Key("sel").OnChange(func(e surfrun.Event) { m.theme = e.Value })),
		surfui.Item(surfui.Cycle(m.mode, "auto", "cells", "hotty").Key("cyc").OnChange(func(e surfrun.Event) { m.mode = e.Value })),
		surfui.Item(surfui.Button("Go").Key("go")),
	).Key("l").Bind("arrow_down", "Next", "Next").Bind("j", "Down", "Down").
		BindScroll("k", "Up", hotty.ScrollUp).Bind("ctrl+s", "Save all", "Save all")
	return surfui.Surface(list).Bind("z", "Zoom", "Zoom").Bind("mod+k", "Find", "Find").
		Bind("ctrl+q", "Quit", surfrun.Quit{}).Bind("enter", "Open", "Open")
}

func (m *bound) Do(act surfui.Action, ev surfrun.Event) {
	switch a := act.(type) {
	case string:
		m.ran = append(m.ran, a)
	case func(surfrun.Event):
		a(ev)
	}
}

const (
	boundField  = "r.kl.0.kf"
	boundSelect = "r.kl.1.ksel"
	boundCycle  = "r.kl.2.kcyc"
	boundButton = "r.kl.3.kgo"
)

// boundStates are the surface's states that change what Key takes.
var boundStates = []struct {
	name string
	to   func(s *surfrun.Surface)
}{
	{"nothing focused", func(*surfrun.Surface) {}},
	{"the field", focus(boundField)},
	{"the field, typed in", func(s *surfrun.Surface) {
		focus(boundField)(s)
		s.Event(hotty.Event{Kind: hotty.EventInput, Target: boundField, Value: "hi", Rev: 1})
	}},
	{"the Select", focus(boundSelect)},
	{"the Select's list", func(s *surfrun.Surface) { focus(boundSelect)(s); s.Key("space") }},
	{"the Cycle", focus(boundCycle)},
	{"the button", focus(boundButton)},
	{"hidden", func(s *surfrun.Surface) { s.Hide() }},
}

func focus(id string) func(*surfrun.Surface) {
	return func(s *surfrun.Surface) { s.Event(hotty.Event{Kind: hotty.EventFocus, Target: id}) }
}

// keyNames are the keys tried in each state: every key an element takes
// of its own, the bound ones, characters, and some nothing takes.
var keyNames = []string{
	"arrow_up", "arrow_down", "arrow_left", "arrow_right", "home", "end", "page_up", "page_down",
	"enter", "space", "escape", "tab", "backspace", "f5",
	"a", "shift+a", "z", "j", "k", "1", ">",
	"ctrl+s", "ctrl+k", "cmd+k", "ctrl+q", "ctrl+x",
}

// Keys says what Key takes, in each state, with NoQuit or without: a key
// is in Keys, or a character where the element picks by them, exactly
// where Key takes it, and the binding listed first for it is the one that
// runs.
func TestKeysAreWhatKeyTakes(t *testing.T) {
	for _, noQuit := range []bool{false, true} {
		for _, st := range boundStates {
			start := func() (*bound, *quits, *surfrun.Surface) {
				m, q := &bound{theme: "dracula", mode: "auto"}, &quits{}
				s := surfrun.NewSurface(m, q)
				s.NoQuit = noQuit
				s.Show()
				st.to(s)
				return m, q, s
			}
			_, _, s := start()
			keys, chars := s.Keys()
			for _, k := range keyNames {
				m, q, s := start()
				took, _ := s.Key(k)
				i := slices.IndexFunc(keys, func(u surfrun.KeyUse) bool { return u.Name == k && !u.Own })
				own := slices.ContainsFunc(keys, func(u surfrun.KeyUse) bool { return u.Name == k && u.Own })
				want := own || i >= 0 || chars == surfrun.CharsPick && typesChar(k)
				if took != want {
					t.Errorf("%s, NoQuit %v: Key(%q) took %v; Keys %+v, %v", st.name, noQuit, k, took, keys, chars)
					continue
				}
				if i < 0 {
					continue
				}
				switch title := keys[i].Title; {
				case title == "Quit":
					if q.n != 1 {
						t.Errorf("%s: %s is Quit's, which ran %d times", st.name, k, q.n)
					}
				case !slices.Contains(m.ran, title):
					t.Errorf("%s: %s is %q's, but %q ran", st.name, k, title, m.ran)
				}
			}
		}
	}
}

// Keys in some of those states, in full: the focused element's keys, the
// bindings up from it, each key once, then the Escape that blurs.
func TestKeysInOrder(t *testing.T) {
	cases := []struct {
		state string
		want  string
		chars surfrun.Chars
	}{
		{"nothing focused", "z:Zoom ctrl+k:Find ctrl+q:Quit enter:Open", surfrun.CharsBound},
		{"the field", "ctrl+s:Send enter:Submit arrow_down:Next j:Down z:Zoom ctrl+k:Find ctrl+q:Quit escape*", surfrun.CharsText},
		{"the field, typed in", "enter* ctrl+s:Send enter:Submit arrow_down:Next j:Down z:Zoom ctrl+k:Find ctrl+q:Quit escape*", surfrun.CharsText},
		{"the Select", "arrow_up* arrow_down* home* page_up* end* page_down* enter* space* ctrl+s:Save all ctrl+k:Find ctrl+q:Quit escape*", surfrun.CharsPick},
		{"the Select's list", "arrow_up* arrow_down* home* page_up* end* page_down* enter* space* escape* ctrl+s:Save all ctrl+k:Find ctrl+q:Quit", surfrun.CharsPick},
		{"the Cycle", "arrow_left* arrow_up* arrow_right* arrow_down* space* enter* home* end* ctrl+s:Save all ctrl+k:Find ctrl+q:Quit escape*", surfrun.CharsPick},
		{"hidden", "z:Zoom ctrl+k:Find ctrl+q:Quit enter:Open", surfrun.CharsBound},
	}
	for _, c := range cases {
		i := slices.IndexFunc(boundStates, func(s struct {
			name string
			to   func(s *surfrun.Surface)
		}) bool {
			return s.name == c.state
		})
		s := surfrun.NewSurface(&bound{theme: "dracula", mode: "auto"}, nil)
		s.Show()
		boundStates[i].to(s)
		keys, chars := s.Keys()
		var got []string
		for _, k := range keys {
			if k.Own {
				got = append(got, k.Name+"*")
			} else {
				got = append(got, k.Name+":"+k.Title)
			}
		}
		if g := strings.Join(got, " "); g != c.want || chars != c.chars {
			t.Errorf("%s: Keys\n got %s, %v\nwant %s, %v", c.state, g, chars, c.want, c.chars)
		}
	}
	// NoQuit leaves Quit's key out, and a mod+k is read on the Platform.
	s := surfrun.NewSurface(&bound{}, nil)
	s.NoQuit, s.Platform = true, hotty.Platform{Mod: "super"}
	s.Show()
	keys, _ := s.Keys()
	var got []string
	for _, k := range keys {
		got = append(got, k.Name)
	}
	if g := strings.Join(got, " "); g != "z super+k enter" {
		t.Errorf("NoQuit, mod super: Keys %s, want z super+k enter", g)
	}
}

// typesChar is whether key types a character with Shift at most, as a
// Select or a Cycle takes it.
func typesChar(key string) bool {
	key = strings.TrimPrefix(key, "shift+")
	r, n := utf8.DecodeRuneInString(key)
	return n > 0 && n == len(key) && unicode.IsPrint(r) && !unicode.IsSpace(r)
}
