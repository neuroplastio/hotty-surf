package surf

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfui"
)

// Register refuses what the two modes and surflua couldn't read alike,
// and adds nothing where it does. (What it accepts is proven by
// examples/widgets: a registration is for good, so a test here that made
// one would add a tag to every test of this package.)
func TestRegisterRefuses(t *testing.T) {
	kids := func(...*surfui.Node) *surfui.Node { return surfui.Stack() }
	none := func() *surfui.Node { return surfui.Stack() }
	ok := Widget{Tag: "x-ok", Form: FormNone, Pkg: "example.com/x", Func: "New", New: none}
	with := func(f func(*Widget)) Widget {
		w := ok
		f(&w)
		return w
	}
	attr := func(a Attr) Widget { return with(func(w *Widget) { w.Attrs = []Attr{a} }) }
	setS := func(*surfui.Node, string) {}
	for name, c := range map[string]struct {
		w    Widget
		want string
	}{
		"no dash":       {with(func(w *Widget) { w.Tag = "gauge" }), "has a dash"},
		"upper case":    {with(func(w *Widget) { w.Tag = "x-Gauge" }), "lowercase"},
		"leading dash":  {with(func(w *Widget) { w.Tag = "-gauge" }), "starts with a letter"},
		"trailing dash": {with(func(w *Widget) { w.Tag = "gauge-" }), "end in a dash"},
		"no pkg":        {with(func(w *Widget) { w.Pkg = "" }), "Pkg and Func"},
		"no func":       {with(func(w *Widget) { w.Func = "" }), "Pkg and Func"},
		"no form":       {with(func(w *Widget) { w.Form = 0 }), "Form is FormKids"},
		"a floor form":  {with(func(w *Widget) { w.Form = FormSwitch }), "Form is FormKids"},
		"no New":        {with(func(w *Widget) { w.New = nil }), "New is <nil>"},
		"wrong New":     {with(func(w *Widget) { w.New = kids }), "New is func(...*surfui.Node)"},
		"an argument":   {attr(Attr{Name: "n", Type: TypeInt, Arg: true, Func: "F", Apply: func(*surfui.Node, int64) {}}), "not a constructor's argument"},
		"no attr func":  {attr(Attr{Name: "n", Type: TypeInt, Apply: func(*surfui.Node, int64) {}}), "Func says where"},
		"wrong Apply":   {attr(Attr{Name: "n", Type: TypeInt, Func: "F", Apply: setS}), "Apply is"},
		"fails unsaid":  {attr(Attr{Name: "n", Type: TypeString, Func: "F", Apply: func(*surfui.Node, string) error { return nil }}), "Apply is"},
		"a list":        {attr(Attr{Name: "n", Type: TypeStrings, Func: "F", Apply: setS}), "Type is TypeString"},
		"values of int": {attr(Attr{Name: "n", Type: TypeInt, Func: "F", Apply: func(*surfui.Node, int64) {}, Values: []string{"a"}}), "Values are a string"},
		"every widget":  {attr(Attr{Name: "gap", Type: TypeInt, Func: "F", Apply: func(*surfui.Node, int64) {}}), "every widget takes"},
		"reserved":      {attr(Attr{Name: "if", Type: TypeBool, Func: "F", Apply: func(*surfui.Node, bool) {}}), "taken, or none"},
		"twice": {with(func(w *Widget) {
			a := Attr{Name: "n", Type: TypeBool, Func: "F", Apply: func(*surfui.Node, bool) {}}
			w.Attrs = []Attr{a, a}
		}), "twice"},
	} {
		t.Run(name, func(t *testing.T) {
			widgets, attrs := len(Widgets), len(Attrs)
			err := Register(c.w)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Register: %v, want %q", err, c.want)
			}
			if len(Widgets) != widgets || len(Attrs) != attrs {
				t.Errorf("a refused registration added to the vocabulary")
			}
		})
	}
	if registered("x-ok") {
		t.Error("x-ok is registered")
	}
}

// MustRegister panics where Register refuses.
func TestMustRegisterPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustRegister didn't panic")
		}
	}()
	MustRegister(Widget{Tag: "plain"})
}
