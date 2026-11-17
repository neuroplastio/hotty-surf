package ui_test

import (
	"context"
	"fmt"
	"github.com/a-h/templ"
	"io"
	"neuroplast.io/hotty-go"
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
//	@ui.KeyScope([]ui.KeyBinding{
//	    {Key: "arrow_down", Label: "Next log line", Name: "log.next"},
//	}, a.MoveLog) {
//	    @ui.Select(&a.sizeMenu, ui.SelectProps{Items: sizes,
//	        Value: a.size, OnChange: a.setSize})
//	}
func ExampleKeyScope() {
	moveLog := func(b ui.KeyBinding) bool { return b.Name == "log.next" }
	var state ui.SelectState
	child := ui.Select(&state, ui.SelectProps{Items: []ui.SelectItem{{Value: "a", Label: "A"}}, Placeholder: "Choose"})
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.KeyScope([]ui.KeyBinding{{Key: "arrow_down", Label: "Next log line", Name: "log.next"}}, moveLog).Render(templ.WithChildren(ctx, child), w)
	})
	_ = surf.New("scoped", view, ui.Stylesheet())
}

// A palette lists what the surface offers now, by label, and runs the
// one the user picks; a program with a hotty host that takes menus sends
// the same list as its menu (SPEC §10.7).
func ExampleBindings() {
	s := surf.New("files", templ.Raw(`<p>Files</p>`), ui.Stylesheet())
	s.Do(func() {
		for _, b := range ui.Bindings(s) {
			fmt.Println(b.Key, b.Label, b.Disabled)
		}
	})
}

func ExampleMenu() {
	s := surf.New("files", templ.Raw(`<p>Files</p>`), ui.Stylesheet())
	var out io.Writer = io.Discard // the terminal
	s.Do(func() {
		io.WriteString(out, hotty.Menu(ui.Menu(s, "Files")))
	})
}

// A host's menu sends the name of the item picked as a command event; the
// program runs it.
func ExamplePick() {
	s := surf.New("files", templ.Raw(`<p>Files</p>`), ui.Stylesheet())
	command := "save" // a command event's name
	s.Do(func() { ui.Pick(s, command) })
}

func ExampleCombobox() {
	var state ui.ComboboxState
	value := ""
	_ = ui.Combobox(&state, ui.ComboboxProps{
		Items:       []ui.SelectItem{{Value: "apple", Label: "Apple"}},
		Value:       value,
		Placeholder: "Choose fruit",
		InputAttrs:  templ.Attributes{"aria-label": "Choose fruit"},
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
