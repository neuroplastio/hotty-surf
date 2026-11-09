package surfrun_test

import (
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// cycler is a setting drawn as a Cycle, which takes the keyboard at the
// start, and a key bound on the root.
type cycler struct {
	value   string
	changes []surfrun.Event
	other   int
}

func (m *cycler) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Cycle(m.value, "auto", "cells", "hotty").Key("render").Autofocus().
			OnChange(func(e surfrun.Event) { m.value = e.Value; m.changes = append(m.changes, e) }),
	).Bind("z", "Other", func(surfrun.Event) { m.other++ }).Bind("ctrl+x", "Other", func(surfrun.Event) { m.other++ })
}

func (m *cycler) Do(act surfui.Action, ev surfrun.Event) {
	if f, ok := act.(func(surfrun.Event)); ok {
		f(ev)
	}
}

const (
	cycleID = "r.krender"
	lessID  = "r.krender.0"
	valueID = "r.krender.1"
	moreID  = "r.krender.2"
)

func startCycler(t *testing.T) (*cycler, *surftest.App) {
	t.Helper()
	m := &cycler{value: "auto"}
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the Cycle has the keyboard", func() bool { return a.Surface().Focused() == cycleID })
	return m, a
}

// shows waits for the Cycle to show want, in the document and the model.
func shows(t *testing.T, a *surftest.App, m *cycler, want string) {
	t.Helper()
	a.Wait("the Cycle showing "+want, func() bool { return a.Surface().TextOf(valueID) == want })
	var got string
	a.Call(func() { got = m.value })
	if got != want {
		t.Fatalf("the model's value is %q, want %q", got, want)
	}
}

// The Cycle's keys step it, round either end, and run its OnChange as a
// Select's pick does.
func TestCycleKeys(t *testing.T) {
	m, a := startCycler(t)
	for _, step := range []struct{ key, want string }{
		{"arrow_right", "cells"},
		{"arrow_down", "hotty"},
		{"arrow_right", "auto"}, // round the end
		{"arrow_left", "hotty"}, // round the start
		{"arrow_up", "cells"},
		{"home", "auto"},
		{"end", "hotty"},
		{"space", "auto"},
		{"enter", "cells"},
		{"h", "hotty"},      // the next that starts with h
		{"shift+a", "auto"}, // an A, as a: case is ignored
		{"c", "cells"},
	} {
		a.Host.Key(step.key)
		shows(t, a, m, step.want)
	}
	var changes []surfrun.Event
	a.Call(func() { changes = m.changes })
	if len(changes) != 12 {
		t.Fatalf("%d changes, want 12", len(changes))
	}
	if e := changes[0]; e.Kind != "change" || e.Value != "cells" || e.Target != cycleID {
		t.Errorf("the first change is %+v, want a change to cells on %s", e, cycleID)
	}
}

// A character no option starts with, or only the value's own, is the
// Cycle's: it changes nothing and goes no further. A key it doesn't use
// goes up the focus chain.
func TestCycleKeepsItsKeys(t *testing.T) {
	m, a := startCycler(t)
	a.Host.Key("arrow_right")
	shows(t, a, m, "cells")
	a.Host.Key("c") // only cells starts with c
	a.Host.Key("z") // nothing does; the root binds z
	a.Host.Key("ctrl+x")
	a.Wait("ctrl+x reached the root", func() bool {
		var n int
		a.Call(func() { n = m.other })
		return n > 0
	})
	var other, changes int
	a.Call(func() { other, changes = m.other, len(m.changes) })
	if other != 1 {
		t.Errorf("the root heard %d keys, want 1 (ctrl+x, not z)", other)
	}
	if changes != 1 {
		t.Errorf("%d changes, want 1", changes)
	}
}

// Without the keyboard, a Cycle takes no keys.
func TestCycleUnfocused(t *testing.T) {
	m, a := startCycler(t)
	a.Host.Key("escape")
	a.Wait("nothing focused", func() bool { return a.Surface().Focused() == "" })
	a.Host.Key("arrow_right")
	a.Host.Key("ctrl+x")
	a.Wait("ctrl+x reached the root", func() bool {
		var n int
		a.Call(func() { n = m.other })
		return n > 0
	})
	shows(t, a, m, "auto")
}

// A click on the value or the › steps on, on the ‹ back, and takes the
// keyboard.
func TestCycleClicks(t *testing.T) {
	m, a := startCycler(t)
	a.Host.Key("escape")
	a.Wait("nothing focused", func() bool { return a.Surface().Focused() == "" })
	for _, step := range []struct{ id, want string }{
		{valueID, "cells"},
		{moreID, "hotty"},
		{moreID, "auto"}, // round the end
		{lessID, "hotty"},
		{lessID, "cells"},
	} {
		if err := a.Host.Click(a.Name, step.id); err != nil {
			t.Fatal(err)
		}
		shows(t, a, m, step.want)
	}
	a.Wait("the Cycle has the keyboard", func() bool { return a.Surface().Focused() == cycleID })
	a.Host.Key("arrow_right")
	shows(t, a, m, "hotty")
}

// A Cycle is as wide as its widest option, gives the program the keys
// a host would scroll with, Space among them, and is no control a=field
// sets.
func TestCycleHTML(t *testing.T) {
	c := surfui.Cycle("1", "0", "1", "center")
	surfui.Resolve(c, "c")
	want := `<div id="c" class="u-cycle" tabindex="0" role="spinbutton" aria-valuemin="1" aria-valuemax="3" ` +
		`aria-valuetext="1" aria-valuenow="2" style="--u-chars:6" ` +
		`data-keys="arrow_left=program arrow_right=program arrow_up=program arrow_down=program home=program end=program space=program">` +
		`<span id="c.0" class="u-cycle-step" aria-label="previous" data-on="click">` + "\U000F0141" + `</span>` +
		`<span id="c.1" class="u-cycle-value" data-on="click">1</span>` +
		`<span id="c.2" class="u-cycle-step" aria-label="next" data-on="click">` + "\U000F0142" + `</span></div>`
	if got := surfui.HTML(c); got != want {
		t.Errorf("Cycle:\n got %s\nwant %s", got, want)
	}
	if _, ok := surfui.ValueOf(c); ok {
		t.Error("a Cycle is a control a=field sets")
	}
}
