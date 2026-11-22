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

// FieldSeparatorProps are a FieldSeparator's attributes.
type FieldSeparatorProps struct {
	// Attrs are more attributes for the div element: its id and handlers,
	// through surf.On, and any other but those the component writes itself
	// (class or data-slot), which are a fault here.
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
