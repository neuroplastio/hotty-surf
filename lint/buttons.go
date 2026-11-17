package lint

import (
	"regexp"
	"strings"
)

// The button-type rule: a <button> in a <form> with no type is a submit
// button (HTML's rule, and hotty's), so a Cancel beside a field runs the
// form's surf.Submit. Saying the type says which it is. The kit's buttons
// (ui.Button and the rest) say theirs; this reads the app's own markup.

// The button-label rule: a button with only an icon in it and no label
// draws as "[ · ]" in the cells fallback. lint reads what it can be sure
// of: a raw <button> holding only icons, with no aria-label or title and
// no spread (which may give one), and a ui.Button with IconOnly and no
// Attrs. surf faults the rest at render time, which surftest fails on.

var iconCallRe = regexp.MustCompile(`@ui\.Icon\([^)]*\)`)
var svgRe = regexp.MustCompile(`(?s)<svg\b.*?</svg>`)

// labels checks a .templ's icon-only buttons.
func (f *checker) labels() {
	src := strings.Join(f.lines, "\n")
	at := func(i int) (int, int) {
		return strings.Count(src[:i], "\n") + 1, i - (strings.LastIndexByte(src[:i], '\n') + 1) + 1
	}
	const msg = "a button with only an icon needs a label (aria-label in its Attrs, or text): the cells fallback draws it as \"[ · ]\""
	for i := 0; i < len(src); i++ {
		rest := src[i:]
		switch {
		case tagAt(rest, "<button"):
			end := startTagEnd(rest)
			tag := rest[:end]
			close := strings.Index(rest[end:], "</button>")
			if close < 0 || strings.Contains(tag, "...") || hasAttr(tag, "aria-label") || hasAttr(tag, "title") || hasAttr(tag, "aria-labelledby") {
				continue
			}
			body := rest[end : end+close]
			if !iconCallRe.MatchString(body) && !svgRe.MatchString(body) {
				continue
			}
			body = svgRe.ReplaceAllString(iconCallRe.ReplaceAllString(body, ""), "")
			if strings.TrimSpace(body) == "" {
				l, c := at(i)
				f.add(l, c, "button-label", msg)
			}
		case strings.HasPrefix(rest, "ui.Button(ui.ButtonProps{"):
			open := len("ui.Button(ui.ButtonProps")
			depth, j := 0, open
			for ; j < len(rest); j++ {
				if rest[j] == '{' {
					depth++
				} else if rest[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			props := rest[open:min(j+1, len(rest))]
			if strings.Contains(props, "IconOnly: true") && !strings.Contains(props, "Attrs") {
				l, c := at(i)
				f.add(l, c, "button-label", msg)
			}
		}
	}
}

// buttons checks a .templ's raw <button> tags inside a raw <form>.
func (f *checker) buttons() {
	src := strings.Join(f.lines, "\n")
	depth := 0
	for i := 0; i < len(src); i++ {
		if src[i] != '<' {
			continue
		}
		rest := src[i:]
		switch {
		case tagAt(rest, "<form"):
			depth++
		case strings.HasPrefix(rest, "</form"):
			depth = max(depth-1, 0)
		case tagAt(rest, "<button") && depth > 0:
			tag := rest[:startTagEnd(rest)]
			if !hasAttr(tag, "type") {
				line := strings.Count(src[:i], "\n") + 1
				col := i - (strings.LastIndexByte(src[:i], '\n') + 1) + 1
				f.add(line, col, "button-type", `a <button> in a form says its type: with none it submits the form (surf.Submit); type="submit" for the form's button, type="button" for any other`)
			}
		}
	}
}

// tagAt says whether s starts with the start tag name (not <formula).
func tagAt(s, name string) bool {
	if !strings.HasPrefix(s, name) || len(s) == len(name) {
		return false
	}
	c := s[len(name)]
	return c == ' ' || c == '\t' || c == '\n' || c == '>' || c == '/'
}

// startTagEnd is where s's start tag ends: its > outside quotes and templ's
// braces.
func startTagEnd(s string) int {
	var quote byte
	braces := 0
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'' || c == '`':
			quote = c
		case c == '{':
			braces++
		case c == '}':
			braces--
		case c == '>' && braces <= 0:
			return i + 1
		}
	}
	return len(s)
}

// hasAttr says whether a start tag has the attribute name.
func hasAttr(tag, name string) bool {
	for i := 0; ; {
		j := strings.Index(tag[i:], name)
		if j < 0 {
			return false
		}
		j += i
		before, after := byte(' '), byte(' ')
		if j > 0 {
			before = tag[j-1]
		}
		if j+len(name) < len(tag) {
			after = tag[j+len(name)]
		}
		if (before == ' ' || before == '\t' || before == '\n') && (after == '=' || after == ' ' || after == '>' || after == '\n') {
			return true
		}
		i = j + len(name)
	}
}
