package surf

import (
	"strings"
	"testing"
	"testing/fstest"

	cellite "neuroplast.io/cel-lite"
	"neuroplast.io/hotty-surf/surfui"
)

// Named colours (docs/surf.md, "Colours from data"): a <palette> in a
// .surf file, and hue= on text, a dot, a glyph and an icon.

var hueShape = &Shape{Name: "crew", Component: true,
	Fields: []Field{{Name: "colour", Type: cellite.String}, {Name: "title", Type: cellite.String}}}

func hueOf(colour string) dynObj {
	return dynObj{hueShape, []cellite.Value{cellite.StringValue(colour), cellite.StringValue("t")}}
}

const huePair = `<palette>
  <hue name="coral" dark="#ff8e7a" light="#a42d1c" cells="9"/>
  <hue name="teal" dark="#4fd1c5" light="#0d645d"/>
</palette>
`

func TestPaletteLoads(t *testing.T) {
	set, err := Load(fstest.MapFS{
		"crew.surf": {Data: []byte(huePair + `<surface><text hue="coral">a</text></surface>`)},
		"more.surf": {Data: []byte(`<palette><hue name="plum" dark="#c9a0ff" light="#6b3fa0"/></palette>`)},
	}, []*Shape{hueShape})
	if err != nil {
		t.Fatal(err)
	}
	p := set.Palette()
	if got := strings.Join(p.Names(), ","); got != "coral,plum,teal" {
		t.Errorf("hues %s", got)
	}
	if h := p["coral"]; h.Dark != "#ff8e7a" || h.Light != "#a42d1c" || h.CellsIndex() != 9 {
		t.Errorf("coral %+v", h)
	}
	// teal has no cells: the nearest chromatic colour by its angle (cyan).
	if got := p["teal"].CellsIndex(); got != 14 {
		t.Errorf("teal's cells %d, want 14", got)
	}
}

func TestHueRenders(t *testing.T) {
	set, err := Load(fstest.MapFS{"crew.surf": {Data: []byte(huePair + `<surface>
  <dot hue="coral"/>
  <text :hue="colour">x</text>
  <glyph :hue="colour">●</glyph>
  <icon name="cog-outline" :hue="colour"/>
</surface>`)}}, []*Shape{hueShape})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ colour, class string }{
		{"teal", "u-hue-teal"},
		{"mauve", "u-hue-mauve"}, // unknown: the class is there, no rule answers it
	} {
		root, errs := set.Render(hueOf(c.colour))
		if len(errs) != 0 {
			t.Fatalf("%s: %v", c.colour, errs)
		}
		html := surfui.HTML(root)
		for _, want := range []string{`class="u-glyph u-dot u-hue-coral"`, `class="u-text u-hue-` + c.colour[:1], `class="u-glyph u-hue-` + c.colour[:1], `class="u-text u-icon u-hue-` + c.colour[:1]} {
			if !strings.Contains(html, want) {
				t.Errorf("%s: no %q in %s", c.colour, want, html)
			}
		}
		if hues := surfui.Hues(root); len(hues) != 2 || hues[1] != c.colour {
			t.Errorf("%s: hues %v", c.colour, hues)
		}
	}
	// An empty name is no hue, and a name that is no identifier is no class.
	for _, colour := range []string{"", "a b", "Coral"} {
		root, errs := set.Render(hueOf(colour))
		if len(errs) != 0 {
			t.Fatalf("%q: %v", colour, errs)
		}
		html := surfui.HTML(root)
		if strings.Count(html, "u-hue-") != 1 { // the literal dot's
			t.Errorf("%q: %s", colour, html)
		}
	}
	// A bound hue that is no string is a render error, as a bound tone is.
	root, errs := set.Render(dynObj{&Shape{Name: "crew", Component: true, Fields: []Field{{Name: "colour", Type: cellite.Dyn}, {Name: "title", Type: cellite.String}}},
		[]cellite.Value{cellite.IntValue(3), cellite.StringValue("t")}})
	if root == nil || len(errs) == 0 {
		t.Errorf("a number as a hue: %v", errs)
	}
}

// refused loads template with the palette's file, and returns what it
// says.
func refused(t *testing.T, files fstest.MapFS) string {
	t.Helper()
	_, err := Load(files, []*Shape{hueShape})
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestHueRefusals(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
	}{
		{"tone and hue", `<surface><text tone="ok" hue="coral">x</text></surface>`, "crew.surf:5:26: <text> takes a tone or a hue, not both"},
		{"a bound tone and a literal hue", `<surface><text :tone="title" hue="coral">x</text></surface>`, "crew.surf:5:30: <text> takes a tone or a hue, not both"},
		{"a literal tone and a bound hue", `<surface><dot tone="ok" :hue="colour"/></surface>`, "crew.surf:5:25: <dot> takes a tone or a hue, not both"},
		{"an unknown name", `<surface><text hue="cora">x</text></surface>`, `crew.surf:5:21: hue "cora" is in no <palette>; the palettes declare coral or teal`},
		{"an empty name", `<surface><text hue="">x</text></surface>`, `crew.surf:5:21: hue "" is in no <palette>; the palettes declare coral or teal`},
		{"hex in a hue", `<surface><text hue="#ff8e7a">x</text></surface>`, `crew.surf:5:21: hue "#ff8e7a" is in no <palette>; the palettes declare coral or teal`},
		{"a button", `<surface><button hue="coral">x</button></surface>`, "crew.surf:5:18: <button> takes no hue; it takes attention, autofocus,"},
		{"a row", `<surface><row :hue="colour"/></surface>`, "crew.surf:5:15: <row> takes no hue; it takes autofocus,"},
		{"a chip", `<surface><chip hue="coral">x</chip></surface>`, "crew.surf:5:16: <chip> takes no hue; it takes autofocus,"},
		{"a bound number", `<surface><text :hue="1">x</text></surface>`, "crew.surf:5:22: :hue is a string, not int"},
	} {
		got := refused(t, fstest.MapFS{"crew.surf": {Data: []byte(huePair + c.src)}})
		if !strings.HasPrefix(got, c.want) {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
	// No palette at all: the literal says so.
	if got := refused(t, fstest.MapFS{"crew.surf": {Data: []byte(`<surface><text hue="coral">x</text></surface>`)}}); !strings.Contains(got, `hue "coral" is in no <palette>: no template declares one`) {
		t.Errorf("no palette: %s", got)
	}
	// A palette in a file declared before it is used, or after, in any file.
	if got := refused(t, fstest.MapFS{
		"crew.surf": {Data: []byte(`<surface><text hue="coral">x</text></surface>`)},
		"zz.surf":   {Data: []byte(huePair)},
	}); got != "" {
		t.Errorf("a palette in another file: %s", got)
	}
}

// What the load says of a palette that can't be read, with the number.
func TestPaletteContrast(t *testing.T) {
	dot := `<palette><hue name="slate" value="#757575"/></palette>` // 3.0 and 3.0: a dot's, not text
	for _, c := range []struct {
		name, src, want string
	}{
		{"dots only, a literal", dot + `<surface><dot hue="slate"/></surface>`, ""},
		{"dots only, bound", dot + `<surface><dot :hue="colour"/></surface>`, ""},
		{"a literal on text", dot + `<surface><dot hue="slate"/><text hue="slate">x</text></surface>`, "crew.surf:1:10: slate's value #757575 is 3.0:1 on the dark ground; text needs 4.5\ncrew.surf:1:10: slate's value #757575 is 3.0:1 on stone; text needs 4.5"},
		{"a bound hue on text asks text of all", dot + `<surface><dot hue="slate"/><text :hue="colour">x</text></surface>`, "crew.surf:1:10: slate's value #757575 is 3.0:1 on the dark ground; text needs 4.5\ncrew.surf:1:10: slate's value #757575 is 3.0:1 on stone; text needs 4.5"},
		{"an icon and a glyph are not dots", dot + `<surface><icon name="cog-outline" hue="slate"/></surface>`, "slate's value #757575 is 3.0:1 on the dark ground; text needs 4.5"},
		{"a hue no template draws asks text", dot + `<surface><dot hue="slate"/></surface>` + "\n", ""},
		{"too faint for a dot", `<palette><hue name="faint" value="#555"/></palette><surface><dot hue="faint"/></surface>`, "faint's value #555 is 1.8:1 on the dark ground; a dot needs 3"},
		{"a pair that reads", huePair + `<surface><text :hue="colour">x</text></surface>`, ""},
		{"the dark one is faint", `<palette><hue name="a" dark="#a33" light="#a42d1c"/></palette><surface><text hue="a">x</text></surface>`,
			"crew.surf:1:10: a's dark #a33 is 2.1:1 on the dark ground; text needs 4.5"},
		{"the light one is faint", `<palette><hue name="a" dark="#ff8e7a" light="#ffeedd"/></palette><surface><text hue="a">x</text></surface>`,
			"crew.surf:1:10: a's light #ffeedd is 1.3:1 on stone; text needs 4.5"},
	} {
		got := refused(t, fstest.MapFS{"crew.surf": {Data: []byte(c.src)}})
		if c.want == "" && got != "" || !strings.Contains(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
	// The one that only dots use is no other file's: a second file's text
	// hue= doesn't make it text, a bound one does.
	if got := refused(t, fstest.MapFS{
		"crew.surf": {Data: []byte(dot + `<surface><dot hue="slate"/><text hue="teal">x</text></surface>`)},
		"b.surf":    {Data: []byte(`<palette><hue name="teal" dark="#4fd1c5" light="#0d645d"/></palette>`)},
	}); got != "" {
		t.Errorf("a dot's hue beside a text's: %s", got)
	}
}

// A palette alone is no template, and a template of a component draws one.
func TestPaletteOnlyFile(t *testing.T) {
	files := fstest.MapFS{
		"crew.surf":    {Data: []byte(`<surface><text hue="sky">x</text></surface>`)},
		"palette.surf": {Data: []byte(`<palette><hue name="sky" dark="#7cc4ff" light="#0a5b9c"/></palette>`)},
	}
	set, err := Load(files, []*Shape{hueShape})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(set.Palette().Names(), ","); got != "sky" {
		t.Errorf("hues %s", got)
	}
	if err := refused(t, fstest.MapFS{"palette.surf": files["palette.surf"], "crew.surf": {Data: []byte(`<surface><palette/></surface>`)}}); !strings.Contains(err, "<palette> comes before the root element") && !strings.Contains(err, "no widget") {
		t.Errorf("a palette in a tree: %s", err)
	}
}
