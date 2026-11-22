// Package surfui is hotty-surf's shared UI foundation: design tokens and base
// CSS (kit.css), the CSS/asset Registry, shared Variant/Size/Mode types,
// keyboard bindings and key scopes, and Kitty detection.
//
// Individual components live in self-contained subpackages under
// surfui/<component> (for example surfui/button, surfui/card, surfui/select,
// surfui/dialog), each with its own templ implementation, optional embedded
// CSS, explicit Register(*surfui.Registry), and storybook showcase in
// surfui/<component>/story. Component-authoring utilities live in surfui/util,
// and internal helpers live under surfui/internal/<pkg>.
//
// Every component follows one pattern:
//
//   - A component is a templ function in surfui/<component>, typically
//     View(Props) or a named variant, with content in templ's children block:
//     @button.View(button.Props{…}) { Save }.
//   - Its attributes are one struct, Props, whose zero value is the default.
//     Every Props has Class for extra classes on the outermost element.
//   - The app's data stays the app's: values come in through Props, and
//     changes go out through On… callbacks or surf.On handlers in Attrs.
//   - Ephemeral UI state (whether a popup is open, highlighted item) lives in
//     State, zero-value ready, kept in the app's state and passed by pointer.
//   - Variants and sizes are shared through surfui.Variant and surfui.Size.
package surfui
