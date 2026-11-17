package ui_test

import (
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// Icon's doc names "plus": the kit has it. A name it hasn't is a fault
// that lists the names it has.
func TestIconNames(t *testing.T) {
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", page(ui.Icon("plus", "")), ui.Stylesheet()))
	if len(f.got) > 0 {
		t.Errorf("ui.Icon(\"plus\", \"\"), the doc's own example: %v", f.got)
	}
	f.got = nil
	surftest.Start(f, surf.New("t", page(ui.Icon("nope", "")), ui.Stylesheet()))
	if len(f.got) != 1 || !strings.Contains(f.got[0], `no icon "nope" (the kit has archive, `) || !strings.Contains(f.got[0], " plus, ") {
		t.Errorf("an icon the kit hasn't: %v", f.got)
	}
}

// A Button's Attrs may give its type, as css.go's doc says; without one it
// is a button that submits nothing.
func TestButtonType(t *testing.T) {
	r := start(t, page(
		ui.Button(ui.ButtonProps{Attrs: templ.Attributes{"type": "submit", "id": "b", "aria-label": "b"}}),
		ui.Button(ui.ButtonProps{Attrs: templ.Attributes{"id": "c", "aria-label": "c"}}),
	))
	if got := attr(r.One("#b"), "type"); got != "submit" {
		t.Errorf("Attrs{type: submit}: the button's type is %q", got)
	}
	if got := attr(r.One("#c"), "type"); got != "button" {
		t.Errorf("no type in Attrs: the button's type is %q", got)
	}
}
