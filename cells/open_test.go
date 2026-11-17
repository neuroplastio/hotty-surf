package cells_test

import (
	"slices"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/ui"
)

// Surface.Open is the same in cells: a menu the pointer opens is open,
// and Escape closes it.
func TestOpenInCells(t *testing.T) {
	var st ui.DropdownMenuState
	items := []ui.MenuItem{{Label: "One", OnSelect: func() {}}}
	r := start(t, 40, 12, ui.DropdownMenu(&st, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Menu"), Items: items}))
	if got := r.s.Open(); got != nil {
		t.Fatalf("Open() = %q before the click, want none", got)
	}
	r.c.Click(r.at("Menu"))
	r.settle()
	if got := r.s.Open(); !slices.Equal(got, []string{"m"}) {
		t.Fatalf("Open() = %q after the click, want the menu; screen:\n%s", got, r.text())
	}
	// The Screen leaves Escape to the program, which offers it to the
	// surface (Surface.Key), as a driver does.
	if !r.c.Key("escape") {
		if _, err := r.s.Key("escape"); err != nil {
			t.Fatal(err)
		}
	}
	r.settle()
	if got := r.s.Open(); got != nil {
		t.Fatalf("Open() = %q after Escape, want none", got)
	}
}
