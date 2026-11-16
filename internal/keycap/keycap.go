// Package keycap writes a key as a menu or a kbd shows it on the host's
// platform: ui.Kbd's text, and the spike's menu shortcuts.
package keycap

import (
	"strings"
	"unicode/utf8"

	"neuroplast.io/hotty-go"
	"neuroplast.io/myelin/myelinkey"
)

// Text is a key as a menu shows it (myelinkey §6) on the host's platform
// (surf.Platform): in words where mod is ctrl, as Linux and Windows write
// them (Ctrl+Shift+P), and in macOS's symbols where it is super (⇧⌘P). keys
// is a key's name as SPEC §10.4 writes it, with mod for the platform's
// command modifier: "mod+shift+p". A name that doesn't read is shown as
// written.
func Text(p hotty.Platform, keys string) string {
	canon, ok := p.ReadKey(keys)
	if !ok {
		return keys
	}
	mods, value, _ := myelinkey.Split(myelinkey.Key(canon))
	if p.Mod == "super" {
		var b strings.Builder
		for _, m := range []struct {
			m myelinkey.Mods
			s string
		}{{myelinkey.Ctrl, "⌃"}, {myelinkey.Alt, "⌥"}, {myelinkey.Shift, "⇧"}, {myelinkey.Super, "⌘"}} {
			if mods&m.m != 0 {
				b.WriteString(m.s)
			}
		}
		b.WriteString(keyName(value, macKeys))
		return b.String()
	}
	var parts []string
	for _, m := range []struct {
		m myelinkey.Mods
		s string
	}{{myelinkey.Ctrl, "Ctrl"}, {myelinkey.Alt, "Alt"}, {myelinkey.Shift, "Shift"}, {myelinkey.Super, "Super"}} {
		if mods&m.m != 0 {
			parts = append(parts, m.s)
		}
	}
	return strings.Join(append(parts, keyName(value, wordKeys)), "+")
}

// The keys shown other than by their names: the arrows by their symbols
// everywhere (myelinkey A.3), and on macOS its own symbols.
var (
	arrowKeys = map[string]string{"arrow_left": "←", "arrow_right": "→", "arrow_up": "↑", "arrow_down": "↓"}
	wordKeys  = map[string]string{"escape": "Esc", "page_up": "PgUp", "page_down": "PgDn", "delete": "Del", "insert": "Ins"}
	macKeys   = map[string]string{"enter": "↩", "backspace": "⌫", "delete": "⌦", "escape": "⎋", "tab": "⇥",
		"page_up": "⇞", "page_down": "⇟", "home": "↖", "end": "↘"}
)

// keyName is a key's value as a keycap shows it: a letter as its capital,
// another character as itself, and a named key by names, then as its name
// in words (page_down: Page Down).
func keyName(v string, names map[string]string) string {
	if s, ok := arrowKeys[v]; ok {
		return s
	}
	if s, ok := names[v]; ok {
		return s
	}
	if utf8.RuneCountInString(v) == 1 {
		return strings.ToUpper(v)
	}
	words := strings.Split(v, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
