package surfrun_test

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

type tabModel struct {
	value    string
	options  []string
	disabled bool
}

func (m *tabModel) Render() *surfui.Node {
	return surfui.Surface(surfui.Tabs(m.value, m.options...).Key("tabs").Disabled(m.disabled).OnChange("pick"))
}

func (m *tabModel) Do(_ surfui.Action, e surfrun.Event) { m.value = e.Value }

func TestTabsKeysAgree(t *testing.T) {
	for _, opts := range [][]string{nil, {"one"}, {"one", "two", "three"}} {
		for _, disabled := range []bool{false, true} {
			m := &tabModel{value: "one", options: opts, disabled: disabled}
			s := surfrun.NewSurface(m, nil)
			s.Show()
			s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ktabs"})
			uses, _ := s.Keys()
			for _, key := range []string{"arrow_left", "arrow_right", "home", "end", "a", "space"} {
				advertised := slices.ContainsFunc(uses, func(k surfrun.KeyUse) bool { return k.Name == key })
				if took, _ := s.Key(key); took != advertised {
					t.Fatalf("%v disabled=%v key=%s: advertised=%v took=%v", opts, disabled, key, advertised, took)
				}
			}
		}
	}
	// Two events before a host draws a frame must use the model's latest
	// value, as a Cycle does, without keeping a second selected value.
	m := &tabModel{value: "one", options: []string{"one", "two", "three"}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ktabs"})
	s.Key("arrow_right")
	s.Key("arrow_right")
	if m.value != "three" {
		t.Fatalf("batched keys picked %q", m.value)
	}
}
