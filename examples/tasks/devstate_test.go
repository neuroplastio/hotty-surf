package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

// TestDevState: what a dev reload carries (package reload) comes back as it
// went, through msgp, into a fresh App: the page shows the same rows.
func TestDevState(t *testing.T) {
	tasks, err := upstream()
	if err != nil {
		t.Fatal(err)
	}
	a := NewApp(tasks)
	a.SetFilter("v")
	a.ToggleFacet("status", "in progress")
	a.Sort("title", true)
	a.SetPageSize(10)
	a.SetPage(1)
	a.Select(a.rows[0], true)
	a.view.Open, a.view.Active = true, 2
	a.facetCmd["priority"].Open = true
	a.facetCmd["priority"].Search = "hi"

	var b bytes.Buffer
	w := msgp.NewWriter(&b)
	if err := a.devState().EncodeMsg(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	var d devState
	if err := d.DecodeMsg(msgp.NewReader(&b)); err != nil {
		t.Fatal(err)
	}
	c := NewApp(tasks)
	c.setDevState(&d)
	for _, f := range []struct {
		name string
		x, y any
	}{
		{"filter", a.filter, c.filter}, {"facet", a.facet, c.facet}, {"sort", a.sortCol, c.sortCol},
		{"desc", a.sortDesc, c.sortDesc}, {"page", a.pageIndex, c.pageIndex}, {"size", a.pageSize, c.pageSize},
		{"selected", a.selected, c.selected}, {"rows", a.rows, c.rows}, {"view", a.view.Open, c.view.Open},
		{"active", a.view.Active, c.view.Active}, {"search", a.facetCmd["priority"].Search, c.facetCmd["priority"].Search},
	} {
		if !reflect.DeepEqual(f.x, f.y) {
			t.Errorf("%s: %v, after the reload %v", f.name, f.x, f.y)
		}
	}
}
