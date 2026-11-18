package surf

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

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
		case kindClick, kindClicks:
			on = append(on, "click")
		case kindInput:
			on = append(on, "input")
		case kindDrag:
			on = append(on, "drag")
		case kindDrop:
			on = append(on, "drop")
		case kindSecondary:
			on = append(on, "secondary")
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
	e := &rt.Handlers{Scope: rt.KeyScopeFrom(ctx)}
	clicks := 0
	for _, h := range hs {
		switch h.kind {
		case kindClick, kindClicks:
			if clicks++; clicks > 1 {
				f.Faults = append(f.Faults, "surf.On: the id "+id+" has two click handlers (Click or Clicks): give it one")
				continue
			}
			if h.kind == kindClick {
				e.Click = h.fn.(func())
				break
			}
			fn := h.fn.(func(int))
			e.Click = func() { fn(max(f.Count, 1)) }
		case kindInput:
			e.Input = h.fn.(func(string))
		case kindCheck:
			e.Check = h.fn.(func(bool))
		case kindKey:
			e.Key = append(e.Key, h.fn.(func(string) bool))
			for _, k := range h.keys {
				c, ok := f.Platform.ReadKey(k)
				if !ok || c == "shift+tab" {
					f.Faults = append(f.Faults, "surf.Key: the id "+id+" declares the key "+strconv.Quote(k)+", which doesn't parse or isn't the program's to take")
					continue
				}
				if !slices.Contains(e.Keys, c) {
					e.Keys = append(e.Keys, c)
				}
			}
		case kindFocus:
			e.Focus = h.fn.(func(bool))
		case kindOutside:
			e.Outside = h.fn.(func())
		case kindHover:
			e.Hover = h.fn.(func(bool))
		case kindSubmit:
			e.Submit = h.fn.(func())
		case kindDrag:
			e.Drag = h.fn
		case kindDrop:
			e.Drop = h.fn
		case kindSecondary:
			e.Secondary = h.fn.(func(hotty.Cell, hotty.Area))
		}
	}
	f.On[id] = e
	f.Order = append(f.Order, id)
	return a
}

// A Handler is one of an element's handlers, for [On]. Click, Clicks,
// Input, Check, Key, Focus, Outside, Hover, Submit, Drag, Drop and
// Secondary make them.
type Handler struct {
	kind kind
	fn   any      // the func the kind takes; nil is no handler
	keys []string // Key's keys, as the app wrote them
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
	kindSubmit
	kindDrag
	kindDrop
	kindSecondary
	kindClicks
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

// Clicks is Click with the click's count: fn hears 1 for a click, 2 for the
// second click of a double click, 3 for a triple click's third (SPEC §9,
// count), as the host or a cells.Screen counts them, and 1 for Enter or
// space. A double click is a click then a click with count 2, each with
// its own press and release, so a Drag on the same element hears them as
// it hears any click, and fn hears both: a row that selects on a click
// and opens on a double click does each in turn. An element takes Click
// or Clicks, not both: two are a fault.
func Clicks(fn func(count int)) Handler {
	return handler(kindClicks, fn, fn == nil)
}

// Refocus asks for the keyboard to go to the element id in this frame, as
// Autofocus does, but whether or not the frame before asked for it: what
// a view uses to take the keyboard back after the user moved it, as a
// palette's query does when Tab leaves it for a stop of its own. A render
// that asks for the element that already has the keyboard sends nothing.
// A handler decides it and the next render asks once: the handler sets a
// flag, and the view asks Refocus and clears the flag (see the example).
// Asked on every render, it would take the keyboard back from wherever
// the user moves it, within the surface or by a click away. id "" is a
// fault.
func Refocus(ctx context.Context, id string) {
	if id == "" {
		rt.Fault(ctx, `surf.Refocus: id "" (ask for an element; nothing focused is not the program's to ask for)`)
		return
	}
	if f := rt.From(ctx); f != nil {
		f.Focus, f.Refocus = id, true
	}
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
// before the page does. Key is for a control's own keyboard (a list's
// arrows, a grid, a field's completion); a shortcut for the whole page,
// which a menu and a palette list, is a ui.KeyScope's binding.
//
// keys are the keys fn takes ahead of the element: while the keyboard is
// on the element or inside it, the host gives each to the program rather
// than using it itself (a field's edit, a button's Space, the surface's
// scroll), so fn hears it. Without keys, fn hears only the keys that
// reach the program anyway: those the element doesn't use. The keys of
// every Key handler on one element are its declaration together. A view
// whose handler takes keys on a condition (a menu open or closed) passes
// that condition's keys, render by render.
//   - They are in fn's notation ("arrow_down", "mod+s"), read for the
//     host's platform, and arrive as fn hears every key: a declared
//     "ctrl+n" arrives as "mod+n" where ctrl is the mod.
//   - A key fn declines goes to the enclosing handlers, those that
//     declared it among them, then to the element's own use, which the
//     surface does as the host would have: a button's activation, a
//     field's edit, a check box's toggle, a select's pick. A use it can't
//     do (the host's scrolling, a date field's) is lost; so is a declined
//     tab on a field, which doesn't move focus.
//   - A printable key ("j", "shift+g") is never taken from an element
//     inside that types: it types there, as under a ui.KeyScope.
//   - escape reaches the program anyway, so a host ignores it in the
//     declaration; it is declared for Surface.OwnKeys, which lists it.
//   - tab is only a text field's own to declare (a completion): on any
//     other element, it would take Tab from every field inside, and it
//     is a fault. shift+tab, and a key that doesn't parse, are faults.
//
// The surface writes the declaration as the element's data-keys (SPEC
// §10.2, key=program): a template that writes data-keys itself is a
// fault, and the attribute is dropped.
//
// A ui.KeyScope naming a key is offered it before the focused kit
// control's default and ordinary Key handlers. A declined key then goes
// to the kit default before it reaches enclosing Key handlers.
//
// Without a scoped override or a declaration, a text field keeps the keys
// it edits with, Enter among them: Enter
// commits the field and submits its form, which the form's [Submit] hears,
// so no Key handler hears it.
//
// fn sees hotty's names ("enter", "arrow_down", "shift+tab"), with one
// change: a key whose modifier is the host's mod (SPEC §10.4: ctrl, or super
// on macOS) arrives as "mod+…". So an app matches "mod+s", and is the same
// on every platform; one that compares "ctrl+s" misses it where ctrl is the
// mod. It is the notation ui.Kbd and ui.MenuItem.Keys show.
func Key(fn func(key string) bool, keys ...string) Handler {
	h := handler(kindKey, fn, fn == nil)
	if fn != nil {
		h.keys = keys
	}
	return h
}

// Focus calls fn with true when the user moves the keyboard onto the
// element or into it, and false when it leaves (SPEC §10.1). It hears the
// user's moves, the program's Surface.Focus among them (the user's Tab
// between panes, which the program carries), but not the focus a view
// asks for with Autofocus, which the host doesn't report: an element
// Autofocus put the keyboard on hears false when it leaves without having
// heard true.
func Focus(fn func(in bool)) Handler {
	return handler(kindFocus, fn, fn == nil)
}

// Submit calls fn when the user submits the element, a form: Enter in one
// of its text fields, after that field's commit (its Input, when the value
// changed), or a click of its submit button (a button with no type or
// type="submit"; ui.Button is a submit button with
// Attrs: templ.Attributes{"type": "submit"}). It is how an app acts on
// Enter in a field: a field outside a form only commits on Enter, and no
// Key handler hears it. A submit button's own Click runs first, so the
// action goes on the form alone, and Enter and the button do the same. The
// form's values are the app's already, through its fields' Input and
// Check, so fn takes none. On an element that isn't a form it is a fault.
func Submit(fn func()) Handler {
	return handler(kindSubmit, fn, fn == nil)
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

// A DragEvent is where a drag is (SPEC §9.1), as Drag's and Drop's
// handlers hear it.
type DragEvent struct {
	// Start is true for the press that starts the drag, and End for its
	// release, or for the host cutting it short (the surface hidden, a new
	// document); both are false for each move between them.
	Start, End bool
	// ID is the element dragged: the one whose On has Drag.
	ID string
	// Over is the element under the pointer: the nearest one, from it
	// outward, whose On has Drag or Drop. It is ID itself at the Start,
	// and "" over nothing that takes a drag (a gap, outside the surface)
	// and at an End the host cut short.
	Over string
	// Drag is where the pointer is, as hotty-go reads the host's detail:
	// its cell (Col, Row, counted from the surface's top left and going
	// on past its edges), its step in ID's box (X, Y) and its step in
	// Over's (TX, TY), each with whether the host said it (HasX, …). Steps
	// are counted only for an element with data-steps (below, Drag).
	hotty.Drag
}

// Drag makes the element draggable (data-on drag) and calls fn as the
// user drags it, with the mouse's or a pen's primary button, or a touch
// on an element whose touch-action allows no pan (SPEC §9.1): once at the
// press (Start), each time the pointer comes onto another element that
// takes a drag, or another step (below), and once at the release (End).
// A drag costs a few calls for each element crossed, not one for every
// move. fn decides what the drag means: a list reorders on End by Over
// and TY, a slider sets its value from X on every call.
//
// Steps: an element that writes data-steps="<x>" or data-steps="<x> <y>"
// beside the attributes On gives it divides its box into that many steps
// across and down, and each call carries the pointer's step in it (X, Y),
// wherever the pointer is; "0" counts none along that axis. So a slider
// from 0 to 20 has data-steps="20", and a row that a drop lands before or
// after has data-steps="0 1" (with Drop, which is what carries TX, TY).
//
// While the pointer is moving the element, from the first move after the
// press to the release, the surface adds the class dragging to it in
// what it sends (the view doesn't write it), so a stylesheet shows it
// lifted (.dragging { opacity: .5 }); a press and release with no move
// in between leaves it off. A release on the element the drag began on is
// also its click, after End, which Click hears. A press with Alt held is
// the program's and starts no drag (SPEC §9.2).
// Escape is not the host's: an app that cancels a drag by key does so
// in a Key handler, then ignores that drag's remaining calls, its End
// and the Drop's, while the element keeps the class dragging until the
// release.
//
// A drag needs a pointer: in cells it is the terminal's mouse, which the
// program hands the Screen (cells.Screen.Press, Move, Release), and from
// the keyboard there is none, so an app that reorders by drag also gives
// the keyboard a way (a Key handler that moves the item).
func Drag(fn func(d DragEvent)) Handler {
	return handler(kindDrag, fn, fn == nil)
}

// Drop makes the element a place to drop on (data-on drop): the host names
// it as a drag's Over while the pointer is over it, though a press on it
// drags nothing, and with data-steps it says where in it the pointer is
// (TX, TY).
//
// fn is called once, at the End of a drag released while the element
// is the drag's Over, after the dragged element's Drag has heard that
// End, with the same DragEvent (ID is what was dragged, Over this
// element). Over is the nearest element that takes a drag, so a Drop
// on a list whose rows have Drag or Drop hears only releases outside
// them; and a release on the dragged element itself calls its own
// Drop, if it has one, with Over == ID. What the drag was over before
// the End, the dragged element's Drag handler hears. An element may
// have both Drag and Drop: the rows of a list that reorders are dragged
// and dropped on.
func Drop(fn func(d DragEvent)) Handler {
	return handler(kindDrop, fn, fn == nil)
}

// Secondary calls fn when the user makes a secondary press on the element
// or inside it (data-on secondary): a mouse's secondary button, Control
// and the primary button on macOS, a pen's barrel button, or a long press
// of a touch, as the user's platform makes one (SPEC §9). at is the cell
// the pointer was on, from the surface's top left, where a context menu
// opens; area is the element's cells. Either is zero when the host
// doesn't say it (surftest's host lays nothing out). The press does
// nothing else: no click, no focus, no drag, and the terminal's own menu
// doesn't open. It tells Outside handlers first, as a primary press
// does, so a menu opened elsewhere closes before fn runs. Shift with it makes it
// the terminal's, and Alt the program's (SPEC §9.2), and a host with no
// secondary gesture sends none, so whatever it opens can also be opened
// another way (a key, a button).
func Secondary(fn func(at hotty.Cell, area hotty.Area)) Handler {
	return handler(kindSecondary, fn, fn == nil)
}

// Autofocus asks for the keyboard to go to the element id once this frame
// is shown. It moves the keyboard only when this frame asks for an element
// the frame before did not, so a view may ask on every render without
// taking the keyboard back from where the user moved it. id "" is a
// fault: a program asks for an element, never for nothing focused (which
// is SPEC's to define, not the program's to ask for).
func Autofocus(ctx context.Context, id string) {
	if id == "" {
		rt.Fault(ctx, `surf.Autofocus: id "" (ask for an element; nothing focused is not the program's to ask for)`)
		return
	}
	if f := rt.From(ctx); f != nil {
		f.Focus = id
	}
}
