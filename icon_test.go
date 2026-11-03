package surf

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

const (
	logoSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" data-glyph="hexagon-outline">
  <title>Logo</title>
  <path d="M12 2L21 7V17L12 22L3 17V7Z"/>
</svg>`
	bareSVG = `<svg viewBox="0 0 24 24"><path d="M3 3H21V21H3Z"/></svg>`
)

// iconSet is src as the board's template, with the assets logo.svg and
// bare.svg (which names no glyph) beside it.
func iconSet(src string) (*Set, error) {
	return Load(fstest.MapFS{
		"board.surf": {Data: []byte(src)},
		"logo.svg":   {Data: []byte(logoSVG)},
		"bare.svg":   {Data: []byte(bareSVG)},
	}, []*Shape{boardShape, noteShape})
}

// An <icon> draws its glyph, a literal name's resolved at load; with an
// svg, a host draws it, and the cells the glyph.
func TestIcon(t *testing.T) {
	for _, c := range []struct {
		src, sel   string
		html, cell string
		err        string
	}{
		{src: "<icon>\U000F08BB</icon>", cell: "\U000F08BB",
			html: "<span class=\"u-text u-icon\">\U000F08BB</span>"},
		{src: `<icon name="cog-outline"/>`, cell: "\U000F08BB",
			html: "<span class=\"u-text u-icon\">\U000F08BB</span>"},
		{src: `<icon name="cog-outline" tone="ok"/>`, cell: "\U000F08BB", // the tone class is the icon's colour (surf.css)
			html: "<span class=\"u-text u-icon u-tone-ok\">\U000F08BB</span>"},
		{src: `<icon name="cog-outline" path="M1 1H23V23H1Z" tone="warn"/>`, cell: "\U000F08BB",
			html: `<span class="u-text u-icon u-svg u-tone-warn"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M1 1H23V23H1Z"></path></svg></span>`},
		{src: `<icon :name="sel"/>`, sel: "restart", cell: "\U000F0709"},
		{src: `<icon :name="sel"/>`, sel: "no-such", cell: "\U000F078B",
			err: `board.surf:1:1: an icon's name is one of MDI's, as cog-outline, not "no-such"`},
		{src: `<icon name="cog-outline" path="M1 1H23V23H1Z"/>`, cell: "\U000F08BB",
			html: `<span class="u-text u-icon u-svg"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M1 1H23V23H1Z"></path></svg></span>`},
		{src: `<icon svg="logo"/>`, cell: "\U000F02D9",
			html: `<span class="u-text u-icon u-svg"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2L21 7V17L12 22L3 17V7Z"></path></svg></span>`},
		{src: `<icon svg="logo" name="cog-outline"/>`, cell: "\U000F08BB"}, // the element's wins
		{src: `<icon svg="bare" name="close"/>`, cell: "\U000F0156"},
		{src: `<icon :svg="sel"/>`, sel: "logo", cell: "\U000F02D9"},
		{src: `<icon :svg="sel" name="close"/>`, sel: "logo", cell: "\U000F0156"},
		{src: `<icon :svg="sel"/>`, sel: "bare", cell: "\U000F078B"}, // no glyph: help-box
		{src: `<icon :svg="sel"/>`, sel: "nope", cell: "\U000F078B",
			err: `board.surf:1:1: no svg icon "nope": an svg is an .svg file beside the templates`},
		{src: `<icon name="close" :path="sel"/>`, sel: "M0 0<script>", cell: "\U000F0156",
			err: `board.surf:1:1: a path is an SVG path's data, as "M12 2L2 22h20Z", not "M0 0<script>"`},
	} {
		set, err := iconSet(c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		b := newBoard()
		b.Sel = c.sel
		n, errs := set.Render(boardObj{b})
		gotErr := ""
		if len(errs) > 0 {
			gotErr = errs[0].Error()
		}
		if gotErr != c.err || len(errs) > 1 {
			t.Errorf("%s: errors %v, want %q", c.src, errs, c.err)
		}
		if c.html != "" {
			surfui.Resolve(n, "r")
			if got := stripID(surfui.HTML(n)); got != c.html {
				t.Errorf("%s:\n got %s\nwant %s", c.src, got, c.html)
			}
		}
		r := cells.New(surfrun.NewSurface(&iconModel{set, b}, nil).Cells())
		r.Show()
		if got := strings.TrimSpace(r.Draw(40, 4).Text()); got != c.cell {
			t.Errorf("%s: cells %q, want %q", c.src, got, c.cell)
		}
	}
}

// stripID is html without its ids.
func stripID(html string) string { return idAttr.ReplaceAllString(html, "") }

var idAttr = regexp.MustCompile(` id="[^"]*"`)

type iconModel struct {
	set *Set
	b   *board
}

func (m *iconModel) Render() *surfui.Node {
	n, _ := m.set.Render(boardObj{m.b})
	return n
}

func (m *iconModel) Do(surfui.Action, surfrun.Event) {}

// A button holds one <icon>, before its text or after it: the cells draw
// its glyph, a space, then the text; one of an icon alone takes a label,
// which a host reads out and the cells don't draw.
func TestButtonIcon(t *testing.T) {
	for _, c := range []struct{ src, cell, html string }{
		{`<button><icon name="restart"/>Restart</button>`, "\U000F0709 Restart",
			"<button class=\"u-btn u-btn-icon\" type=\"button\"><span class=\"u-text u-icon\">\U000F0709</span>Restart</button>"},
		{`<button>Next <icon name="arrow-right"/></button>`, "Next \U000F0054",
			"<button class=\"u-btn u-btn-icon\" type=\"button\">Next<span class=\"u-text u-icon\">\U000F0054</span></button>"},
		{`<button label="Close"><icon name="close"/></button>`, "\U000F0156",
			"<button class=\"u-btn u-btn-icon\" type=\"button\" aria-label=\"Close\"><span class=\"u-text u-icon\">\U000F0156</span></button>"},
		{`<button :label="title"><icon svg="logo"/></button>`, "\U000F02D9", ""},
		{`<button><icon :if="sel == 'x'" name="close"/>Close</button>`, "Close",
			`<button class="u-btn" type="button">Close</button>`},
	} {
		set, err := iconSet(c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		b := newBoard()
		n, errs := set.Render(boardObj{b})
		if len(errs) > 0 {
			t.Errorf("%s: %v", c.src, errs)
		}
		if c.html != "" {
			surfui.Resolve(n, "r")
			if got := stripID(surfui.HTML(n)); got != c.html {
				t.Errorf("%s:\n got %s\nwant %s", c.src, got, c.html)
			}
		}
		r := cells.New(surfrun.NewSurface(&iconModel{set, b}, nil).Cells())
		r.Show()
		if got := strings.TrimSpace(r.Draw(40, 4).Text()); !strings.Contains(got, c.cell) {
			t.Errorf("%s: cells %q, want %q in them", c.src, got, c.cell)
		}
	}
}

// What an <icon> refuses, and a button's, and an .svg file's.
func TestIconErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`<icon name="cog_outline"/>`, `board.surf:1:1: an icon's name is one of MDI's, as cog-outline, not "cog_outline"`},
		{`<icon name="cog-outline">x</icon>`, `board.surf:1:1: an <icon> is its glyph, &#xF08BB;, or takes name=, svg= or path=, not both`},
		{`<icon svg="logo" path="M1 1Z" name="close"/>`, `board.surf:1:1: an <icon> takes svg= or path=, not both`},
		{`<icon path="M1 1H23V23H1Z"/>`, `board.surf:1:1: an svg icon needs a glyph for the cells: name=…`},
		{`<icon :path="sel"/>`, `board.surf:1:1: an svg icon needs a glyph for the cells: name=…`},
		{`<icon svg="bare"/>`, `board.surf:1:1: an svg icon needs a glyph for the cells: name=…, or data-glyph in bare.svg`},
		{`<icon svg="nope" name="close"/>`, `board.surf:1:1: no svg icon "nope": an svg is an .svg file beside the templates, nope.svg`},
		{`<icon name="close" path="M1 1 url(x)"/>`, `board.surf:1:1: a path is an SVG path's data, as "M12 2L2 22h20Z", not "M1 1 url(x)"`},
		{`<icon name="close"><b/></icon>`, `board.surf:1:20: <icon> holds text, not <b>`},
		{`<text><icon name="close"/></text>`, `board.surf:1:7: <text> holds text, not <icon>`},
		{`<button><icon name="close"/><icon name="restart"/>X</button>`, `board.surf:1:29: a <button> has one <icon>; this is a second`},
		{`<button>A<icon name="close"/>B</button>`, `board.surf:1:30: a <button>'s <icon> goes before its text or after it, not in it`},
		{`<button><icon name="close"/></button>`, `board.surf:1:1: a <button> of an <icon> alone takes label=, which a host reads out: <button label="Close">`},
		{`<button><icon :each="n in notes" name="close"/>X</button>`, `board.surf:1:15: a <button> has one <icon>, not :each`},
		{`<icon name="close" tone="loud"/>`, `board.surf:1:26: a tone is accent, ok, warn, error, info, muted or plain, not "loud"`},
	} {
		_, err := iconSet(c.src)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.src, err, c.want)
		}
	}
}

func TestSVGFile(t *testing.T) {
	a, err := parseSVG("logo.svg", logoSVG)
	if err != nil || a.Path != "M12 2L21 7V17L12 22L3 17V7Z" || a.Glyph != "\U000F02D9" {
		t.Errorf("logo.svg: %+v %v", a, err)
	}
	for _, c := range []struct{ src, want string }{
		{`<svg viewBox="0 0 16 16"><path d="M1 1Z"/></svg>`, `x.svg:1:15: an svg icon's viewBox is 0 0 24 24, not "0 0 16 16"`},
		{`<svg><path d="M1 1Z"/></svg>`, `x.svg:1:1: an svg icon's box is 24×24: viewBox="0 0 24 24"`},
		{`<svg viewBox="0 0 24 24"><path d="M1 1Z"/><path d="M2 2Z"/></svg>`, `x.svg:1:43: an svg icon holds one <path>; this is a second: join them into one`},
		{`<svg viewBox="0 0 24 24"><circle r="3"/></svg>`, `x.svg:1:26: an svg icon holds one <path>, not <circle>: draw it as one path`},
		{`<svg viewBox="0 0 24 24"><path d="M1 1Z" fill-rule="evenodd"/></svg>`, `x.svg:1:42: <path> takes no fill-rule; it takes d or id`},
		{`<svg viewBox="0 0 24 24" style="x"><path d="M1 1Z"/></svg>`, `x.svg:1:26: <svg> takes no style; it takes data-glyph, fill, height, id, version, viewBox, width, xmlns or xmlns:xlink`},
		{`<svg viewBox="0 0 24 24" data-glyph="gearwheel"><path d="M1 1Z"/></svg>`, `x.svg:1:38: data-glyph is one of MDI's names, as cog-outline, not "gearwheel"`},
		{`<svg viewBox="0 0 24 24"></svg>`, `x.svg:1:1: an svg icon holds one <path d="M…">`},
		{`<svg viewBox="0 0 24 24"><path/></svg>`, `x.svg:1:26: <path> has its data, d="M…"`},
	} {
		if _, err := parseSVG("x.svg", c.src); err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.src, err, c.want)
		}
	}
}

// A button's click target is the whole of it, its icon and its text: a
// press on its glyph runs its handler, as one on its text's last letter.
func TestButtonIconHit(t *testing.T) {
	set, err := iconSet(`<surface><row><button @click="open('b')"><icon name="restart"/>Restart</button></row></surface>`)
	if err != nil {
		t.Fatal(err)
	}
	m := &renderOf{set: set, b: newBoard()}
	r := cells.New(surfrun.NewSurface(m, nil).Cells())
	r.Show()
	sc := r.Draw(40, 4)
	for y, l := range strings.Split(sc.Text(), "\n") {
		line := []rune(l)
		g := slices.Index(line, '\U000F0709')
		if g < 0 {
			continue
		}
		end := g + len([]rune(" Restart")) // its last letter
		if end >= len(line) || line[end] != 't' {
			t.Fatalf("row %d: %q", y, l)
		}
		for _, x := range []int{g, g + 1, end} {
			if sc.HitAt(x, y) == "" || sc.HitAt(x, y) != sc.HitAt(g, y) {
				t.Errorf("x %d: hit %q, the glyph's %q", x, sc.HitAt(x, y), sc.HitAt(g, y))
			}
			m.b.Log = nil
			r.Press(x, y, 0)
			r.Release(x, y, 0)
			if len(m.b.Log) != 1 || m.b.Log[0] != "open b" {
				t.Errorf("x %d: a click ran %v", x, m.b.Log)
			}
		}
		return
	}
	t.Fatalf("no glyph in\n%s", sc.Text())
}
