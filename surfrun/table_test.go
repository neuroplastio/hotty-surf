package surfrun_test

import (
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// grid is a table of three rows, the second selected, each with an
// @activate; the selection is the model's, and a click moves it.
type grid struct {
	sel int
	got []surfrun.Event
}

func (m *grid) Render() *surfui.Node {
	t := surfui.Table(surfui.Col("name"), surfui.Col("n").Align("end"))
	for i, name := range []string{"alpha", "beta", "gamma"} {
		t.Add(surfui.Item(surfui.Text(name), surfui.Text("1")).Key(name).Selected(i == m.sel).
			OnClick("pick").OnActivate("open"))
	}
	return surfui.Surface(t.Key("t"))
}

func (m *grid) Do(act surfui.Action, ev surfrun.Event) {
	m.got = append(m.got, ev)
	if act == "pick" {
		m.sel = map[string]int{"r.kt.2.kalpha": 0, "r.kt.2.kbeta": 1, "r.kt.2.kgamma": 2}[ev.Target]
	}
}

// Activating a row: Enter on the table runs the selected row's @activate
// with the row's key; a click on the selected row does, after its own
// click; a click on another row only selects it; a double click activates
// it whichever it is.
func TestActivateARow(t *testing.T) {
	m := &grid{sel: 1}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	kinds := func() (out []string) {
		for _, e := range m.got {
			out = append(out, e.Kind+" "+e.Value)
		}
		m.got = nil
		return out
	}
	same := func(got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("heard %q, want %q", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("heard %q, want %q", got, want)
			}
		}
	}
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.kt.2"}) // the body, which the keys' binding is on
	if took, _ := s.Key("enter"); !took {
		t.Fatal("Enter on a table with a selected row is not taken")
	}
	same(kinds(), "activate beta")
	s.Update()

	click := func(key string, count int) {
		s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kt.2.k" + key, Count: count})
		s.Update()
	}
	click("beta", 1) // the selected row
	same(kinds(), "click ", "activate beta")
	click("gamma", 1) // another: it selects
	same(kinds(), "click ")
	click("alpha", 2) // a double click
	same(kinds(), "click ", "activate alpha")
}
