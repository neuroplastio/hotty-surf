package mdi

import "testing"

func TestGlyph(t *testing.T) {
	for name, want := range map[string]rune{
		"ab-testing":   0xF01C9, // the first
		"cog-outline":  0xF08BB,
		"help-box":     HelpBox,
		"content-copy": 0xF018F,
		"restart":      0xF0709,
		"close":        0xF0156,
	} {
		if got, ok := Glyph(name); !ok || got != want {
			t.Errorf("%s: %U %v, want %U", name, got, ok, want)
		}
	}
	for _, name := range []string{"", "cog_outline", "Cog-Outline", "nope", "zzzz", "a"} {
		if g, ok := Glyph(name); ok {
			t.Errorf("%q: %U, want none", name, g)
		}
	}
	if Len() < 6000 {
		t.Errorf("%d names", Len())
	}
	for i := 1; i < Len(); i++ { // sorted, which the search needs
		if nameAt(i-1) >= nameAt(i) {
			t.Fatalf("%s before %s", nameAt(i-1), nameAt(i))
		}
	}
}
