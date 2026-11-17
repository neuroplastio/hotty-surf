package surftest_test

import (
	"testing"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
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
	r.Key("enter") // the field's form submits: its surf.Submit adds the task
	if len(r.Find("li:text(Write the docs)")) != 1 {
		t.Fatal("the task wasn't added")
	}
}

// Every app keeps a lint test: in TestLint(t *testing.T),
func ExampleLint() {
	var t *testing.T // the test's

	surftest.Lint(t, ".")
}

// A test of a list that reorders by drag (surf.Drag, surf.Drop): in
// TestReorder(t *testing.T),
func ExampleRig_DragStep() {
	var t *testing.T // the test's

	r := surftest.Start(t, surf.New("list", page, ui.Stylesheet()))
	r.DragStep("#row-a", "#row-c", 0, 1) // released over the bottom half of c
	if rows := r.Find("li"); attr(rows[len(rows)-1], "id") != "row-a" {
		t.Fatal("a isn't last")
	}
}

// A test of a row's context menu (surf.Secondary): in TestMenu(t *testing.T),
func ExampleRig_Secondary() {
	var t *testing.T // the test's

	r := surftest.Start(t, surf.New("list", page, ui.Stylesheet()))
	r.Secondary("#row-a")
	r.Click("[role=menuitem]:text(Rename)")
}

// A list row that opens on a double click (surf.Clicks).
func ExampleRig_DoubleClick() {
	var t *testing.T // the test's

	r := surftest.Start(t, surf.New("list", page, ui.Stylesheet()))
	r.DoubleClick("#row-a")
}

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}
