package ui

import "github.com/a-h/templ"

// TableProps are a Table's attributes.
type TableProps struct {
	// Attrs are more attributes for the table element: its id and
	// handlers, through surf.On, and any other but those the component
	// writes itself (class or data-slot), which are a fault here.
	Attrs templ.Attributes
	// Class is more classes for the table element, after its own.
	Class string
}

// Table is Basecoat's table, in the container that scrolls it sideways
// when it is wider than its place. Its rows and cells are plain elements
// in its children, as Basecoat's markup has them: <thead>, <tbody> and
// <tfoot>, their <tr>s (data-state="selected" marks one chosen), <th>s and
// <td>s, and a <caption>. Attrs and Class are the table's, as shadcn's
// Table takes them, not the container's.
func Table(p TableProps) templ.Component {
	return scoped(table(p), false)
}
