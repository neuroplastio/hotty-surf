package surf

import (
	"context"

	"github.com/a-h/templ"
)

// On names an element id and registers its handlers for this render. It
// returns the attributes the element takes, spread in templ:
//
//	<li { surf.On(ctx, "row-"+t.ID, surf.Click(open), surf.Key(rowKey))... }>
//
// The id is the element's, unique on the page. On an element that isn't a
// button, a link, a summary or a field, a [Click] handler makes it report
// clicks (data-on). A render registers each id once.
func On(ctx context.Context, id string, hs ...Handler) templ.Attributes {
	panic("L0")
}

// A Handler is one of an element's handlers, for [On]. Click, Input, Check,
// Key, Focus, Outside and Hover make them.
type Handler struct {
	_ struct{}
}

// Click calls fn when the user activates the element: a click, or Enter or
// space while it has the keyboard.
func Click(fn func()) Handler {
	panic("L0")
}

// Input calls fn with all a text field holds after each edit.
func Input(fn func(value string)) Handler {
	panic("L0")
}

// Check calls fn with a check box's or a radio button's state when the user
// changes it.
func Check(fn func(checked bool)) Handler {
	panic("L0")
}

// Key offers fn the keys the surface hears while the keyboard is on the
// element or inside it; fn says whether it took the key. An element's
// handler is asked before its ancestors', so an open menu hears the arrows
// before the page does. Shortcuts for the whole page go on its outermost
// element.
func Key(fn func(key string) bool) Handler {
	panic("L0")
}

// Focus calls fn with true when the keyboard comes onto the element or into
// it, and false when it leaves (SPEC §10.1).
func Focus(fn func(in bool)) Handler {
	panic("L0")
}

// Outside calls fn when the user clicks anywhere on the surface outside the
// element: what closes a popover or a menu.
func Outside(fn func()) Handler {
	panic("L0")
}

// Hover calls fn with true when the pointer comes onto the element, and
// false when it leaves (SPEC §9.4): what moves a menu's highlighted item with
// the pointer. A style that only changes on hover is CSS's :hover, which
// needs no handler.
func Hover(fn func(over bool)) Handler {
	panic("L0")
}

// Autofocus asks for the keyboard to go to the element id once this frame
// is shown. It moves the keyboard only when this frame asks for an element
// the frame before did not, so a view may ask on every render without
// taking the keyboard back from where the user moved it.
func Autofocus(ctx context.Context, id string) {
	panic("L0")
}
