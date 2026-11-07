package surflua

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
	lua "github.com/yuin/gopher-lua"
)

// stderr is what a test's run writes of its warnings.
func stderr(t *testing.T) *strings.Builder {
	var b strings.Builder
	was := surfrun.Stderr
	surfrun.Stderr = &b
	t.Cleanup(func() { surfrun.Stderr = was })
	return &b
}

const hueFiles = `local surf = require("surf")
crew = { agents = { { name = "ada", colour = "coral" }, { name = "bo", colour = "ochre" }, { name = "cy", colour = "mauve" } } }
view = surf.load("."):view("crew", crew)
`

// A palette in a template, the one surf.palette declares at run time, a
// literal hue and a bound one, in a script.
func TestLuaHues(t *testing.T) {
	warned := stderr(t)
	write(t, map[string]string{
		"crew.surf": `<palette>
  <hue name="coral" dark="#ff8e7a" light="#a42d1c" cells="9"/>
</palette>
<surface>
  <text hue="coral">literal</text>
  <row :each="a in agents" :key="a.name"><dot :hue="a.colour"/><text :hue="a.colour">{{ a.name }}</text></row>
</surface>`,
	})
	L := state(t)
	runChunk(t, L, "/t/crew.lua", hueFiles)

	html := render(t, L)
	for _, want := range []string{
		`<span id="r.0" class="u-text u-hue-coral">literal</span>`,
		`<span id="r.kada.0" class="u-glyph u-dot u-hue-coral">`,
		`<span id="r.kbo.1" class="u-text u-hue-ochre">bo</span>`,
		`<span id="r.kcy.1" class="u-text u-hue-mauve">cy</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the markup lacks %q:\n%s", want, html)
		}
	}
	// ochre and mauve are in no palette yet: each is warned of once, however
	// often the view draws.
	render(t, L)
	render(t, L)
	if got := warned.String(); strings.Count(got, "is in no palette") != 2 || !strings.Contains(got, `hue "ochre"`) || !strings.Contains(got, `hue "mauve"`) {
		t.Errorf("warnings:\n%s", got)
	}

	// Declared at run time, ochre is one: the stylesheet it returns is the
	// floor's with every hue's rules, and surf.css is that now. The
	// template's coral stays.
	r := eval(t, L, `local surf = require("surf")
		local css = surf.palette{ ochre = { dark = "#f2c14e", light = "#7b5000", cells = 3 } }
		return css, surf.css`)
	css := r[0].String()
	if r[1].String() != css || !strings.HasPrefix(css, surfui.CSS) {
		t.Error("surf.css is not the stylesheet surf.palette returned, the floor's and the hues'")
	}
	if !strings.Contains(css, "--u-hue-ochre: #f2c14e;") || !strings.Contains(css, ".u-hue-ochre { color: var(--u-hue-ochre); }") {
		t.Errorf("no rules for ochre:\n%s", css[len(surfui.CSS):])
	}
	v, ok := ViewOf(L.GetGlobal("view"))
	if !ok {
		t.Fatal("no view")
	}
	if got := strings.Join(v.Palette().Names(), ","); got != "coral,ochre" {
		t.Errorf("the view's hues %s, want the template's and the script's", got)
	}
	render(t, L)
	if got := warned.String(); strings.Count(got, "is in no palette") != 2 {
		t.Errorf("ochre is declared, and still warned of:\n%s", got)
	}

	// Declared again, it replaces the set; a hue whose text can't be read
	// warns once, however often it is declared.
	faint := `require("surf").palette{ plum = { value = "#757575" } }`
	for range 2 {
		r = eval(t, L, "return "+faint)
	}
	if strings.Contains(r[0].String(), "ochre") || !strings.Contains(r[0].String(), "--u-hue-plum-ink: var(--u-ink);") {
		t.Errorf("the set is not replaced, or plum has no ink:\n%s", r[0].String()[len(surfui.CSS):])
	}
	if got := strings.Join(v.Palette().Names(), ","); got != "coral,plum" {
		t.Errorf("the view's hues %s", got)
	}
	want := "surf: hue plum's value #757575 is 3.0:1 on the dark ground; text needs 4.5; its text draws in the ink there, its dots keep the hue\n"
	if got := warned.String(); strings.Count(got, want) != 1 || !strings.Contains(got, "is 3.0:1 on stone") {
		t.Errorf("warnings:\n%s", got)
	}

	// A hue that is no hue raises, and declares nothing.
	for _, c := range []struct{ src, want string }{
		{`{ x = { dark = "red", light = "#fff" } }`, `hue x: dark is #rgb or #rrggbb, not "red"`},
		{`{ X = { value = "#fff" } }`, "a hue's name is a lower-case identifier"},
		{`{ x = { dark = "#fff" } }`, "hue x takes dark and light, or value"},
		{`{ x = { value = "#fff", cells = 16 } }`, "cells is 0 to 15, not 16"},
		{`{ x = { value = "#fff", cells = 1.5 } }`, "cells is a whole number"},
		{`{ x = { value = 3 } }`, "value is a string"},
		{`{ x = "#fff" }`, "a hue is name = { dark"},
	} {
		err := L.DoString(`require("surf").palette(` + c.src + `)`)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.src, err, c.want)
		}
	}
	if got := strings.Join(v.Palette().Names(), ","); got != "coral,plum" {
		t.Errorf("a refused palette changed the hues: %s", got)
	}
	_ = lua.LNil
}
