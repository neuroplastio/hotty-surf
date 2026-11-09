package surfkeys_test

import (
	"testing"

	"neuroplast.io/hotty-surf/surfkeys"
)

// The names of one side and the other, which FromTea and ToTea carry
// over: margin's bindings and hotty-go's key names.
var teaPairs = []struct{ tea, myelin string }{
	{"j", "j"},
	{"J", "shift+j"},
	{"esc", "escape"},
	{"pgdown", "page_down"},
	{"pgup", "page_up"},
	{"up", "arrow_up"},
	{"down", "arrow_down"},
	{"left", "arrow_left"},
	{"right", "arrow_right"},
	{"menu", "context_menu"},
	{"space", "space"},
	{"yr", "y r"},
	{"ctrl+b v", "ctrl+b v"},
	{"ctrl+b J", "ctrl+b shift+j"},
	{"gg", "g g"},
	{"enter", "enter"},
	{"tab", "tab"},
	{"shift+tab", "shift+tab"},
	{"backspace", "backspace"},
	{"delete", "delete"},
	{"home", "home"},
	{"end", "end"},
	{"f5", "f5"},
	{"ctrl+c", "ctrl+c"},
	{"alt+x", "alt+x"},
	{"ctrl+alt+shift+up", "ctrl+alt+shift+arrow_up"},
	{"shift+up", "shift+arrow_up"},
	{"ctrl+shift+j", "ctrl+shift+j"},
	{"ctrl+pgdown", "ctrl+page_down"},
	{"/", "/"},
	{"?", "?"},
	{"+", "+"},
	{"ctrl++", "ctrl++"},
	{"1", "1"},
	{"é", "é"},
	{"É", "shift+é"},
}

// Each name goes to myelin's and back, and no two of Bubble Tea's go to
// one: a capital is shift and its letter, and "j" is never "J".
func TestTeaKeys(t *testing.T) {
	seen := map[string]string{}
	for _, c := range teaPairs {
		if got := surfkeys.FromTea(c.tea); got != c.myelin {
			t.Errorf("FromTea(%q) = %q, want %q", c.tea, got, c.myelin)
		}
		if got := surfkeys.ToTea(c.myelin); got != c.tea {
			t.Errorf("ToTea(%q) = %q, want %q", c.myelin, got, c.tea)
		}
		if other, dup := seen[c.myelin]; dup {
			t.Errorf("%q and %q are both %q", other, c.tea, c.myelin)
		}
		seen[c.myelin] = c.tea
	}
	if surfkeys.FromTea("j") == surfkeys.FromTea("J") || surfkeys.ToTea("j") == surfkeys.ToTea("shift+j") {
		t.Error("j and J are one key")
	}
}

// Bubble Tea 1's " " is space, and a name with no rule stays as it is.
func TestTeaKeysOther(t *testing.T) {
	for in, want := range map[string]string{" ": "space", "escape": "escape", "arrow_up": "arrow_up", "meta+x": "super+x", "ctrl+J": "ctrl+J", "": ""} {
		if got := surfkeys.FromTea(in); got != want {
			t.Errorf("FromTea(%q) = %q, want %q", in, got, want)
		}
	}
	if got := surfkeys.ToTea("f12"); got != "f12" {
		t.Errorf("ToTea(f12) = %q", got)
	}
}
