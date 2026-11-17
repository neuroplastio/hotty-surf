package ui_test

import (
	"context"
	"github.com/a-h/templ"
	"io"
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
	var state ui.SelectState
	child := ui.Select(&state, ui.SelectProps{Items: []ui.SelectItem{{Value: "a", Label: "A"}}, Placeholder: "Choose"})
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.KeyScope([]string{"arrow_down"}, moveLog).Render(templ.WithChildren(ctx, child), w)
	})
	_ = surf.New("scoped", view, ui.Stylesheet())
}

// A row's context menu: a right click on the row opens it at the
// pointer, shift+f10 or the Menu key on the row's button under the row.
// In templ:
//
//	@ui.ContextMenu(&row.menu, ui.ContextMenuProps{ID: "menu-" + row.ID,
//	    Items: []ui.MenuItem{{Label: "Rename", OnSelect: row.rename}}}) {
//	    @ui.Item(ui.ItemProps{Attrs: surf.On(ctx, row.ID, surf.Click(row.open))}) {
//	        { row.Name }
//	    }
//	}
func ExampleContextMenu() {
	var state ui.ContextMenuState
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		row := templ.Raw(`<p>README.md</p>`)
		menu := ui.ContextMenu(&state, ui.ContextMenuProps{
			ID: "menu-readme",
			Items: []ui.MenuItem{
				{Label: "Open", OnSelect: func() {}},
				{Label: "Delete", Variant: ui.Destructive, OnSelect: func() {}},
			},
		})
		return menu.Render(templ.WithChildren(ctx, row), w)
	})
	_ = surf.New("files", view, ui.Stylesheet())
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

// A menu the app places itself (a surface of its own at the pointer): the
// surface's whole view is a Menu, its highlight the app's.
func ExampleMenu() {
	hi := -1
	items := []ui.MenuItem{
		{Label: "Split right", Keys: "mod+d", OnSelect: func() {}},
		{Label: "Close", Variant: ui.Destructive, OnSelect: func() {}},
	}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Menu(ui.MenuProps{
			ID: "pane-menu", Items: items, Highlight: hi,
			OnHighlight: func(i int) { hi = i },
		}).Render(ctx, w)
	})
	_ = surf.New("pane-menu", view, ui.Stylesheet())
}

// Rows of the app's own: a host's colour in the icon's place, its tabs in
// the keys' place, a rule, and a choice drawn as a switch with an aside.
func ExampleMenuItem_slots() {
	wrap := true
	_ = []ui.MenuItem{
		{Label: "build-01",
			Leading:  templ.Raw(`<span class="dot" style="color:#3a7">●</span>`),
			Trailing: templ.Raw(`<span>3 tabs</span>`),
			OnSelect: func() {}},
		{Separator: true},
		{Label: "Wrap",
			Leading:  ui.Switch(ui.SwitchProps{Checked: wrap}),
			Content:  templ.Raw(`<span>Wrap <small>long lines</small></span>`),
			OnSelect: func() { wrap = !wrap }},
	}
}
