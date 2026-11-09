package surfrun_test

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// committer is a setting drawn as a Field that applies on its OnChange
// alone, as a settings page's does, with a button beside it and a count
// the model changes by itself.
type committer struct {
	value   string
	commits []string
	refuse  bool // the model does not take what is committed
	ticks   int
}

func (m *committer) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Field().Value(m.value).Key("f").Autofocus().OnChange(func(e surfrun.Event) {
			m.commits = append(m.commits, e.Value)
			if !m.refuse {
				m.value = e.Value
			}
		}),
		surfui.Button("Other").Key("b").OnClick(func() {}),
		surfui.Text(strconv.Itoa(m.ticks)).Key("t"),
	)
}

func (m *committer) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func(surfrun.Event):
		f(ev)
	case func():
		f()
	}
}

const (
	commitField  = "r.kf"
	commitButton = "r.kb"
	commitCount  = "r.kt"
)

func startCommitter(t *testing.T, m *committer) *surftest.App {
	t.Helper()
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the field has the keyboard", func() bool { return a.Surface().Focused() == commitField })
	return a
}

// commits waits for the model to have heard want, and no more.
func commits(t *testing.T, a *surftest.App, m *committer, want ...string) {
	t.Helper()
	var got []string
	a.Wait(fmt.Sprintf("commits %q", want), func() bool {
		a.Call(func() { got = slices.Clone(m.commits) })
		return len(got) >= len(want)
	})
	// Anything more would come with the events already sent.
	a.Call(func() { got = slices.Clone(m.commits) })
	if !slices.Equal(got, want) {
		t.Fatalf("commits %q, want %q", got, want)
	}
}

func fill(t *testing.T, a *surftest.App, text string) {
	t.Helper()
	if err := a.Host.Fill(a.Name, commitField, text); err != nil {
		t.Fatal(err)
	}
}

// Enter commits what the field holds, and leaving it after commits
// nothing more: one OnChange for one value.
func TestFieldCommitsOnEnter(t *testing.T) {
	m := &committer{value: "80"}
	a := startCommitter(t, m)
	fill(t, a, "120")
	a.Host.Key("enter")
	commits(t, a, m, "120")
	a.Host.Key("tab")
	a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == commitButton })
	a.Call(func() { m.ticks++ }) // an update after the change has been heard
	a.Wait("the count", func() bool { return a.Surface().TextOf(commitCount) == "1" })
	commits(t, a, m, "120")
}

// Leaving the field commits it, however it is left: Tab, a click on
// another control, the surface losing the keyboard.
func TestFieldCommitsOnLeaving(t *testing.T) {
	for _, c := range []struct {
		name  string
		leave func(a *surftest.App) error
	}{
		{"tab", func(a *surftest.App) error { a.Host.Key("tab"); return nil }},
		{"a click elsewhere", func(a *surftest.App) error { return a.Host.Click(a.Name, commitButton) }},
		{"the surface losing the keyboard", func(a *surftest.App) error { return a.Host.Blur(a.Name) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := &committer{value: "80"}
			a := startCommitter(t, m)
			fill(t, a, "65")
			if err := c.leave(a); err != nil {
				t.Fatal(err)
			}
			commits(t, a, m, "65")
			if got, _ := a.Surface().Value(commitField); got != "65" {
				t.Errorf("the field shows %q", got)
			}
		})
	}
}

// Until it commits, the field keeps what the user types through the
// model's other changes, and leaving it unchanged commits nothing.
func TestFieldKeepsWhatIsTyped(t *testing.T) {
	m := &committer{value: "80"}
	a := startCommitter(t, m)
	fill(t, a, "9")
	a.Call(func() { m.ticks++ })
	a.Wait("the count", func() bool { return a.Surface().TextOf(commitCount) == "1" })
	if got, _ := a.Surface().Value(commitField); got != "9" {
		t.Fatalf("the field shows %q after an update, want what was typed", got)
	}
	fill(t, a, "80") // back to the value it had
	a.Host.Key("enter")
	a.Host.Key("tab")
	a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == commitButton })
	a.Call(func() { m.ticks++ })
	a.Wait("the count", func() bool { return a.Surface().TextOf(commitCount) == "2" })
	commits(t, a, m)
}

// A model that refuses what is committed has its own value put back in
// the field, and the same value committed again is heard again.
func TestFieldRefusalPutsTheValueBack(t *testing.T) {
	m := &committer{value: "80", refuse: true}
	a := startCommitter(t, m)
	fill(t, a, "30")
	a.Host.Key("enter")
	commits(t, a, m, "30")
	a.Wait("the value in force again", func() bool { v, _ := a.Surface().Value(commitField); return v == "80" })
	fill(t, a, "30")
	a.Host.Key("tab")
	commits(t, a, m, "30", "30")
}
