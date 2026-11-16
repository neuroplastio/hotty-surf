package ui

import "github.com/a-h/templ"

// DialogProps are a Dialog's attributes.
type DialogProps struct {
	ID string
	// Open shows the dialog. The app keeps it, and sets it false in
	// OnClose.
	Open        bool
	Title       string
	Description string
	// OnClose hears that the user dismissed the dialog: Escape, the close
	// button, or a click on the backdrop.
	OnClose func()
	Class   string
}

// Dialog is Basecoat's dialog, modal: its markup generated from
// dialog.html.jinja. The children are its body; a <footer> among them holds
// its buttons, as Basecoat's markup has it. The keyboard goes into it when
// it opens and back where it was when it closes.
func Dialog(p DialogProps) templ.Component {
	panic("L0")
}
