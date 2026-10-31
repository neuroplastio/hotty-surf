package surfkeys_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// plx's seed style: glyph caps, lower case, glue on, then a space.
var seed = surfkeys.Style{Caps: "glyphs", Case: "lower", Sep: "auto", Then: "space", Glue: "on", OS: "auto", Mod: "auto"}

// Each of plx's keys.style settings, on a host and in cells.
func TestCaps(t *testing.T) {
	const (
		ctrl  = "\U000F0634"
		shift = "\U000F0636"
		cmd   = "\U000F0633"
		ret   = "\U000F0311"
	)
	for _, c := range []struct {
		name string
		st   surfkeys.Style
		host bool
		seq  string
		want []string
	}{
		{"the seed", seed, true, "ctrl+b p shift+x", []string{ctrl + "b", "p", shift + "x"}},
		{"the seed, mod on Linux", seed, true, "mod+enter", []string{ctrl + "enter"}},
		{"the defaults on a host", surfkeys.Style{}, true, "ctrl+shift+t", []string{ctrl + "+" + shift + "+T"}},
		{"the defaults in cells", surfkeys.Style{}, false, "ctrl+shift+t", []string{"Ctrl+Shift+T"}},
		{"text, title", surfkeys.Style{Caps: "text", Case: "title"}, true, "ctrl+b escape", []string{"Ctrl+B", "Esc"}},
		{"lower", surfkeys.Style{Caps: "text", Case: "lower"}, true, "ctrl+shift+l", []string{"ctrl+shift+l"}},
		{"shifted", surfkeys.Style{Caps: "text", Case: "shifted"}, true, "ctrl+shift+p", []string{"ctrl+P"}},
		{"sep space", surfkeys.Style{Caps: "text", Sep: "space"}, true, "alt+f4", []string{"Alt F4"}},
		{"sep -", surfkeys.Style{Caps: "text", Sep: "-"}, true, "ctrl+alt+delete", []string{"Ctrl-Alt-Del"}},
		{"mac's glyphs, glued", surfkeys.Style{OS: "mac"}, true, "mod+shift+p", []string{cmd + shift + "P"}},
		{"mac's text", surfkeys.Style{OS: "mac", Caps: "text"}, true, "mod+enter", []string{"Meta-Enter"}},
		{"mac's mod as ctrl", surfkeys.Style{OS: "mac", Mod: "ctrl", Caps: "text"}, true, "mod+c", []string{"Ctrl-C"}},
		{"mac's return", surfkeys.Style{OS: "mac"}, true, "enter", []string{ret}},
		{"glue off on mac", surfkeys.Style{OS: "mac", Glue: "off"}, true, "mod+p", []string{cmd + "-P"}},
		{"words before glyphs", surfkeys.Style{Glue: "on"}, true, "super+alt+ctrl+shift+r", []string{"Meta+Alt+" + ctrl + shift + "R"}},
		{"a label of the person's", surfkeys.Style{Caps: "text", Labels: map[string]string{"super": "\U000F033D"}}, true, "super+l", []string{"\U000F033D+L"}},
		{"arrows as words' arrows", surfkeys.Style{Caps: "text"}, true, "alt+arrow_up", []string{"Alt+↑"}},
		{"names in words", surfkeys.Style{Caps: "text"}, true, "page_down context_menu f12", []string{"PgDn", "Menu", "F12"}},
		// A plus sign is a key: pane.grow's `ctrl+b p +`.
		{"a plus", surfkeys.Style{Caps: "text", Case: "title"}, true, "ctrl+b p +", []string{"Ctrl+B", "P", "+"}},
		{"ctrl and plus", surfkeys.Style{Caps: "text"}, true, "ctrl++", []string{"Ctrl++"}},
		{"shift+delete", surfkeys.Style{Caps: "text"}, true, "shift+delete", []string{"Shift+Del"}},
	} {
		if got := surfkeys.Caps(c.st, c.host, c.seq); !slices.Equal(got, c.want) {
			t.Errorf("%s: %q is %q, want %q", c.name, c.seq, got, c.want)
		}
	}
}

// OnPlatform labels for macOS where mod is super, as a host says it is.
func TestOnPlatform(t *testing.T) {
	if got := surfkeys.OnPlatform(surfkeys.Style{}, "super").OS; got != "mac" {
		t.Errorf("super: %q", got)
	}
	if got := surfkeys.OnPlatform(surfkeys.Style{}, "ctrl").OS; got != "" {
		t.Errorf("ctrl: %q", got)
	}
	if got := surfkeys.OnPlatform(surfkeys.Style{OS: "windows"}, "super").OS; got != "windows" {
		t.Errorf("a set OS: %q", got)
	}
}

// Then: > by default, a gap for space; Text has it in a line.
func TestThen(t *testing.T) {
	text := surfkeys.Style{Caps: "text"}
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "Ctrl+B > V" {
		t.Errorf("default %q", got)
	}
	text.Then = "space"
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "Ctrl+B V" {
		t.Errorf("space %q", got)
	}
	text.Then = "then"
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "Ctrl+B then V" {
		t.Errorf("a word %q", got)
	}
}

// Expand draws each key of a tree in place, its sequence kept, and draws
// it again in another style; a Node keeps its own.
func TestExpand(t *testing.T) {
	text := surfkeys.Style{Caps: "text"}
	k := surfui.Key("ctrl+b v").Key("k")
	own := surfkeys.Node(seed, true, "ctrl+k").Key("own")
	tree := surfui.Row(surfui.Text("Split"), k, own)
	surfkeys.Expand(tree, text, true)
	surfui.Resolve(tree, "r")
	want := `<div id="r" class="u-row"><span id="r.0" class="u-text">Split</span>` +
		`<kbd id="r.kk" class="u-keys" aria-label="Ctrl+B V" data-seq="ctrl+b v">` +
		`<kbd id="r.kk.0" class="u-key">Ctrl+B</kbd><span id="r.kk.1" class="u-then">&gt;</span><kbd id="r.kk.2" class="u-key">V</kbd></kbd>` +
		`<kbd id="r.kown" class="u-keys" aria-label="ctrl+k">` +
		`<kbd id="r.kown.0" class="u-key">` + "\U000F0634" + `k</kbd></kbd></div>`
	if got := surfui.HTML(tree); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	surfkeys.Expand(tree, seed, true)
	if got := surfui.TextOf(k); got != "\U000F0634"+"bv" {
		t.Errorf("again, in the seed: %q", got)
	}
	if strings.Contains(surfui.HTML(k), "u-then") {
		t.Error("then is a space in the seed: no mark")
	}
	if seq, ok := surfkeys.SeqOf(k); !ok || seq != "ctrl+b v" {
		t.Errorf("the sequence %q %v", seq, ok)
	}
	if _, ok := surfkeys.SeqOf(own); ok {
		t.Error("a Node has a sequence to draw again")
	}
}
