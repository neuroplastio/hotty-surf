package surfui

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottyedit"
)

// readlineCases are the readline keys: what each does to "one two three",
// the caret after "two" (7), as the SDK's keymap has it.
var readlineCases = []struct {
	key, want string
	caret     int
}{
	{"ctrl+backspace", "one  three", 4},
	{"alt+backspace", "one  three", 4},
	{"ctrl+w", "one  three", 4},
	{"ctrl+u", " three", 0},
	{"ctrl+k", "one two", 7},
	{"ctrl+a", "one two three", 0},
	{"ctrl+e", "one two three", 13},
	{"alt+b", "one two three", 4},
	{"alt+f", "one two three", 13},
	{"alt+d", "one two", 7},
}

// edit is the field's text and caret after key, on the keymap a host
// reads for it: the data-keys of the elements from the root down.
func edit(f *Node, plat hotty.Platform, key string) (string, int) {
	var values []string
	for p := f; p != nil; p = p.parent {
		if k := dataKeys(p); k != "" {
			values = append([]string{k}, values...)
		}
	}
	ed := hottyedit.Field{Value: "one two three", Caret: 7}
	ed.Key(hotty.Resolve(plat, false, values...), key)
	return ed.Value, ed.Caret
}

// Readline puts the SDK's keymap into data-keys, on a Surface for every
// field in it and on a Field for that field; off, a field has the
// platform's default, which on a Mac (super) has no ctrl+backspace, ctrl+w
// or ctrl+u as readline has them.
func TestReadline(t *testing.T) {
	for _, plat := range []hotty.Platform{{Mod: "super"}, {Mod: "ctrl"}} {
		for _, where := range []string{"surface", "field"} {
			for _, c := range readlineCases {
				f := Field()
				root := Surface(Stack(f))
				if where == "surface" {
					root.Readline()
				} else {
					f.Readline()
				}
				Resolve(root, "r")
				if got, caret := edit(f, plat, c.key); got != c.want || caret != c.caret {
					// ctrl+a is select-all on a ctrl platform (SPEC §10.2).
					if plat.Mod == "ctrl" && c.key == "ctrl+a" {
						continue
					}
					t.Errorf("%s readline on the %s, %s: %q at %d, want %q at %d", plat.Mod, where, c.key, got, caret, c.want, c.caret)
				}
			}
		}
	}
	root := Surface(Field().Key("f")).Readline()
	Resolve(root, "r")
	if got := DataKeys(root); got != hotty.TerminalKeys {
		t.Errorf("a Surface's data-keys are %q", got)
	}
}

// Off, a Mac's ctrl+backspace does not delete a word: the default.
func TestReadlineOffIsTheDefault(t *testing.T) {
	f := Field()
	root := Surface(f)
	Resolve(root, "r")
	if strings.Contains(DataKeys(f), "ctrl") {
		t.Errorf("data-keys %q", DataKeys(f))
	}
	if got, _ := edit(f, hotty.Platform{Mod: "super"}, "ctrl+backspace"); got != "one two three" {
		t.Errorf("super, no readline: ctrl+backspace made %q", got)
	}
	if got, _ := edit(f, hotty.Platform{Mod: "ctrl"}, "ctrl+backspace"); got != "one  three" {
		t.Errorf("ctrl, no readline: ctrl+backspace made %q", got)
	}
}

// A program's Bind on ctrl+w wins over readline's, from the field, its
// parent and the surface; a key it reserves (Give) too, ctrl+b on a Mac.
func TestReadlineBindWins(t *testing.T) {
	for _, where := range []string{"field", "parent", "surface"} {
		f := Field()
		box := Stack(f)
		root := Surface(box).Readline()
		on := map[string]*Node{"field": f, "parent": box, "surface": root}[where]
		on.Bind("ctrl+w", "Close", "close").Give("ctrl+b")
		Resolve(root, "r")
		got := DataKeys(f)
		if !strings.Contains(got, "ctrl+w=program") || !strings.Contains(got, "ctrl+b=program") {
			t.Errorf("%s: data-keys %q", where, got)
		}
		for _, plat := range []hotty.Platform{{Mod: "super"}, {Mod: "ctrl"}} {
			if text, _ := edit(f, plat, "ctrl+w"); text != "one two three" {
				t.Errorf("%s, %s: ctrl+w edited, %q", where, plat.Mod, text)
			}
			if text, c := edit(f, plat, "ctrl+b"); text != "one two three" || c != 7 {
				t.Errorf("%s, %s: ctrl+b edited, %q at %d", where, plat.Mod, text, c)
			}
		}
	}
}

// The reservation comes ahead of the platform default and of readline in
// the field's data-keys, the field's own last.
func TestReadlineOrder(t *testing.T) {
	f := Field().Readline()
	root := Surface(f).Readline()
	root.Give("ctrl+b")
	Resolve(root, "r")
	got := DataKeys(f)
	if !strings.HasPrefix(got, hotty.TerminalKeys+" ") || !strings.HasSuffix(got, "ctrl+b=program") {
		t.Errorf("data-keys %q", got)
	}
}
