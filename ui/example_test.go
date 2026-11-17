package ui_test

import (
	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// An app keeps its data and each component's state; a view passes both in.
// A view is usually a .templ file (examples/hello):
//
//	@ui.Select(&a.sizeMenu, ui.SelectProps{ID: "size", Items: sizes,
//		Value: a.size, OnChange: a.setSize})
//	@ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "save", surf.Click(a.save))}) { Save }
//	@ui.Toaster(&a.toasts)
func Example() {
	type app struct {
		size     string
		sizeMenu ui.SelectState
		toasts   ui.ToasterState
	}
	a := &app{size: "m"}
	sizes := []ui.SelectItem{
		{Value: "s", Label: "Small"},
		{Value: "m", Label: "Medium"},
		{Value: "l", Label: "Large"},
	}
	view := ui.Select(&a.sizeMenu, ui.SelectProps{
		ID:    "size",
		Items: sizes,
		Value: a.size,
		OnChange: func(v string) {
			a.size = v
			a.toasts.Show(ui.Toast{Title: "Size changed"})
		},
	})
	_ = surf.New("sizes", view, ui.Stylesheet())
}

// In templ, a scope around a control can take a key before its default:
//
//	@ui.KeyScope([]string{"arrow_down"}, a.MoveLog) {
//	    @ui.Select(&a.sizeMenu, ui.SelectProps{Items: sizes,
//	        Value: a.size, OnChange: a.setSize})
//	}
func ExampleKeyScope() {
	moveLog := func(key string) bool { return key == "arrow_down" }
	_ = ui.KeyScope([]string{"arrow_down"}, moveLog)
}

func ExampleCombobox() {
	var state ui.ComboboxState
	value := ""
	_ = ui.Combobox(&state, ui.ComboboxProps{
		Items:       []ui.SelectItem{{Value: "apple", Label: "Apple"}},
		Value:       value,
		Placeholder: "Choose fruit",
		OnChange:    func(v string) { value = v },
	})
}

func ExampleDrawer() {
	var state ui.DrawerState
	open := true
	_ = ui.Drawer(&state, ui.DrawerProps{
		Open: open, Side: "left", OnClose: func() { open = false },
	})
}

func ExampleAlertDialog() {
	_ = ui.AlertDialog(ui.AlertDialogProps{
		ID: "confirm", Open: true, Title: "Delete item?",
		Description: "This cannot be undone.", OnClose: func() {},
	})
}

func ExampleTooltip() {
	_ = ui.Tooltip(ui.TooltipProps{Text: "Add to library", Side: "top"})
}

func ExampleSlider() {
	_ = ui.Slider(ui.SliderProps{
		ID: "volume", Attrs: templ.Attributes{"aria-label": "Volume"}, Min: 0, Max: 100, Step: 5,
		Value: 20, OnChange: func(float64) {},
	})
}

func ExampleRangeSlider() {
	_ = ui.RangeSlider(ui.RangeSliderProps{
		ID: "price", StartAttrs: templ.Attributes{"aria-label": "Price, start"}, EndAttrs: templ.Attributes{"aria-label": "Price, end"}, Min: 0, Max: 100, Step: 5,
		Start: 20, End: 70, OnChange: func(float64, float64) {},
	})
}

func ExampleToastAction() {
	_ = ui.Toast{Title: "Deleted", Action: &ui.ToastAction{
		Label: "Undo", OnSelect: func() {},
	}}
}
