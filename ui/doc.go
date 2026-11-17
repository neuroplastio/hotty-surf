// Package ui is every Basecoat 1.1.0 component as a typed Go API for templ:
// shadcn's look, in Basecoat's classes, with Basecoat's behaviour scripts
// translated to Go. An app imports this package and no other for its
// components.
//
// Every component follows one pattern:
//
//   - A component is a function returning a templ.Component, named as
//     Basecoat names it: Button, DropdownMenu, Select. Content goes in
//     templ's children block: @ui.Button(ui.ButtonProps{…}) { Save }.
//   - Its attributes are one struct, <Name>Props, whose zero value is
//     Basecoat's default. Every Props has Class, for Tailwind classes added
//     to the component's outermost element.
//   - A component made from CSS alone is one element, and its Props has
//     Attrs for any attribute of it: its handlers, through surf.On, as on
//     any element. A component made from a Jinja macro has ID, which names
//     its inner elements, and reports its own events through On…
//     callbacks.
//   - The app's data stays the app's: the value a component shows comes in
//     through Props, and a change goes out through an On… callback in Props
//     (a component with behaviour of its own), or through surf.Input or
//     surf.Check in Attrs (a native control: Input, Textarea, NativeSelect,
//     Checkbox, Radio, Switch), which the app applies to its own state.
//   - What only the component needs to remember (whether a menu is open, the
//     item the arrows highlighted) is in <Name>State. The
//     app keeps one per component in its state, zero value ready, and passes
//     a pointer: @ui.Select(&a.size, ui.SelectProps{…}).
//   - Items are plain data: SelectItem, Tab, MenuItem.
//   - Variants and sizes are shared, as Basecoat's data-variant and
//     data-size are: Variant and Size. A value only one component has is
//     that component's own field. Sides, alignments and orientations are
//     strings, their values in the field's doc. surftest fails a test that
//     renders a value the component doesn't take.
//   - Basecoat's parts that are plain elements stay plain elements: a card's
//     <header>, <section> and <footer> are written in the card's children,
//     as Basecoat's markup has them.
//
// Basecoat's coverage, component by component, is docs/coverage.md. This
// package's L0 shape holds a few components, one of each kind (written from
// Basecoat's CSS, generated from its Jinja macros, with a translated
// behaviour); the rest follow the pattern, each batch in its own API review.
package ui
