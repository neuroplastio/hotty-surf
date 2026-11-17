package main

import (
	"testing"
	"time"

	"neuroplast.io/hotty-surf/surftest"
)

var t0 = time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)

// feed adds lines as the service would, with no clock: one a second from
// t0, at the levels given.
func feed(a *App, r *surftest.Rig, lines ...Line) {
	for _, l := range lines {
		l.Time = t0.Add(time.Duration(a.Next) * time.Second)
		a.Add(l)
	}
	r.Update()
}

func TestWatch(t *testing.T) {
	a := NewApp()
	r := surftest.Start(t, newSurface("logs", Page(a)))
	r.One("h3:text(No lines)")

	feed(a, r,
		Line{Level: 0, Text: "heartbeat from depot north-3"},
		Line{Level: 1, Text: "parcel P-00001 routed via depot east-7"},
		Line{Level: 2, Text: "depot east-7 slow to answer (420 ms)"},
		Line{Level: 3, Text: "timeout reaching depot east-7"},
		Line{Level: 1, Text: "batch of 12 parcels sealed"},
	)
	// Info and up by default: the debug line is hidden, the error set apart.
	if n := len(r.Find("div[data-level]")); n != 4 {
		t.Fatalf("%d rows shown, want 4:\n%s", n, r.HTML())
	}
	r.One("div[data-level=error]")
	r.One("span:text(timeout reaching depot east-7)")
	r.One("span:text(09:00:03.000)")

	// Errors only.
	r.Click("#level-trigger")
	r.Click("[role=option]:text(Errors only)")
	if n := len(r.Find("div[data-level]")); n != 1 {
		t.Fatalf("%d rows at errors only, want 1:\n%s", n, r.HTML())
	}
	// Debug and up, then a search: the text matched is marked.
	r.Click("#level-trigger")
	r.Click("[role=option]:text(Debug and up)")
	r.Type("#q", "East-7")
	if n := len(r.Find("div[data-level]")); n != 3 {
		t.Fatalf("%d rows for east-7, want 3:\n%s", n, r.HTML())
	}
	if n := len(r.Find("mark:text(east-7)")); n != 3 {
		t.Fatalf("%d marks, want 3", n)
	}

	// Paused, a new line doesn't come in; resumed, the next does.
	r.Click("button:text(Pause)")
	r.One("span:text(Paused)")
	feed(a, r, Line{Level: 3, Text: "depot east-7 refused manifest"})
	if len(r.Find("span:text(depot east-7 refused manifest)")) != 0 {
		t.Fatal("a line came in while paused")
	}
	r.Click("button:text(Resume)")
	feed(a, r, Line{Level: 3, Text: "depot east-7 refused manifest"})
	r.One("span:text(depot east-7 refused manifest)")

	// Clear empties the view.
	r.Click("button:text(Clear)")
	r.One("h3:text(No lines)")
	if len(r.Find("div[data-level]")) != 0 {
		t.Fatal("rows left after Clear")
	}
}

func TestFollow(t *testing.T) {
	a := NewApp()
	r := surftest.Start(t, newSurface("logs", Page(a)))
	feed(a, r, Line{Level: 1, Text: "one"}, Line{Level: 1, Text: "two"}, Line{Level: 1, Text: "three"})

	// Scrolled up, the view keeps its bottom line as more come in.
	r.Key("arrow_up")
	r.One("span:text(Scrolled back)")
	feed(a, r, Line{Level: 1, Text: "four"}, Line{Level: 1, Text: "five"})
	if len(r.Find("span:text(four)")) != 0 {
		t.Fatalf("a scrolled-back view moved on:\n%s", r.HTML())
	}
	r.One("button:text(3 newer lines)")
	r.Click("button:text(3 newer lines)")
	r.One("span:text(Following)")
	r.One("span:text(five)")

	// Follow off by its switch: the same.
	r.Click("#follow")
	if r.Checked("#follow") {
		t.Fatal("follow still on")
	}
	feed(a, r, Line{Level: 1, Text: "six"})
	if len(r.Find("span:text(six)")) != 0 {
		t.Fatal("follow off, but the view moved to the new line")
	}
	// End follows again.
	r.Key("end")
	r.One("span:text(six)")
	if !r.Checked("#follow") {
		t.Fatal("End didn't turn follow back on")
	}
}

func TestLint(t *testing.T) { surftest.Lint(t, ".") }
