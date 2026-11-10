package surf

import "strings"

// The event's names: $value, the event's value, and $event, the event,
// which an @ handler's arguments may be (docs/spec.md, "Handlers"); and a
// drag's, which a drag's handlers' may be (@dragstart, @dragend, @drop):
// $drag.from, the dragged element's key, $drag.where, where in the target
// it was dropped, and $drag.cancel, whether Escape ended it. `$` is
// no character of a CEL name, so cel-lite never sees them: the template
// layer finds them in an expression's source, outside its strings and
// comments, before cel-lite parses it. Anywhere but as a whole argument
// of an @ handler's call they are a load error. There, each is taken out
// of the source and a `null` of the same length put in its place, which
// cel-lite parses as the call's argument; the handler knows that
// argument by its place in the source, never by a name, so no name a
// template may have (a field, a prop, a :each's local, a Lua key) can be
// taken for one, and every error's column stays where it was.

// dollar is a $name in an expression's source, outside its strings and
// comments.
type dollar struct {
	off  int    // the $'s byte offset
	name string // the name after it: value for $value; "" for none
	// whole says it is an argument of the outermost call by itself:
	// inside that call's parentheses and no other bracket, between a (
	// or a , and a , or a ).
	whole bool
}

// dollars are the $names in src, in order. It reads src as cel-lite's
// lexer does: a string, raw or not, triple-quoted or not, with a b or r
// prefix, and a // comment hold no name. Where a string doesn't end,
// cel-lite's error is due, and it stops.
func dollars(src string) []dollar {
	var out []dollar
	var open []byte // the brackets open: ( [ {
	var prev byte   // the last byte before here that isn't space
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f':
			i++
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		case c == '"' || c == '\'':
			end, ok := stringEnd(src, i)
			if !ok {
				return out
			}
			i = end
		case letter(c) || c == '_':
			if n := stringPrefix(src[i:]); n > 0 {
				end, ok := stringEnd(src, i+n)
				if !ok {
					return out
				}
				i = end
				break
			}
			for i < len(src) && (letter(src[i]) || digit(src[i]) || src[i] == '_') {
				i++
			}
		case c == '$':
			j := i + 1
			for j < len(src) && (letter(src[j]) || digit(src[j]) || src[j] == '_') {
				j++
			}
			if src[i+1:j] == "drag" && j < len(src) && src[j] == '.' {
				// $drag's member: $drag.from is one name.
				j++
				for j < len(src) && (letter(src[j]) || digit(src[j]) || src[j] == '_') {
					j++
				}
			}
			next := nextByte(src, j)
			out = append(out, dollar{off: i, name: src[i+1 : j],
				whole: len(open) == 1 && open[0] == '(' && (prev == '(' || prev == ',') && (next == ',' || next == ')')})
			i = j
		case c == '(' || c == '[' || c == '{':
			open = append(open, c)
			i++
		case c == ')' || c == ']' || c == '}':
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
			i++
		default:
			i++
		}
		prev = c
	}
	return out
}

// nextByte is the first byte at or after i that is no space and in no
// comment, 0 at the end.
func nextByte(src string, i int) byte {
	for i < len(src) {
		switch c := src[i]; {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f':
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		default:
			return c
		}
	}
	return 0
}

// stringPrefix is the length of a string literal's prefix at the start
// of s (b, r, br, any case), 0 where no quote follows one: cel-lite's.
func stringPrefix(s string) int {
	n := 0
	if n < len(s) && (s[n] == 'b' || s[n] == 'B') {
		n++
	}
	if n < len(s) && (s[n] == 'r' || s[n] == 'R') {
		n++
	}
	if n > 0 && n < len(s) && (s[n] == '"' || s[n] == '\'') {
		return n
	}
	return 0
}

// stringEnd is the offset just past the string literal whose quote is at
// src[q], after any prefix: cel-lite's. A single quote stops at a line
// break, a triple one doesn't, and a raw string reads a backslash as it
// is.
func stringEnd(src string, q int) (int, bool) {
	raw := q > 0 && (src[q-1] == 'r' || src[q-1] == 'R')
	quote := src[q]
	triple := q+2 < len(src) && src[q+1] == quote && src[q+2] == quote
	i := q + 1
	if triple {
		i = q + 3
	}
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\\' && !raw:
			i += 2
			continue
		case triple:
			if c == quote && i+2 < len(src) && src[i+1] == quote && src[i+2] == quote {
				return i + 3, true
			}
		case c == quote:
			return i + 1, true
		case c == '\n' || c == '\r':
			return 0, false
		}
		i++
	}
	return 0, false
}

func letter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func digit(c byte) bool  { return c >= '0' && c <= '9' }

// EventArg is where a handler's argument comes from: its expression,
// evaluated at render, or the event, when it runs.
type EventArg uint8

const (
	// ArgExpr is an expression, evaluated at render.
	ArgExpr EventArg = iota
	// ArgValue is $value: the event's Value, a string.
	ArgValue
	// ArgEvent is $event: the event (EventValue).
	ArgEvent
	// ArgRows is $value in a @height handler: the field's rows, an int
	// (surfrun.Event's Rows).
	ArgRows
	// ArgDragFrom is $drag.from: the dragged element's key, a string
	// (surfrun.Drag's From).
	ArgDragFrom
	// ArgDragWhere is $drag.where: where in the target the drag was
	// dropped, a string: "before", "after", "into" or "".
	ArgDragWhere
	// ArgDragCancel is $drag.cancel: whether Escape ended the drag, a
	// bool.
	ArgDragCancel
)

// eventArg is what the $name name is as a handler's argument, if it is
// one of the event's.
func eventArg(name string) (EventArg, bool) {
	switch name {
	case "value":
		return ArgValue, true
	case "event":
		return ArgEvent, true
	case "drag.from":
		return ArgDragFrom, true
	case "drag.where":
		return ArgDragWhere, true
	case "drag.cancel":
		return ArgDragCancel, true
	}
	return ArgExpr, false
}

// isDragArg reports whether k is one of a drag's names, $drag.….
func isDragArg(k EventArg) bool { return k == ArgDragFrom || k == ArgDragWhere || k == ArgDragCancel }

// eventFill is one of the event's names in a handler's source: its
// offset, and which.
type eventFill struct {
	off int
	k   EventArg
}

// eventArgs is a handler's source with its $value and $event taken out,
// each a null of its length, and each, in order; or the error at the
// first that can't be an argument there, and its offset. In a <bind>'s
// do (handler false) neither may be. $drag's names may be only where drag
// says: a drag's handler's.
func eventArgs(src string, handler, drag bool) (string, []eventFill, int, string) {
	ds := dollars(src)
	if len(ds) == 0 {
		return src, nil, 0, ""
	}
	b := []byte(src)
	var fills []eventFill
	for _, d := range ds {
		k, ok := eventArg(d.name)
		switch {
		case !ok && d.name == "":
			continue // a $ alone: cel-lite's error
		case !ok && (d.name == "drag" || strings.HasPrefix(d.name, "drag.")):
			return "", nil, d.off, "$" + d.name + " is none of $drag.from, $drag.where and $drag.cancel"
		case !ok:
			return "", nil, d.off, "$" + d.name + " is neither $value nor $event"
		case !handler:
			return "", nil, d.off, "$" + d.name + " is an @ handler's argument only"
		case isDragArg(k) && !drag:
			return "", nil, d.off, "$" + d.name + " is a drag's handler's argument only: @dragstart's, @dragend's or @drop's"
		case !d.whole:
			return "", nil, d.off, "$" + d.name + " is a whole argument, not part of an expression"
		}
		fills = append(fills, eventFill{d.off, k})
		copy(b[d.off:], "null")
		for i := d.off + 4; i < d.off+1+len(d.name); i++ {
			b[i] = ' '
		}
	}
	return string(b), fills, 0, ""
}
