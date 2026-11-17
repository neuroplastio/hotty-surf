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
//	    {Key: "arrow_down", Label: "Next log line", Name: "log.next", Run: a.nextLine},
//	    {Key: "mod+s", Label: "Save", Name: "save", Run: a.save, Disabled: !a.dirty},
//	    {Label: "About", Name: "about", Run: a.about},
//	}) {
//	    @ui.Select(&a.sizeMenu, ui.SelectProps{Items: sizes,
//	        Value: a.size, OnChange: a.setSize})
//	}
func ExampleKeyScope() {
	nextLine := func() bool { return true }
	var state ui.SelectState
	child := ui.Select(&state, ui.SelectProps{Items: []ui.SelectItem{{Value: "a", Label: "A"}}, Placeholder: "Choose"})
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.KeyScope([]ui.KeyBinding{{Key: "arrow_down", Label: "Next log line", Name: "log.next", Run: nextLine}}).Render(templ.WithChildren(ctx, child), w)
	})
	_ = surf.New("scoped", view, ui.Stylesheet())
}

// files is a view with a scope, for the examples below.
func files(save func() bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		scope := ui.KeyScope([]ui.KeyBinding{
			{Key: "mod+s", Label: "Save", Name: "save", Run: save},
			{Label: "About", Name: "about", Run: func() bool { return true }},
		})
		return scope.Render(templ.WithChildren(ctx, templ.Raw(`<p>Files</p>`)), w)
	})
}

// A palette is a Command whose items are the surface's bindings: each
// picks its binding by name.
func ExampleBindings() {
	s := surf.New("files", files(func() bool { return true }), ui.Stylesheet())
	var palette []ui.MenuItem
	s.Do(func() {
		palette = palette[:0]
		for _, b := range ui.Bindings(s) {
			if b.Name == "" {
				continue // only its key runs it
			}
			name := b.Name
			palette = append(palette, ui.MenuItem{Label: b.Label, Keys: b.Key, Disabled: b.Disabled,
				OnSelect: func() { ui.Pick(s, name) }})
		}
	})
	var state ui.CommandState
	_ = ui.Command(&state, ui.CommandProps{Items: palette})
}

// A host that takes menus shows the surface's keys as its which-key level
// and lists its actions in its palette (SPEC §10.7).
func ExampleHostMenu() {
	s := surf.New("files", files(func() bool { return true }), ui.Stylesheet())
	var out io.Writer = io.Discard // the terminal
	s.Do(func() {
		fmt.Fprint(out, hotty.Menu(ui.HostMenu(s, "Files")))
	})
}

// A host's menu sends the name of the item picked as a command event; the
// program runs it.
func ExamplePick() {
	saved := false
	s := surf.New("files", files(func() bool { saved = true; return true }), ui.Stylesheet())
	command := "save" // a command event's name
	s.Do(func() { ui.Pick(s, command) })
	_ = saved
}

// A pane's context menu: a right click in the pane opens it at the
// pointer, shift+f10 or the Menu key on an element in it under the pane.
// In templ:
//
//	@ui.ContextMenu(&pane.menu, ui.ContextMenuProps{ID: "files-menu",
//	    Items: []ui.MenuItem{{Label: "New file", OnSelect: pane.create}}}) {
//	    @ui.Card(ui.CardProps{}) {
//	        @files(pane)
//	    }
//	}
func ExampleContextMenu() {
	var state ui.ContextMenuState
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		pane := templ.Raw(`<p>Files</p>`)
		menu := ui.ContextMenu(&state, ui.ContextMenuProps{
			ID: "files-menu",
			Items: []ui.MenuItem{
				{Label: "New file", OnSelect: func() {}},
				{Label: "Refresh", OnSelect: func() {}},
			},
		})
		return menu.Render(templ.WithChildren(ctx, pane), w)
	})
	_ = surf.New("pane", view, ui.Stylesheet())
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

// hostDot is a host's colour as a dot: a component that escapes what it
// is given (the colour is config's, not constant markup).
func hostDot(color string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		style := templ.SanitizeCSS("color", color)
		_, err := io.WriteString(w, `<span class="dot" style="`+templ.EscapeString(string(style))+`">●</span>`)
		return err
	})
}

// Rows of the app's own: a host's colour in the icon's place, its tabs in
// the keys' place, a rule, and an on/off row with an aside. templ.Raw is
// for constant markup only.
func ExampleMenuItem_slots() {
	wrap := true
	_ = []ui.MenuItem{
		{Label: "build-01",
			Leading:  hostDot("#3a7"),
			Trailing: templ.Raw(`<span>3 tabs</span>`),
			OnSelect: func() {}},
		{Role: "separator"},
		{Label: "Wrap", Role: "menuitemcheckbox", Checked: wrap,
			Content:         templ.Raw(`<span>Wrap <small>long lines</small></span>`),
			OnCheckedChange: func(on bool) { wrap = on }},
		{Label: "Here"},
	}
}
