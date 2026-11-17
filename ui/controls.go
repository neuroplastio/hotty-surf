package ui

import (
	"strconv"
	"strings"
	"time"
)

// Control is one value in Controls. Name is its visible label and the
// name sent to OnChange. Kind is "bool" for a switch, "enum" for a
// select, or "string" or "int" for a text field. Value is its value
// as text; Options are the enum's choices. Any other Kind renders as a
// text field; a bool is checked exactly when Value is "true".
type Control struct {
	// Name is its label and the name sent to OnChange.
	Name string
	// Kind selects its input.
	Kind string
	// Value is the controlled value as text.
	Value string
	// Options are an enum's choices.
	Options []string
}

// ControlsProps are a Controls' attributes. OnChange is required when a
// control is edited. Names must produce distinct ControlIDs.
type ControlsProps struct {
	// ID prefixes the generated element IDs.
	ID string
	// Controls are the values to edit, in display order.
	Controls []Control
	// OnChange receives the control's Name and new value as text.
	OnChange func(name, value string)
	// Class adds classes to the grid.
	Class string
}

// EventLogProps are an EventLog's attributes.
type EventLogProps struct {
	// ID prefixes the clear button's element ID.
	ID string
	// Entries are in chronological order, oldest first.
	Entries []LogEntry
	// OnClear is called by the Clear button; nil hides it.
	OnClear func()
	// Empty is the text shown when Entries is empty.
	Empty string
}

func controlID(id, name string) string { return id + "-ctl-" + strings.ReplaceAll(name, ".", "-") }

// ControlID is the element ID for the control named name in Controls with
// ID id. Names must be chosen so their IDs are distinct: dots become hyphens.
func ControlID(id, name string) string { return controlID(id, name) }

// LogEntry is an EventLog entry: At is its time, Kind is the displayed
// category, Text is the message, and KindClass adds CSS classes to the
// category. The caller supplies the corresponding stylesheet.
type LogEntry struct {
	// At is the event's time.
	At time.Time
	// Kind is the displayed category.
	Kind string
	// Text is the event message.
	Text string
	// KindClass adds CSS classes to the category.
	KindClass string
}

func logKindClass(class string) string {
	if class == "" {
		return "text-muted-foreground"
	}
	return class
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
