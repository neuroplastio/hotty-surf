package cells

import (
	"slices"
	"testing"

	"github.com/clipperhouse/displaywidth"
	"github.com/clipperhouse/uax29/v2/graphemes"
)

// charsRef is charsOf as it was: graphemes, then each one's width.
func charsRef(s string) []char {
	var out []char
	for g := graphemes.FromString(s); g.Next(); {
		v := g.Value()
		if v == "\t" {
			out = append(out, char{" ", 1})
			continue
		}
		w := displaywidth.String(v)
		if w <= 0 {
			continue
		}
		out = append(out, char{v, min(w, 2)})
	}
	return out
}

func TestCharsOfIsTheReference(t *testing.T) {
	for _, s := range []string{
		"", "a", "hello world", "a\tb", "a\r\nb", "\x00\x1b[0m\x7f", "\u00a0\u00a0\u00a0",
		"╭ ─╮│", "e\u0301", "é", "日本語", "👍🏽", "👨\u200d👩\u200d👧", "🇩🇪🇫🇷", "a\u200db", "\u200d",
		"x\u0300\u0301y", "한국어", "각", "…●×", "a\u00a0\u0301", "\ufe0f", "☺\ufe0f",
		"tab 12", "\t\t", "A\u0308\u0323",
	} {
		got, want := charsOf(s), charsRef(s)
		if !slices.Equal(got, want) {
			t.Errorf("charsOf(%q) = %v, want %v", s, got, want)
		}
	}
}

// TestCharsOfOneCellRunes: each rune the fast path takes, alone, before
// and after a joiner, a combining mark, a variation selector and another.
func TestCharsOfOneCellRunes(t *testing.T) {
	var rs []rune
	for r := rune(0); r < 0x3000; r++ {
		if oneCell(r) {
			rs = append(rs, r)
		}
	}
	for _, r := range rs {
		for _, s := range []string{
			string(r), string(r) + "\u0301", string(r) + "\u200d", string(r) + "\ufe0f",
			string(r) + "a", "a" + string(r), string(r) + "\u00a0", string(r) + "\U0001F44D",
			string(r) + "\u0903", "\u1100" + string(r),
		} {
			if got, want := charsOf(s), charsRef(s); !slices.Equal(got, want) {
				t.Fatalf("charsOf(%q) = %v, want %v", s, got, want)
			}
		}
	}
}
