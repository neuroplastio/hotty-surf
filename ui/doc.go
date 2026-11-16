// Package ui is every Basecoat 1.1.0 component as a typed Go API for templ:
// shadcn's look, in Basecoat's classes, with Basecoat's behaviour scripts
// translated to Go. An app imports this package and no other for its
// components.
//
// Every component follows one pattern:
//
//   - A component is a function returning a templ.Component, named as
//     Basecoat names it: Button, DropdownMenu, NativeSelect. Content goes in
//     templ's children block: @ui.Button(ui.ButtonProps{…}) { Save }.
//   - Its attributes are one struct, <Name>Props, whose zero value is
//     Basecoat's default. Every Props has Class, for Tailwind classes added
//     to the component's outermost element, and a component that reports
//     anything has ID, the id its elements are named from (see surf.On).
//   - The app's data stays the app's: the value a component shows comes in
//     through Props, and a change goes out through an On… callback in Props,
//     which the app applies to its own state.
//   - What only the component needs to remember (whether a menu is open, the
//     item the arrows highlighted, a select's filter) is in <Name>State. The
//     app keeps one per component in its state, zero value ready, and passes
//     a pointer: @ui.Select(&a.size, ui.SelectProps{…}).
//   - Items are plain data: SelectItem, Tab, MenuItem.
//   - Variants and sizes are shared, not per component: Variant and Size.
//   - Basecoat's parts that are plain elements stay plain elements: a card's
//     <header>, <section> and <footer> are written in the card's children,
//     as Basecoat's markup has them.
//
// Basecoat's coverage, component by component, is docs/coverage.md. This
// package's L0 shape holds a few components, one of each kind (written from
// Basecoat's CSS, generated from its Jinja macros, with a translated
// behaviour); the rest follow the pattern, each batch in its own API review.
package ui
