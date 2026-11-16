package surftest_test

import (
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

var page templ.Component // the app's view

// An app's test drives its view as the user would: in TestAdd(t *testing.T),
func Example() {
	var t *testing.T // the test's

	r := surftest.Start(t, surf.New("tasks", page, ui.Stylesheet()), surftest.Size(120, 40))
	r.Type("input[name=title]", "Write the docs")
	r.Key("enter")
	if len(r.Find("li:text(Write the docs)")) != 1 {
		t.Fatal("the task wasn't added")
	}
}

// Every app keeps a lint test: in TestLint(t *testing.T),
func ExampleLint() {
	var t *testing.T // the test's

	surftest.Lint(t, ".")
}
