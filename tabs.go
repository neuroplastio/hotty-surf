package surf

import "neuroplast.io/hotty-surf/surfui"

// NewTabs builds a template's row of options, value selected.
func NewTabs(value string, options []string) *surfui.Node { return surfui.Tabs(value, options...) }
