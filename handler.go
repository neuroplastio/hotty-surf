package surf

import (
	"context"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/rt"
)

// On names an element id and registers its handlers for this render. It
// returns the attributes the element takes, spread in templ:
//
//	<li { surf.On(ctx, "row-"+t.ID, surf.Click(open), surf.Key(rowKey))... }>
//
// The id is the element's, unique on the page. On an element that isn't a
// button, a link, a summary or a field, a [Click] handler makes it report
// clicks (data-on).
//
// A render registers each id once. A second On for an id in the same render
// fails a surftest test; elsewhere the first one's handlers stay and the
// second's are dropped. A list gives each item an id of its own
// ("row-"+item.ID).
func On(ctx context.Context, id string, hs ...Handler) templ.Attributes {
	a := templ.Attributes{"id": id}
	var on []string
	for _, h := range hs {
		switch h.kind {
		case kindClick:
			on = append(on, "click")
		case kindInput:
			on = append(on, "input")
		}
	}
	if len(on) > 0 {
		a["data-on"] = strings.Join(on, " ")
	}
	f := rt.From(ctx)
	if f == nil {
		// Rendered outside a surface (a static render, a test of a
		// component): the handlers go nowhere.
		return a
	}
	if f.On[id] != nil {
		f.Faults = append(f.Faults, "surf.On: the id "+id+" is registered twice in one render")
		return a
	}
	e := &rt.Handlers{}
	for _, h := range hs {
		switch h.kind {
		case kindClick:
			e.Click = h.fn.(func())
		case kindInput:
			e.Input = h.fn.(func(string))
		case kindCheck:
			e.Check = h.fn.(func(bool))
		case kindKey:
			e.Key = append(e.Key, h.fn.(func(string) bool))
		case kindFocus:
			e.Focus = h.fn.(func(bool))
		case kindOutside:
			e.Outside = h.fn.(func())
		case kindHover:
			e.Hover = h.fn.(func(bool))
		}
	}
	f.On[id] = e
	f.Order = append(f.Order, id)
	return a
}

// A Handler is one of an element's handlers, for [On]. Click, Input, Check,
// Key, Focus, Outside and Hover make them.
type Handler struct {
	kind kind
	fn   any // the func the kind takes; nil is no handler
}

type kind uint8

const (
	kindNone kind = iota
	kindClick
	kindInput
	kindCheck
	kindKey
	kindFocus
	kindOutside
	kindHover
)

// handler is fn as a Handler of kind k, or no handler when absent (a nil
// callback, which a view may pass through), so that it adds nothing.
func handler(k kind, fn any, absent bool) Handler {
	if absent {
		return Handler{}
	}
	return Handler{kind: k, fn: fn}
}

// Click calls fn when the user activates the element: a click, or Enter or
// space while it has the keyboard.
func Click(fn func()) Handler {
	return handler(kindClick, fn, fn == nil)
}

// Input calls fn with all a text field holds after each edit, and with the
// value of a select's chosen option.
func Input(fn func(value string)) Handler {
	return handler(kindInput, fn, fn == nil)
}

// Check calls fn with a check box's or a radio button's state when the user
// changes it.
func Check(fn func(checked bool)) Handler {
	return handler(kindCheck, fn, fn == nil)
}

// Key offers fn the keys the surface hears while the keyboard is on the
// element or inside it; fn says whether it took the key. An element's
// handler is asked before its ancestors', so an open menu hears the arrows
// before the page does. Shortcuts for the whole page go on its outermost
// element.
//
// fn sees hotty's names ("enter", "arrow_down", "shift+tab"), with one
// change: a key whose modifier is the host's mod (SPEC §10.4: ctrl, or super
// on macOS) arrives as "mod+…". So an app matches "mod+s", and is the same
// on every platform; one that compares "ctrl+s" misses it where ctrl is the
// mod. It is the notation ui.Kbd and ui.MenuItem.Keys show.
func Key(fn func(key string) bool) Handler {
	return handler(kindKey, fn, fn == nil)
}

// Focus calls fn with true when the user moves the keyboard onto the
// element or into it, and false when it leaves (SPEC §10.1). It hears the
// user's moves only: the host reports no focus the program gives
// (Autofocus), so an element Autofocus put the keyboard on hears false
// when it leaves without having heard true.
func Focus(fn func(in bool)) Handler {
	return handler(kindFocus, fn, fn == nil)
}

// Outside calls fn when the user clicks anywhere on the surface outside the
// element, and when the surface loses the keyboard (a blur: a click on the
// program's own cells or on another surface): what closes a popover or a
// menu. It needs the surface placed with Press on (see Surface.Attach).
func Outside(fn func()) Handler {
	return handler(kindOutside, fn, fn == nil)
}

// Hover calls fn with true when the pointer comes onto the element, and
// false when it leaves (SPEC §9.4; it needs the surface placed with Hover
// on, see Surface.Attach): what moves a menu's highlighted item with
// the pointer. A style that only changes on hover is CSS's :hover, which
// needs no handler.
func Hover(fn func(over bool)) Handler {
	return handler(kindHover, fn, fn == nil)
}

// Autofocus asks for the keyboard to go to the element id once this frame
// is shown. It moves the keyboard only when this frame asks for an element
// the frame before did not, so a view may ask on every render without
// taking the keyboard back from where the user moved it.
func Autofocus(ctx context.Context, id string) {
	if f := rt.From(ctx); f != nil {
		f.Focus = id
	}
}
