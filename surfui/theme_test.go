package surfui

import (
	"maps"
	"regexp"
	"strings"
	"testing"
)

// tokens are the hex custom properties of the rule whose selector line
// is sel, exactly as it stands in surf.css (its indent included).
func tokens(t *testing.T, sel string) map[string]string {
	t.Helper()
	i := strings.Index(CSS, "\n"+sel+" {\n")
	if i < 0 {
		t.Fatalf("no rule %q in surf.css", sel)
	}
	body := CSS[i+len(sel)+4:]
	body = body[:strings.Index(body, "}")]
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`--u-([a-z0-9-]+):\s*(#[0-9a-fA-F]{6});`).FindAllStringSubmatch(body, -1) {
		out[m[1]] = m[2]
	}
	return out
}

// mixOver is a over b at share t, as color-mix(in srgb, a t, transparent)
// is drawn over b.
func mixOver(t *testing.T, a, b string, share float64) string {
	t.Helper()
	ar, ag, ab, ok1 := ParseHex(a)
	br, bg, bb, ok2 := ParseHex(b)
	if !ok1 || !ok2 {
		t.Fatalf("mix of %s and %s", a, b)
	}
	ch := func(x, y uint8) uint8 { return uint8(float64(x)*share + float64(y)*(1-share) + 0.5) }
	const hex = "0123456789abcdef"
	var s strings.Builder
	s.WriteByte('#')
	for _, c := range []uint8{ch(ar, br), ch(ag, bg), ch(ab, bb)} {
		s.WriteByte(hex[c>>4])
		s.WriteByte(hex[c&15])
	}
	return s.String()
}

// Graphite and Stone are fixed colours, so every pair's contrast is a
// number (plx's chrome contract, A0): text 4.5:1 on every ground and
// state fill it is drawn on, an action's label on its fill, a badge's
// text on its fill, a tone on its soft fill; an input's edge and a
// program tile's glyph 3:1. A light terminal's Graphite is Stone.
func TestGraphiteStoneContrast(t *testing.T) {
	stone := tokens(t, ".u-surface.u-theme-stone")
	if light := tokens(t, "  .u-surface.u-theme-graphite"); !maps.Equal(light, stone) {
		t.Errorf("Graphite on a light terminal is not Stone:\n%v\n%v", light, stone)
	}
	themes := []struct {
		name       string
		tok        map[string]string
		muted, dim string
	}{
		{"graphite", tokens(t, ".u-surface.u-theme-graphite"), "#9b9b9b", "#989898"},
		{"stone", stone, "#484848", "#494949"},
	}
	for _, th := range themes {
		tok := th.tok
		tok["muted"], tok["dim"] = th.muted, th.dim
		check := func(what, fg, bg string, need float64) {
			t.Helper()
			r, err := ContrastOf(fg, bg)
			if err != nil {
				t.Fatalf("%s %s: %v", th.name, what, err)
			}
			if r < need {
				t.Errorf("%s: %s is %.2f:1 (%s on %s), want %.1f", th.name, what, r, fg, bg, need)
			}
		}
		get := func(k string) string {
			t.Helper()
			v, ok := tok[k]
			if !ok {
				t.Fatalf("%s has no --u-%s", th.name, k)
			}
			return v
		}
		for _, fg := range []string{"ink", "muted", "dim", "accent", "ok", "warn", "error", "info"} {
			for _, bg := range []string{"canvas", "bg", "raised", "popover", "field", "float", "hover", "selected", "accent-soft"} {
				check(fg+" on "+bg, get(fg), get(bg), TextContrast)
			}
		}
		check("an action's label", get("on-solid"), get("solid"), TextContrast)
		check("an action's label under the pointer", get("on-solid"), get("solid-hover"), TextContrast)
		check("a badge's text", get("on-accent"), get("accent"), TextContrast)
		check("a warn badge's text", get("canvas"), get("warn"), TextContrast)
		check("an error badge's text", get("canvas"), get("error"), TextContrast)
		check("a muted badge's text", get("ink"), get("badge-muted"), TextContrast)
		// The soft fills are mixed over the ground (at most 16%).
		check("warn on its soft fill", get("warn"), mixOver(t, get("warn"), get("bg"), 0.16), TextContrast)
		check("error on its soft fill", get("error"), mixOver(t, get("error"), get("bg"), 0.16), TextContrast)
		for _, bg := range []string{"field", "bg", "popover"} {
			check("an input's edge on "+bg, get("input"), get(bg), DotContrast)
		}
		for _, tile := range []string{"agent", "code", "branch", "book", "server", "settings"} {
			check("a tile's glyph on "+tile, get("tile-ink"), get("tile-"+tile), DotContrast)
		}
		check("the shell tile's glyph", get("tile-shell-ink"), get("tile-shell"), DotContrast)
	}
}

// Graphite's and Stone's muted and dim are their own on every layer, not
// mixed from the ink, and are the ones the contrast test measures.
func TestGraphiteStoneMutedPinned(t *testing.T) {
	for _, want := range []string{
		".u-theme-graphite .u-dialog, .u-theme-graphite .u-options, .u-theme-graphite .u-menu, .u-theme-graphite .u-card { --u-muted: #9b9b9b; --u-dim: #989898; }",
		".u-theme-stone .u-dialog, .u-theme-stone .u-options, .u-theme-stone .u-menu, .u-theme-stone .u-card { --u-muted: #484848; --u-dim: #494949; }",
	} {
		if !strings.Contains(CSS, want) {
			t.Errorf("surf.css doesn't pin %q", want)
		}
	}
}

// A checked checkbox draws its mark and its hover and focus rings in the
// ink the kit puts on the accent (--u-on-accent), which the badge's text
// uses: 3:1 at least for the mark against the fill, in Graphite and Stone
// (paper and the neuroplastio theme's accents are #5b4bd6 and #9d90ff
// under #ffffff and the canvas: 6.1:1 and about 6.7:1). The old ink,
// --u-on-solid, was 1.2:1 on Graphite's accent.
func TestCheckboxMarkInk(t *testing.T) {
	for _, sel := range []string{".u-checkbox.is-on {", ".u-checkbox.is-on:hover {", ".u-checkbox.is-on:focus {"} {
		i := strings.Index(CSS, "\n"+sel)
		if i < 0 {
			t.Fatalf("no rule %q in surf.css", sel)
		}
		rule := CSS[i : i+strings.Index(CSS[i:], "}")]
		if strings.Contains(rule, "on-solid") || strings.Contains(rule, "ground-line") || !strings.Contains(rule, "--u-on-accent") && !strings.Contains(rule, "--u-focus") {
			t.Errorf("%s draws its ink with %q, want --u-on-accent", sel, rule)
		}
	}
	for name, tok := range map[string]map[string]string{
		"graphite": tokens(t, ".u-surface.u-theme-graphite"),
		"stone":    tokens(t, ".u-surface.u-theme-stone"),
	} {
		r, err := ContrastOf(tok["on-accent"], tok["accent"])
		if err != nil || r < DotContrast {
			t.Errorf("%s: the mark is %.2f:1 on the accent (%v), want %.1f", name, r, err, DotContrast)
		}
	}
}
