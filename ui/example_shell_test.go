package ui_test

import (
	"context"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// A program keeps which item is selected and handles activation itself.
func ExampleListItem() {
	selected := 0
	_ = ui.List("Sessions")
	_ = ui.ListItem(ui.ListItemProps{
		ID: "session-0", Selected: selected == 0,
		OnClick: func() { selected = 0 },
		Leading: ui.Symbol(ui.ShapeTerminal, "size-4"),
	})
	_ = ui.ItemTitle(ui.ItemTitleProps{})
	_ = ui.ItemSubtitle(ui.ItemSubtitleProps{})
	_ = ui.ItemDetail(ui.ItemDetailProps{})
}

// A symbol is decorative; the button carries its accessible name.
func ExampleIconButton() {
	attrs := surf.On(context.Background(), "hosts", surf.Click(func() {}))
	attrs["aria-label"], attrs["title"] = "Hosts", "Hosts"
	_ = ui.IconButton(ui.IconButtonProps{Shape: ui.ShapePublic, Attrs: attrs})
	_ = ui.Symbols()
}

// A named button may omit its click handler or use aria-labelledby.
func ExampleIconButton_labelledBy() {
	_ = ui.IconButton(ui.IconButtonProps{
		Shape: ui.ShapeTerminal,
		Attrs: map[string]any{"aria-labelledby": "terminal-label"},
	})
}
