package conformance

// TabsExample draws the panel chosen by a model-owned value.
type TabsExample struct {
	Value    string
	Disabled bool
}

func (c *TabsExample) Pick(value string) { c.Value = value }
