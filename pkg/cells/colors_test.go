package cells

import (
	"strings"
	"testing"
)

func TestParseColors(t *testing.T) {
	in := "\x1b]10;rgb:d0d0/d0d0/d0d0\x1b\\junk\x1b]11;rgb:10/20/30\a" +
		"\x1b]4;5;rgb:f/0/f\a\x1b]4;16;rgb:0/0/0\a\x1b]4;9;#ff0000\x1b\\\x1b]12;rgb:1/1/1\a"
	p := ParseColors([]byte(in))
	for role, want := range map[Role]RGB{
		Fg: {0xd0, 0xd0, 0xd0}, Bg: {0x10, 0x20, 0x30},
		AnsiRole(5): {255, 0, 255}, Accent: {255, 0, 255},
		AnsiRole(9): {255, 0, 0}, Error: {255, 0, 0},
	} {
		if p[role] != want {
			t.Errorf("%s: %v, want %v", role, p[role], want)
		}
	}
	if _, ok := p[AnsiRole(16)]; ok || len(p) != 6 {
		t.Errorf("unexpected roles in %v", p)
	}
	if ParseColors([]byte("\x1b[0m")) != nil {
		t.Error("no answer is nil")
	}
	if !strings.HasPrefix(QueryColors, "\x1b]10;?\a\x1b]11;?\a\x1b]4;0;?\a") || strings.Count(QueryColors, "\x1b]4;") != 16 {
		t.Errorf("QueryColors %q", QueryColors)
	}
}

func TestPaintWith(t *testing.T) {
	s := Screen{Lines: [][]Run{{
		{Text: "a", Fg: Fg},
		{Text: "b", Fg: Accent, Bg: Field, Attr: Bold},
		{Text: "c", Fg: Fg, Bg: Accent},
	}}}
	p := Palette{Fg: {200, 200, 200}, Bg: {0, 0, 0}, Accent: {255, 0, 255}}
	got := PaintWith(s, p)[0]
	want := "\x1b[0;38;2;200;200;200ma" +
		"\x1b[0;48;2;20;20;20;38;2;255;0;255;1mb" +
		"\x1b[0;48;2;255;0;255;38;2;0;0;0mc\x1b[0m"
	if got != want {
		t.Errorf("%q\nwant %q", got, want)
	}
	if _, ok := p[Field]; ok {
		t.Error("Derive changed p")
	}
}
