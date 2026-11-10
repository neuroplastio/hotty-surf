package surfui_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfui"
)

// A field's Label and Error are drawn by the stack its parent holds in its
// place, and the field keeps the same id as a message comes and goes: a
// field that changed its id would lose the keyboard as the user types.
func TestFieldLabelAndErrorKeepTheFieldsID(t *testing.T) {
	build := func(msg string) (surfui.Index, string) {
		root := surfui.Surface(surfui.Form(
			surfui.Field().Key("name").Label("Name").Error(msg),
		).Key("form"))
		return surfui.Resolve(root, "r"), surfui.HTML(root)
	}
	ix, html := build("")
	f := ix["r.kform.kname.0.kname"]
	if f == nil || surfui.Tag(f) != "input" {
		t.Fatalf("no field at r.kform.kname.0.kname: %v", keys(ix))
	}
	if strings.Contains(html, "u-tone-error") || strings.Contains(html, "is-invalid") {
		t.Errorf("an error with none: %s", html)
	}
	if !strings.Contains(html, `class="u-text u-tone-muted u-label u-keep">Name</span><input`) {
		t.Errorf("the label is not before the field: %s", html)
	}
	ix, html = build("Required")
	if ix["r.kform.kname.0.kname"] == nil {
		t.Errorf("the field moved with its message: %v", keys(ix))
	}
	if !strings.Contains(html, `>Required</span>`) || !strings.Contains(html, "is-invalid") || !strings.Contains(html, `aria-invalid="true"`) {
		t.Errorf("the message or invalid is missing: %s", html)
	}
	if i, j := strings.Index(html, "<input"), strings.Index(html, "Required"); i > j {
		t.Errorf("the message is above the field: %s", html)
	}
}

// A Bare field sits inside another control: its label is the accessible
// one only, and it is drawn as it was.
func TestBareFieldLabelIsAccessibleOnly(t *testing.T) {
	root := surfui.Surface(surfui.Row(surfui.Field().Bare().Label("Search")).Key("q"))
	surfui.Resolve(root, "r")
	html := surfui.HTML(root)
	if strings.Contains(html, "u-labelled") || strings.Contains(html, ">Search<") {
		t.Errorf("a bare field drew its label: %s", html)
	}
	if !strings.Contains(html, `aria-label="Search"`) {
		t.Errorf("no accessible label: %s", html)
	}
}

// AriaLabel is the accessible label alone, whatever the element is.
func TestAriaLabelDrawsNothing(t *testing.T) {
	root := surfui.Surface(surfui.Field().Key("f").AriaLabel("Filter"))
	surfui.Resolve(root, "r")
	html := surfui.HTML(root)
	if strings.Contains(html, "u-labelled") || !strings.Contains(html, `aria-label="Filter"`) {
		t.Errorf("html %s", html)
	}
}

func TestPasswordAndSubmit(t *testing.T) {
	root := surfui.Surface(surfui.Form(
		surfui.Field().Key("p").Password(),
		surfui.Button("Go").Key("go").Submit(),
		surfui.Button("Cancel").Key("no"),
	).Key("form"))
	ix := surfui.Resolve(root, "r")
	html := surfui.HTML(root)
	for _, want := range []string{`type="password"`, `<button id="r.kform.kgo" class="u-btn" type="submit">`, `<button id="r.kform.kno" class="u-btn" type="button">`} {
		if !strings.Contains(html, want) {
			t.Errorf("no %s in %s", want, html)
		}
	}
	if !surfui.Submits(ix["r.kform.kgo"]) || surfui.Submits(ix["r.kform.kno"]) {
		t.Error("Submits is wrong")
	}
	if surfui.FormOf(ix["r.kform.kp"]) != ix["r.kform"] || surfui.FormOf(root) != nil {
		t.Error("FormOf is wrong")
	}
}

// A field of one line in a form keeps Enter: the host commits it and
// submits the form. Out of a form, a field with an OnChange gives Enter
// to the program, which commits it; a field of several lines never gives
// it, so that its Enter is a line break.
func TestGivesEnter(t *testing.T) {
	loose := surfui.Field().Key("a").OnChange(func() {})
	area := surfui.Field().Key("b").Rows(2, 4).OnChange(func() {})
	inForm := surfui.Field().Key("c").OnChange(func() {})
	surfui.Resolve(surfui.Surface(loose, area, surfui.Form(inForm).Key("f")), "r")
	for _, c := range []struct {
		name string
		n    *surfui.Node
		want bool
	}{{"a field out of a form", loose, true}, {"a field of several lines", area, false}, {"a field in a form", inForm, false}} {
		if got := surfui.GivesEnter(c.n); got != c.want {
			t.Errorf("%s: GivesEnter %v, want %v", c.name, got, c.want)
		}
		if strings.Contains(surfui.DataKeys(c.n), "enter=program") != c.want {
			t.Errorf("%s: data-keys %q", c.name, surfui.DataKeys(c.n))
		}
	}
}

func keys(ix surfui.Index) []string {
	var out []string
	for k := range ix {
		out = append(out, k)
	}
	return out
}
