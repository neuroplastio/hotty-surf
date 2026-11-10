package surfrun

import "neuroplast.io/hotty-surf/surfui"

// A Field with an OnChange commits what the user typed (surfui.Commits): on
// Enter, which the field gives the program, and when the user leaves it,
// which the host reports as change (SPEC §9: Tab, a click elsewhere, the
// surface losing the keyboard). Either way the surface runs the OnChange
// once for each value: a change after Enter committed the same value
// runs nothing.
//
// Such a field reports its edits, so that Enter commits what it holds.
// They are not what the model shows (its Value) until a commit: the
// field keeps what the user types, and after a commit the model's value
// is the field's, as after a select's pick. A model that does not take
// the value has it put back (a=field), as a refusal. A value the model
// puts in the field (a refusal's, or "" after a commit) is what the
// field holds then: Enter commits nothing until the user types again.

// fieldEdit hears an input or a change of a committing field, and
// reports whether the event is done with: an edit only the surface
// wanted, or a change, which commits.
func (s *Surface) fieldEdit(n *surfui.Node, kind, id string, h heard) bool {
	s.edits[id] = h
	s.due = true
	if kind == "change" {
		s.commit(n, id, h)
		return true
	}
	if _, ok := surfui.Handler(n, "input"); ok {
		s.heard[id] = h
		return false
	}
	return true
}

// fieldEnter commits the focused field on Enter, if it commits and the
// user has typed in it, and reports whether it did. The key goes on to
// the bindings either way.
func (s *Surface) fieldEnter(key string) bool {
	if key != "enter" || !s.shown || s.focused == "" {
		return false
	}
	n := s.index[s.focused]
	if n == nil || !surfui.Commits(n) || surfui.Tag(n) == "textarea" {
		return false // Enter is a newline in a field of several lines
	}
	h, ok := s.edits[s.focused]
	if ok {
		s.commit(n, s.focused, h)
	}
	return ok
}

// commit runs n's OnChange with what the user committed, unless that is
// the value the field had when they began: as they last committed it, or
// as the program last put it there.
func (s *Surface) commit(n *surfui.Node, id string, h heard) {
	s.heard[id] = h
	s.due = true
	if b, ok := s.base[id]; ok && b == h.value {
		return
	}
	s.base[id] = h.value
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return
	}
	if act, ok := surfui.Handler(n, "change"); ok {
		s.do(act, Event{Kind: "change", Target: id, Value: h.value})
	}
}
