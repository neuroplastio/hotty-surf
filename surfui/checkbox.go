package surfui

// Checkbox is a boolean drawn as a square with a check. The model owns
// checked: its OnClick changes that state, as a Switch's does. The
// button supplies the same click for a pointer, Space and Enter. Give
// it a Label; a Text beside it can show that label on screen.
func Checkbox(checked bool) *Node {
	return El("button", Icon("\U000F012C").Class("u-check-mark").Attr("aria-hidden", "true")).
		Class("u-checkbox").Attr("type", "button").Attr("role", "checkbox").Checked(checked)
}

// CheckboxOf reports whether n is a Checkbox, and whether it is checked
// and enabled. The runtime drives its change: a click says what it is
// now (not checked, if it was).
func CheckboxOf(n *Node) (checked, enabled, ok bool) {
	if n == nil || !hasClass(n, "u-checkbox") {
		return false, false, false
	}
	_, off := AttrOf(n, "disabled")
	return hasClass(n, "is-on"), !off, true
}
