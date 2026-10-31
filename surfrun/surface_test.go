package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// These drive a surfrun.Surface as a host that is no surfrun.Runtime does, with
// no terminal and no host: the test hands it the host's events and the
// keys, and reads the frames it gives back.

// opOn is whether ops hold one of kind on target.
func opOn(ops []hotty.Op, kind, target string) bool {
	return slices.ContainsFunc(ops, func(o hotty.Op) bool { return o.Op == kind && o.Target == target })
}

// A field's input is kept, and Enter commits it: the key is taken, a
// redraw is due, and the model hears the value once, leaving it after
// too.
func TestSurfaceFieldCommitsOnEnter(t *testing.T) {
	m := &committer{value: "80"}
	s := surfrun.NewSurface(m, nil)
	if f := s.Show(); f.Focus != commitField || s.Focused() != commitField {
		t.Fatalf("the field is not the keys' home: frame %+v, focused %q", f, s.Focused())
	}
	if took, redraw := s.Event(hotty.Event{Kind: hotty.EventInput, Target: commitField, Value: "120", Rev: 3}); !took || !redraw {
		t.Errorf("the field's input: took %v, redraw %v", took, redraw)
	}
	s.Update()
	if len(m.commits) != 0 {
		t.Fatalf("an input committed %q", m.commits)
	}
	if took, redraw := s.Key("enter"); !took || !redraw {
		t.Errorf("enter: took %v, redraw %v", took, redraw)
	}
	if !slices.Equal(m.commits, []string{"120"}) {
		t.Fatalf("commits %q, want 120", m.commits)
	}
	if f := s.Update(); len(f.Fields) != 0 {
		t.Errorf("the model took the value, and a=field puts it back: %+v", f.Fields)
	}
	s.Event(hotty.Event{Kind: hotty.EventChange, Target: commitField, Value: "120", Rev: 3})
	s.Update()
	if !slices.Equal(m.commits, []string{"120"}) {
		t.Errorf("leaving after Enter committed again: %q", m.commits)
	}

	// A refusal: the model keeps its value, and the frame puts it back.
	m.refuse = true
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: commitField, Value: "30", Rev: 5})
	s.Key("enter")
	f := s.Update()
	if len(f.Fields) != 1 || f.Fields[0].ID != commitField || *f.Fields[0].Change.Value != "120" || f.Fields[0].Change.Rev != 5 {
		t.Errorf("a refused commit's a=field: %+v", f.Fields)
	}
	// The field holds the model's value again: Enter commits nothing
	// until the user types.
	s.Key("enter")
	if !slices.Equal(m.commits, []string{"120", "30"}) {
		t.Errorf("Enter after the a=field committed again: %q", m.commits)
	}
}

// unbound is a field the model draws no value in, whose edits and
// commits it hears.
type unbound struct{ inputs, commits []string }

func (m *unbound) Render() *surfui.Node {
	return surfui.Surface(surfui.Field().Key("f").Autofocus().
		OnInput(func(e surfrun.Event) { m.inputs = append(m.inputs, e.Value) }).
		OnChange(func(e surfrun.Event) { m.commits = append(m.commits, e.Value) }))
}

func (m *unbound) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// A field the model draws no value in is the user's: what they type is
// not put back, and Enter commits it.
func TestSurfaceUnboundField(t *testing.T) {
	m := &unbound{}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: "r.kf", Value: "abc", Rev: 1})
	if f := s.Update(); len(f.Fields) != 0 {
		t.Errorf("the input put back: %+v", f.Fields)
	}
	s.Key("enter")
	if !slices.Equal(m.inputs, []string{"abc"}) || !slices.Equal(m.commits, []string{"abc"}) {
		t.Errorf("inputs %q, commits %q, want abc each", m.inputs, m.commits)
	}
}

// A Cycle's keys step it, a click on its parts too, and each change is a
// frame's ops.
func TestSurfaceCycle(t *testing.T) {
	m := &cycler{value: "auto"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	if took, _ := s.Key("arrow_right"); !took || m.value != "cells" {
		t.Fatalf("→: took %v, value %q", took, m.value)
	}
	if f := s.Update(); !opOn(f.Ops, "text", valueID) {
		t.Errorf("the step's frame: %+v", f.Ops)
	}
	if took, redraw := s.Event(hotty.Event{Kind: hotty.EventClick, Target: lessID}); !took || !redraw || m.value != "auto" {
		t.Errorf("a click on ‹: took %v, redraw %v, value %q", took, redraw, m.value)
	}
	s.Update()
	// z, which no option starts with, the Cycle keeps; ctrl+x goes up to
	// the root's binding.
	if took, _ := s.Key("z"); !took || m.other != 0 || m.value != "auto" {
		t.Errorf("z: took %v, the root heard %d, value %q", took, m.other, m.value)
	}
	if took, _ := s.Key("ctrl+x"); !took || m.other != 1 {
		t.Errorf("ctrl+x: took %v, the root heard %d", took, m.other)
	}
}

// A click on a Select opens its list: the frame hands it to the host as
// its tree and the Select's cells, the click's area. The keys move in it;
// a press on an option takes the keyboard, whose blur on the surface is
// no leaving, and its click picks: the list closes, and the Select has
// the keyboard again.
func TestSurfaceSelectList(t *testing.T) {
	m := &chooser{value: "neuroplastio"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	area := &hotty.Area{Col: 10, Row: 3, W: 30, H: 2}
	if took, redraw := s.Event(hotty.Event{Kind: hotty.EventClick, Target: selectID, Area: area}); !took || !redraw {
		t.Fatalf("the click: took %v, redraw %v", took, redraw)
	}
	f := s.Update()
	l := f.List
	if l == nil || !l.New || l.Closed || l.Tree == nil {
		t.Fatalf("no new list in the frame: %+v", l)
	}
	if l.Anchor == nil || *l.Anchor != *area {
		t.Errorf("the list's anchor %+v, want the click's %+v", l.Anchor, area)
	}
	if l.Rows != len(themes)+1 || l.Cols != len("neuroplastio")+6 || l.At != 3 {
		t.Errorf("the list wants %d×%d, at %d: want %d×%d at 3", l.Cols, l.Rows, l.At, len("neuroplastio")+6, len(themes)+1)
	}
	want := surfui.SelectList(s.Tree(), "neuroplastio", themes, 3)
	surfui.Resolve(want, surfui.ListID)
	if got := surfui.HTML(l.Tree); got != surfui.HTML(want) {
		t.Errorf("the list's tree:\n got %s\nwant %s", got, surfui.HTML(want))
	}
	if !slices.ContainsFunc(f.Ops, func(o hotty.Op) bool {
		return o.Op == "attr" && o.Target == selectID && o.Key == "aria-expanded" && *o.Value == "true"
	}) {
		t.Errorf("the Select is not drawn open: %+v", f.Ops)
	}

	if took, _ := s.Key("arrow_down"); !took {
		t.Error("the open list did not take ↓")
	}
	if l := s.Update().List; l == nil || l.New || l.At != 4 || len(l.Ops) == 0 {
		t.Errorf("↓'s list: %+v", l)
	}
	if took, _ := s.ListEvent(hotty.Event{Kind: hotty.EventPress, Target: surfui.OptionID(4)}); !took {
		t.Error("the press on an option was not taken")
	}
	s.Event(hotty.Event{Kind: hotty.EventBlur})
	if took, _ := s.ListEvent(hotty.Event{Kind: hotty.EventClick, Target: surfui.OptionID(4)}); !took {
		t.Error("the click on an option was not taken")
	}
	if !slices.Equal(m.changes, []string{"nord"}) {
		t.Errorf("changes %q, want nord", m.changes)
	}
	f = s.Update()
	if f.List == nil || !f.List.Closed || f.List.Tree != nil {
		t.Errorf("the list did not close: %+v", f.List)
	}
	if f.Focus != selectID || s.Focused() != selectID {
		t.Errorf("the Select has not the keyboard: frame's %q, focused %q", f.Focus, s.Focused())
	}

	// A key opens it too, with no anchor; DropList takes it away at once.
	s.Key("space")
	if l := s.Update().List; l == nil || !l.New || l.Anchor != nil {
		t.Fatalf("space's list: %+v", l)
	}
	if !s.DropList() {
		t.Error("DropList: the list's surface was up")
	}
	if f := s.Update(); f.List != nil || !opOn(f.Ops, "attr", selectID) {
		t.Errorf("after DropList: list %+v, ops %+v", f.List, f.Ops)
	}
}

// binds is a field in a list in a box, each binding keys: heard counts
// what each binding heard. A key bound to surfrun.Quit ends the run.
type binds struct{ heard []string }

func (m *binds) Render() *surfui.Node {
	hear := func(what string) func() { return func() { m.heard = append(m.heard, what) } }
	field := surfui.Field().Key("f").Autofocus().Bind("ctrl+s", "Save", hear("field ctrl+s"))
	list := surfui.List(surfui.Item(field)).Key("l").Bind("arrow_down", "Next", hear("list ↓"))
	return surfui.Surface(surfui.Box(list).Key("b")).Bind("z", "Zoom", hear("root z")).
		Bind("mod+k", "Find", hear("root mod+k")).Bind("ctrl+q", "Quit", surfrun.Quit{})
}

func (m *binds) Do(act surfui.Action, ev surfrun.Event) { act.(func())() }

// quits counts Quit, and posts nothing.
type quits struct{ n int }

func (q *quits) Post(func())  {}
func (q *quits) Quit()        { q.n++ }
func (q *quits) Focus(string) {}

// A key goes up the focus chain from the focused element to the first
// binding of it; a key nothing binds is not taken, and nothing is due; a
// binding to Quit ends the run; an Escape nothing binds blurs.
func TestSurfaceKeyClimbs(t *testing.T) {
	m, q := &binds{}, &quits{}
	s := surfrun.NewSurface(m, q)
	s.Show()
	const field = "r.kb.kl.0.kf"
	if s.Focused() != field {
		t.Fatalf("focused %q, want the field", s.Focused())
	}
	for _, k := range []string{"ctrl+s", "arrow_down", "z", "ctrl+k"} {
		if took, redraw := s.Key(k); !took || !redraw {
			t.Errorf("%s: took %v, redraw %v", k, took, redraw)
		}
		s.Update()
	}
	if want := []string{"field ctrl+s", "list ↓", "root z", "root mod+k"}; !slices.Equal(m.heard, want) {
		t.Errorf("heard %q, want %q", m.heard, want)
	}
	if took, redraw := s.Key("x"); took || redraw {
		t.Errorf("x, which nothing binds: took %v, redraw %v", took, redraw)
	}
	s.Platform = hotty.Platform{Mod: "super"}
	if took, _ := s.Key("ctrl+k"); took {
		t.Error("mod+k took ctrl+k where mod is super")
	}
	if took, _ := s.Key("ctrl+q"); !took || q.n != 1 {
		t.Errorf("ctrl+q: took %v, Quit %d times", took, q.n)
	}
	if took, _ := s.Key("escape"); !took {
		t.Error("escape did not blur")
	}
	if f := s.Update(); !f.Blur || f.Focus != "" || s.Focused() != "" {
		t.Errorf("escape's frame: %+v, focused %q", f, s.Focused())
	}
	// With nothing focused, keys start at the keys' home, the field.
	if took, _ := s.Key("arrow_down"); !took || m.heard[len(m.heard)-1] != "list ↓" {
		t.Errorf("↓ from the home: took %v, heard %q", took, m.heard)
	}
}

// Hidden, as in the cells, a surface takes its bindings only, from the
// keys' home, and its frames are empty; Show sends the document again.
func TestSurfaceHidden(t *testing.T) {
	m := &cycler{value: "auto"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Hide()
	if took, _ := s.Key("arrow_right"); took || m.value != "auto" {
		t.Errorf("a hidden Cycle took →: value %q", m.value)
	}
	if took, _ := s.Key("z"); !took || m.other != 1 {
		t.Errorf("the root's z, hidden: took %v, heard %d", took, m.other)
	}
	if f := s.Update(); len(f.Ops) != 0 || f.Focus != "" || f.List != nil {
		t.Errorf("a hidden surface's frame: %+v", f)
	}
	if f := s.Show(); f.Focus != cycleID || !strings.HasPrefix(surfui.HTML(s.Tree()), `<div id="r"`) {
		t.Errorf("Show again: %+v", f)
	}
}
