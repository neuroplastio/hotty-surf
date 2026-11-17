package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// AlertDialogProps are an AlertDialog's attributes.
type AlertDialogProps struct {
	// ID names its elements; "" is "alert-dialog". It is unique among the
	// surface's dialogs.
	ID string
	// Open shows the dialog; the app sets it false in OnClose.
	Open bool
	// Title and Description label the alert dialog.
	Title, Description string
	// Size is "" (default) or "sm".
	Size string
	// OnClose hears Escape or a press on the backdrop.
	OnClose func()
	// Class adds classes to the dialog element.
	Class string
}

// AlertDialog is a modal alert with the application's actions as children.
// A footer among them holds its buttons. It has no automatic close button.
// Focus enters it on opening and returns to the opener when it closes.
func AlertDialog(p AlertDialogProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return alertDialog(p, templ.GetChildren(ctx)).Render(templ.ClearChildren(ctx), w)
	})
}
