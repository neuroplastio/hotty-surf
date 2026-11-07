package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// lender is a field the keys start in, a Select, and a button whose focus
// it counts; later an autofocus element of its own.
type lender struct {
	theme   string
	focuses int
	late    bool // a list drawn late, with autofocus
	field   bool // the field is drawn
}

func (m *lender) Render() *surfui.Node {
	return surfui.Surface(
		surfui.If(m.field, surfui.Field().Key("f").Autofocus()),
		surfui.Select(m.theme, themes...).Key("sel").OnChange(func(e surfrun.Event) { m.theme = e.Value }),
		surfui.Button("Go").Key("go").OnFocus(func(surfrun.Event) { m.focuses++ }),
		surfui.If(m.late, surfui.List(surfui.Item(surfui.Text("late"))).Key("late").Autofocus()),
	)
}

func (m *lender) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// The host takes the keyboard for keys of its own and gives it back: the
// blur that comes back keeps the focused element, and giving it back
// focuses it again on the host, with no focus event and no OnFocus.
func TestLendKeepsTheFocus(t *testing.T) {
	m := &lender{theme: "dracula", field: true}
	r := surftest.NewRelay(t, m, surfrun.Options{})
	s, h := r.Surface(), r.Host
	if err := h.Click(r.Name, "r.kgo"); err != nil {
		t.Fatal(err)
	}
	r.Settle()
	if s.Focused() != "r.kgo" || m.focuses != 1 {
		t.Fatalf("the click: focused %q, OnFocus %d", s.Focused(), m.focuses)
	}

	f := s.Lend(true)
	if !f.Blur || f.Focus != "" {
		t.Fatalf("Lend(true)'s frame: %+v", f)
	}
	r.Send(f)
	r.Settle()
	if got := h.Surface(r.Name).Focused(); got != "" {
		t.Errorf("the host's surface kept the keyboard, at %q", got)
	}
	if s.Focused() != "r.kgo" {
		t.Errorf("the blur that came back took the focus: %q", s.Focused())
	}
	// The model adds an autofocus element meanwhile: it does not take the
	// keyboard from the host.
	m.late = true
	if f := s.Update(); f.Focus != "" {
		t.Errorf("an autofocus element took the lent keyboard: %+v", f)
	}

	f = s.Lend(false)
	if f.Blur || f.Focus != "r.kgo" {
		t.Fatalf("Lend(false)'s frame: %+v", f)
	}
	r.Send(f)
	r.Settle()
	if got := h.Surface(r.Name).Focused(); got != "r.kgo" {
		t.Errorf("the host focused %q, want the button", got)
	}
	if s.Focused() != "r.kgo" || m.focuses != 1 {
		t.Errorf("given back: focused %q, OnFocus %d, want the button and 1", s.Focused(), m.focuses)
	}
}

// Lending closes the open list; given back with nothing focused, the
// keyboard goes to the keys' home, or else the first element that takes
// focus; and the user focusing an element meanwhile ends the loan.
func TestLendCases(t *testing.T) {
	m := &lender{theme: "dracula"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.ksel"})
	if l := s.Update().List; l == nil || !l.New {
		t.Fatalf("no list: %+v", l)
	}
	if f := s.Lend(true); f.List == nil || !f.List.Closed || !f.Blur {
		t.Errorf("Lend(true) with the list open: %+v", f)
	}
	s.Event(hotty.Event{Kind: hotty.EventBlur})
	if f := s.Lend(false); f.Focus != "r.ksel" {
		t.Errorf("given back after the list: %+v, want the Select", f)
	}

	// Nothing focused, no home: the first element that takes focus.
	m = &lender{theme: "dracula"}
	s = surfrun.NewSurface(m, nil)
	s.Show()
	s.Lend(true)
	if f := s.Lend(false); f.Focus != "r.ksel" || s.Focused() != "r.ksel" {
		t.Errorf("no home: %+v, focused %q, want the Select", f, s.Focused())
	}
	// With a home: the home.
	m = &lender{theme: "dracula", field: true}
	s = surfrun.NewSurface(m, nil)
	s.Show()
	s.Key("escape")
	s.Update()
	s.Lend(true)
	if f := s.Lend(false); f.Focus != "r.kf" {
		t.Errorf("with a home: %+v, want the field", f)
	}
	// The focused element gone meanwhile: the home it left.
	s.Lend(true)
	m.field, m.late = false, true
	if f := s.Update(); f.Focus != "" {
		t.Errorf("the new home took the lent keyboard: %+v", f)
	}
	if f := s.Lend(false); f.Focus != "r.klate" {
		t.Errorf("the field gone: %+v, want the new home", f)
	}

	// The user comes back by a click: the loan is over, and the next blur
	// takes the focus as any does.
	s.Lend(true)
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ksel"})
	s.Event(hotty.Event{Kind: hotty.EventBlur})
	if s.Focused() != "" {
		t.Errorf("a blur after the user came back kept %q", s.Focused())
	}

	// Hidden, there is nothing to lend.
	s.Hide()
	if f := s.Lend(true); f.Blur || len(f.Ops) != 0 {
		t.Errorf("Lend(true), hidden: %+v", f)
	}
}

// Theme draws the root in a theme of the host's, whatever the model's.
func TestSurfaceTheme(t *testing.T) {
	m := &lender{theme: "dracula"}
	s := surfrun.NewSurface(m, nil)
	s.Theme = "terminal"
	s.Show()
	if c := surfui.Classes(s.Tree()); !slices.Contains(c, "u-theme-terminal") || slices.Contains(c, "u-theme-graphite") {
		t.Errorf("the root's classes %q", c)
	}
	m.theme = "nord"
	if f := s.Update(); slices.ContainsFunc(f.Ops, func(o hotty.Op) bool { return o.Target == "r" }) {
		t.Errorf("an Update touched the root: %+v", f.Ops)
	}
}

// NoQuit leaves a binding to Quit as if nothing bound it: the key goes
// up to an outer binding of it, or is not taken, and the run goes on.
func TestSurfaceNoQuit(t *testing.T) {
	m, q := &binds{}, &quits{}
	s := surfrun.NewSurface(m, q)
	s.NoQuit = true
	s.Show()
	if took, redraw := s.Key("ctrl+q"); took || redraw || q.n != 0 {
		t.Errorf("ctrl+q with NoQuit: took %v, redraw %v, Quit %d times", took, redraw, q.n)
	}
	o := &outerQuit{}
	s = surfrun.NewSurface(o, q)
	s.NoQuit = true
	s.Show()
	if took, _ := s.Key("ctrl+q"); !took || q.n != 0 || strings.Join(o.heard, " ") != "outer" {
		t.Errorf("ctrl+q, bound outside Quit's: took %v, Quit %d, heard %q", took, q.n, o.heard)
	}
}

// outerQuit binds ctrl+q to Quit inside, and to an action of its own
// around it.
type outerQuit struct{ heard []string }

func (m *outerQuit) Render() *surfui.Node {
	return surfui.Surface(surfui.Box(surfui.Button("b").Key("b").Autofocus()).Key("in").Bind("ctrl+q", "Quit", surfrun.Quit{})).
		Bind("ctrl+q", "Close", func() { m.heard = append(m.heard, "outer") })
}

func (m *outerQuit) Do(act surfui.Action, ev surfrun.Event) { act.(func())() }
