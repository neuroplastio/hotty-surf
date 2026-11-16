package ui

import "github.com/a-h/templ"

// DropdownMenuProps are a DropdownMenu's attributes.
type DropdownMenuProps struct {
	ID string
	// Trigger is what opens the menu: Basecoat's button, holding it.
	Trigger templ.Component
	Items   []MenuItem
	// Align is the menu's side along its trigger: "start" (the default),
	// "center" or "end".
	Align string
	Class string
}

// A MenuItem is one of a menu's items. Consecutive items with the same
// Group are shown under its label, the groups apart.
type MenuItem struct {
	Label string
	// Icon is a Lucide icon's name, shown before the label.
	Icon string
	// Keys is the item's shortcut, shown as Kbd shows it. The menu shows it
	// only: the app binds the key itself.
	Keys     string
	Group    string
	Variant  Variant // Destructive, or Default
	Disabled bool
	// OnSelect runs when the user chooses the item; the menu closes.
	OnSelect func()
}

// DropdownMenuState is what a DropdownMenu remembers between renders:
// whether it is open and the item highlighted. The zero value is closed.
type DropdownMenuState struct {
	_ struct{}
}

// Open opens the menu with its first item highlighted, as a shortcut that
// opens it does.
func (st *DropdownMenuState) Open() {
	panic("L0")
}

// DropdownMenu is Basecoat's dropdown menu: its markup generated from
// dropdown-menu.html.jinja, its behaviour dropdown-menu.js's, translated to
// Go (the arrows, Home, End, Enter, Escape, typing to find, a click outside
// closing it).
func DropdownMenu(st *DropdownMenuState, p DropdownMenuProps) templ.Component {
	panic("L0")
}
