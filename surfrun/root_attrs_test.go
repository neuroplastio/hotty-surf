package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

type rootModel struct{}

func (*rootModel) Render() *surfui.Node {
	return surfui.Surface(surfui.List(surfui.Item(surfui.Text("Current")).Selected(true)))
}
func (*rootModel) Do(surfui.Action, surfrun.Event) {}

type fineRootModel struct {
	rootModel
	ops []hotty.Op
}

func (m *fineRootModel) Changes() ([]hotty.Op, bool) {
	ops := m.ops
	m.ops = nil
	return ops, false
}

func TestHostRootAttrs(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    surfrun.Model
	}{{"ordinary", &rootModel{}}, {"fine", &fineRootModel{}}} {
		t.Run(tc.name, func(t *testing.T) {
			s := surfrun.NewSurface(tc.m, nil)
			s.RootAttrs = map[string]string{"data-region": "focused", "data-window": "focused"}
			s.Show()
			if got, _ := surfui.AttrOf(s.Tree(), "data-region"); got != "focused" {
				t.Fatalf("initial region %q", got)
			}
			s.RootAttrs["data-region"] = "unfocused"
			delete(s.RootAttrs, "data-window")
			ops := s.Update().Ops
			if !slices.ContainsFunc(ops, func(o hotty.Op) bool {
				return o.Op == hotty.OpAttr("", "", "").Op && o.Target == "r" && o.Key == "data-region" && o.Value != nil && *o.Value == "unfocused"
			}) ||
				!slices.ContainsFunc(ops, func(o hotty.Op) bool {
					return o.Op == hotty.OpUnattr("", "").Op && o.Target == "r" && o.Key == "data-window"
				}) {
				t.Fatalf("root changes: %+v", ops)
			}
			if got, _ := surfui.AttrOf(s.Tree(), "data-region"); got != "unfocused" {
				t.Fatalf("updated region %q", got)
			}
			if _, ok := surfui.AttrOf(s.Tree(), "data-window"); ok {
				t.Fatal("window attribute remains")
			}
			if len(s.Update().Ops) != 0 {
				t.Fatal("unchanged root sent ops")
			}
		})
	}
}

func TestHostRootAttrsFollowFineRootMorph(t *testing.T) {
	m := &fineRootModel{}
	s := surfrun.NewSurface(m, nil)
	s.RootAttrs = map[string]string{"data-region": "focused"}
	s.Show()
	m.ops = []hotty.Op{hotty.OpMorph("r", "<div id=\"r\"></div>")}
	ops := s.Update().Ops
	if len(ops) != 2 || ops[1].Op != hotty.OpAttr("", "", "").Op || ops[1].Key != "data-region" {
		t.Fatalf("root attributes after morph: %+v", ops)
	}
}

func TestHostRootAttrsReachCells(t *testing.T) {
	s := surfrun.NewSurface(&fineRootModel{}, nil)
	s.RootAttrs = map[string]string{"data-region": "focused", "data-window": "focused"}
	r := cells.New(s.Cells())
	r.Show()
	find := func() cells.Role {
		for _, line := range r.Draw(20, 4).Lines {
			for _, run := range line {
				if strings.Contains(run.Text, "Current") {
					return run.Bg
				}
			}
		}
		return ""
	}
	if got := find(); got != cells.Active {
		t.Fatalf("focused fill %q", got)
	}
	s.RootAttrs["data-window"] = "unfocused"
	r.Update()
	if got := find(); got != cells.Selected {
		t.Fatalf("unfocused fill %q", got)
	}
}

func TestHostRootVars(t *testing.T) {
	for _, m := range []surfrun.Model{&rootModel{}, &fineRootModel{}} {
		s := surfrun.NewSurface(m, nil)
		s.RootVars = map[string]string{"u-r": "4px", "u-transition": "67ms"}
		s.Show()
		s.RootVars["u-r"] = "8px"
		delete(s.RootVars, "u-transition")
		ops := s.Update().Ops
		style, _ := surfui.StyleOf(s.Tree())
		if !strings.Contains(style, "--u-r:8px") || strings.Contains(style, "--u-transition") {
			t.Fatalf("style after host update: %q", style)
		}
		if !slices.ContainsFunc(ops, func(o hotty.Op) bool { return o.Key == "style" && o.Op == hotty.OpAttr("", "", "").Op }) {
			t.Fatalf("removal did not update style: %+v", ops)
		}
	}
}

func TestExternalKeyboardOnOrdinaryAndFineTrees(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model surfrun.Model
	}{
		{"ordinary", &focusRoot{}}, {"fine", &fineFocusRoot{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := surfrun.NewSurface(tc.model, nil)
			s.ExternalKeyboard = true
			if f := s.Show(); f.Focus != "" {
				t.Fatalf("show took focus: %+v", f)
			}
			row := surfui.Kids(s.Tree())[0]
			if _, ok := surfui.AttrOf(row, "tabindex"); ok || surfui.Focused(row) {
				t.Fatal("row still requests keyboard focus")
			}
			if _, ok := surfui.Handler(row, "click"); !ok {
				t.Fatal("click handler was removed")
			}
			if b := surfui.Kids(s.Tree())[1]; surfui.Tag(b) != "button" {
				t.Fatal("native control was rewritten")
			}
			s.Focus("row")
			if f := s.Update(); f.Focus != "" {
				t.Fatalf("program focus escaped policy: %+v", f)
			}
		})
	}
}

type focusRoot struct{}

func (*focusRoot) Render() *surfui.Node {
	return surfui.Surface(surfui.Item(surfui.Text("row")).Key("row").Focusable().Autofocus().OnClick("go"), surfui.Button("action"))
}
func (*focusRoot) Do(surfui.Action, surfrun.Event) {}

type fineFocusRoot struct{ focusRoot }

func (*fineFocusRoot) Changes() ([]hotty.Op, bool) { return nil, false }
