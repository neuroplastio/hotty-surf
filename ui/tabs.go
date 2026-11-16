package ui

import "github.com/a-h/templ"

// TabsProps are a Tabs' attributes.
type TabsProps struct {
	ID   string
	Tabs []Tab
	// Value is the value of the tab shown.
	Value string
	// OnChange hears the tab the user chose.
	OnChange func(value string)
	Class    string
}

// A Tab is one of a Tabs' tabs.
type Tab struct {
	Value, Label string
	Disabled     bool
}

// Tabs is Basecoat's tabs: its markup generated from tabs.html.jinja, its
// keys tabs.js's, translated to Go (← → between the tabs, Home, End). The
// children are the shown tab's panel.
func Tabs(p TabsProps) templ.Component {
	panic("L0")
}
