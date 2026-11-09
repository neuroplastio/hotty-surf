package surfrun_test

import (
	"errors"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// pane is a scroll box holding a list whose keys a host would scroll
// with, a row's button, and a field under an element that binds ←, Home
// and Space: each counts what it heard.
type pane struct {
	heard map[string]int
	name  string
}

func (m *pane) Render() *surfui.Node {
	hear := func(what string) func(surfrun.Event) { return func(surfrun.Event) { m.heard[what]++ } }
	list := surfui.List(
		surfui.Item(surfui.Text("a"), surfui.Button("Open").Key("open").OnClick(hear("open"))).Key("a"),
		surfui.Item(surfui.Text("b")).Key("b"),
	).Focusable().Autofocus().Key("list").
		Bind("arrow_down", "Next", hear("next")).Bind("space", "Mark", hear("mark"))
	form := surfui.Stack(
		surfui.Field().Key("name").Value(m.name).OnInput(func(e surfrun.Event) { m.name = e.Value }),
	).Key("form").Bind("arrow_left", "Back", hear("back")).Bind("home", "First", hear("first")).
		Bind("space", "Mark", hear("space"))
	return surfui.Surface(surfui.Stack(list, form).Scroll().Key("box"))
}

func (m *pane) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// The keys a pane binds reach its bindings in a document that scrolls,
// what is in the pane keeps its own: a button Space, a field ←, Home and
// Space.
func TestBoundKeysInAScrollBox(t *testing.T) {
	m := &pane{heard: map[string]int{}}
	a := surftest.Start(t, m, surfrun.Options{Scroll: hotty.ScrollVertical})
	const list, open, field = "r.kbox.klist", "r.kbox.klist.ka.kopen", "r.kbox.kform.kname"
	a.Wait("the list has the keyboard", func() bool { return a.Surface().Focused() == list })
	heard := func(what string, n int) func() bool {
		return func() bool {
			var got int
			a.Call(func() { got = m.heard[what] })
			return got == n
		}
	}

	a.Host.Key("arrow_down")
	a.Wait("the list's ↓", heard("next", 1))
	a.Host.Key("space")
	a.Wait("the list's Space", heard("mark", 1))

	// In the list, the row's button keeps Space, which presses it, and
	// gives ↓ to the list's binding.
	if err := a.Host.Click(a.Name, open); err != nil {
		t.Fatal(err)
	}
	a.Wait("the click", heard("open", 1))
	a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == open })
	a.Host.Key("space")
	a.Wait("Space presses the button", heard("open", 2))
	a.Host.Key("arrow_down")
	a.Wait("the button's ↓, the list's", heard("next", 2))
	a.Call(func() {
		if m.heard["mark"] != 1 {
			t.Errorf("the list's Space heard the button's: %d", m.heard["mark"])
		}
	})

	// A field under ←, Home and Space keeps them: it edits with them. A
	// click focuses it, and reports nothing.
	if err := a.Host.Click(a.Name, field); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		t.Fatal(err)
	}
	a.Wait("the field has the keyboard", func() bool { return a.Surface().Focused() == field })
	for _, k := range []string{"a", "b", "arrow_left", "c", "home", "x", "space"} {
		a.Host.Key(k)
	}
	a.Wait("the field edited", func() bool { v, _ := a.Surface().Value(field); return v == "x acb" })
	a.Call(func() {
		if m.heard["back"] != 0 || m.heard["first"] != 0 || m.heard["space"] != 0 || m.heard["mark"] != 1 {
			t.Errorf("the field gave its keys away: %v", m.heard)
		}
	})
}

// pager is a scroll box that scrolls with j, under a root that binds j:
// heard counts the root's.
type pager struct{ heard int }

func (m *pager) Render() *surfui.Node {
	box := surfui.Stack(surfui.Text("a"), surfui.Text("b")).Scroll().Focusable().Autofocus().Key("box").
		BindScroll("j", "Down", hotty.ScrollDown)
	return surfui.Surface(box).Bind("j", "Jump", 1)
}

func (m *pager) Do(act surfui.Action, ev surfrun.Event) { m.heard += act.(int) }

// A key bound to a scroll action is the host's. Where it reaches the
// program, as on a host that doesn't scroll, it goes on past that binding
// to the next one out.
func TestScrollBindingPassesOn(t *testing.T) {
	m := &pager{}
	a := surftest.Start(t, m, surfrun.Options{Scroll: hotty.ScrollVertical})
	a.Wait("the box has the keyboard", func() bool { return a.Surface().Focused() == "r.kbox" })
	if got, _ := a.Surface().Attr("r.kbox", "data-keys"); got != "j=scroll-down" {
		t.Errorf("the box's data-keys are %q", got)
	}
	a.Host.Key("j")
	a.Wait("the root's j", func() bool { var n int; a.Call(func() { n = m.heard }); return n == 1 })
}
