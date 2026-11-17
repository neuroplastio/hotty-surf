package lint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"neuroplast.io/hotty-surf/ui"
)

// The class rule: a class in a .templ's class attribute that no stylesheet
// defines does nothing, and nothing else says so. The kit's stylesheet has
// the classes the kit's own markup uses, not all of Tailwind's, so an app's
// class is defined there, in the app's own CSS, or nowhere.

// kitClasses are the classes ui.Stylesheet defines.
var kitClasses = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	cssClasses(ui.Stylesheet(), m)
	return m
})

var classSelRe = regexp.MustCompile(`\.((?:[A-Za-z0-9_-]|\\[0-9a-fA-F]{1,6} ?|\\[^0-9a-fA-F\s])+)`)

// cssClasses adds to m the class names src's selectors name, unescaped
// (.hover\:bg-accent is hover:bg-accent). A number such as 0.5rem adds a
// name no class has, which costs nothing.
func cssClasses(src string, m map[string]bool) {
	for _, g := range classSelRe.FindAllStringSubmatch(src, -1) {
		m[unescape(g[1])] = true
	}
}

// unescape undoes CSS's escapes in an identifier: \: is :, \32 is "2".
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		j := i + 1
		for j < len(s) && j-i <= 6 && isHex(s[j]) {
			j++
		}
		if j == i+1 {
			b.WriteByte(s[j])
			i = j
			continue
		}
		r, _ := strconv.ParseUint(s[i+1:j], 16, 32)
		b.WriteRune(rune(r))
		if j < len(s) && s[j] == ' ' {
			j++
		}
		i = j - 1
	}
	return b.String()
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// marker is a class that is used by other classes' selectors and has no
// rule of its own: Tailwind's group and peer.
func marker(c string) bool {
	return c == "group" || c == "peer" || strings.HasPrefix(c, "group/") || strings.HasPrefix(c, "peer/")
}

// classes checks a class attribute's literal against the defined classes.
func (f *checker) classes(line int, s literal) {
	if f.defined == nil || s.attr != "class" {
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
			f.add(line, s.at+i+1, "class", fmt.Sprintf("class %q has no rule: neither the kit's stylesheet (ui.Stylesheet) nor the app's CSS defines it, so it does nothing; use a class the kit has, or define it in the app's CSS (style/app.css)", c))
		}
		i = j
	}
}
