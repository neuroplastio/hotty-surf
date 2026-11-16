package ui

import "github.com/a-h/templ"

// The components Basecoat makes with CSS alone: typed from a table of their
// classes and variants (docs/coverage.md, markup css). Each is one element
// the app may give any attribute through Attrs: handlers with surf.On, a
// type, aria-*.

// ButtonProps are a Button's attributes.
type ButtonProps struct {
	// Variant is Primary, Secondary, Outline, Ghost, Destructive or Link.
	Variant Variant
	// Size is Medium, XSmall, Small or Large.
	Size Size
	// IconOnly makes the button a square that holds only an icon.
	IconOnly bool
	Disabled bool
	// Attrs are more attributes for the button element: its handlers,
	// surf.On(ctx, "save", surf.Click(a.Save)), and any other.
	Attrs templ.Attributes
	Class string
}

// Button is Basecoat's button, holding its children:
//
//	@ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "save", surf.Click(a.Save))}) {
//		Save
//	}
func Button(p ButtonProps) templ.Component {
	return button(p)
}

// BadgeProps are a Badge's attributes.
type BadgeProps struct {
	// Variant is Primary, Secondary, Outline, Ghost, Destructive or Link.
	Variant Variant
	Attrs   templ.Attributes
	Class   string
}

// Badge is Basecoat's badge, holding its children.
func Badge(p BadgeProps) templ.Component {
	return badge(p)
}

// CardProps are a Card's attributes.
type CardProps struct {
	Attrs templ.Attributes
	Class string
}

// Card is Basecoat's card. Its parts are plain elements in its children, as
// Basecoat's markup has them: <header> (an <h2> and a <p>), <section> and
// <footer>.
func Card(p CardProps) templ.Component {
	return card(p)
}
