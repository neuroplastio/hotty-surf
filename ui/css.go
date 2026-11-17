package ui

import "github.com/a-h/templ"

// The components Basecoat makes with CSS alone. Those that are one element
// holding their children are generated from a table of their classes and
// attributes (internal/uigen, css_gen.go); the few with parts of their own
// are here. Each takes Attrs, any attribute of its element: handlers
// through surf.On, an id, a name, aria-*. A class goes in Class, not
// Attrs: the element's own class comes first, and of two the first wins.

//go:generate go run ../internal/uigen .
//go:generate templ generate -f css_gen.templ

// NativeSelectProps are a NativeSelect's attributes.
type NativeSelectProps struct {
	// Items are its options. Consecutive items with one Group are an
	// <optgroup> of that label.
	Items []SelectItem
	// Value is the Value of the option it shows chosen, which the surface
	// sets on the host whenever it differs from what the host last
	// reported. A Value no item has chooses none, and leaves the choice to
	// the host: it shows its first option.
	Value    string
	Disabled bool
	// Invalid marks it as failing its form's check (aria-invalid), in the
	// destructive colour.
	Invalid bool
	// Attrs are more attributes for the select element: its id and
	// handlers, through surf.On, and any other. A surf.Input handler hears
	// the user's choice, its option's Value.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// NativeSelect is Basecoat's native select: the host's own <select>, which
// a host may show with no list of its own (SPEC §10.2), so that the arrow
// keys choose. Select is the kit's, with its list drawn in the page.
func NativeSelect(p NativeSelectProps) templ.Component {
	return nativeSelect(p)
}

// RangeProps are a Range's attributes.
type RangeProps struct {
	// Value is where it is set, which the surface sets on the host
	// whenever it differs from what the host last reported.
	Value float64
	// Min and Max are its ends; a Max of 0 is 100, as HTML's default.
	Min, Max float64
	// Step is how far a step moves it; 0 is 1, as HTML's default.
	Step     float64
	Disabled bool
	// Attrs are more attributes for the input element: its id and
	// handlers, through surf.On, and any other. A surf.Input handler hears
	// each move, its value as text.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// Range is Basecoat's range slider, a native one the host moves, with its
// track filled up to Value as range.js fills it.
func Range(p RangeProps) templ.Component {
	return rangeInput(p)
}

// FieldSeparator is Basecoat's field separator: a line between two Fields
// or FieldSets, with label ("Or") in its middle, or none for "".
func FieldSeparator(label string) templ.Component {
	return fieldSeparator(label)
}
