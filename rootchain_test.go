package surf

import (
	"strings"
	"testing"
	"testing/fstest"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
)

// rootChainFS draws a board whose partials' roots are :if chains: a
// field that is the same element in either arm (only its attributes
// differ), a tab that is a button in one arm and a chip in the other,
// and a sibling field that is no part of any chain.
var rootChainFS = fstest.MapFS{
	"board.surf": {Data: []byte(`
<surface>
  <field key="side" placeholder="side"/>
  <name-field key="name" :on="sel == 'n1'"/>
  <name-tab key="tab" :on="sel == 'n1'"/>
</surface>`)},
	"name-field.surf": {Data: []byte(`
<props on="bool"/>
<field :if="on" autofocus bold placeholder="shouting"/>
<field :else autofocus placeholder="plain"/>`)},
	"name-tab.surf": {Data: []byte(`
<props on="bool"/>
<chip :if="on" fill="accent">selected</chip>
<button :else-if="!on" @click="open('n1')">plain</button>`)},
}

// A partial whose root is a :if chain draws the arm that holds, and
// when the condition flips, in both directions, the element at the
// partial's key is morphed in place: the same id, its attributes (or
// its kind) changed. Nothing around it is replaced, so the keyboard
// stays where it was, and what was typed into a field of the partial
// that keeps its kind stays too.
func TestRootChainFlip(t *testing.T) {
	set, err := Load(rootChainFS, []*Shape{boardShape, noteShape})
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard()
	b.Sel = "n2"
	a := surftest.Start(t, surflive.New(Component(set, boardObj{b})), surfrun.Options{})
	val := func(id string) string { v, _ := a.Surface().Value(id); return v }
	has := func(id, c string) bool { return a.Has(id, c) }

	a.Wait("the first draw", func() bool { return a.Surface().Placed() && has("r.kname", "u-field") })
	if s := a.Surface().HTML(); !strings.Contains(s, "plain") || strings.Contains(s, "shouting") {
		t.Fatalf("not the :else arms at the start:\n%s", s)
	}
	if !has("r.ktab.0", "u-btn") && !has("r.ktab", "u-btn") {
		t.Fatalf("the tab is not a button at the start:\n%s", a.Surface().HTML())
	}
	a.Wait("the field has the keyboard", func() bool { return a.Surface().Focused() == "r.kname" })
	if err := a.Host.Fill(a.Name, "r.kname", "abc"); err != nil {
		t.Fatal(err)
	}
	a.Wait("typed", func() bool { return val("r.kname") == "abc" })

	for round := 0; round < 2; round++ {
		// Flip: the first arms hold.
		before := len(a.Ops())
		a.Call(func() { b.Sel = "n1" })
		a.Wait("the first arms", func() bool { return strings.Contains(a.Surface().HTML(), "shouting") })
		html := a.Surface().HTML()
		if !strings.Contains(html, "selected") || strings.Contains(html, ">plain<") {
			t.Fatalf("round %d: the tab did not flip to selected:\n%s", round, html)
		}
		for _, op := range a.Ops()[before:] {
			if op.Op == "remove" && (op.Target == "r.kname" || op.Target == "r.ktab") {
				t.Errorf("round %d: %s was removed, not morphed: %v", round, op.Target, op)
			}
		}
		if f := a.Surface().Focused(); f != "r.kname" {
			t.Errorf("round %d: the keyboard moved to %q", round, f)
		}
		if v := val("r.kname"); v != "abc" {
			t.Errorf("round %d: what was typed is %q after the flip", round, v)
		}

		// And back.
		a.Call(func() { b.Sel = "n2" })
		a.Wait("the second arms", func() bool { return !strings.Contains(a.Surface().HTML(), "shouting") })
		html = a.Surface().HTML()
		if strings.Contains(html, "selected") || !strings.Contains(html, ">plain<") {
			t.Fatalf("round %d: the tab did not flip back to plain:\n%s", round, html)
		}
		if f := a.Surface().Focused(); f != "r.kname" {
			t.Errorf("round %d: back, the keyboard moved to %q", round, f)
		}
		if v := val("r.kname"); v != "abc" {
			t.Errorf("round %d: back, what was typed is %q", round, v)
		}
	}
}
