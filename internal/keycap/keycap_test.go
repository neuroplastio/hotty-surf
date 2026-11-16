package keycap_test

import (
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/keycap"
)

// TestKeycap: keys in words where mod is ctrl, in macOS's symbols where it
// is super; the zero Platform is ctrl.
func TestKeycap(t *testing.T) {
	for _, c := range []struct{ mod, keys, want string }{
		{"", "mod+shift+p", "Ctrl+Shift+P"},
		{"ctrl", "mod+s", "Ctrl+S"},
		{"ctrl", "ctrl+alt+delete", "Ctrl+Alt+Del"},
		{"ctrl", "mod+backspace", "Ctrl+Backspace"},
		{"ctrl", "alt+arrow_up", "Alt+↑"},
		{"ctrl", "page_down", "PgDn"},
		{"ctrl", "f2", "F2"},
		{"ctrl", "?", "?"},
		{"ctrl", "mod+ctrl+a", "mod+ctrl+a"}, // doesn't read: as written
		{"super", "mod+shift+p", "⇧⌘P"},
		{"super", "mod+s", "⌘S"},
		{"super", "mod+alt+k", "⌥⌘K"},
		{"super", "ctrl+shift+k", "⌃⇧K"},
		{"super", "mod+backspace", "⌘⌫"},
		{"super", "escape", "⎋"},
	} {
		if got := keycap.Text(hotty.Platform{Mod: c.mod}, c.keys); got != c.want {
			t.Errorf("keycap.Text(%q, %q) = %q, want %q", c.mod, c.keys, got, c.want)
		}
	}
}
