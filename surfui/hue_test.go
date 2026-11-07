package surfui

import (
	"strings"
	"testing"
)

func TestParseHex(t *testing.T) {
	for _, c := range []struct {
		in      string
		r, g, b uint8
		ok      bool
	}{
		{"#fff", 255, 255, 255, true},
		{"#a33", 0xaa, 0x33, 0x33, true},
		{"#FF8E7a", 0xff, 0x8e, 0x7a, true},
		{"#ff8e7", 0, 0, 0, false},
		{"ff8e7a", 0, 0, 0, false},
		{"#ggg", 0, 0, 0, false},
		{"red", 0, 0, 0, false},
		{"#ff8e7a00", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	} {
		r, g, b, ok := ParseHex(c.in)
		if ok != c.ok || r != c.r || g != c.g || b != c.b {
			t.Errorf("ParseHex(%q) = %d %d %d %v, want %d %d %d %v", c.in, r, g, b, ok, c.r, c.g, c.b, c.ok)
		}
	}
}

func TestContrastOf(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want float64
	}{{"#000", "#fff", 21}, {"#fff", "#000", 21}, {"#123456", "#123456", 1}} {
		got, err := ContrastOf(c.a, c.b)
		if err != nil || got < c.want-0.01 || got > c.want+0.01 {
			t.Errorf("ContrastOf(%s, %s) = %v, %v, want %v", c.a, c.b, got, err, c.want)
		}
	}
	if _, err := ContrastOf("#000", "black"); err == nil {
		t.Error("a colour that is no hex has a contrast")
	}
	// A hue that passes on both grounds: the Q-0001 examples.
	for _, c := range []struct{ dark, light string }{{"#ff8e7a", "#b4321f"}, {"#4fd1c5", "#0f766e"}} {
		d, _ := ContrastOf(c.dark, DarkGround)
		l, _ := ContrastOf(c.light, PaperGround)
		if d < TextContrast || l < TextContrast {
			t.Errorf("%s/%s: %.2f and %.2f, want 4.5 and over", c.dark, c.light, d, l)
		}
	}
}

// No single colour is text on both grounds: their own contrast is under
// 4.5 squared, so a value= is a dot's, or refused.
func TestNoValueIsText(t *testing.T) {
	for _, v := range []string{"#000", "#fff", "#808a9c", "#777", "#9a9aa0"} {
		weak := Palette{"v": {Value: v}}.Contrast()
		if len(weak) == 0 {
			t.Errorf("%s reads as text on both grounds", v)
		}
	}
	if weak := (Palette{"v": {Value: "#808a9c"}}).ContrastAt(func(string) float64 { return DotContrast }); len(weak) != 0 {
		t.Errorf("a mid grey fails as a dot's: %v", weak)
	}
}

func TestNearestCell(t *testing.T) {
	for _, c := range []struct {
		hex  string
		want int
	}{
		{"#ff0000", 9}, {"#ff8e7a", 9}, // red, coral
		{"#ffff00", 11}, {"#f2c14e", 11}, // yellow, amber
		{"#00ff00", 10},                  // green
		{"#4fd1c5", 14}, {"#00ffff", 14}, // cyan, teal
		{"#0000ff", 12}, {"#5b8cff", 12}, // blue
		{"#ff00ff", 13}, {"#ff5fd7", 13}, // magenta, pink
		{"#888888", 9}, // a grey has no angle: red's
	} {
		if got := NearestCell(c.hex); got != c.want {
			t.Errorf("NearestCell(%s) = %d, want %d", c.hex, got, c.want)
		}
	}
}

func TestHueCheck(t *testing.T) {
	for _, c := range []struct {
		name string
		h    Hue
		err  string
	}{
		{"coral", Hue{Dark: "#ff8e7a", Light: "#b4321f"}, ""},
		{"coral", Hue{Dark: "#ff8e7a", Light: "#b4321f", Cells: Cell(9)}, ""},
		{"coral", Hue{Dark: "#ff8e7a", Light: "#b4321f", Cells: Cell(0)}, ""},
		{"sea_green2", Hue{Value: "#2e8b57"}, ""},
		{"Coral", Hue{Value: "#fff"}, "a hue's name is a lower-case identifier"},
		{"sea-green", Hue{Value: "#fff"}, "a hue's name is a lower-case identifier"},
		{"2fast", Hue{Value: "#fff"}, "a hue's name is a lower-case identifier"},
		{"", Hue{Value: "#fff"}, "a hue's name is a lower-case identifier"},
		{"a", Hue{Dark: "#fff"}, "hue a takes dark and light, or value"},
		{"a", Hue{}, "hue a takes dark and light, or value"},
		{"a", Hue{Value: "#fff", Light: "#000"}, "hue a takes value, or dark and light, not both"},
		{"a", Hue{Dark: "red", Light: "#000"}, `hue a: dark is #rgb or #rrggbb, not "red"`},
		{"a", Hue{Value: "rgb(1 2 3)"}, `hue a: value is #rgb or #rrggbb`},
		{"a", Hue{Value: "#fff", Cells: Cell(16)}, "hue a: cells is 0 to 15, not 16"},
		{"a", Hue{Value: "#fff", Cells: Cell(-1)}, "hue a: cells is 0 to 15, not -1"},
	} {
		err := c.h.Check(c.name)
		switch {
		case c.err == "" && err != nil:
			t.Errorf("%s %+v: %v", c.name, c.h, err)
		case c.err != "" && (err == nil || !strings.Contains(err.Error(), c.err)):
			t.Errorf("%s %+v: %v, want %q", c.name, c.h, err, c.err)
		}
	}
}

// A contrast failure says its number the way Q-0001 has it, cut where it
// would round up to what is asked.
func TestWeakMessage(t *testing.T) {
	weak := Palette{"coral": {Dark: "#a33", Light: "#b4321f"}}.Contrast()
	if len(weak) != 1 {
		t.Fatalf("weak: %v", weak)
	}
	got := weak[0].String()
	want := "coral's dark #a33 is 2.1:1 on the dark ground; text needs 4.5"
	if got != want {
		t.Errorf("%q, want %q", got, want)
	}
	paper := Palette{"x": {Dark: "#ff8e7a", Light: "#fff"}}.Contrast()
	if len(paper) != 1 || paper[0].String() != "x's light #fff is 1.0:1 on paper; text needs 4.5" {
		t.Errorf("paper: %v", paper)
	}
	dot := Palette{"x": {Value: "#222"}}.ContrastAt(func(string) float64 { return DotContrast })
	if len(dot) == 0 || !strings.HasSuffix(dot[0].String(), "; a dot needs 3") {
		t.Errorf("a dot's: %v", dot)
	}
	// 4.49 shows as 4.4, never as the 4.5 that is asked.
	if s := (Weak{Hue: "h", Side: "dark", Colour: "#abc", Ratio: 4.496, Need: TextContrast}).String(); !strings.Contains(s, "is 4.4:1") {
		t.Errorf("cut: %s", s)
	}
}

func TestPaletteCSS(t *testing.T) {
	p := Palette{
		"teal":  {Dark: "#4fd1c5", Light: "#0f766e"},
		"coral": {Dark: "#ff8e7a", Light: "#b4321f", Cells: Cell(9)},
	}
	css := p.CSS()
	for _, want := range []string{
		".u-surface {\n  --u-hue-coral: #ff8e7a;\n  --u-hue-teal: #4fd1c5;\n}",
		"@media (prefers-color-scheme: light) {\n  .u-surface {\n    --u-hue-coral: #b4321f;\n    --u-hue-teal: #0f766e;\n  }\n}",
		".u-surface.u-theme-paper, .u-surface.u-theme-stone {\n  --u-hue-coral: #b4321f;\n  --u-hue-teal: #0f766e;\n}",
		".u-hue-coral { color: var(--u-hue-coral); }",
		".u-hue-teal { color: var(--u-hue-teal); }",
		// A hue's dot is a thing's own colour, 8px, where a state's is 6.
		".u-dot.u-hue-coral { --u-dw: 8px; }",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("the css lacks %q:\n%s", want, css)
		}
	}
	if strings.Contains(css, "-ink") || strings.Contains(css, "--u-dw: 8px; color") {
		t.Errorf("a palette that reads has no ink fallback:\n%s", css)
	}
	if (Palette{}).CSS() != "" || Palette(nil).CSS() != "" {
		t.Error("no hues, rules")
	}
}

// Where a hue's text can't be read on a ground, its rule is the ink there
// and the dot's keeps the hue; a hue that is no hue is left out whole.
func TestPaletteCSSFaint(t *testing.T) {
	p := Palette{
		"ok":    {Dark: "#ff8e7a", Light: "#b4321f"},
		"faint": {Dark: "#7a8499", Light: "#b4321f"},
		"junk":  {Value: "red}"},
	}
	css := p.CSS()
	for _, want := range []string{
		"  --u-hue-faint: #7a8499;\n  --u-hue-faint-ink: var(--u-ink);\n",
		"    --u-hue-faint: #b4321f;\n    --u-hue-faint-ink: var(--u-hue-faint);\n",
		".u-hue-faint { color: var(--u-hue-faint-ink); }",
		".u-dot.u-hue-faint { --u-dw: 8px; color: var(--u-hue-faint); }",
		".u-hue-ok { color: var(--u-hue-ok); }",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("the css lacks %q:\n%s", want, css)
		}
	}
	if strings.Contains(css, "junk") || strings.Contains(css, "red}") {
		t.Errorf("a hue that is none got into the css:\n%s", css)
	}
	if strings.Contains(css, "--u-hue-ok-ink") {
		t.Errorf("a hue that reads has an ink:\n%s", css)
	}
	faults := p.Faults()
	if len(faults) != 2 || !strings.Contains(faults[0], "junk") || !strings.Contains(faults[1], "faint's dark #7a8499 is 3.7:1 on the dark ground; text needs 4.5; its text draws in the ink there, its dots keep the hue") {
		t.Errorf("faults: %q", faults)
	}
	if f := (Palette{"dim": {Dark: "#a33", Light: "#b4321f"}}).Faults(); len(f) != 1 || !strings.HasSuffix(f[0], "its text draws in the ink there, and a dot in it is faint (it needs 3)") {
		t.Errorf("a faint dot: %q", f)
	}
}

func TestHueNode(t *testing.T) {
	n := Text("x").Hue("coral")
	if got := HueOf(n); got != "coral" {
		t.Errorf("HueOf = %q", got)
	}
	if got := Classes(n); len(got) != 2 || got[1] != "u-hue-coral" {
		t.Errorf("classes %v", got)
	}
	// A name that is no identifier is no class (it could add classes of its
	// own), and is still the hue the runtime warns of.
	bad := Text("x").Hue("a b")
	if got := Classes(bad); len(got) != 1 || HueOf(bad) != "a b" {
		t.Errorf("a bad name: classes %v, hue %q", got, HueOf(bad))
	}
	none := Text("x").Hue("")
	if len(Classes(none)) != 1 || HueOf(none) != "" {
		t.Errorf("no name: %v", Classes(none))
	}
	root := Stack(Row(Dot().Hue("coral"), Text("a").Hue("coral")), Glyph("●").Hue("teal"), Text("b").Hue("mauve"), Text("c"))
	if got := strings.Join(Hues(root), ","); got != "coral,teal,mauve" {
		t.Errorf("Hues = %s", got)
	}
	if html := HTML(Resolve0(root)); !strings.Contains(html, `class="u-glyph u-dot u-hue-coral"`) {
		t.Errorf("html: %s", html)
	}
}

// Resolve0 resolves root under "r" and returns it.
func Resolve0(n *Node) *Node { Resolve(n, "r"); return n }

func TestPaletteMergeEqualValid(t *testing.T) {
	a := Palette{"a": {Value: "#111"}, "b": {Dark: "#222", Light: "#333", Cells: Cell(3)}}
	b := Palette{"b": {Value: "#444"}, "c": {Value: "#555"}}
	m := a.Merge(b)
	if len(m) != 3 || m["b"].Value != "#444" || m["a"].Value != "#111" {
		t.Errorf("merge: %v", m)
	}
	if len(a) != 2 || a["b"].Dark != "#222" {
		t.Errorf("merge changed its receiver: %v", a)
	}
	if !a.Merge(nil).Equal(a) || !Palette(nil).Merge(a).Equal(a) {
		t.Error("merge with none")
	}
	if !a.Equal(Palette{"a": {Value: "#111"}, "b": {Dark: "#222", Light: "#333", Cells: Cell(3)}}) {
		t.Error("equal palettes are unequal")
	}
	if a.Equal(Palette{"a": {Value: "#111"}, "b": {Dark: "#222", Light: "#333", Cells: Cell(4)}}) || a.Equal(Palette{"a": a["a"]}) {
		t.Error("unequal palettes are equal")
	}
	good, errs := Palette{"a": {Value: "#111"}, "B": {Value: "#222"}}.Valid()
	if len(good) != 1 || len(errs) != 1 {
		t.Errorf("valid: %v %v", good, errs)
	}
}
