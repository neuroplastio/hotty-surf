package conformance

// CheckboxExample holds a model-owned boolean in both template modes:
// one a click changes, one a change event says.
type CheckboxExample struct {
	Enabled  bool
	Disabled bool
	Marked   bool
	Heard    []bool
}

func (c *CheckboxExample) Toggle() { c.Enabled = !c.Enabled }

// Mark is what a checkbox's change says: whether it is now checked.
func (c *CheckboxExample) Mark(checked bool) {
	c.Marked = checked
	c.Heard = append(c.Heard, checked)
}
