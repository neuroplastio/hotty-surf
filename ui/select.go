package ui

import "github.com/a-h/templ"

// SelectProps are a Select's attributes.
type SelectProps struct {
	ID    string
	Items []SelectItem
	// Value is the value of the item shown as chosen; "" is none, and the
	// trigger shows Placeholder.
	Value       string
	Placeholder string
	Disabled    bool
	// OnChange hears the value the user chose.
	OnChange func(value string)
	Class    string
}

// A SelectItem is one of a Select's options. Consecutive items with the same
// Group are shown under its label, the groups apart.
type SelectItem struct {
	Value, Label string
	Group        string
	Disabled     bool
}

// SelectState is what a Select remembers between renders: whether its list
// is open and the item the arrows highlighted. The zero value is closed.
type SelectState struct {
	_ struct{}
}

// Select is Basecoat's select: its markup generated from select.html.jinja,
// its behaviour select.js's, translated to Go (opening, the arrows, Home,
// End, choosing, Escape, a click outside closing it).
func Select(st *SelectState, p SelectProps) templ.Component {
	panic("L0")
}
