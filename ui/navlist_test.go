package ui_test

import (
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/ui"
)

func TestNavListKeysAndFilter(t *testing.T) {
	nav := &ui.NavList{ID: "nav", Label: "Sections", FilterLabel: "Filter", Current: "one",
		Groups: []ui.NavGroup{{Name: "Account", Items: []ui.NavItem{
			{ID: "one", Label: "Profile"}, {ID: "two", Label: "Security"},
		}}}}
	r := start(t, templ.Join(nav.FilterInput(""), nav.List("overflow-y-auto")))
	r.Click("#nav-item-one")
	r.Key("arrow_down")
	if nav.Current != "two" {
		t.Fatalf("arrow down selected %q", nav.Current)
	}
	r.Type("#nav-filter", "profile")
	if nav.Current != "one" || len(r.Find("button:text(Security)")) != 0 {
		t.Fatalf("filter selected %q or left Security visible", nav.Current)
	}
}

func TestFuzzyByteBehavior(t *testing.T) {
	// The bytes of é occur across two different UTF-8 sequences here.
	if !ui.Fuzzy("ã©", "é") || !ui.Fuzzy("anything", "  ") {
		t.Fatal("Fuzzy no longer follows its documented byte matching")
	}
}

func TestControlIDCollision(t *testing.T) {
	if ui.ControlID("x", "A.B") != ui.ControlID("x", "A-B") {
		t.Fatal("dot-to-hyphen ID behavior changed")
	}
}
