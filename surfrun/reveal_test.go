package surfrun_test

import (
	"strconv"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// rows is a list of 40 rows in a scroll box, the arrows moving its
// selection; keys counts them.
type rows struct{ sel, keys int }

func (m *rows) Render() *surfui.Node {
	l := surfui.List().Focusable().Autofocus().Key("list").
		Bind("arrow_down", "Next", 1).Bind("arrow_up", "Previous", -1)
	for i := range 40 {
		l.Add(surfui.Item(surfui.Text("row " + strconv.Itoa(i))).Key(strconv.Itoa(i)).Selected(i == m.sel))
	}
	return surfui.Surface(surfui.Stack(l).Scroll().Key("box"))
}

func (m *rows) Do(act surfui.Action, ev surfrun.Event) {
	m.sel = min(max(m.sel+act.(int), 0), 39)
	m.keys++
}

const rowsList = "r.kbox.klist"

// reveals are the targets of the reveal ops in ops.
func reveals(ops []hotty.Op) []string {
	var out []string
	for _, o := range ops {
		if o.Op == "reveal" {
			out = append(out, o.Target)
		}
	}
	return out
}

// In a run whose document scrolls, the runtime reveals the row the
// selection moves to, after the ops that select it, and the selected row
// of a new document; where it doesn't scroll, it sends no reveal.
func TestSelectionKeptInView(t *testing.T) {
	m := &rows{sel: 30}
	a := surftest.Start(t, m, surfrun.Options{Scroll: hotty.ScrollVertical})
	a.Wait("the list has the keyboard", func() bool { return a.Surface().Focused() == rowsList })
	a.Call(func() {}) // the document's update, written
	if got := reveals(a.Ops()); len(got) != 1 || got[0] != rowsList+".k30" {
		t.Errorf("the document's reveals are %v, want the selected row 30", got)
	}

	before := len(a.Ops())
	a.Host.Key("arrow_down")
	a.Wait("row 31 selected", func() bool { return a.Has(rowsList+".k31", "is-sel") })
	a.Call(func() {})
	ops := a.Ops()[before:]
	if len(ops) == 0 || ops[len(ops)-1].Op != "reveal" || ops[len(ops)-1].Target != rowsList+".k31" || ops[len(ops)-1].Value != nil {
		t.Errorf("↓ sent %+v, want its ops then a reveal of row 31, by as little as it can", ops)
	}
	if got := reveals(ops); len(got) != 1 {
		t.Errorf("↓ revealed %v, want row 31 only", got)
	}

	// A key that moves nothing reveals nothing.
	a.Call(func() { m.sel = 39 })
	a.Wait("row 39 selected", func() bool { return a.Has(rowsList+".k39", "is-sel") })
	before = len(a.Ops())
	a.Host.Key("arrow_down")
	a.Wait("the key heard", func() bool { var n int; a.Call(func() { n = m.keys }); return n == 2 })
	a.Call(func() {}) // its update written
	if got := a.Ops()[before:]; len(got) != 0 {
		t.Errorf("↓ at the end sent %v", got)
	}

	// Where the document doesn't scroll, nothing is revealed.
	still := &rows{sel: 30}
	b := surftest.Start(t, still, surfrun.Options{})
	b.Wait("the list has the keyboard", func() bool { return b.Surface().Focused() == rowsList })
	b.Host.Key("arrow_up")
	b.Wait("row 29 selected", func() bool { return b.Has(rowsList+".k29", "is-sel") })
	b.Call(func() {})
	if got := reveals(b.Ops()); len(got) != 0 {
		t.Errorf("a run that doesn't scroll revealed %v", got)
	}
}
