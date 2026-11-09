package surfkeys

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Bubble Tea names keys as a program's Key.String gives them ("esc",
// "pgdown", "up", "ctrl+a", "J", " " or "space") and its keymaps write a
// sequence as letters run together ("yr"). myelin's names, which hotty
// and this package use, spell a key out ("escape", "page_down",
// "arrow_up"), write a capital as shift and its letter ("shift+j") and
// space the chords of a sequence ("y r"). FromTea and ToTea go between
// them from a table of names, so that a program with Bubble Tea's
// bindings needs neither Bubble Tea nor a list of its own.

// teaKeys are the keys whose names differ, Bubble Tea's to myelin's. A
// key of the same name in both (enter, tab, backspace, delete, insert,
// home, end, f1 to f20, a character) is not listed.
var teaKeys = map[string]string{
	"esc":    "escape",
	"escape": "escape",
	"pgup":   "page_up",
	"pgdown": "page_down",
	"up":     "arrow_up",
	"down":   "arrow_down",
	"left":   "arrow_left",
	"right":  "arrow_right",
	"menu":   "context_menu",
	" ":      "space",
	"space":  "space",
}

// myelinKeys are teaKeys the other way, the names Bubble Tea writes: for
// each myelin name, the one it prints (space is "space" as Bubble Tea 2
// has it, where 1 printed " ", which FromTea reads too).
var myelinKeys = map[string]string{
	"escape":       "esc",
	"page_up":      "pgup",
	"page_down":    "pgdown",
	"arrow_up":     "up",
	"arrow_down":   "down",
	"arrow_left":   "left",
	"arrow_right":  "right",
	"context_menu": "menu",
	"space":        "space",
}

// The modifiers, in the order each side writes them: Bubble Tea's
// ctrl, alt, shift, meta, super; myelin's super, ctrl, alt, shift.
var (
	teaMods    = [...]string{"ctrl", "alt", "shift", "meta", "super"}
	myelinMods = [...]string{"super", "ctrl", "alt", "shift"}
)

// FromTea is a Bubble Tea key name, or a sequence of them, as myelin
// names it: "J" is "shift+j", "esc" "escape", "pgdown" "page_down", "up"
// "arrow_up", " " "space", "ctrl+up" "ctrl+arrow_up" and a sequence, two
// lower-case letters ("yr") or chords spaced ("ctrl+b v"), "y r". A
// lower-case letter stays what it is: "j" is never "J". A name it has no
// rule for is returned as it is, which myelin may still read. ToTea is
// its inverse.
func FromTea(name string) string {
	if name == " " {
		return "space"
	}
	if f := strings.Fields(name); len(f) > 1 {
		for i, c := range f {
			f[i] = FromTea(c)
		}
		return strings.Join(f, " ")
	}
	if len(name) == 2 && lowerLetter(name[0]) && lowerLetter(name[1]) {
		if _, named := teaKeys[name]; !named {
			return name[:1] + " " + name[1:]
		}
	}
	mods, key := splitTea(name)
	if k, ok := teaKeys[key]; ok {
		key = k
	} else if r, n := utf8.DecodeRuneInString(key); n == len(key) && len(mods) == 0 && unicode.IsUpper(r) && unicode.ToLower(r) != r {
		// Only a bare capital: Bubble Tea writes shift with the others
		// ("ctrl+shift+j"), which is already myelin's.
		key, mods = string(unicode.ToLower(r)), []string{"shift"}
	}
	return joinMods(mods, key, myelinMods[:])
}

// ToTea is a myelin key name, or a sequence, as Bubble Tea names it:
// "shift+j" is "J", "escape" "esc", "page_down" "pgdown", "arrow_up"
// "up", and a sequence of two lower-case letters ("y r") "yr", where
// chords of any other kind stay spaced. A name it has no rule for is
// returned as it is. FromTea is its inverse.
func ToTea(name string) string {
	if f := strings.Fields(name); len(f) > 1 {
		short := len(f) == 2
		for i, c := range f {
			f[i] = ToTea(c)
			short = short && len(c) == 1 && lowerLetter(c[0])
		}
		if short {
			return f[0] + f[1]
		}
		return strings.Join(f, " ")
	}
	mods, key := splitTea(name)
	if k, ok := myelinKeys[key]; ok {
		key = k
	} else if r, n := utf8.DecodeRuneInString(key); n == len(key) && unicode.IsLower(r) && unicode.ToUpper(r) != r && hasMod(mods, "shift") {
		// shift+j is J; with another modifier, Bubble Tea writes the
		// modifiers and the small letter ("ctrl+shift+j"), which stays.
		if len(mods) == 1 {
			mods, key = nil, string(unicode.ToUpper(r))
		}
	}
	return joinMods(mods, key, teaMods[:])
}

func lowerLetter(b byte) bool { return b >= 'a' && b <= 'z' }

// splitTea is a chord's modifiers and its key. A "+" is the key where it
// ends the chord ("+", "ctrl++").
func splitTea(chord string) (mods []string, key string) {
	for {
		i := strings.IndexByte(chord, '+')
		if i <= 0 || i == len(chord)-1 && chord[i-1:] == "++" {
			break
		}
		m := strings.ToLower(chord[:i])
		if m == "meta" {
			m = "super"
		}
		if !isMod(m) {
			break
		}
		mods = addMod(mods, m)
		chord = chord[i+1:]
	}
	return mods, chord
}

func isMod(m string) bool {
	for _, x := range teaMods {
		if m == x {
			return true
		}
	}
	return false
}

func addMod(mods []string, m string) []string {
	if hasMod(mods, m) {
		return mods
	}
	return append(mods, m)
}

func hasMod(mods []string, m string) bool {
	for _, x := range mods {
		if x == m {
			return true
		}
	}
	return false
}

// joinMods writes mods in order, then key.
func joinMods(mods []string, key string, order []string) string {
	var b strings.Builder
	for _, m := range order {
		if hasMod(mods, m) {
			b.WriteString(m + "+")
		}
	}
	b.WriteString(key)
	return b.String()
}
