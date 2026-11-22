package surf_test

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// ExampleSubmit's form does what its doc says: Enter in the field and
// the submit button both add, and the field is cleared.
func TestExampleSubmit(t *testing.T) {
	a := &todo{}
	r := surftest.Start(t, surf.New("todo", a.view(), surfui.Stylesheet()))
	r.Type("input[name=title]", "Buy milk")
	r.Key("enter")
	if v := r.Value("input[name=title]"); v != "" {
		t.Errorf("the field after Enter: %q", v)
	}
	r.Type("input[name=title]", "Walk")
	r.Click("button[type=submit]")
	if !slices.Equal(a.tasks, []string{"Buy milk", "Walk"}) {
		t.Errorf("tasks: %q", a.tasks)
	}
	if v := r.Value("input[name=title]"); v != "" {
		t.Errorf("the field after the button: %q", v)
	}
}
