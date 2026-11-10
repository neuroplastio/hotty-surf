package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestCheckboxInput(t *testing.T) {
	on, disabled, clicks := false, false, 0
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(
			surfui.Checkbox(on).Key("check").Label("Enabled").Disabled(disabled).
				OnClick(func() { on = !on; clicks++ }),
			surfui.Text("Enabled"),
		).Gap(1))
	}}
	r := show(m)
	r.Draw(20, 3)
	keys(r, "tab", "space")
	if !on || clicks != 1 || !strings.Contains(r.Draw(20, 3).Text(), "[x]") {
		t.Fatalf("Space: on=%v clicks=%d\n%s", on, clicks, r.Draw(20, 3).Text())
	}
	keys(r, "enter")
	if on || clicks != 2 {
		t.Fatalf("Enter: on=%v clicks=%d", on, clicks)
	}
	press(t, r, "r.0.kcheck")
	if !on || clicks != 3 {
		t.Fatalf("click: on=%v clicks=%d", on, clicks)
	}
	disabled = true
	r.Show()
	r.Draw(20, 3)
	keys(r, "space", "enter", "tab")
	r.Press(1, 0, 0)
	r.Release(1, 0, 0)
	if clicks != 3 || r.Focused() == "r.0.kcheck" {
		t.Fatalf("disabled control acted or took focus: clicks=%d focus=%s", clicks, r.Focused())
	}
}

func TestCheckboxGoldens(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(
			surfui.Row(surfui.Checkbox(false), surfui.Text("Off")).Gap(1),
			surfui.Row(surfui.Checkbox(true), surfui.Text("On")).Gap(1),
			surfui.Row(surfui.Checkbox(true).Disabled(true), surfui.Text("Disabled")).Gap(1),
			surfui.Row(surfui.Checkbox(false).Invalid(true), surfui.Text("Invalid")).Gap(1),
		))
	}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	goldenText(t, "checkbox.html", surfui.HTML(s.Tree())+"\n")
	golden(t, "checkbox", show(m).Draw(20, 5))
}
