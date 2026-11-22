package keydefault

// Check returns a check control's next state and whether it changed.
// A selected radio cannot be unchecked by activating it again.
func Check(checked, radio bool) (next, changed bool) {
	if checked && radio {
		return true, false
	}
	return !checked, true
}
