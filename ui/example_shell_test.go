package ui_test

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

// A program keeps which item is selected; the List asks it to change
// with OnChange, and a row's own click opens it.
func ExampleList() {
	sessions := []string{"main", "docs", "build"}
	selected := 0
	open := func(name string) func() { return func() { fmt.Println("open", name) } }
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for i, name := range sessions {
				item := ui.ListItem(ui.ListItemProps{
					Selected: selected == i,
					Leading:  ui.Symbol(ui.ShapeTerminal, "size-4"),
					Attrs:    surf.On(ctx, "session-"+name, surf.Click(open(name))),
				})
				title := ui.ItemTitle(ui.ItemTitleProps{})
				err := item.Render(templ.WithChildren(ctx,
					templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						return title.Render(templ.WithChildren(ctx, templ.Raw(name)), w)
					})), w)
				if err != nil {
					return err
				}
			}
			return nil
		})
		list := ui.List(ui.ListProps{
			Attrs:    templ.Attributes{"aria-label": "Sessions"},
			OnChange: func(i int) { selected = i },
		})
		return list.Render(templ.WithChildren(ctx, items), w)
	})
	_ = view
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
