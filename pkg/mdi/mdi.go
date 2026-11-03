// Package mdi is the Material Design Icons' glyphs by name, as the font
// hotty hosts ship (AdwaitaMono Nerd Font Propo, hotty HOST.md §2.1) has
// them: Nerd Fonts' md-* glyphs, at MDI's own code points. A template's
// <icon name="cog-outline"/> is looked up here.
//
// The table is generated (go generate; gen.go says how) from Nerd Fonts
// 3.5.1's glyphnames.json, its md-* entries only (md.tsv). The names and
// code points are MDI's (Pictogrammers, Apache-2.0); the file they come
// from is Nerd Fonts' (MIT). NOTICE at the repository's root says so.
//
// It is constant data: a program that never calls Glyph links none of
// it. A template's literal name is looked up when the template compiles,
// so only a name bound at run time links the table.
package mdi

//go:generate go run gen.go

// HelpBox is MDI's help-box, the glyph drawn for a name it doesn't have.
const HelpBox = '\U000F078B'

// Glyph is the glyph MDI's name stands for ("cog-outline"), and whether
// it has one by that name.
func Glyph(name string) (rune, bool) {
	lo, hi := 0, len(ends)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if nameAt(m) < name {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(ends) && nameAt(lo) == name {
		return 0xF0000 + rune(codes[lo]), true
	}
	return 0, false
}

// nameAt is the i'th name.
func nameAt(i int) string {
	start := uint32(0)
	if i > 0 {
		start = ends[i-1]
	}
	return names[start:ends[i]]
}

// Len is how many names it has.
func Len() int { return len(ends) }
