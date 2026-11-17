package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/internal/rt"
)

// AlertDialogProps are an AlertDialog's attributes.
type AlertDialogProps struct {
	// ID names its elements; "" is "alert-dialog". It is unique among the
	// surface's dialogs.
	ID string
	// Open shows the dialog; the app sets it false in OnClose.
	Open bool
	// Title is the visible heading and accessible name; it must be nonempty.
	// Description is optional and is referenced only when rendered.
	Title, Description string
	// Size is Medium (the default) or Small (data-size). surftest fails
	// any other size.
	Size Size
	// OnClose requests closing on Escape or a backdrop press; the app sets
	// Open false.
	OnClose func()
	// Class adds classes to the dialog element.
	Class string
}

func checkAlertDialog(ctx context.Context, p AlertDialogProps) string {
	if p.Title == "" {
		rt.Fault(ctx, "ui.AlertDialog: Title is required")
	}
	return ""
}

// AlertDialog is a modal alert with the application's actions as children.
// A footer among them holds its buttons. It has no automatic close button.
// Focus enters it on opening and returns to the opener when it closes.
func AlertDialog(p AlertDialogProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return alertDialog(p, templ.GetChildren(ctx)).Render(templ.ClearChildren(ctx), w)
	})
}
