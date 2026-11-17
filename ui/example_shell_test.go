package ui_test

import (
	"github.com/a-h/templ"
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
	_ = ui.ItemTitle()
	_ = ui.ItemSubtitle()
	_ = ui.ItemDetail()
}

// A symbol is decorative; the button carries its accessible name.
func ExampleIconButton() {
	_ = ui.IconButton("Hosts", ui.ShapePublic, "", templ.Attributes{"id": "hosts"})
	_ = ui.Symbols()
}
