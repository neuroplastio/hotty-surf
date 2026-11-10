package surfrun

import (
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// KeyUse is a key the surface's Key would take (Surface.Keys): Name as
// Key takes it, the binding's Title ("" for a key of the focused
// element's own), and Own for a key the focused element takes itself (a
// field's Enter, which commits it and goes on to a binding of Enter; a
// Select's keys, its open list's, a Cycle's; the Escape that blurs). It
// is cells.KeyUse, which a renderer's Keys lists too.
type KeyUse = cells.KeyUse

// Chars is what the focused element does with a key that types a
// character, with Shift at most: "a", "shift+a", "1", ">" (Surface.Keys).
// It is cells.Chars.
type Chars = cells.Chars

const (
	// CharsBound is no element that takes characters: such a key goes to
	// the bindings, as any key does.
	CharsBound = cells.CharsBound
	// CharsPick is a Select, its open list or a Cycle, which takes each
	// such key ahead of the bindings, and goes to the next option that
	// starts with it.
	CharsPick = cells.CharsPick
	// CharsText is a text field, which types them, ahead of the bindings
	// (SPEC §10.2): a host keeps them for it, and Key never hears one.
	CharsText = cells.CharsText
)

// Keys are the keys Key would take now, in the order it tries them, and
// what the focused element does with characters: for a host that decides
// a key without calling Key (a multiplexer, whose program is in another
// process), from what it last heard. A key that is in neither, Key
// refuses. They are good after Show, Update, Event, ListEvent, Key or
// Lend, as a Frame is, and until the model changes off them (Post).
//
// First, while a drag is on, the Escape that ends it (Dragging); then
// the focused element's own keys, then the bindings from it up
// (or from the keys' home, with none focused), each key once, as the
// first binding of it has it; last the Escape that blurs. A field's Enter
// goes on to a binding of Enter, which is listed after it. A key the
// first binding gives the program (Program) is none of them.
func (s *Surface) Keys() ([]KeyUse, Chars) {
	s.fresh()
	var keys []KeyUse
	chars := CharsBound
	stops := map[string]bool{}
	own := func(names []string) {
		for _, k := range names {
			if !stops[k] {
				stops[k] = true
				keys = append(keys, KeyUse{Name: k, Own: true})
			}
		}
	}
	if s.Dragging() {
		stops["escape"] = true
		keys = append(keys, KeyUse{Name: "escape", Title: "Cancel the drag", Own: true}) // cancelDrag
	}
	if n := s.index[s.focused]; s.shown && n != nil {
		if surfui.IsTextField(n) {
			chars = CharsText
		}
		if _, typed := s.edits[s.focused]; typed && surfui.Commits(n) {
			keys = append(keys, KeyUse{Name: "enter", Own: true}) // fieldEnter
		}
	}
	if s.shown && s.menu != nil {
		// An open menu holds the keyboard: Key takes every key, and
		// ignores those it has no use for. Only these do something.
		own(menuKeys)
		return keys, CharsPick
	}
	if s.shown && s.list != nil {
		own(listKeys)
		chars = CharsPick
	} else if s.focusedSelect() != nil {
		own(selectKeys)
		chars = CharsPick
	}
	if s.shown && s.menu == nil {
		if _, ok := s.Menu(MenuAt{Focus: true}); ok {
			own([]string{"shift+f10", "context_menu"}) // menuKey
		}
	}
	if s.focusedCycle() != nil {
		own(cycleKeys)
		chars = CharsPick
	}
	if s.focusedSlider() != nil {
		own(sliderKeys)
	}
	s.eachBinding(func(k string, b surfui.Binding) bool {
		if !stops[k] && !(chars == CharsPick && isCharacter(k)) {
			stops[k] = true
			if !surfui.Gives(b) {
				keys = append(keys, KeyUse{Name: k, Title: b.Title})
			}
		}
		return true
	})
	if s.blurs() {
		own([]string{"escape"})
	}
	return keys, chars
}

// isCharacter reports whether key types a character (character).
func isCharacter(key string) bool {
	_, ok := character(key)
	return ok
}
