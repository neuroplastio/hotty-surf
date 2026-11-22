// Package cssclass reads the class names a stylesheet's selectors define:
// what lint's class rule checks an app's classes against, and what
// `hotty-surf classes` lists.
package cssclass

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	commentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	classRe   = regexp.MustCompile(`\.((?:[A-Za-z0-9_-]|\\[0-9a-fA-F]{1,6} ?|\\[^0-9a-fA-F\s])+)`)
)

// Add adds to m the class names css's selectors name, unescaped
// (.hover\:bg-accent is hover:bg-accent). A selector is the text before a
// {, so a number in a declaration (0.5rem) names no class.
func Add(css string, m map[string]bool) {
	css = commentRe.ReplaceAllString(css, "")
	for start := 0; ; {
		open := strings.IndexByte(css[start:], '{')
		if open < 0 {
			return
		}
		open += start
		// The selector starts after the last }, ; or { before it.
		from := strings.LastIndexAny(css[start:open], "};{") + 1 + start
		sel := css[from:open]
		if !strings.HasPrefix(strings.TrimSpace(sel), "@") {
			for _, g := range classRe.FindAllStringSubmatch(sel, -1) {
				m[Unescape(g[1])] = true
			}
		}
		start = open + 1
	}
}

// List is the class names css defines, sorted.
func List(css string) []string {
	m := map[string]bool{}
	Add(css, m)
	out := make([]string, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

// Unescape undoes CSS's escapes in an identifier: \: is :, \32 is "2".
func Unescape(s string) string {
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
