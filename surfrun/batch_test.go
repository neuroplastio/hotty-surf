package surfrun_test

import (
	"reflect"
	"testing"

	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// form is a field that commits and empties, a Select, a Cycle, a button
// and a binding: what a host's acts reach between two frames.
type form struct {
	Draft, Theme, Mode string
	Sent               []string
	Clicks, Saves      int
}

func (m *form) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Field().Value(m.Draft).Key("draft").Autofocus().
			OnChange(func(e surfrun.Event) { m.Sent = append(m.Sent, e.Value); m.Draft = "" }),
		surfui.Select(m.Theme, themes...).Key("theme").OnChange(func(e surfrun.Event) { m.Theme = e.Value }),
		surfui.Cycle(m.Mode, "auto", "cells", "hotty").Key("render").OnChange(func(e surfrun.Event) { m.Mode = e.Value }),
		surfui.Button("Count").Key("count").OnClick(func(surfrun.Event) { m.Clicks++ }),
	).Bind("ctrl+s", "Save", func(surfrun.Event) { m.Saves++ })
}

func (m *form) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// Many events and keys, then one Update, leave the model and the host as
// an Update after each does (plx draws in frames): a field typed in and
// committed, which the model empties; two steps of a Select and of a
// Cycle, each from where the last left it; two clicks; a binding; an
// Escape that blurs.
func TestManyEventsThenOneUpdate(t *testing.T) {
	acts := []func(t *testing.T, h *hottytest.Host, name string){
		func(t *testing.T, h *hottytest.Host, name string) {
			if err := h.Fill(name, "r.kdraft", "hello"); err != nil {
				t.Fatal(err)
			}
		},
		key("enter"), key("tab"), key("g"), key("n"),
		key("tab"), key("arrow_right"), key("arrow_right"),
		func(t *testing.T, h *hottytest.Host, name string) {
			for range 2 {
				if err := h.Click(name, "r.kcount"); err != nil {
					t.Fatal(err)
				}
			}
		},
		key("ctrl+s"), key("escape"),
	}
	start := func() (*form, *surftest.Relay) {
		m := &form{Theme: "dracula", Mode: "auto"}
		return m, surftest.NewRelay(t, m, surfrun.Options{})
	}
	em, each := start()
	bm, batch := start()
	for _, act := range acts {
		act(t, each.Host, each.Name)
		each.Input(true)
		act(t, batch.Host, batch.Name)
		batch.Input(false)
	}
	each.Settle()
	batch.Update()
	batch.Settle()

	want := form{Theme: "neuroplastio", Mode: "hotty", Sent: []string{"hello"}, Clicks: 2, Saves: 1}
	if !reflect.DeepEqual(*em, want) {
		t.Fatalf("an Update after each: %+v, want %+v", *em, want)
	}
	if !reflect.DeepEqual(*bm, *em) {
		t.Errorf("one Update: %+v, an Update after each: %+v", *bm, *em)
	}
	es, bs := each.Host.Surface(each.Name), batch.Host.Surface(batch.Name)
	if b, e := bs.HTML(), es.HTML(); b != e {
		t.Errorf("the host's document, one Update:\n%s\nan Update after each:\n%s", b, e)
	}
	bv, _ := bs.Value("r.kdraft")
	ev, _ := es.Value("r.kdraft")
	if bv != ev || bv != "" || bs.Focused() != es.Focused() {
		t.Errorf("the field %q, focused %q: an Update after each's %q, %q", bv, bs.Focused(), ev, es.Focused())
	}
}

// key is an act that presses name on the host.
func key(name string) func(*testing.T, *hottytest.Host, string) {
	return func(_ *testing.T, h *hottytest.Host, _ string) { h.Key(name) }
}
