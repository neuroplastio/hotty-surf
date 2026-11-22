package surftui_test

import (
	"slices"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/surfui/menu"
)

// Surface.Overlays is the same in cells: a menu the pointer opens is open,
// and Escape closes it.
func TestOverlaysInCells(t *testing.T) {
	var st menu.DropdownState
	items := []menu.Item{{Label: "One", OnSelect: func() {}}}
	r := start(t, 40, 12, menu.Dropdown(&st, menu.DropdownProps{ID: "m", Trigger: templ.Raw("Menu"), Items: items}))
	if got := r.s.Overlays(); got != nil {
		t.Fatalf("Overlays() = %q before the click, want none", got)
	}
	r.c.Click(r.at("Menu"))
	r.settle()
	if got := r.s.Overlays(); !slices.Equal(got, []string{"m"}) {
		t.Fatalf("Overlays() = %q after the click, want the menu; screen:\n%s", got, r.text())
	}
	// The Screen leaves Escape to the program, which offers it to the
	// surface (Surface.Key), as a driver does.
	if !r.c.Key("escape") {
		if _, err := r.s.Key("escape"); err != nil {
			t.Fatal(err)
		}
	}
	r.settle()
	if got := r.s.Overlays(); got != nil {
		t.Fatalf("Overlays() = %q after Escape, want none", got)
	}
}
