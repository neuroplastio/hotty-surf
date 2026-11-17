package ui_test

import (
	"fmt"
	"time"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/ui"
)

func ExampleNavList() {
	nav := &ui.NavList{
		ID: "sections", Label: "Sections", FilterLabel: "Filter sections",
		Groups: []ui.NavGroup{{Name: "Account", Items: []ui.NavItem{
			{ID: "profile", Label: "Profile"}, {ID: "security", Label: "Security"},
		}}}, Current: "profile", OnSelect: func(id string) { fmt.Println(id) },
	}
	// Place these two components in the app's view; the app supplies CSS.
	_ = templ.Join(nav.FilterInput(""), nav.List("overflow-y-auto"))
}

func ExampleControls() {
	cs := []ui.Control{{Name: "Label", Kind: "string", Value: "Save"}}
	_ = ui.Controls(ui.ControlsProps{ID: "demo", Controls: cs, OnChange: func(name, value string) {}})
}

func ExampleEventLog() {
	entries := []ui.LogEntry{{At: time.Unix(0, 0), Kind: "click", Text: "save"}}
	_ = ui.EventLog(ui.EventLogProps{ID: "demo", Entries: entries, OnClear: func() {}, Empty: "Nothing yet"})
}

func ExampleFuzzy() {
	fmt.Println(ui.Fuzzy("Account Security", "as"))
	// Output: true
}

func ExampleControlID() {
	fmt.Println(ui.ControlID("demo", "Menu.Open"))
	// Output: demo-ctl-Menu-Open
}
