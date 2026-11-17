package ui_test

import (
	"context"
	"io"
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
		return with(ui.KeyScope([]string{"arrow_right"}, func(string) bool { intercepted = true; return true }), ui.Slider(p)).Render(ctx, w)
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
