// Package surfkeys draws keys as the person reads them (H5): a key, a chord
// or a sequence in myelin's names ("ctrl+shift+t", "ctrl+b v"), each
// chord on a cap of its own, labelled in the person's Style: for their
// system, as glyphs or as words, in their case, joined and spaced as
// they set. A surfui.Key holds a sequence; a renderer draws it in its Style
// (Expand): surfrun.Surface on a host, cells.Renderer in a terminal's cells.
package surfkeys

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Style is how keys are drawn on their caps: plx's keys.style, keys.os,
// keys.mod and keys.labels. Each field's "" (or "auto", as plx's settings
// say it) is the system's, or the renderer's, default; the zero Style is
// the defaults throughout.
type Style struct {
	// OS is the system the keys are labelled for: "mac", "linux" or
	// "windows". "" is the platform's (OnPlatform): mac where mod is
	// super, linux elsewhere, as no host tells Windows from Linux.
	OS string
	// Mod is what myelin's mod stands for: "ctrl" or "super". "" is the
	// system's: super on macOS, ctrl elsewhere.
	Mod string
	// Caps is how a cap draws its keys: "glyphs", as symbols where the
	// system has them (⌘, ⇧, the arrows) in the Nerd Font hosts ship
	// (hotty HOST.md §2.1), or "text", as words in any font. "" is the
	// renderer's: glyphs on a host, text in cells, whose font is the
	// terminal's.
	Caps string
	// Case is how a cap's words are written: "lower" (ctrl+shift+l, esc),
	// "title" (Ctrl+Shift+L, Esc), "shifted", lower but for a letter with
	// Shift, which is its capital with no Shift (ctrl+L), or "system", as
	// the system writes them (title's, but for macOS's esc, which stays
	// lower in glyphs). "" is lower, as the kit's design board has it.
	Case string
	// Sep joins a chord's labels: "+", "-", or "space". "" is the
	// system's: "-" on macOS, as Apple writes keys, "+" elsewhere.
	Sep string
	// Then stands between a sequence's chords: ">", or "space", a gap
	// with no character. "" is ">".
	Then string
	// Glue is whether a chord's modifiers drawn as glyphs sit tight
	// against its key, after those written as words (⌘⇧P, Meta+⌃⇧R):
	// "on" or "off". "" is the system's: on in macOS's glyphs, off
	// elsewhere.
	Glue string
	// Labels are the person's own labels for keys, by myelin's name,
	// over the system's: super as 󰌽.
	Labels map[string]string
}

// IsZero is whether st says nothing of its own: the defaults throughout.
func (st Style) IsZero() bool {
	return st.OS == "" && st.Mod == "" && st.Caps == "" && st.Case == "" &&
		st.Sep == "" && st.Then == "" && st.Glue == "" && len(st.Labels) == 0
}

// OnPlatform is st on a platform whose mod is mod (hotty.Platform's:
// "super" on macOS): its OS, where st leaves it, the platform's.
func OnPlatform(st Style, mod string) Style {
	if auto(st.OS) && mod == "super" {
		st.OS = "mac"
	}
	return st
}

// Caps are seq's caps in st, a chord's label each, as a host draws them
// (host) or cells do: "ctrl+b v" is ^b and v in plx's style.
func Caps(st Style, host bool, seq string) []string {
	var out []string
	for _, chord := range strings.Fields(seq) {
		out = append(out, chordCap(st, host, false, chord))
	}
	return out
}

// Between is what stands between two caps of a sequence in st: "" for a
// gap with no character.
func Between(st Style) string {
	switch st.Then {
	case "space", " ":
		return ""
	case "", "auto":
		return ">"
	}
	return st.Then
}

// Text is seq's caps as a line of text has them, where a cap is no box:
// the caps apart by a space, or by Between with a space either side.
func Text(st Style, host bool, seq string) string {
	sep := " "
	if b := Between(st); b != "" {
		sep = " " + b + " "
	}
	return strings.Join(Caps(st, host, seq), sep)
}

// Words are seq's caps as a screen reader says them (aria-label): in
// words whatever the caps are drawn in, as no reader says a Nerd Font's
// glyph, the person's own labels left out for the same reason, and the
// caps apart by spaces. The arrows are the text's arrows; Enter is the
// word, as no reader says ↵.
func Words(st Style, seq string) string {
	st.Caps, st.Labels = "text", nil
	var out []string
	for _, chord := range strings.Fields(seq) {
		out = append(out, chordCap(st, false, true, chord))
	}
	return strings.Join(out, " ")
}

// The labels a cap's keys take, as the kit's Keycaps has them for a PC
// and for macOS: ctrl's and shift's glyphs are Material Design Icons in
// the font hosts ship (^, an outlined ⇧); alt and the rest are words.
var (
	pcGlyphs = map[string]string{"ctrl": "\U000F0634", "shift": "\U000F0636", "backspace": "\U000F0B5C",
		"arrow_up": "\U000F005D", "arrow_down": "\U000F0045", "arrow_left": "\U000F004D", "arrow_right": "\U000F0054",
		"enter": "\U000F0311"}
	pcArrows = map[string]string{"arrow_up": "↑", "arrow_down": "↓", "arrow_left": "←", "arrow_right": "→"}
	pcReturn = "↵" // Enter in text, as the arrows are
	pcWords  = map[string]string{"super": "Meta", "ctrl": "Ctrl", "alt": "Alt", "shift": "Shift", "enter": "Enter",
		"escape": "Esc", "backspace": "Backspace", "delete": "Del", "tab": "Tab", "space": "Space",
		"page_up": "PgUp", "page_down": "PgDn", "insert": "Ins", "context_menu": "Menu"}
	macGlyphs = map[string]string{"ctrl": "\U000F0634", "alt": "\U000F0635", "shift": "\U000F0636",
		"super": "\U000F0633", "enter": "\U000F0311", "backspace": "\U000F0B5C", "escape": "esc"}
	modOrder = []string{"super", "ctrl", "alt", "shift"}
)

// auto is whether a setting is left to the system: "" or "auto".
func auto(v string) bool { return v == "" || v == "auto" }

// mac is whether st labels keys for macOS.
func mac(st Style) bool { return st.OS == "mac" }

// glyphs is whether caps draw their keys as glyphs, host the renderer's
// default.
func glyphs(st Style, host bool) bool {
	switch st.Caps {
	case "glyphs":
		return true
	case "text":
		return false
	}
	return host
}

// splitChord is a chord's modifiers and its key: `ctrl+b` is ctrl and b.
// A key that is a plus sign is the chord's last +, alone (`+`) or after
// its modifiers (`ctrl++`).
func splitChord(chord string) (mods []string, key string) {
	i := strings.LastIndex(strings.TrimSuffix(chord, "+"), "+")
	if i < 0 {
		return nil, chord
	}
	return strings.Split(chord[:i], "+"), chord[i+1:]
}

// chordCap is one chord as its cap reads, as the kit's Keycaps draws it:
// the modifiers in myelin's order, mod as the system's; with glue on,
// those drawn as one glyph go last, tight against the key, and the rest
// are joined by sep.
func chordCap(st Style, host, spoken bool, chord string) string {
	mods, key := splitChord(chord)
	for i, m := range mods {
		if m == "mod" {
			mods[i] = "ctrl"
			if st.Mod == "super" || st.Mod != "ctrl" && mac(st) {
				mods[i] = "super"
			}
		}
	}
	slices.SortStableFunc(mods, func(a, b string) int { return slices.Index(modOrder, a) - slices.Index(modOrder, b) })
	label := keyLabel(st, host, spoken, key)
	if _, own := st.Labels[key]; st.Case == "shifted" && !own && utf8.RuneCountInString(key) == 1 && slices.Contains(mods, "shift") {
		label = strings.ToUpper(key)
		mods = slices.DeleteFunc(mods, func(m string) bool { return m == "shift" })
	}
	sep := st.Sep
	switch {
	case sep == "space":
		sep = " "
	case !auto(sep):
	case mac(st):
		sep = "-"
	default:
		sep = "+"
	}
	glue := st.Glue == "on" || auto(st.Glue) && mac(st) && glyphs(st, host)
	var words, marks []string
	for _, m := range mods {
		l := keyLabel(st, host, spoken, m)
		if glue && utf8.RuneCountInString(l) == 1 {
			marks = append(marks, l)
		} else {
			words = append(words, l)
		}
	}
	return strings.Join(append(words, strings.Join(marks, "")+label), sep)
}

// keyLabel is a key's label on a cap: the person's own, else the system's
// in the style's case. Spoken, Enter is the word, not ↵.
func keyLabel(st Style, host, spoken bool, key string) string {
	if l, ok := st.Labels[key]; ok {
		return l
	}
	g := glyphs(st, host)
	var l string
	switch {
	case mac(st) && g && macGlyphs[key] != "":
		l = macGlyphs[key]
	case g && pcGlyphs[key] != "":
		l = pcGlyphs[key]
	case key == "enter" && !spoken:
		l = pcReturn
	case pcArrows[key] != "":
		l = pcArrows[key]
	case pcWords[key] != "":
		l = pcWords[key]
	case utf8.RuneCountInString(key) == 1 || len(key) <= 3 && len(key) > 1 && key[0] == 'f' && strings.Trim(key[1:], "0123456789") == "":
		l = strings.ToUpper(key)
	default:
		l = titled(strings.ReplaceAll(key, "_", " "))
	}
	switch st.Case {
	case "system":
		return l
	case "title":
		return titled(l)
	}
	return strings.ToLower(l)
}

// titled is s with each word's first letter a capital.
func titled(s string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if r, n := utf8.DecodeRuneInString(w); n > 0 {
			words[i] = string(unicode.ToUpper(r)) + w[n:]
		}
	}
	return strings.Join(words, " ")
}
