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
		{"the seed, mod on Linux", seed, true, "mod+enter", []string{ctrl + ret}},
		{"the defaults on a host", surfkeys.Style{}, true, "ctrl+shift+t", []string{ctrl + "+" + shift + "+t"}},
		{"the defaults in cells", surfkeys.Style{}, false, "ctrl+shift+t", []string{"ctrl+shift+t"}},
		{"lower is the default", surfkeys.Style{Caps: "text"}, false, "ctrl+c esc f5 page_up", []string{"ctrl+c", "esc", "f5", "pgup"}},
		{"the default's return in cells", surfkeys.Style{}, false, "enter ctrl+enter", []string{"↵", "ctrl+↵"}},
		{"the default's return on a host", surfkeys.Style{}, true, "enter", []string{ret}},
		{"a return glyph on a PC", surfkeys.Style{Caps: "glyphs", OS: "linux"}, false, "enter", []string{ret}},
		{"a return of the person's", surfkeys.Style{Labels: map[string]string{"enter": "go"}}, true, "ctrl+enter", []string{ctrl + "+go"}},
		{"a return of the person's, in text", surfkeys.Style{Caps: "text", Labels: map[string]string{"enter": "go"}}, false, "enter", []string{"go"}},
		{"title, the capitals", surfkeys.Style{Caps: "text", Case: "title"}, false, "ctrl+c esc f5 page_up enter", []string{"Ctrl+C", "Esc", "F5", "PgUp", "↵"}},
		{"system, as it was", surfkeys.Style{Caps: "text", Case: "system"}, false, "ctrl+c esc f5 page_up", []string{"Ctrl+C", "Esc", "F5", "PgUp"}},
		{"system, mac's esc stays", surfkeys.Style{OS: "mac", Case: "system"}, true, "escape", []string{"esc"}},
		{"title, mac's esc", surfkeys.Style{OS: "mac", Case: "title"}, true, "escape", []string{"Esc"}},
		{"text, title", surfkeys.Style{Caps: "text", Case: "title"}, true, "ctrl+b escape", []string{"Ctrl+B", "Esc"}},
		{"lower", surfkeys.Style{Caps: "text", Case: "lower"}, true, "ctrl+shift+l", []string{"ctrl+shift+l"}},
		{"shifted", surfkeys.Style{Caps: "text", Case: "shifted"}, true, "ctrl+shift+p", []string{"ctrl+P"}},
		{"sep space", surfkeys.Style{Caps: "text", Sep: "space"}, true, "alt+f4", []string{"alt f4"}},
		{"sep -", surfkeys.Style{Caps: "text", Sep: "-"}, true, "ctrl+alt+delete", []string{"ctrl-alt-del"}},
		{"mac's glyphs, glued", surfkeys.Style{OS: "mac"}, true, "mod+shift+p", []string{cmd + shift + "p"}},
		{"mac's text", surfkeys.Style{OS: "mac", Caps: "text"}, true, "mod+enter", []string{"meta-↵"}},
		{"mac's mod as ctrl", surfkeys.Style{OS: "mac", Mod: "ctrl", Caps: "text"}, true, "mod+c", []string{"ctrl-c"}},
		{"mac's return", surfkeys.Style{OS: "mac"}, true, "enter", []string{ret}},
		{"glue off on mac", surfkeys.Style{OS: "mac", Glue: "off"}, true, "mod+p", []string{cmd + "-p"}},
		{"words before glyphs", surfkeys.Style{Glue: "on"}, true, "super+alt+ctrl+shift+r", []string{"meta+alt+" + ctrl + shift + "r"}},
		{"a label of the person's", surfkeys.Style{Caps: "text", Labels: map[string]string{"super": "\U000F033D"}}, true, "super+l", []string{"\U000F033D+l"}},
		{"arrows as words' arrows", surfkeys.Style{Caps: "text"}, true, "alt+arrow_up", []string{"alt+↑"}},
		{"names in words", surfkeys.Style{Caps: "text"}, true, "page_down context_menu f12", []string{"pgdn", "menu", "f12"}},
		// A plus sign is a key: pane.grow's `ctrl+b p +`.
		{"a plus", surfkeys.Style{Caps: "text", Case: "title"}, true, "ctrl+b p +", []string{"Ctrl+B", "P", "+"}},
		{"ctrl and plus", surfkeys.Style{Caps: "text"}, true, "ctrl++", []string{"ctrl++"}},
		{"shift+delete", surfkeys.Style{Caps: "text"}, true, "shift+delete", []string{"shift+del"}},
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
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "ctrl+b > v" {
		t.Errorf("default %q", got)
	}
	text.Then = "space"
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "ctrl+b v" {
		t.Errorf("space %q", got)
	}
	text.Then = "then"
	if got := surfkeys.Text(text, false, "ctrl+b v"); got != "ctrl+b then v" {
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
		`<kbd id="r.kk" class="u-keys" aria-label="ctrl+b v" data-seq="ctrl+b v">` +
		`<kbd id="r.kk.0" class="u-key">ctrl+b</kbd><span id="r.kk.1" class="u-then">&gt;</span><kbd id="r.kk.2" class="u-key">v</kbd></kbd>` +
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

// Words say Enter as a word whatever the caps draw, and the arrows as the
// text draws them.
func TestWords(t *testing.T) {
	for _, c := range []struct {
		st   surfkeys.Style
		seq  string
		want string
	}{
		{surfkeys.Style{}, "enter", "enter"},
		{surfkeys.Style{}, "ctrl+enter", "ctrl+enter"},
		{surfkeys.Style{Caps: "glyphs", Labels: map[string]string{"enter": "go"}}, "enter", "enter"},
		{surfkeys.Style{OS: "mac"}, "mod+enter", "meta-enter"},
		{surfkeys.Style{Case: "title"}, "ctrl+enter", "Ctrl+Enter"},
		{surfkeys.Style{}, "alt+arrow_up", "alt+↑"},
	} {
		if got := surfkeys.Words(c.st, c.seq); got != c.want {
			t.Errorf("%+v %q is %q, want %q", c.st, c.seq, got, c.want)
		}
	}
}
