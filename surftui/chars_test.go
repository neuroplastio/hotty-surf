package surftui

import (
	"math/rand/v2"
	"slices"
	"strings"
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

// charsAlphabet are pieces of text of every class a cluster's rules
// read: ASCII and its controls, C1, Latin-1 and the soft hyphen,
// combining marks, joiners and selectors, regional indicators, emoji and
// their modifiers, Hangul jamo and syllables, Indic conjuncts, Prepend
// and SpacingMark, tags, wide and fullwidth, box drawing, and invalid
// UTF-8 (the verifier's).
var charsAlphabet = []string{
	"a", "Z", " ", "\t", "\r", "\n", "\x00", "\x1b", "\x7f", "\u0080", "\u009f",
	"\u00a0", "\u00ad", "\u00e9", "\u00ff", "\u0300", "\u0301", "\u036f", "\u0483",
	"\u200d", "\u200b", "\u200c", "\ufe0e", "\ufe0f", "\U000E0100",
	"\U0001F1E9", "\U0001F1EA", "\U0001F44D", "\U0001F3FD", "\U0001F468", "\U0001F469",
	"\u263a", "\u2764", "\u00a9", "\u2122", "\u2500", "\u259f", "\u25a0", "\u25b6",
	"\u1100", "\u1161", "\u11a8", "\uac00", "\uac01", "\ud55c",
	"\u0915", "\u094d", "\u0937", "\u0903", "\u0600", "\u0e33",
	"\u65e5", "\uff21", "\x80", "\xff", "\xe2\x94", "\ufffd", "\U0001F600", "e\u0301\u0301",
	"\u2003", "\u3000", "\U000E0041", "\U000E007F", "\U0001F3F4",
}

// Random strings of up to seven pieces of the alphabet.
func TestCharsOfRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	n := 300000
	if testing.Short() {
		n = 20000
	}
	for range n {
		var b strings.Builder
		for k := r.IntN(7) + 1; k > 0; k-- {
			b.WriteString(charsAlphabet[r.IntN(len(charsAlphabet))])
		}
		if s := b.String(); !slices.Equal(charsOf(s), charsRef(s)) {
			t.Fatalf("charsOf(%q) = %v, want %v", s, charsOf(s), charsRef(s))
		}
	}
}

// Each one-cell rune before every rune to U+30FF and a few past it, and
// every ASCII string of three bytes (the third a step of 3).
func TestCharsOfPairs(t *testing.T) {
	if testing.Short() {
		t.Skip("4.4 million strings")
	}
	var after []rune
	for y := rune(0); y < 0x3100; y++ {
		after = append(after, y)
	}
	after = append(after, 0xfe0f, 0x1f44d, 0x1f3fd, 0x1f1e9, 0xac00, 0xe0100, 0xfffd)
	for x := rune(0); x < 0x2600; x++ {
		if !oneCell(x) {
			continue
		}
		for _, y := range after {
			if s := string(x) + string(y); !slices.Equal(charsOf(s), charsRef(s)) {
				t.Fatalf("charsOf(%q) = %v, want %v", s, charsOf(s), charsRef(s))
			}
		}
	}
	buf := make([]byte, 3)
	for a := range 128 {
		for b := range 128 {
			for c := 0; c < 128; c += 3 {
				buf[0], buf[1], buf[2] = byte(a), byte(b), byte(c)
				if s := string(buf); !slices.Equal(charsOf(s), charsRef(s)) {
					t.Fatalf("charsOf(%q) = %v, want %v", s, charsOf(s), charsRef(s))
				}
			}
		}
	}
}

// FuzzCharsOf: charsOf is the two-pass composition for any bytes. Its
// seeds are the alphabet; go test -fuzz FuzzCharsOf ./cells explores.
func FuzzCharsOf(f *testing.F) {
	for _, s := range charsAlphabet {
		f.Add(s)
		f.Add(s + s)
		f.Add("x" + s + "\u0301")
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := charsOf(s), charsRef(s); !slices.Equal(got, want) {
			t.Fatalf("charsOf(%q) = %v, want %v", s, got, want)
		}
	})
}
