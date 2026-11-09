package screens

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
)

// A mode is a way the screens are drawn (conformance's modes): its Mode,
// and each screen's component in it.
type mode struct {
	Mode
	agents   func(*Agents) surflive.Component
	notices  func(*Notices) surflive.Component
	palette  func(*Palette) surflive.Component
	settings func(*Settings) surflive.Component
}

// modes are the ways the screens are drawn: each test of a screen holds
// in both.
var modes = []mode{
	{Interpreted, surfInterpAgents, surfInterpNotices, surfInterpPalette, surfInterpSettings},
	{Compiled, surfCompiledAgents, surfCompiledNotices, surfCompiledPalette, surfCompiledSettings},
}

// inModes runs f in each mode, as a subtest named for it.
func inModes(t *testing.T, f func(t *testing.T, m mode)) {
	for _, m := range modes {
		t.Run(string(m.Mode), func(t *testing.T) { f(t, m) })
	}
}

// app is one of the screens running on the mock, on a hottytest host.
type app struct {
	*surftest.App
	t   *testing.T
	src actor
}

// actor is a screen's source as a test reads it: the acts the screen
// asked it for, in order.
type actor interface{ Acts() []string }

// start runs root, a screen on the mock src, on a host cols by rows cells,
// as cmd/surfdemo runs it, and waits for the keyboard to be in its place.
func start(t *testing.T, root surflive.Component, src actor, scroll bool, cols, rows int) *app {
	t.Helper()
	opts := surfrun.Options{Name: "a"}
	if scroll {
		opts.Scroll = hotty.Axes(1)
	}
	a := &app{App: surftest.Start(t, surflive.New(root), opts, hottytest.Size(cols, rows)), t: t, src: src}
	a.Wait("the keyboard in its place", func() bool { return a.Surface().Focused() != "" })
	return a
}

// id is the element's id under the key path keys, each a key (a string:
// its "k" and the key as the id has it) or a child's index (an int).
func id(keys ...any) string {
	out := "r"
	for _, k := range keys {
		switch k := k.(type) {
		case int:
			out += "." + strconv.Itoa(k)
		case string:
			out += ".k" + safeID(k)
		}
	}
	return out
}

// safeID is a key as an id has it (surfui's idSafe): as it is where it is
// letters, digits, - and _, else each other byte, and _, as _ and its hex.
func safeID(s string) string {
	letter := func(c byte) bool {
		return c == '-' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
	}
	if strings.IndexFunc(s, func(r rune) bool { return r > 127 || r != '_' && !letter(byte(r)) }) < 0 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if c := s[i]; letter(c) {
			b.WriteByte(c)
		} else {
			b.WriteString("_" + strconv.FormatInt(int64(c), 16))
		}
	}
	return b.String()
}

func (a *app) click(id string) {
	a.t.Helper()
	if err := a.Host.Click(a.Name, id); err != nil {
		a.t.Fatal(err)
	}
}

func (a *app) double(id string) {
	a.t.Helper()
	if err := a.Host.DoubleClick(a.Name, id); err != nil {
		a.t.Fatal(err)
	}
}

func (a *app) fill(id, text string) {
	a.t.Helper()
	if err := a.Host.Fill(a.Name, id, text); err != nil {
		a.t.Fatal(err)
	}
}

func (a *app) keys(keys ...string) {
	for _, k := range keys {
		a.Host.Key(k)
	}
}

// text is the element's text on the host.
func (a *app) text(id string) string { return a.Surface().TextOf(id) }

// value is a field's value on the host.
func (a *app) value(id string) string {
	v, _ := a.Surface().Value(id)
	return v
}

// html is the surface's document on the host.
func (a *app) html() string { return a.Surface().HTML() }

// until waits for ok, saying what it waits for, and shows the document
// if it never holds.
func (a *app) until(what string, ok func() bool) {
	a.t.Helper()
	a.Wait(what, func() bool {
		var held bool
		a.Call(func() { held = ok() })
		return held
	})
}

// acted waits until the mock has recorded want as its acts.
func (a *app) acted(want ...string) {
	a.t.Helper()
	a.Wait("the acts "+strings.Join(want, "; "), func() bool { return slices.Equal(a.src.Acts(), want) })
}

// cells switches the run to its cells UI, as a host's failure does.
func (a *app) cells() { a.Call(a.Run.Fail) }

// screenHas waits until the terminal's cells show text.
func (a *app) screenHas(text string) {
	a.t.Helper()
	a.Wait("the cells to show "+text, func() bool { return strings.Contains(a.Host.Screen(), text) })
}

// opsSince are the ops sent after the first n.
func (a *app) opsSince(n int) []hotty.Op { return a.Ops()[n:] }

// Each of Names is a screen, in each mode; any other name is none.
func TestNew(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		for _, name := range Names {
			if s, ok := New(name, m.Mode); !ok || s.Name != name || s.Model == nil {
				t.Errorf("New(%s) = %+v, %v", name, s, ok)
			}
		}
		if _, ok := New("nope", m.Mode); ok {
			t.Error("a screen called nope")
		}
	})
}
