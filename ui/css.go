package ui

import "github.com/a-h/templ"

// The components Basecoat makes with CSS alone: typed from a table of their
// classes and variants (docs/coverage.md, markup css).

// ButtonProps are a Button's attributes.
type ButtonProps struct {
	ID       string
	Variant  Variant
	Size     Size
	Disabled bool
	// OnClick makes the button report clicks; it needs ID.
	OnClick func()
	Class   string
}

// Button is Basecoat's button, holding its children.
func Button(p ButtonProps) templ.Component {
	panic("L0")
}

// BadgeProps are a Badge's attributes.
type BadgeProps struct {
	Variant Variant
	Class   string
}

// Badge is Basecoat's badge, holding its children.
func Badge(p BadgeProps) templ.Component {
	panic("L0")
}

// CardProps are a Card's attributes.
type CardProps struct {
	Class string
}

// Card is Basecoat's card. Its parts are plain elements in its children, as
// Basecoat's markup has them: <header> (an <h2> and a <p>), <section> and
// <footer>.
func Card(p CardProps) templ.Component {
	panic("L0")
}
