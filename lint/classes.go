package lint

import (
	"fmt"
	"strings"
	"sync"

	"neuroplast.io/hotty-surf/internal/cssclass"
	"neuroplast.io/hotty-surf/ui"
)

// The class rule: a class in a .templ that no stylesheet defines does
// nothing, and nothing else says so. The kit's stylesheet has the classes
// the kit's own markup uses, not all of Tailwind's (`hotty-surf classes`
// lists them), so an app's class is defined there, in the app's own CSS,
// or nowhere.

// kitClasses are the classes ui.Stylesheet defines.
var kitClasses = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	cssclass.Add(ui.Stylesheet(), m)
	return m
})

// cssClasses adds to m the classes src's selectors define.
func cssClasses(src string, m map[string]bool) { cssclass.Add(src, m) }

// marker is a class that is used by other classes' selectors and has no
// rule of its own: Tailwind's group and peer.
func marker(c string) bool {
	return c == "group" || c == "peer" || strings.HasPrefix(c, "group/") || strings.HasPrefix(c, "peer/")
}

// classes checks a literal of classes against the defined ones: a class
// attribute's value, or a Go string where templ takes classes
// (templ.KV's key, templ.Classes' arguments).
func (f *checker) classes(line int, s literal) {
	if f.defined == nil || !s.classes {
		return
	}
	for i := 0; i < len(s.text); {
		for i < len(s.text) && (s.text[i] == ' ' || s.text[i] == '\t') {
			i++
		}
		j := i
		for j < len(s.text) && s.text[j] != ' ' && s.text[j] != '\t' {
			j++
		}
		// A px or colour finding on the class says more than this would.
		if c := s.text[i:j]; c != "" && !marker(c) && !f.defined[c] && !kitClasses()[c] &&
			!pxArbRe.MatchString(c) && !colourRe.MatchString(c) {
			f.add(line, s.at+i+1, "class", fmt.Sprintf("class %q has no rule: neither the kit's stylesheet nor the app's CSS defines it, so it does nothing; use one of the kit's classes (hotty-surf classes lists them), or define it in the app's CSS (style/app.css)", c))
		}
		i = j
	}
}
