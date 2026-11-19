package cells

import (
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
)

// Copy and cut in cells (for plx). The Screen has no
// clipboard: the copy and cut keys aren't its (Key leaves them to the
// program), so the program asks it for the field's selection on its own
// copy key and puts it on the terminal's clipboard, as it pastes with
// Paste.

// Selection is the text selected in the text field or textarea that has
// the keyboard: what the program copies on its copy key. It is "" where
// nothing is selected, no text field has the keyboard, or the field is a
// password's, which a browser won't copy either.
func (c *Screen) Selection() string {
	n := c.focused
	if c.body == nil || !c.active || !isTextField(n) || isPassword(n) {
		return ""
	}
	f := c.field(n)
	lo, hi := f.ed.Selection()
	if lo >= hi {
		return ""
	}
	return strings.Join(graphemesOf(f.ed.Value)[lo:hi], "")
}

// Cut is Selection, deleted from the field as the user's edit (its input
// where it listens for one, its change when it commits): what the program
// does on its cut key. It is "", and the field is as it was, in a field
// that is readonly or disabled, or a password's. Lines changes with it:
// the program draws the Screen again.
func (c *Screen) Cut() string {
	n := c.focused
	text := c.Selection()
	if text == "" || has(n, "readonly") || disabled(n) {
		return ""
	}
	f := c.field(n)
	if f.ed.Do(hotty.DeleteCharBackward) {
		c.edited(n, f)
	}
	c.lay = nil
	c.flush()
	return text
}

// isPassword is a password field.
func isPassword(n *html.Node) bool {
	return strings.EqualFold(attr(n, "type"), "password")
}
