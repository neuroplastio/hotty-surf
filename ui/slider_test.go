package ui_test

import (
	"context"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

func TestSliderKeysAndDrag(t *testing.T) {
	value := 20.0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Slider(ui.SliderProps{ID: "volume", Attrs: templ.Attributes{"aria-label": "Volume"}, Min: 0, Max: 100, Step: 5, Value: value, OnChange: func(v float64) { value = v }}).Render(ctx, w)
	})
	s := surf.New("slider", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#volume-thumb")
	r.Key("arrow_right")
	if value != 25 {
		t.Fatalf("right: %v", value)
	}
	r.Key("page_up")
	if value != 75 {
		t.Fatalf("page up: %v", value)
	}
	r.Key("home")
	if value != 0 {
		t.Fatalf("home: %v", value)
	}
	event := func(kind, target string, x int) {
		t.Helper()
		if err := s.Event(hotty.Event{Kind: kind, Target: target, Drag: &hotty.Drag{X: x, HasX: true}}); err != nil {
			t.Fatal(err)
		}
		r.Update()
	}
	event(hotty.EventDragStart, "volume", 4)
	if value != 20 {
		t.Fatalf("dragstart: %v", value)
	}
	event(hotty.EventDrag, "elsewhere", 14)
	if value != 70 {
		t.Fatalf("drag outside: %v", value)
	}
	event(hotty.EventDragEnd, "elsewhere", 16)
	if value != 80 {
		t.Fatalf("dragend: %v", value)
	}
	r.Click("#volume-notch-2")
	if value != 10 {
		t.Fatalf("tap after release outside: %v", value)
	}
}

func TestRangeSliderKeysAndDrag(t *testing.T) {
	lo, hi := 20.0, 70.0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.RangeSlider(ui.RangeSliderProps{ID: "range", StartAttrs: templ.Attributes{"aria-label": "Price, start"}, EndAttrs: templ.Attributes{"aria-label": "Price, end"}, Min: 0, Max: 100, Step: 5, Start: lo, End: hi, OnChange: func(a, b float64) { lo, hi = a, b }}).Render(ctx, w)
	})
	s := surf.New("range", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#range-start")
	if got := rt.Peek.Focused(s); got != "range-start" {
		t.Fatalf("first focus: %q", got)
	}
	r.Key("end")
	if lo != 70 || hi != 70 {
		t.Fatalf("end clamps start: %v %v", lo, hi)
	}
	r.Key("tab")
	if got := rt.Peek.Focused(s); got != "range-end" {
		t.Fatalf("second focus: %q", got)
	}
	r.Key("home")
	if lo != 70 || hi != 70 {
		t.Fatalf("home clamps end: %v %v", lo, hi)
	}
	if err := s.Event(hotty.Event{Kind: hotty.EventDragStart, Target: "range", Drag: &hotty.Drag{X: 14, HasX: true}}); err != nil {
		t.Fatal(err)
	}
	r.Update()
	if err := s.Event(hotty.Event{Kind: hotty.EventDrag, Drag: &hotty.Drag{X: 10, HasX: true}}); err != nil {
		t.Fatal(err)
	}
	r.Update()
	if lo != 50 || hi != 70 {
		t.Fatalf("drag joined thumbs: %v %v", lo, hi)
	}
}

func TestSliderKeyScopeAndDisabled(t *testing.T) {
	value, intercepted := 20.0, false
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		p := ui.SliderProps{ID: "volume", Attrs: templ.Attributes{"aria-label": "Volume"}, Min: 0, Max: 100, Step: 5, Value: value, OnChange: func(v float64) { value = v }}
		return with(ui.KeyScope([]ui.KeyBinding{{Key: "arrow_right"}}, func(ui.KeyBinding) bool { intercepted = true; return true }), ui.Slider(p)).Render(ctx, w)
	})
	r := surftest.Start(t, surf.New("slider", view, ui.Stylesheet()))
	r.Click("#volume-thumb")
	r.Key("arrow_right")
	if !intercepted || value != 20 {
		t.Fatalf("scope took=%v value=%v", intercepted, value)
	}
	r.Key("arrow_left")
	if value != 15 {
		t.Fatalf("unscoped key: %v", value)
	}
	disabled := ui.Slider(ui.SliderProps{ID: "off", Attrs: templ.Attributes{"aria-label": "Off"}, Min: 0, Max: 10, Value: 5, Disabled: true})
	d := surftest.Start(t, surf.New("disabled", disabled, ui.Stylesheet()))
	d.One("#off-thumb[disabled]")
	if len(d.Find("#off[data-on]")) != 0 {
		t.Fatal("disabled track accepts drags")
	}
}

func TestModalBlocksBackgroundSliderDrag(t *testing.T) {
	value := 20.0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		slider := ui.Slider(ui.SliderProps{ID: "volume", Attrs: templ.Attributes{"aria-label": "Volume"}, Min: 0, Max: 100, Step: 5, Value: value, OnChange: func(v float64) { value = v }})
		modal := with(ui.Dialog(ui.DialogProps{ID: "dialog", Open: true, Title: "Confirm"}), button("inside", "OK", func() {}))
		return page(slider, modal).Render(ctx, w)
	})
	s := surf.New("modal-slider", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	if err := s.Event(hotty.Event{Kind: hotty.EventDragStart, Target: "volume", Drag: &hotty.Drag{X: 15, HasX: true}}); err != nil {
		t.Fatal(err)
	}
	r.Update()
	if value != 20 {
		t.Fatalf("background drag changed value to %v", value)
	}
}

func TestSliderInvalidNumbersAndAttrs(t *testing.T) {
	f := &faults{T: t}
	attrs := templ.Attributes{"aria-label": "Value", "ID": "wrong", "DATA-KEYS": "home=field", "ARIA-VALUENOW": "bad", "Data-Hotty-KeyScope": "999"}
	slider := ui.Slider(ui.SliderProps{ID: "number", Min: 0, Max: 10, Value: math.NaN(), Attrs: attrs})
	r := surftest.Start(f, surf.New("slider", with(ui.KeyScope([]ui.KeyBinding{{Key: "home"}}, func(ui.KeyBinding) bool { return false }), slider), ui.Stylesheet()))
	r.One(`#number-thumb[aria-valuenow="0"][aria-label="Value"]`)
	if got := attr(r.One("#number-thumb"), "data-keys"); !strings.Contains(got, "home=program") || strings.Contains(got, "home=field") {
		t.Fatalf("key bindings %q", got)
	}
	if got := attr(r.One("#number-thumb"), "data-hotty-keyscope"); got == "" || got == "999" {
		t.Fatalf("scope marker %q", got)
	}
	if len(f.got) < 4 {
		t.Fatalf("expected value and attribute faults: %v", f.got)
	}
	if attrs["ID"] != "wrong" || attrs["DATA-KEYS"] != "home=field" || attrs["Data-Hotty-KeyScope"] != "999" {
		t.Fatalf("mutated attrs: %v", attrs)
	}
	f2 := &faults{T: t}
	ranged := ui.RangeSlider(ui.RangeSliderProps{ID: "range", Min: 0, Max: 10, Start: math.Inf(1), End: math.NaN(), StartAttrs: templ.Attributes{"aria-label": "Start"}, EndAttrs: templ.Attributes{"aria-label": "End"}})
	r2 := surftest.Start(f2, surf.New("range", ranged, ui.Stylesheet()))
	r2.One(`#range-start[aria-valuenow="0"]`)
	r2.One(`#range-end[aria-valuenow="10"]`)
	if len(f2.got) == 0 {
		t.Fatal("nonfinite range did not fault")
	}
}

func TestRangeSliderClampsInvalidValuesWithoutCallback(t *testing.T) {
	f := &faults{T: t}
	called := false
	p := ui.RangeSliderProps{ID: "range", Min: 0, Max: 10, Start: 8, End: 2, StartAttrs: templ.Attributes{"aria-label": "Start"}, EndAttrs: templ.Attributes{"aria-label": "End"}, OnChange: func(float64, float64) { called = true }}
	r := surftest.Start(f, surf.New("range", ui.RangeSlider(p), ui.Stylesheet()))
	r.One(`#range-start[aria-valuenow="8"]`)
	r.One(`#range-end[aria-valuenow="8"]`)
	if len(f.got) == 0 || called {
		t.Fatalf("faults=%v called=%v", f.got, called)
	}
}

func TestDisabledRangeThumbsRejectCaseInsensitiveAttrs(t *testing.T) {
	f := &faults{T: t}
	start := templ.Attributes{"aria-label": "Start", "ID": "wrong-start", "DATA-KEYS": "enter=field", "ArIa-ValueNow": "bad"}
	end := templ.Attributes{"aria-label": "End", "Id": "wrong-end", "DATA-HOTTY-KEYSCOPE": "999"}
	p := ui.RangeSliderProps{ID: "range", Min: 0, Max: 10, Start: 2, End: 8, Disabled: true, StartAttrs: start, EndAttrs: end}
	r := surftest.Start(f, surf.New("range", ui.RangeSlider(p), ui.Stylesheet()))
	r.One(`#range-start[disabled][aria-valuenow="2"]`)
	r.One(`#range-end[disabled][aria-valuenow="8"]`)
	if len(f.got) < 4 {
		t.Fatalf("expected owned-attribute faults: %v", f.got)
	}
	if start["ID"] != "wrong-start" || end["Id"] != "wrong-end" {
		t.Fatalf("mutated attrs: %v %v", start, end)
	}
}

// The slider's styles are the kit stylesheet's, so a page has them once,
// however many sliders it draws, and no slider writes a <style>.
func TestSliderStylesInStylesheet(t *testing.T) {
	if !strings.Contains(ui.Stylesheet(), "[data-hotty-slider-thumb]") {
		t.Fatal("the kit stylesheet has no slider styles")
	}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			ui.Slider(ui.SliderProps{ID: "a", Min: 0, Max: 10, Value: 5, Attrs: templ.Attributes{"aria-label": "A"}}),
			ui.Slider(ui.SliderProps{ID: "b", Min: 0, Max: 10, Value: 2, Attrs: templ.Attributes{"aria-label": "B"}}),
		).Render(ctx, w)
	})
	r := surftest.Start(t, surf.New("slider", view, ui.Stylesheet()))
	if n := len(r.Find("style")); n != 0 {
		t.Fatalf("two sliders wrote %d <style> elements, want none", n)
	}
}
