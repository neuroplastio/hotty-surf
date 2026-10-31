package screens

import (
	"strings"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// The cells UIs' drawing (F7): plain lines, with ANSI colours for the
// tones, bold, and reverse video for a selection.

// seg is a run of a cells line's text, in a tone.
type seg struct {
	text string
	tone surfui.Tone
	bold bool
}

// sgr is each tone's ANSI colour.
var sgr = map[surfui.Tone]string{surfui.Accent: "35", surfui.OK: "32", surfui.Warn: "33", surfui.Error: "31", surfui.Info: "36", surfui.Muted: "90"}

// cellsLine is a line of cols cells: left's runs, then right's at the
// line's end, left cut with … where they don't both fit. A selected line
// is in reverse video from end to end, its tones left out.
func cellsLine(cols int, selected bool, left []seg, right ...seg) string {
	var b strings.Builder
	if selected {
		b.WriteString("\x1b[7m")
	}
	rw := 0
	for _, s := range right {
		rw += width(s.text)
	}
	rw = min(rw, cols)
	used := writeSegs(&b, left, cols-rw, selected)
	b.WriteString(strings.Repeat(" ", max(cols-rw-used, 0)))
	writeSegs(&b, right, rw, selected)
	if selected {
		b.WriteString("\x1b[27m")
	}
	return b.String()
}

// writeSegs writes segs in at most room cells, and says how many it took.
func writeSegs(b *strings.Builder, segs []seg, room int, plain bool) int {
	used := 0
	for _, s := range segs {
		text := s.text
		if w := width(text); used+w > room {
			text = cut(text, room-used)
		}
		if text == "" {
			continue
		}
		if s.bold {
			b.WriteString("\x1b[1m")
		}
		if c, ok := sgr[s.tone]; ok && !plain {
			b.WriteString("\x1b[" + c + "m" + text + "\x1b[39m")
		} else {
			b.WriteString(text)
		}
		if s.bold {
			b.WriteString("\x1b[22m")
		}
		used += width(text)
	}
	return used
}

// boxed is a line of a box drawn in cells: content, a line as wide as the
// box's inside, between the box's sides with a cell of room either side,
// at left cells in.
func boxed(left int, content string) string {
	return strings.Repeat(" ", left) + "\x1b[90m│\x1b[39m " + content + " \x1b[90m│\x1b[39m"
}

// boxEdge is the box's top (╭─╮) or bottom (╰─╯) edge.
func boxEdge(left, inner int, top bool) string {
	l, r := "╰", "╯"
	if top {
		l, r = "╭", "╮"
	}
	return strings.Repeat(" ", left) + "\x1b[90m" + l + strings.Repeat("─", inner+2) + r + "\x1b[39m"
}

// cut is s in at most n cells, ending in … where it was cut.
func cut(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if width(s) <= n {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := width(string(r))
		if w+rw > n-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

// wrap is s in lines of at most n cells, broken at spaces, a word longer
// than a line cut.
func wrap(s string, n int) []string {
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		switch {
		case line == "":
			line = w
		case width(line)+1+width(w) <= n:
			line += " " + w
		default:
			out = append(out, cut(line, n))
			line = w
		}
	}
	if line != "" {
		out = append(out, cut(line, n))
	}
	return out
}

// width is s's width in cells: an emoji, ✋ and the ideographic space take
// two, a joiner or a variation selector none, the rest one, the private
// use planes' icons (a key cap's ^ and ⇧) among them. Enough for the
// demo's data; a real app takes a width table.
func width(s string) int {
	n := 0
	for _, r := range s {
		switch {
		case r == 0x200d || r >= 0xfe00 && r <= 0xfe0f:
		case r >= 0x1f000 && r <= 0x1faff, r == '✋', r == 0x3000:
			n += 2
		default:
			n++
		}
	}
	return n
}

// typed is the text a key types into a field in cells, if it types one.
func typed(key string) (string, bool) {
	switch {
	case key == "space":
		return " ", true
	case strings.HasPrefix(key, "shift+") && len([]rune(key)) == 7:
		return strings.ToUpper(key[6:]), true
	case len([]rune(key)) == 1:
		return key, true
	}
	return "", false
}

// backspaced is s without its last character.
func backspaced(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}
