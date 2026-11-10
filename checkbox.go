package surf

import "neuroplast.io/hotty-surf/surfui"

// NewCheckbox builds a template's checkbox; its checked modifier sets
// the model's state after construction, in both modes.
func NewCheckbox() *surfui.Node { return surfui.Checkbox(false) }
