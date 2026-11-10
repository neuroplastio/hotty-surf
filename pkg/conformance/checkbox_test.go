package conformance

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

func TestCheckboxModes(t *testing.T) {
	for _, mode := range []func(*CheckboxExample) surflive.Component{surfInterpCheckboxExample, surfCompiledCheckboxExample} {
		state := &CheckboxExample{}
		s := surfrun.NewSurface(surflive.New(mode(state)), nil)
		r := cells.New(s.Cells())
		r.Show()
		r.Draw(20, 3)
		r.Key("tab")
		r.Key("space")
		if !state.Enabled {
			t.Fatal("Space did not toggle the model")
		}
		n := s.Index()["r.kcheck"]
		if checked, _ := surfui.AttrOf(n, "aria-checked"); checked != "true" {
			t.Fatal("model change was not rendered")
		}
		s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kcheck"})
		s.Update()
		if state.Enabled {
			t.Fatal("host click did not toggle the same model")
		}

		// A change says whether the checkbox is now checked, as a bool,
		// to a click of the pointer and to Space alike; the click
		// handler of another is not run by it.
		click := func() {
			s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.kmark"})
			s.Update()
		}
		click()
		click()
		click()
		if !state.Marked || !slices.Equal(state.Heard, []bool{true, false, true}) {
			t.Fatalf("change heard %v, marked %v", state.Heard, state.Marked)
		}
		if state.Enabled {
			t.Fatal("the other checkbox's click ran")
		}
		r.Draw(20, 3)
		r.Key("tab")
		r.Key("space")
		if state.Marked || len(state.Heard) != 4 {
			t.Fatalf("Space: heard %v, marked %v", state.Heard, state.Marked)
		}
	}
}
