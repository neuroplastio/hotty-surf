package cells

import (
	"strconv"
	"strings"
)

// RGB is a colour in 8 bits a channel.
type RGB struct{ R, G, B uint8 }

// Palette maps roles to the colours a screen is painted in (PaintWith),
// truecolour where Paint (surfrun) has the terminal's sixteen. It needs
// only what a program knows: the text (Fg), the ground (Bg) and, where it
// has them, the tones, which ParseColors fills from the terminal's
// answers. A role it lacks is derived, as Derive says.
type Palette map[Role]RGB

// Mix is a toward b by t, 0 to 1: a ground mixed toward the ink is a
// surface a step from it.
func Mix(a, b RGB, t float64) RGB {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return RGB{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B)}
}

// The ground and the text a palette that names neither gets.
var (
	defaultFg = RGB{0xd0, 0xd0, 0xd0}
	defaultBg = RGB{0x10, 0x10, 0x10}
)

// Derive is p with every role there is: those p has are kept, and the
// rest are mixed from its Fg and Bg, as the terminal's palette draws
// them in sixteen colours: the surface, a control's field, tonal, a
// selection and sel steps from the ground toward the text; the border and
// a rule's line a quarter of the way; muted and faint toward the ground;
// text on an accent fill the ground's own; a tone, if p has none, the
// text's. p is not changed.
func (p Palette) Derive() Palette {
	out := make(Palette, len(p)+16)
	for r, c := range p {
		out[r] = c
	}
	fg, ok := out[Fg]
	if !ok {
		fg = defaultFg
		out[Fg] = fg
	}
	bg, ok := out[Bg]
	if !ok {
		bg = defaultBg
		out[Bg] = bg
	}
	for _, d := range [...]struct {
		role Role
		to   RGB
		t    float64
	}{
		{Surface, fg, 0.04}, {Field, fg, 0.10}, {Tonal, fg, 0.08},
		{Selected, fg, 0.12}, {Sel, fg, 0.20},
		{Border, fg, 0.25}, {Line, fg, 0.25},
		{Muted, bg, 0.40}, {Faint, bg, 0.60},
	} {
		if _, ok := out[d.role]; !ok {
			from := bg
			if d.to == bg {
				from = fg
			}
			out[d.role] = Mix(from, d.to, d.t)
		}
	}
	if _, ok := out[OnAccent]; !ok {
		out[OnAccent] = bg
	}
	for _, r := range [...]Role{Accent, OK, Warn, Error, Info} {
		if _, ok := out[r]; !ok {
			out[r] = fg
		}
	}
	return out
}

// PaintWith is a screen in truecolour in the colours of p (Derive), one
// string a row, each run's text in its colours and a fill, the row ending
// in a reset: Paint's shape, for a program that has its own colours or
// asked the terminal for them (QueryColors). A tone's fill carries the
// ground's colour as text; a run with no fill keeps the terminal's own.
// No cursor moves, no modes.
//
// The runs that carry a link (Run.Href) are written as OSC 8 hyperlinks,
// as surfrun.Paint writes them; s.NoLinks() leaves them out, for a
// program whose host keeps the links itself.
func PaintWith(s Screen, p Palette) []string {
	p = p.Derive()
	out := make([]string, len(s.Lines))
	for i, line := range s.Lines {
		var b strings.Builder
		link := ""
		for _, run := range line {
			link = SetLink(&b, link, run.Href)
			b.WriteString(sgrWith(run, p))
			b.WriteString(run.Text)
		}
		SetLink(&b, link, "")
		b.WriteString("\x1b[0m")
		out[i] = b.String()
	}
	return out
}

// SetLink writes to b what takes a row's text from the link it is in
// (from, "" for none) to the link next is: an OSC 8 that closes one and
// opens the other, nothing where they are the same. It returns next,
// the link the text after it is in. A painter of a screen's rows calls
// it before each run, and once more with "" at the row's end, so that a
// link never reaches into the next row's cells.
func SetLink(b *strings.Builder, from, next string) string {
	if from == next {
		return next
	}
	b.WriteString("\x1b]8;;")
	b.WriteString(linkURI(next))
	b.WriteString("\x1b\\")
	return next
}

// linkURI is href as an OSC 8's URI has it: printable ASCII only. A
// control character, which would end the sequence early, and a byte
// outside it, are written as % and their two hex digits, so that an href
// from a program or a peer can write no other escape sequence.
func linkURI(href string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(href); i++ {
		c := href[i]
		if c <= 0x20 || c >= 0x7f {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

var sgrAttrs = [...]struct {
	attr Attr
	sgr  string
}{{Bold, "1"}, {Dim, "2"}, {Italic, "3"}, {Underline, "4"}, {Reverse, "7"}, {Strike, "9"}, {Overline, "53"}}

// sgrWith is a run's look in p, which has every role.
func sgrWith(run Run, p Palette) string {
	ps := []string{"0"}
	fg := run.Fg
	if fg == "" {
		fg = Fg
	}
	if run.Bg != "" {
		if c, ok := roleColour(run.Bg, p); ok {
			ps = append(ps, "48;2;"+rgbParams(c))
		}
		switch {
		case run.Bg == Accent || run.Bg == OK || run.Bg == Warn || run.Bg == Error || run.Bg == Info:
			fg = OnAccent // a tone's fill: the ground's own colour as text
		}
	}
	if c, ok := roleColour(fg, p); ok {
		ps = append(ps, "38;2;"+rgbParams(c))
	}
	for _, a := range sgrAttrs {
		if run.Attr&a.attr != 0 {
			ps = append(ps, a.sgr)
		}
	}
	if run.Ul != "" && run.Attr&(Underline|Overline) != 0 {
		if c, ok := roleColour(run.Ul, p); ok {
			ps = append(ps, "58;2;"+rgbParams(c))
		}
	}
	return "\x1b[" + strings.Join(ps, ";") + "m"
}

// roleColour is a role's colour in p; an ansi role p has no colour for
// has none, so that the terminal draws it.
func roleColour(r Role, p Palette) (RGB, bool) {
	c, ok := p[r]
	return c, ok
}

func rgbParams(c RGB) string {
	return strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
}

// QueryColors is what a program writes to the terminal to ask for its
// colours: the default text (OSC 10), the ground (OSC 11) and the sixteen
// of its palette (OSC 4, 0 to 15), each ending in BEL. The terminal
// answers on its input; ParseColors reads the answers, and a terminal that
// has none of them is silent about it, so a program waits a moment (the
// time it allows for a reply) and then goes on with what came.
var QueryColors = func() string {
	var b strings.Builder
	b.WriteString("\x1b]10;?\a\x1b]11;?\a")
	for n := 0; n < 16; n++ {
		b.WriteString("\x1b]4;" + strconv.Itoa(n) + ";?\a")
	}
	return b.String()
}()

// ParseColors reads the terminal's answers to QueryColors in b: any of
// OSC 10, 11 and 4, in any order, each ended by BEL or ST (ESC \), the
// colour as rgb:R/G/B with one to four hex digits a channel, or #RRGGBB.
// It skips what it doesn't know, and returns what it found as a Palette:
// Fg, Bg and AnsiRole(n), and the roles the runtime maps to a terminal
// colour: accent to 5, ok to 10, warn to 11, error to 9, info to 14,
// border and line to 8, where those answered. Anything else is derived
// when it is painted (Palette.Derive). It is nil if b had none.
func ParseColors(b []byte) Palette {
	var p Palette
	s := string(b)
	for {
		i := strings.Index(s, "\x1b]")
		if i < 0 {
			break
		}
		s = s[i+2:]
		end, next := len(s), len(s)
		if j := strings.IndexByte(s, '\a'); j >= 0 {
			end, next = j, j+1
		}
		if j := strings.Index(s, "\x1b\\"); j >= 0 && j < end {
			end, next = j, j+2
		}
		body := s[:end]
		s = s[next:]
		role, val, ok := oscColour(body)
		if !ok {
			continue
		}
		c, ok := parseColour(val)
		if !ok {
			continue
		}
		if p == nil {
			p = Palette{}
		}
		p[role] = c
	}
	for _, m := range [...]struct {
		n    int
		role []Role
	}{{5, []Role{Accent}}, {10, []Role{OK}}, {11, []Role{Warn}}, {9, []Role{Error}}, {14, []Role{Info}}, {8, []Role{Border, Line}}} {
		if c, ok := p[AnsiRole(m.n)]; ok {
			for _, r := range m.role {
				p[r] = c
			}
		}
	}
	return p
}

// oscColour splits an OSC body, "11;rgb:…" or "4;9;rgb:…", into the role
// it answers for and the colour's text.
func oscColour(body string) (Role, string, bool) {
	cmd, rest, ok := strings.Cut(body, ";")
	if !ok {
		return "", "", false
	}
	switch cmd {
	case "10":
		return Fg, rest, true
	case "11":
		return Bg, rest, true
	case "4":
		idx, val, ok := strings.Cut(rest, ";")
		n, err := strconv.Atoi(idx)
		if !ok || err != nil || n < 0 || n > 15 {
			return "", "", false
		}
		return AnsiRole(n), val, true
	}
	return "", "", false
}

// parseColour reads rgb:R/G/B (one to four hex digits a channel, scaled
// to 8 bits) or #RRGGBB.
func parseColour(s string) (RGB, bool) {
	if h, ok := strings.CutPrefix(s, "#"); ok {
		if len(h) != 6 {
			return RGB{}, false
		}
		v, err := strconv.ParseUint(h, 16, 24)
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, err == nil
	}
	h, ok := strings.CutPrefix(s, "rgb:")
	if !ok {
		return RGB{}, false
	}
	parts := strings.Split(h, "/")
	if len(parts) != 3 {
		return RGB{}, false
	}
	var ch [3]uint8
	for i, part := range parts {
		if len(part) < 1 || len(part) > 4 {
			return RGB{}, false
		}
		v, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return RGB{}, false
		}
		max := uint64(1)<<(4*len(part)) - 1
		ch[i] = uint8((v*255 + max/2) / max)
	}
	return RGB{ch[0], ch[1], ch[2]}, true
}
