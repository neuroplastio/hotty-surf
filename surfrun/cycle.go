package surfrun

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"neuroplast.io/hotty-surf/surfui"
)

// A Cycle (surfui.Cycle) is no host control: the surface drives it. Its keys
// reach the program while it has the keyboard (its data-keys give the
// program those a host would scroll with), and its parts report their
// clicks. Either way the surface works out the next value and runs the
// Cycle's OnChange with it, as a Select's pick runs its own.

// cycleKey steps the Cycle that has the keyboard by key, and reports
// whether the Cycle took key: the arrows, Home, End, Space, Enter and a
// character, as the Cycle keeps what it uses, as a Select does.
func (s *Surface) cycleKey(key string) bool {
	c := s.focusedCycle()
	if c == nil {
		return false
	}
	_, value, options, _, _ := surfui.CycleOf(c)
	next, took := cycleKeyed(value, options, key)
	if took {
		s.cycleTo(c, next)
	}
	return took
}

// focusedCycle is the Cycle with the keyboard, where it takes keys: it
// has options, and is not disabled; nil for none.
func (s *Surface) focusedCycle() *surfui.Node {
	if !s.shown || s.focused == "" {
		return nil
	}
	n := s.index[s.focused]
	if n == nil {
		return nil
	}
	c, _, options, _, ok := surfui.CycleOf(n)
	if !ok || len(options) == 0 {
		return nil
	}
	if _, off := surfui.AttrOf(c, "disabled"); off {
		return nil
	}
	return c
}

// cycleKeys are the keys a Cycle takes besides a character (cycleKeyed).
var cycleKeys = []string{"arrow_left", "arrow_up", "arrow_right", "arrow_down", "space", "enter", "home", "end"}

// cycleTo runs c's OnChange with value, unless it is c's value already
// or c is disabled.
func (s *Surface) cycleTo(c *surfui.Node, value string) {
	_, now, _, _, ok := surfui.CycleOf(c)
	if !ok || now == value {
		return
	}
	if _, off := surfui.AttrOf(c, "disabled"); off {
		return
	}
	if act, ok := surfui.Handler(c, "change"); ok {
		s.do(act, Event{Kind: "change", Target: surfui.IDOf(c), Value: value})
	}
}

// cycleKeyed is the value key takes a Cycle to from value, and whether
// the Cycle takes key at all: ←/↑ go back and →/↓/Space/Enter on, round
// either end; Home and End go to the ends; a character goes to the next
// option after value that starts with it, ignoring case, round to the
// first (and stays put, taken, where none does).
func cycleKeyed(value string, options []string, key string) (string, bool) {
	switch key {
	case "arrow_left", "arrow_up":
		return cycleBy(value, options, -1), true
	case "arrow_right", "arrow_down", "space", "enter":
		return cycleBy(value, options, 1), true
	case "home":
		return options[0], true
	case "end":
		return options[len(options)-1], true
	}
	ch, ok := character(key)
	if !ok {
		return value, false
	}
	at := slices.Index(options, value)
	for i := 1; i <= len(options); i++ {
		o := options[(at+i+len(options))%len(options)]
		if first, _ := utf8.DecodeRuneInString(o); unicode.ToLower(first) == ch {
			return o, true
		}
	}
	return value, true
}

// cycleBy is the option step places from value, round either end; from a
// value that is none of options, the first or the last.
func cycleBy(value string, options []string, step int) string {
	if len(options) == 0 {
		return value
	}
	at := slices.Index(options, value)
	if at < 0 {
		if step > 0 {
			return options[0]
		}
		return options[len(options)-1]
	}
	n := len(options)
	return options[((at+step)%n+n)%n]
}

// character is the character a key types, lower case, if it types one
// with Shift at most: "a", "shift+a" (an A), "1", ">".
func character(key string) (rune, bool) {
	key = strings.TrimPrefix(key, "shift+")
	r, size := utf8.DecodeRuneInString(key)
	if size == 0 || size != len(key) || !unicode.IsPrint(r) || unicode.IsSpace(r) {
		return 0, false
	}
	return unicode.ToLower(r), true
}
