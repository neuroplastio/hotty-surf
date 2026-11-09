package main

import (
	"testing"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
)

// The second screen, in each mode: keys on the list and on the field, a
// row's switch, the filter, the stateful child and a slot's button.
func TestTasks(t *testing.T) {
	for _, m := range []struct {
		name string
		view func(*Tasks) surflive.Component
	}{{"interpreted", surfInterpTasks}, {"compiled", surfCompiledTasks}} {
		t.Run(m.name, func(t *testing.T) { testTasks(t, m.view) })
	}
}

func testTasks(t *testing.T, view func(*Tasks) surflive.Component) {
	s := NewTasks("Write the guide", "Write the spec", "Run make check")
	a := surftest.Start(t, surflive.New(view(s)), surfrun.Options{Name: "tasks"})
	a.Wait("the list has the keyboard", func() bool {
		return a.Surface().Focused() == "r.0.klist"
	})

	// The list's keys: the arrow moves the selection, enter marks it done.
	a.Host.Key("arrow_down")
	a.Wait("the second task selected", func() bool {
		return a.Has("r.0.klist.kt2", "is-sel")
	})
	a.Host.Key("enter")
	a.Wait("the second task done", func() bool {
		v, _ := a.Surface().Attr("r.0.klist.kt2.0", "aria-checked")
		return v == "true"
	})

	// A click on a row's switch is the switch's handler, not the row's.
	before := len(a.Ops())
	if err := a.Host.Click(a.Name, "r.0.klist.kt1.0"); err != nil {
		t.Fatal(err)
	}
	a.Wait("one left", func() bool {
		return a.Surface().TextOf("r.0.khead.kleft") == "1 left"
	})
	if !a.Has("r.0.klist.kt2", "is-sel") {
		t.Error("the switch's click selected its row")
	}
	for _, o := range a.Ops()[before:] {
		if o.Op != "attr" && o.Op != "text" {
			t.Errorf("a switch's click sent %s %s, want attr and text ops only",
				o.Op, o.Target)
		}
	}

	// The field's enter adds what it holds, and the field is emptied.
	if err := a.Host.Fill(a.Name, "r.0.kdraft", "Ship it"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the draft", func() bool {
		var d string
		a.Call(func() { d = s.Draft })
		return d == "Ship it"
	})
	a.Host.Key("enter")
	a.Wait("the new task, selected", func() bool { return a.Has("r.0.klist.kt4", "is-sel") })
	a.Wait("the field emptied", func() bool { v, _ := a.Surface().Value("r.0.kdraft"); return v == "" })

	// The filter: a click opens the select's list, a click on an option
	// picks it.
	a.Pick("r.0.khead.kfilter", "open")
	a.Wait("the done tasks hidden", func() bool {
		_, t1 := a.Surface().Element("r.0.klist.kt1")
		_, t3 := a.Surface().Element("r.0.klist.kt3")
		return !t1 && t3
	})

	// The stateful child flips its own state.
	if err := a.Host.Click(a.Name, "r.0.khelp.1.0"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the keys shown", func() bool { return !a.Has("r.0.khelp.1.kkeys", "u-tone-muted") })
	var open bool
	a.Call(func() { open = s.Help.Open })
	if !open {
		t.Error("the panel's switch didn't open it")
	}

	// A slot's button calls its user's handler.
	a.Pick("r.0.khead.kfilter", "all")
	a.Wait("the done tasks shown", func() bool { _, t1 := a.Surface().Element("r.0.klist.kt1"); return t1 })
	if err := a.Host.Click(a.Name, "r.0.khelp.0.2"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the done tasks gone", func() bool {
		var n int
		a.Call(func() { n = len(s.Items) })
		return n == 2
	})

	// Escape, which nothing nearer binds, is the surface's: it ends the run.
	a.Host.Key("escape")
	a.Wait("the surface gone", func() bool { return a.Surface() == nil })
}
