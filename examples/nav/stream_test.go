package main

import (
	"os"
	"testing"

	"neuroplast.io/hotty-go"
)

// TestStream writes what the surface sends for a click, the arrows and
// Enter to NAV_STREAM=<file>, for comparing with another runtime's stream
// of the same acts (internal/streamdump).
func TestStream(t *testing.T) {
	path := os.Getenv("NAV_STREAM")
	if path == "" {
		t.Skip("NAV_STREAM=<file>")
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := newSurface(surface, Page(NewApp(Mock())))
	s.Attach(f, hotty.Caps{})
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	doc, err := s.Document()
	must(err)
	_, err = f.WriteString(hotty.Doc(surface, doc))
	must(err)
	must(s.Update())
	must(s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "row-s4"}))
	must(s.Event(hotty.Event{Kind: hotty.EventClick, Target: "row-s4"}))
	for _, k := range []string{"arrow_down", "arrow_down", "arrow_up", "arrow_up", "arrow_up", "arrow_up", "arrow_up", "arrow_down", "enter"} {
		_, err := s.Key(k)
		must(err)
	}
	must(s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "row-s5"}))
	must(s.Event(hotty.Event{Kind: hotty.EventClick, Target: "row-s5"}))
}
