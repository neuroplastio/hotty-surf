package surf

import (
	_ "embed"
	cmdflag "flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The spec's tables (docs/spec.md, "Widgets" and "Attributes") are
// written from the vocabulary, Widgets and Attrs, and their HTML from
// what each tag's canonical instance renders now. This test fails where
// the spec says otherwise; -update writes the tables again.

var updateSpec = cmdflag.Bool("update", false, "write docs/spec.md's Widgets and Attributes tables again")

const specFile = "docs/spec.md"

// specDoc is the spec, embedded: TinyGo's wasm, where the package's tests
// also run (make wasm), has no file system to read it from. -update
// writes the file, for the next build.
//
//go:embed docs/spec.md
var specDoc string

// canonWidgets is each tag in a canonical instance: its HTML is the
// table's.
var canonWidgets = map[string]string{
	"surface": `<surface/>`,
	"stack":   `<stack/>`,
	"row":     `<row/>`,
	"spacer":  `<spacer/>`,
	"text":    `<text>Hi</text>`,
	"line":    `<line><text muted>Pane: </text><text>New</text></line>`,
	"heading": `<heading>Agents</heading>`,
	"caption": `<caption>Panes</caption>`,
	"chip":    `<chip>3</chip>`,
	"glyph":   `<glyph>●</glyph>`,
	"icon":    "<icon>\U000F08BB</icon>",
	"dot":     `<dot tone="ok"/>`,
	"key":     `<key>ctrl+b v</key>`,
	"button":  `<button>Save</button>`,
	"field":   `<field/>`,
	"switch":  `<switch on/>`,
	"select":  `<select value="b" :options="['a', 'b']"/>`,
	"cycle":   `<cycle value="b" :options="['a', 'b']"/>`,
	"list":    `<list/>`,
	"item":    `<item/>`,
	"section": `<section heading="Agents"/>`,
	"rule":    `<rule/>`,
	"box":     `<box/>`,
	"dialog":  `<dialog/>`,
	"face":    `<face icon="source-branch" value="main" unit="+2"/>`,
}

// canonAttrs is each attribute in a canonical instance, by its name and,
// where it is only on some tags, the first of them.
var canonAttrs = map[string]string{
	"drag/surface":      `<item drag/>`,
	"drop/surface":      `<item drop="halves"/>`,
	"on/switch":         `<switch on/>`,
	"value/select":      `<select value="b" :options="['a', 'b']"/>`,
	"options/select":    `<select :options="['a', 'b']"/>`,
	"heading/section":   `<section heading="Agents"/>`,
	"name/icon":         `<icon name="cog-outline"/>`,
	"svg/icon":          `<icon svg="spec"/>`,
	"path/icon":         `<icon name="hexagon-outline" path="M12 2L21 7V17L12 22L3 17V7Z"/>`,
	"icon/face":         `<face icon="memory" value="23%"/>`,
	"unit/face":         `<face value="84" unit="°C"/>`,
	"meter/face":        `<face icon="memory" value="23%" meter="23"/>`,
	"stale/face":        `<face icon="server-off" value="—" unit="offline" stale/>`,
	"popover/box":       `<box popover/>`,
	"theme/surface":     `<surface theme="paper"/>`,
	"density/surface":   `<surface density="dense"/>`,
	"key":               `<row><text key="a">x</text></row>`,
	"tone":              `<text tone="warn">x</text>`,
	"hue/text":          `<text :hue="'coral'">x</text>`,
	"fill":              `<chip fill="ok">x</chip>`,
	"bold":              `<text bold>x</text>`,
	"semibold":          `<text semibold>x</text>`,
	"muted":             `<text muted>x</text>`,
	"small":             `<text small>x</text>`,
	"wrap":              `<text wrap>x</text>`,
	"grow":              `<text grow>x</text>`,
	"keep":              `<row keep/>`,
	"fit/row":           `<row fit/>`,
	"fit-start/row":     `<row fit-start/>`,
	"readline/surface":  `<surface readline/>`,
	"program/surface":   `<surface program="ctrl+b"/>`,
	"gap":               `<row gap="2"/>`,
	"pad":               `<box pad="2 1"/>`,
	"indent":            `<text indent="2">x</text>`,
	"down":              `<box down="4"/>`,
	"top":               `<row top/>`,
	"width":             `<chip width="6">x</chip>`,
	"height":            `<stack height="3"/>`,
	"selected":          `<item selected/>`,
	"checked":           `<switch checked/>`,
	"disabled":          `<button disabled>x</button>`,
	"invalid":           `<field invalid/>`,
	"value/field":       `<field value="hi"/>`,
	"placeholder/field": `<field placeholder="Find"/>`,
	"name":              `<field name="q"/>`,
	"label":             `<button label="Close">x</button>`,
	"title":             `<text title="Copies the row">x</text>`,
	"reveal":            `<button reveal>x</button>`,
	"focusable":         `<list focusable/>`,
	"bare":              `<button bare>x</button>`,
	"chrome":            `<stack chrome/>`,
	"pill/button":       `<button pill>app</button>`,
	"raised/button":     `<button pill raised>main</button>`,
	"current/button":    `<button pill current>editor</button>`,
	"badge/button":      `<button pill badge="3">Tests</button>`,
	"badge-dot/button":  `<button pill badge-dot>Files</button>`,
	"badge-tone/button": `<button pill badge="1" badge-tone="warn">Agents</button>`,
	"card/box":          `<box card/>`,
	"ring/dot":          `<dot ring/>`,
	"target/dot":        `<dot target/>`,
	"pulse/dot":         `<dot pulse/>`,
	"search/row":        `<row search><field bare/></row>`,
	"attention/button":  `<button pill attention>build</button>`,
	"cap/key":           `<key cap>ctrl+b</key>`,
	"strip/row":         `<row strip><row/><row/><row/></row>`,
	"grid/row":          `<row grid><button>a</button></row>`,
	"segmented/row":     `<row segmented><button selected>All</button><button>Errors</button></row>`,
	"nav/list":          `<list nav/>`,
	"entries/list":      `<list entries><item/></list>`,
	"scale":             `<box scale="80"/>`,
	"scroll":            `<box scroll/>`,
	"autofocus":         `<list autofocus/>`,
}

// specSVG is the asset svg="spec" names.
const specSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" data-glyph="hexagon-outline"><path d="M12 2L21 7V17L12 22L3 17V7Z"/></svg>`

func attrKey(a *Attr) string {
	if len(a.Tags) > 0 {
		return a.Name + "/" + a.Tags[0]
	}
	return a.Name
}

// specCase is the component a canonical instance is the template of: no
// fields, no methods.
type specCase struct{}

var specShape = &Shape{Name: "specCase", Component: true}

func (specCase) Shape() *Shape                           { return specShape }
func (specCase) Field(int) cellite.Value                 { return cellite.NullValue() }
func (specCase) Call(int, []cellite.Value) cellite.Value { return cellite.NullValue() }

// renderCanon is src's HTML, rendered as a template's root with its ids
// resolved.
func renderCanon(src string) (string, error) {
	set, err := Load(fstest.MapFS{
		TemplateName(specShape.Name) + ".surf": {Data: []byte(src)},
		"spec.svg":                             {Data: []byte(specSVG)},
	}, []*Shape{specShape})
	if err != nil {
		return "", err
	}
	n, errs := set.Render(specCase{})
	if len(errs) > 0 {
		return "", fmt.Errorf("%v", errs)
	}
	surfui.Resolve(n, "r")
	return surfui.HTML(n), nil
}

func formName(f Form) string {
	switch f {
	case FormKids:
		return "kids"
	case FormText:
		return "text"
	case FormNone:
		return "none"
	case FormSwitch:
		return "switch"
	case FormChoice:
		return "choice"
	case FormSection:
		return "section"
	case FormIcon:
		return "icon"
	case FormFace:
		return "face"
	}
	return "?"
}

func typeText(t Type) string {
	switch t {
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeStrings:
		return "list(string)"
	case TypePad:
		return "pad"
	}
	return "?"
}

// code is s as a Markdown code span in a table's cell.
func code(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	if strings.Contains(s, "`") {
		return "`` " + s + " ``"
	}
	return "`" + s + "`"
}

func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// takes is what a widget is built from: its constructor's arguments,
// then its children or its text.
func takes(w *Widget) string {
	var args []string
	for _, a := range Attrs {
		if a.Arg && contains(a.Tags, w.Tag) {
			args = append(args, code(a.Name))
		}
	}
	switch w.Form {
	case FormKids, FormSection:
		args = append(args, "children")
	case FormText:
		args = append(args, "text")
		if w.Tag == "button" {
			args = append(args, "an `<icon>`")
		}
	case FormIcon:
		return "text or `name`, and `svg` or `path`"
	}
	if len(args) == 0 {
		return "nothing"
	}
	return strings.Join(args, ", ")
}

func widgetTable(t *testing.T) string {
	var b strings.Builder
	b.WriteString("| tag | form | takes | events | what | canonical instance | HTML |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for i := range Widgets {
		w := &Widgets[i]
		src, ok := canonWidgets[w.Tag]
		if !ok {
			t.Errorf("<%s> has no canonical instance in spec_test.go's canonWidgets", w.Tag)
			continue
		}
		if !strings.HasPrefix(src, "<"+w.Tag+">") && !strings.HasPrefix(src, "<"+w.Tag+" ") && !strings.HasPrefix(src, "<"+w.Tag+"/") {
			t.Errorf("<%s>'s canonical instance %s is not a <%s>", w.Tag, src, w.Tag)
		}
		html, err := renderCanon(src)
		if err != nil {
			t.Errorf("<%s>'s canonical instance %s: %v", w.Tag, src, err)
			continue
		}
		events := "none"
		if len(w.Events) > 0 {
			events = strings.Join(w.Events, ", ")
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s |\n",
			w.Tag, formName(w.Form), takes(w), events, cell(w.Doc), code(src), code(html))
	}
	for tag := range canonWidgets {
		if _, ok := widget(tag); !ok {
			t.Errorf("canonWidgets has <%s>, which is no widget", tag)
		}
	}
	return b.String()
}

func attrTable(t *testing.T) string {
	var b strings.Builder
	b.WriteString("| order | attribute | type | on | as | what | canonical instance | HTML |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	seen := map[string]bool{}
	for i := range Attrs {
		a := &Attrs[i]
		k := attrKey(a)
		seen[k] = true
		src, ok := canonAttrs[k]
		if !ok {
			t.Errorf("%s has no canonical instance in spec_test.go's canonAttrs", k)
			continue
		}
		if !strings.Contains(src, " "+a.Name) && !strings.Contains(src, ":"+a.Name) {
			t.Errorf("%s's canonical instance %s doesn't give it", k, src)
		}
		html, err := renderCanon(src)
		if err != nil {
			t.Errorf("%s's canonical instance %s: %v", k, src, err)
			continue
		}
		on := "every widget"
		if len(a.Tags) > 0 {
			var tags []string
			for _, tg := range a.Tags {
				tags = append(tags, "`"+tg+"`")
			}
			on = strings.Join(tags, ", ")
		}
		as := "modifier"
		if a.Arg {
			as = "argument"
		}
		fmt.Fprintf(&b, "| %d | `%s` | %s | %s | %s | %s | %s | %s |\n",
			i+1, a.Name, typeText(a.Type), on, as, cell(a.Doc), code(src), code(html))
	}
	for k := range canonAttrs {
		if !seen[k] {
			t.Errorf("canonAttrs has %s, which is no attribute", k)
		}
	}
	return b.String()
}

// region is the text between doc's markers for name, and where it is.
func region(doc, name string) (start, end int, ok bool) {
	begin := "<!-- begin " + name + " table: spec_test.go writes it (go test -run TestSpecTables -update) -->\n"
	stop := "<!-- end " + name + " table -->"
	i := strings.Index(doc, begin)
	j := strings.Index(doc, stop)
	if i < 0 || j < i {
		return 0, 0, false
	}
	return i + len(begin), j, true
}

// TestSpecTables checks docs/spec.md's Widgets and Attributes tables
// against Widgets and Attrs, and their HTML against what each canonical
// instance renders now.
func TestSpecTables(t *testing.T) {
	doc := specDoc
	for _, tb := range []struct {
		name string
		make func(*testing.T) string
	}{{"widgets", widgetTable}, {"attributes", attrTable}} {
		want := tb.make(t)
		start, end, ok := region(doc, tb.name)
		if !ok {
			t.Errorf("%s has no %s table's markers", specFile, tb.name)
			continue
		}
		if doc[start:end] == want {
			continue
		}
		if *updateSpec {
			doc = doc[:start] + want + doc[end:]
			continue
		}
		t.Errorf("%s's %s table is not the vocabulary's (go test -run TestSpecTables -update writes it):\n got\n%s\nwant\n%s",
			specFile, tb.name, doc[start:end], want)
	}
	if *updateSpec && doc != specDoc {
		if err := os.WriteFile(specFile, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
