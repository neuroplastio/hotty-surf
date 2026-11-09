package surftest_test

import (
	"strconv"
	"testing"

	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// rows is a list of two rows and a count, with a selection the arrows
// move.
type rows struct {
	sel   int
	count int
}

func (m *rows) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Text(string(rune('0'+m.count))).Key("count"),
		surfui.List(
			surfui.Item(surfui.Text("alpha")).Key("alpha").Selected(m.sel == 0),
			surfui.Item(surfui.Text("beta")).Key("beta").Selected(m.sel == 1),
		).Key("list").Focusable().Autofocus().Bind("arrow_down", "Next", "next"),
	)
}

func (m *rows) Do(act surfui.Action, ev surfrun.Event) {
	if act == "next" {
		m.sel = (m.sel + 1) % 2
	}
}

func TestApp(t *testing.T) {
	m := &rows{}
	a := surftest.Start(t, m, surfrun.Options{})
	if !a.Has("r.klist.kalpha", "is-sel") {
		t.Fatalf("alpha not selected at the start:\n%s", a.Surface().HTML())
	}
	a.Wait("the list has the keyboard", func() bool { return a.Surface().Focused() == "r.klist" })
	a.Host.Key("arrow_down")
	a.Wait("beta selected", func() bool { return a.Has("r.klist.kbeta", "is-sel") })
	var sel int
	a.Call(func() { sel = m.sel })
	if sel != 1 {
		t.Errorf("the model's selection is %d, want 1", sel)
	}

	before := len(a.Ops())
	a.Call(func() { m.count = 7 })
	a.Wait("the count shows 7", func() bool { return a.Surface().TextOf("r.kcount") == "7" })
	ops := a.Ops()[before:]
	if len(ops) != 1 || ops[0].Op != "text" {
		t.Errorf("the count's change went as %v, want one text op", ops)
	}
	if u := a.Updates(); len(u) == 0 || u[len(u)-1].Ops != 1 {
		t.Errorf("the last update's ops: %v", u)
	}
}

// sized draws the size the runtime told it (surfrun.Sizer).
type sized struct{ cols, rows int }

func (m *sized) Render() *surfui.Node {
	return surfui.Surface(surfui.Text(strconv.Itoa(m.cols) + "x" + strconv.Itoa(m.rows)).Key("size"))
}
func (m *sized) Do(surfui.Action, surfrun.Event) {}
func (m *sized) Resize(cols, rows int)           { m.cols, m.rows = cols, rows }

// Resize reaches the model as a window resized does, and the surface is
// drawn at the new size.
func TestResize(t *testing.T) {
	a := surftest.Start(t, &sized{}, surfrun.Options{}, hottytest.Size(90, 30))
	a.Wait("the first size", func() bool { return a.Surface().TextOf("r.ksize") == "90x30" })
	a.Resize(150, 40)
	a.Wait("the new size", func() bool { return a.Surface().TextOf("r.ksize") == "150x40" })
}
