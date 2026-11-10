package conformance

import (
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

func TestTabsModes(t *testing.T) {
	for _, mode := range []func(*TabsExample) surflive.Component{surfInterpTabsExample, surfCompiledTabsExample} {
		state := &TabsExample{Value: "Source"}
		s := surfrun.NewSurface(surflive.New(mode(state)), nil)
		r := cells.New(s.Cells())
		r.Show()
		r.Draw(40, 4)
		r.Key("tab")
		r.Key("arrow_right")
		if state.Value != "Result" {
			t.Fatalf("cell key picked %q", state.Value)
		}
		s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.ktabs.k2"})
		s.Update()
		if state.Value != "Log" {
			t.Fatalf("host click picked %q", state.Value)
		}
	}
}
