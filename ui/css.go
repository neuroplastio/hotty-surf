package ui

import (
	"bytes"
	"context"
	"io"

	"github.com/a-h/templ"
)

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
	// Value is the Value of the option shown chosen, one of Items'
	// Values; surftest fails any other. A select that starts with no
	// choice has an item whose Value is "" ({Label: "Choose…"}), as an
	// HTML select does. The surface sets it on the host whenever it
	// differs from what the host last reported.
	Value string
	// Size is Medium or Small (data-size).
	Size     Size
	Disabled bool
	// Invalid marks it as failing its form's check (aria-invalid), in the
	// destructive colour.
	Invalid bool
	// Attrs are more attributes for the select element: its id and
	// handlers, through surf.On, and any other. A surf.Input handler hears
	// the user's choice, its option's Value. An attribute the component
	// writes itself (its class, data-slot, or a field's) is a fault here.
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

// FieldSeparatorProps are a FieldSeparator's attributes.
type FieldSeparatorProps struct {
	// Attrs are more attributes for the div element: its id and handlers,
	// through surf.On, and any other. An attribute the component writes
	// itself (its class or data-slot) is a fault here.
	Attrs templ.Attributes
	// Class is more classes for the element, after its own.
	Class string
}

// FieldSeparator is Basecoat's field separator: a line between two Fields
// or FieldSets, with its children in the middle ("Or"), or a bare line
// without them:
//
//	@ui.FieldSeparator(ui.FieldSeparatorProps{}) {
//		Or
//	}
func FieldSeparator(p FieldSeparatorProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var b bytes.Buffer
		if kids := templ.GetChildren(ctx); kids != nil {
			if err := kids.Render(outOfGroup(ctx), &b); err != nil {
				return err
			}
		}
		return fieldSeparator(p, b.String()).Render(templ.ClearChildren(ctx), w)
	})
}
