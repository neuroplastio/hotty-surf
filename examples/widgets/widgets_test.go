package widgets

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	lua "github.com/yuin/gopher-lua"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/examples/widgets/gauge"
	"neuroplast.io/hotty-surf/pkg/conformance"
	"neuroplast.io/hotty-surf/pkg/surfgen"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surflua"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

var update = flag.Bool("update", false, "write each case's document, ops and log as the interpreter gives them")

// The widgets' own cases (cases/*.txt), in both modes on the runtime, a
// host of its own and the cells: the conformance suite, run by the
// package that registers them.
func TestCases(t *testing.T) {
	root := func(draw func(*Dash) surflive.Component) func(testing.TB, string, []byte) (surflive.Component, func() []string) {
		return func(t testing.TB, name string, state []byte) (surflive.Component, func() []string) {
			if name != "" && name != "Dash" {
				t.Fatalf("root: no fixture %s", name)
			}
			d := &Dash{}
			if err := json.Unmarshal(state, d); err != nil {
				t.Fatal(err)
			}
			return draw(d), func() []string { return d.Log }
		}
	}
	s := conformance.Suite{Cases: os.DirFS("."), Dir: ".", Update: *update, Modes: []conformance.Mode{
		{Name: "interpreted", Root: root(surfInterpDash)},
		{Name: "compiled", Root: root(surfCompiledDash)},
	}}
	s.Run(t)
}

// The registrations are in the vocabulary: the tags, their attributes
// and events, read as the floor's are.
func TestRegistered(t *testing.T) {
	var tags []string
	for _, w := range surf.Registered() {
		tags = append(tags, w.Tag)
		if w.Pkg != gauge.Pkg {
			t.Errorf("<%s> is from %q", w.Tag, w.Pkg)
		}
	}
	if got := strings.Join(tags, " "); got != "x-gauge x-callout x-tagline x-spark" {
		t.Errorf("registered: %s", got)
	}
}

// A tag registered twice is refused, and adds nothing: not a second
// entry, not a second sheet.
func TestDuplicate(t *testing.T) {
	n, sheet := len(surf.Registered()), surfui.Sheet()
	err := surf.Register(surf.Widget{Tag: "x-gauge", Form: surf.FormNone, Pkg: gauge.Pkg, Func: "Gauge", New: gauge.Gauge, CSS: ".other {}"})
	if err == nil || !strings.Contains(err.Error(), "the tag is a widget's already") {
		t.Errorf("Register: %v", err)
	}
	if len(surf.Registered()) != n || surfui.Sheet() != sheet {
		t.Errorf("a refused registration added something")
	}
}

// The widgets' stylesheet goes after surf.css's, once, and before the
// palette's rules.
func TestSheet(t *testing.T) {
	sheet := surfui.Sheet()
	if !strings.HasPrefix(sheet, surfui.CSS) || strings.Count(sheet, ".x-gauge-fill {") != 1 {
		t.Fatalf("the sheet doesn't carry the widgets' CSS once after surf.css")
	}
	if strings.Index(sheet, ".x-gauge {") < len(surfui.CSS) {
		t.Errorf("the widgets' CSS comes before surf.css's end")
	}
	for _, m := range []func(*Dash) surflive.Component{surfInterpDash, surfCompiledDash} {
		a := surftest.Start(t, surflive.New(m(&Dash{Title: "T", Level: 20})), surfrun.Options{Name: "dash"})
		_, got, ok := a.Host.Resource(surfui.StyleID)
		if !ok || strings.Count(string(got), ".x-gauge-fill {") != 1 || !strings.HasPrefix(string(got), surfui.CSS) {
			t.Errorf("the stylesheet resource sent is not surf.css with the widgets' CSS once after it")
		}
	}
}

// Both modes give one document, and a click on a registered widget runs
// its handler, which changes the level the gauge shows.
func TestBothModes(t *testing.T) {
	var docs []string
	for _, m := range []func(*Dash) surflive.Component{surfInterpDash, surfCompiledDash} {
		d := &Dash{Title: "Dash", Level: 30, Note: "disk is full"}
		a := surftest.Start(t, surflive.New(m(d)), surfrun.Options{Name: "dash"})
		if err := a.Host.Click(a.Name, "r.kraise"); err != nil {
			t.Fatal(err)
		}
		a.Wait("the level", func() bool { return strings.Contains(a.Surface().HTML(), "40%") })
		if err := a.Host.Click(a.Name, "r.knote"); err != nil {
			t.Fatal(err)
		}
		a.Wait("the pick", func() bool {
			var n int
			a.Call(func() { n = len(d.Log) })
			return n == 2
		})
		docs = append(docs, a.Surface().HTML())
		if !strings.Contains(docs[len(docs)-1], `class="x-callout x-callout-warn"`) {
			t.Errorf("the callout has no kind's class:\n%s", docs[len(docs)-1])
		}
	}
	if docs[0] != docs[1] {
		t.Errorf("the modes differ:\n%s\n%s", docs[0], docs[1])
	}
}

// A template with a registered widget's tag can't load: a tag is a
// widget's or a template's.
func TestClash(t *testing.T) {
	fsys := fstest.MapFS{
		"dash.surf":    {Data: []byte("<surface><x-gauge level=\"1\"/></surface>")},
		"x-gauge.surf": {Data: []byte("<surface/>")},
	}
	_, err := surf.Load(fsys, []*surf.Shape{{Name: "Dash", Component: true}})
	if err == nil || !strings.Contains(err.Error(), "x-gauge.surf: <x-gauge> is a registered widget's tag") {
		t.Errorf("Load: %v", err)
	}
}

// A literal a registered attribute refuses is a load error, as the
// floor's attributes' are; a bad value of a bound one is a render error.
func TestAttrRefusals(t *testing.T) {
	for src, want := range map[string]string{
		`<surface><x-callout kind="loud"/></surface>`:           `"loud"`,
		`<surface><x-gauge level="x"/></surface>`:               "level",
		`<surface><x-gauge colour="x"/></surface>`:              "<x-gauge> takes no colour",
		`<surface><x-tagline>a<x-gauge/></x-tagline></surface>`: "<x-tagline> holds text",
		`<surface><x-gauge>text</x-gauge></surface>`:            "<x-gauge> has no children",
	} {
		fsys := fstest.MapFS{"dash.surf": {Data: []byte(src)}}
		_, err := surf.Load(fsys, []*surf.Shape{{Name: "Dash", Component: true}})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", src, err, want)
		}
	}
}

// surflua reads the same registry: the widget is in the Lua host's
// templates with no more said.
func TestLua(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	surflua.PreloadFS(L, fstest.MapFS{"t/dash.surf": {Data: []byte(
		`<surface><x-gauge level="40" key="g"/><x-callout kind="warn" key="c"><x-tagline loud>Hi</x-tagline></x-callout><x-spark key="s">1 2 3</x-spark></surface>`)}})
	if err := L.DoString(`
local surf = require("surf")
dash = {}
view = surf.load("/t"):view("dash", dash)`); err != nil {
		t.Fatal(err)
	}
	if err := L.DoString(`html, errs = view:html()`); err != nil {
		t.Fatal(err)
	}
	html := L.GetGlobal("html").String()
	for _, want := range []string{`class="x-gauge"`, `style="--x-level:0.4"`, `40%`, `x-callout-warn`, `class="x-tagline x-loud"`, `<polyline points="2,14.0 6,8.0 10,2.0">`} {
		if !strings.Contains(html, want) {
			t.Errorf("surflua's document has no %s:\n%s", want, html)
		}
	}
	if e := L.GetGlobal("errs"); e != lua.LNil {
		t.Errorf("render errors: %v", e)
	}
}

// surf_gen.go is what surfgen writes with the widgets linked.
func TestGenerated(t *testing.T) {
	got, err := surfgen.Generate(".", false)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("surf_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("surf_gen.go is stale: go generate")
	}
}
