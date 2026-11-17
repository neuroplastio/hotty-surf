package main

import (
	"testing"

	"neuroplast.io/hotty-surf/surftest"
)

// TestRange switches the range as a user would, by a click and by the
// arrows, and checks that a number, both charts and the table follow.
func TestRange(t *testing.T) {
	r := surftest.Start(t, newSurface("dashboard", Page(NewApp())), surftest.Size(160, 48))

	// 7 days: the week's revenue, a bar per channel, no plate set in the top five.
	r.One("h2:text($12,933)")
	r.One("rect[data-value=128]")
	r.One("polyline[data-values=2128 2154 1617 1501 1625 1763 2145]")
	if len(r.Find("td:text(Dinner plate set)")) != 0 {
		t.Error("the plate set is in the 7 days' top five")
	}

	r.Click("button:text(30 days)")
	r.One("h2:text($62,342)")
	if n := len(r.Find("span:text(vs the 30 days before)")); n != 4 {
		t.Errorf("%d changes against the 30 days before, want 4", n)
	}
	r.One("rect[data-value=676]")
	if n := len(r.Find("span:text(30)")); n != 1 {
		t.Errorf("the line's labels: %d labelled 30, want 1", n)
	}
	r.One("td:text(Dinner plate set)")
	r.One("td:text($20,736)")

	// The arrows move along the tabs.
	r.Key("arrow_right")
	r.One("h2:text($682,725)")
	r.One("span:text(Dec)")
	r.One("rect[data-value=8061]")
	r.One("td:text(Stoneware bowl)")
	r.Key("arrow_right") // wraps to the first
	r.One("h2:text($12,933)")
}

// TestLint: the app keeps the kit's rules (package lint).
func TestLint(t *testing.T) { surftest.Lint(t, ".") }
