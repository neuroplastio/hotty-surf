package ui_test

import (
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
