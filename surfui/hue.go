package surfui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
)

// Hues are the named colours a view draws data in: an agent's own, a
// user's tag. The look's tones stay the theme's; a hue is declared, by a
// <palette> in a .surf file or at run time (Palette), and the surface
// carries it as a custom property, --u-hue-<name>, which a theme may set
// again. Colours are hex, and appear nowhere but in a palette.

// The reference grounds a hue's contrast is checked against. The dark
// ground is the terminal's, which a host sets (--hotty-bg), so no value
// is certain: DarkGround is hotty-blitz's default theme's, which the
// fidelity gate draws on. A light value is checked on LightGround, the
// darkest light ground the kit has, Stone's (surf.css): what clears it
// clears paper's (PaperGround) too.
const (
	DarkGround  = "#282c34"
	PaperGround = "#fbfbfd"
	StoneGround = "#d2d2d2"
	LightGround = StoneGround
)

// The contrast a hue needs on both grounds (WCAG 2.x): text 4.5:1, a
// dot's fill, which is a graphic, 3:1.
const (
	TextContrast = 4.5
	DotContrast  = 3.0
)

// Hue is a named colour: a pair, one value for each ground (Dark, Light:
// the light one is drawn wherever the ground is light, on paper and under
// a light terminal), or one Value for both, which has to pass both. Cells
// is the terminal colour a cells host draws it in, 0 to 15; nil is the
// nearest chromatic one by hue angle (NearestCell). Colours are "#rgb" or
// "#rrggbb".
type Hue struct {
	Dark, Light string
	Value       string
	Cells       *int
}

// Cell is a pointer to n, for a Hue's Cells.
func Cell(n int) *int { return &n }

// Palette is a set of hues by name. A name is a lower-case identifier:
// a letter, then letters, digits and _.
type Palette map[string]Hue

// ValidHueName reports whether s is a name a hue may have.
func ValidHueName(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || i > 0 && (c >= '0' && c <= '9' || c == '_')) {
			return false
		}
	}
	return s != ""
}

// ParseHex reads "#rgb" or "#rrggbb".
func ParseHex(s string) (r, g, b uint8, ok bool) {
	if len(s) != 4 && len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, false
	}
	digit := func(c byte) (uint8, bool) {
		switch {
		case c >= '0' && c <= '9':
			return c - '0', true
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10, true
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10, true
		}
		return 0, false
	}
	var v [6]uint8
	n := (len(s) - 1) / 3
	for i := 0; i < len(s)-1; i++ {
		d, ok := digit(s[i+1])
		if !ok {
			return 0, 0, 0, false
		}
		v[i] = d
	}
	if n == 1 {
		return v[0] * 17, v[1] * 17, v[2] * 17, true
	}
	return v[0]<<4 | v[1], v[2]<<4 | v[3], v[4]<<4 | v[5], true
}

// ContrastOf is the WCAG contrast ratio of two hex colours, 1 to 21.
func ContrastOf(a, b string) (float64, error) {
	la, err := luminance(a)
	if err != nil {
		return 0, err
	}
	lb, err := luminance(b)
	if err != nil {
		return 0, err
	}
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05), nil
}

func luminance(s string) (float64, error) {
	r, g, b, ok := ParseHex(s)
	if !ok {
		return 0, fmt.Errorf("a colour is #rgb or #rrggbb, not %q", s)
	}
	lin := func(c uint8) float64 {
		v := float64(c) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b), nil
}

// ansiHues are the chromatic colours of the terminal's palette and their
// standard hue angles: red, yellow, green, cyan, blue, magenta. Each is
// ANSI's, at index i (1 to 6) and its bright one at i+8. A hue's default
// cells colour is the one of these nearest its own angle, the bright one:
// the colours text is drawn in on a dark terminal, as the tones' are.
var ansiHues = [...]struct {
	angle  float64
	bright int
}{
	{0, 9},    // red     1, 9
	{60, 11},  // yellow  3, 11
	{120, 10}, // green   2, 10
	{180, 14}, // cyan    6, 14
	{240, 12}, // blue    4, 12
	{300, 13}, // magenta 5, 13
}

// NearestCell is the terminal colour nearest the hex colour's hue angle:
// one of the bright chromatic ones, 9 to 14. A grey has no angle, and is
// taken as 0, red; write cells for such a hue.
func NearestCell(hex string) int {
	r, g, b, ok := ParseHex(hex)
	if !ok {
		return 9
	}
	angle := hueAngle(float64(r)/255, float64(g)/255, float64(b)/255)
	best, bestD := 9, 361.0
	for _, a := range ansiHues {
		d := math.Abs(angle - a.angle)
		if d > 180 {
			d = 360 - d
		}
		if d < bestD {
			best, bestD = a.bright, d
		}
	}
	return best
}

func hueAngle(r, g, b float64) float64 {
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	if d == 0 {
		return 0
	}
	var h float64
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h
}

// Pair is h's dark and light values: Value for both where it has one.
func (h Hue) Pair() (dark, light string) {
	if h.Value != "" {
		return h.Value, h.Value
	}
	return h.Dark, h.Light
}

// CellsIndex is the terminal colour h draws in, 0 to 15.
func (h Hue) CellsIndex() int {
	if h.Cells != nil {
		return *h.Cells
	}
	d, _ := h.Pair()
	return NearestCell(d)
}

// Check is the shape of the hue called name: its name, its colours (one
// Value, or a Dark and a Light) and its cells.
func (h Hue) Check(name string) error {
	if !ValidHueName(name) {
		return fmt.Errorf("a hue's name is a lower-case identifier, as coral or sea_green, not %q", name)
	}
	switch {
	case h.Value != "" && (h.Dark != "" || h.Light != ""):
		return fmt.Errorf("hue %s takes value, or dark and light, not both", name)
	case h.Value == "" && (h.Dark == "" || h.Light == ""):
		return fmt.Errorf("hue %s takes dark and light, or value", name)
	}
	for _, c := range [...]struct{ what, v string }{{"value", h.Value}, {"dark", h.Dark}, {"light", h.Light}} {
		if c.v == "" {
			continue
		}
		if _, _, _, ok := ParseHex(c.v); !ok {
			return fmt.Errorf("hue %s: %s is #rgb or #rrggbb, not %q", name, c.what, c.v)
		}
	}
	if h.Cells != nil && (*h.Cells < 0 || *h.Cells > 15) {
		return fmt.Errorf("hue %s: cells is 0 to 15, not %d", name, *h.Cells)
	}
	return nil
}

// Check is the first error of the palette's shape, the names in order.
func (p Palette) Check() error {
	for _, n := range p.names() {
		if err := p[n].Check(n); err != nil {
			return err
		}
	}
	return nil
}

func (p Palette) names() []string {
	out := make([]string, 0, len(p))
	for n := range p {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Names are the hues' names, in order.
func (p Palette) Names() []string { return p.names() }

// Valid is the hues of p that are hues (Hue.Check), and an error for each
// that isn't, in the names' order. It is p itself where all are.
func (p Palette) Valid() (Palette, []error) {
	var errs []error
	var bad []string
	for _, n := range p.names() {
		if err := p[n].Check(n); err != nil {
			errs = append(errs, err)
			bad = append(bad, n)
		}
	}
	if len(bad) == 0 {
		return p, nil
	}
	out := make(Palette, len(p))
	for n, h := range p {
		out[n] = h
	}
	for _, n := range bad {
		delete(out, n)
	}
	return out, errs
}

// Equal reports whether q has the same hues as p.
func (p Palette) Equal(q Palette) bool {
	if len(p) != len(q) {
		return false
	}
	for n, h := range p {
		o, ok := q[n]
		if !ok || h.Dark != o.Dark || h.Light != o.Light || h.Value != o.Value || (h.Cells == nil) != (o.Cells == nil) || h.Cells != nil && *h.Cells != *o.Cells {
			return false
		}
	}
	return true
}

// Weak is a colour of a hue that is too faint on its ground.
type Weak struct {
	Hue    string
	Side   string // "dark", "light" or "value"
	Colour string
	Paper  bool // the ground is the light one (LightGround), not the dark one
	Ratio  float64
	Need   float64 // what was asked of it: TextContrast, or DotContrast of a hue only dots use
}

// Dot is whether a dot in it is faint too (under 3:1).
func (w Weak) Dot() bool { return w.Ratio < DotContrast }

// String is the failure, with the number:
// "coral's dark #a33 is 2.9:1 on the dark ground; text needs 4.5". The
// number is cut, not rounded, so that a ratio short of the need never
// reads as the need.
func (w Weak) String() string {
	ground := "the dark ground"
	if w.Paper {
		ground = "stone"
	}
	need := "text needs 4.5"
	if w.Need == DotContrast {
		need = "a dot needs 3"
	}
	return fmt.Sprintf("%s's %s %s is %.1f:1 on %s; %s", w.Hue, w.Side, w.Colour, math.Floor(w.Ratio*10)/10, ground, need)
}

// Faults are what is wrong with a palette that is declared at run time,
// as the warnings that say it, once each is due: a hue that is no hue is
// left out; one whose text can't be read on a ground draws its text in
// the ink there, and its dots in the hue.
func (p Palette) Faults() []string {
	good, errs := p.Valid()
	var out []string
	for _, err := range errs {
		out = append(out, err.Error())
	}
	for _, w := range good.Contrast() {
		msg := "hue " + w.String() + "; its text draws in the ink there, its dots keep the hue"
		if w.Dot() {
			msg = "hue " + w.String() + "; its text draws in the ink there, and a dot in it is faint (it needs 3)"
		}
		out = append(out, msg)
	}
	return out
}

// Contrast lists each colour of the palette that text can't be read in,
// against the dark ground and the light one (Stone's): a dark value on the
// first, a light one on the second, a single value on both. It is the names' order. Call
// it on a palette that Check passes.
func (p Palette) Contrast() []Weak { return p.ContrastAt(nil) }

// ContrastAt is Contrast where need says what a hue is asked: TextContrast,
// or DotContrast for a hue that only dots are drawn in. Nil asks text of
// every hue.
func (p Palette) ContrastAt(need func(name string) float64) []Weak {
	var out []Weak
	for _, n := range p.names() {
		h := p[n]
		for _, c := range h.grounds() {
			ground := DarkGround
			if c.paper {
				ground = LightGround
			}
			want := TextContrast
			if need != nil {
				want = need(n)
			}
			if r, err := ContrastOf(c.colour, ground); err == nil && r < want {
				out = append(out, Weak{Hue: n, Side: c.side, Colour: c.colour, Paper: c.paper, Ratio: r, Need: want})
			}
		}
	}
	return out
}

type groundOf struct {
	side, colour string
	paper        bool
}

// grounds are the colours of h, each with the ground it is drawn on.
func (h Hue) grounds() []groundOf {
	if h.Value != "" {
		return []groundOf{{"value", h.Value, false}, {"value", h.Value, true}}
	}
	return []groundOf{{"dark", h.Dark, false}, {"light", h.Light, true}}
}

// CSS is the palette as rules to follow the stylesheet:
//   - each hue is a custom property on the surface, --u-hue-<name>: its
//     dark value, its light one where the ground is light (under
//     prefers-color-scheme: light, and the paper and stone themes), so that a theme
//     sets it again as it sets --u-accent, in a rule of its own;
//   - an element of the hue (class u-hue-<name>) is coloured from it, and
//     a dot of a hue is 8px, a thing's own colour (a machine's), where a
//     state's dot is 6.
//
// Where a colour is too faint for text on its ground (Contrast), the
// hue's text draws in the ink there, through --u-hue-<name>-ink; a dot
// keeps the hue. The palette's rules come after the stylesheet's, so that
// the hue wins over what an element is otherwise (an icon's muted).
func (p Palette) CSS() string {
	p, _ = p.Valid()
	if len(p) == 0 {
		return ""
	}
	weak := map[string][2]bool{} // the hue's text is faint: on the dark ground, on the light one
	for _, w := range p.Contrast() {
		f := weak[w.Hue]
		if w.Paper {
			f[1] = true
		} else {
			f[0] = true
		}
		weak[w.Hue] = f
	}
	var dark, light, rules strings.Builder
	decl := func(b *strings.Builder, name, colour string, faint bool) {
		fmt.Fprintf(b, "  --u-hue-%s: %s;\n", name, colour)
		if _, any := weak[name]; any {
			ink := "var(--u-hue-" + name + ")"
			if faint {
				ink = "var(--u-ink)"
			}
			fmt.Fprintf(b, "  --u-hue-%s-ink: %s;\n", name, ink)
		}
	}
	for _, n := range p.names() {
		d, l := p[n].Pair()
		f := weak[n]
		decl(&dark, n, d, f[0])
		decl(&light, n, l, f[1])
		if _, any := weak[n]; any {
			fmt.Fprintf(&rules, ".u-hue-%s { color: var(--u-hue-%s-ink); }\n.u-dot.u-hue-%s { --u-dw: 8px; color: var(--u-hue-%s); }\n", n, n, n, n)
		} else {
			fmt.Fprintf(&rules, ".u-hue-%s { color: var(--u-hue-%s); }\n.u-dot.u-hue-%s { --u-dw: 8px; }\n", n, n, n)
		}
	}
	var b strings.Builder
	b.WriteString("\n/* The palette's hues (surfui.Palette). */\n.u-surface {\n" + dark.String() + "}\n")
	b.WriteString("@media (prefers-color-scheme: light) {\n  .u-surface {\n" + indent(light.String()) + "  }\n}\n")
	b.WriteString(".u-surface.u-theme-paper, .u-surface.u-theme-stone {\n" + light.String() + "}\n")
	b.WriteString(rules.String())
	return b.String()
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(strings.TrimSuffix(s, "\n"), "\n", "\n  ") + "\n"
}

// Merge is p with q's hues over it: q's wins for a name both have. Either
// may be nil; the result is p or q itself where the other is empty.
func (p Palette) Merge(q Palette) Palette {
	switch {
	case len(q) == 0:
		return p
	case len(p) == 0:
		return q
	}
	out := make(Palette, len(p)+len(q))
	for n, h := range p {
		out[n] = h
	}
	for n, h := range q {
		out[n] = h
	}
	return out
}

// Hue sets the hue the text of n is drawn in, by name: a name the surface's
// palette declares, as the sheet colours it (u-hue-<name>), and the cells
// by its terminal colour. A name no palette declares draws the ink, and is
// the runtime's to warn of, once. "" is no hue. A hue and a Tone are
// exclusive; a node given a hue after a tone keeps both classes, so the
// vocabulary refuses the pair where it can.
func (n *Node) Hue(name string) *Node {
	n.hue = name
	if ValidHueName(name) {
		n.Class("u-hue-" + name)
	}
	return n
}

// HueOf is the hue n was given, "" for none.
func HueOf(n *Node) string { return n.hue }

// Hues are the names of the hues the tree under n uses, each once, in
// the order they first appear.
func Hues(n *Node) []string {
	var out []string
	var walk func(*Node)
	walk = func(n *Node) {
		stackprobe.Probe()
		if n.hue != "" {
			dup := false
			for _, h := range out {
				dup = dup || h == n.hue
			}
			if !dup {
				out = append(out, n.hue)
			}
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(n)
	return out
}
