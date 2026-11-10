package cells_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestTabsInput(t *testing.T) {
	value, disabled := "Source", false
	var picked []string
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(
			surfui.Tabs(value, "Source", "Result", "Log").Key("tabs").Disabled(disabled).
				OnChange(func(e surfrun.Event) { value = e.Value; picked = append(picked, value) }),
			surfui.Button("Next").Key("next"),
		)
	}}
	r := show(m)
	r.Draw(40, 5)
	keys(r, "tab")
	if r.Focused() != "r.ktabs" {
		t.Fatalf("tabs not the single Tab stop: %s", r.Focused())
	}
	keys(r, "arrow_right", "end", "arrow_right", "arrow_left", "home")
	if !slices.Equal(picked, []string{"Result", "Log", "Source", "Log", "Source"}) {
		t.Fatalf("keys picked %q", picked)
	}
	press(t, r, "r.ktabs.k1")
	if value != "Result" || r.Focused() != "r.ktabs" {
		t.Fatalf("click: value %s focus %s", value, r.Focused())
	}
	keys(r, "tab")
	if r.Focused() != "r.knext" {
		t.Fatalf("Tab stopped inside the options: %s", r.Focused())
	}
	before := len(picked)
	disabled = true
	r.Show()
	r.Draw(40, 5)
	press(t, r, "r.ktabs.k0")
	keys(r, "arrow_right", "home")
	if len(picked) != before {
		t.Fatal("disabled tabs changed")
	}
}

func TestTabsGoldens(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Tabs("Result", "Source", "Result", "Log").Label("View"))
	}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	goldenText(t, "tabs.html", surfui.HTML(s.Tree())+"\n")
	r := show(m)
	golden(t, "tabs", r.Draw(32, 3))
	if got := r.Draw(12, 3).Text(); !strings.Contains(got, "Source") {
		t.Fatalf("narrow tabs lost their start: %q", got)
	}
}
