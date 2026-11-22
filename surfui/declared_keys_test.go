package surfui_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
	"neuroplast.io/hotty-surf/surfui/combobox"
	"neuroplast.io/hotty-surf/surfui/input"
	"neuroplast.io/hotty-surf/surfui/menu"
)

type faults struct {
	*testing.T
	got []string
}

func (f *faults) Error(args ...any) {
	f.got = append(f.got, fmt.Sprint(args...))
}

func (f *faults) Errorf(format string, args ...any) {
	f.got = append(f.got, fmt.Sprintf(format, args...))
}

type menuApp struct {
	st  menu.DropdownState
	ran string
}

func (a *menuApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return menu.Dropdown(&a.st, menu.DropdownProps{
			ID:      "m",
			Trigger: templ.Raw("Menu"),
			Items:   []menu.Item{{Label: "One", OnSelect: func() { a.ran = "one" }}, {Label: "Two", OnSelect: func() { a.ran = "two" }}},
		}).Render(ctx, w)
	})
}

type comboboxApp struct {
	st    combobox.State
	value string
}

func (a *comboboxApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return combobox.View(&a.st, combobox.Props{
			ID:          "fruit",
			Placeholder: "Choose fruit",
			Value:       a.value,
			Items:       []combobox.Item{{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"}},
			OnChange:    func(v string) { a.value = v },
		}).Render(ctx, w)
	})
}

// DropdownMenu's keys are its trigger's, where the keyboard is (K32): the
// handler is registered there with them, and the root declares none, so
// no key is asked twice.
func TestDropdownMenuDeclaresOnTrigger(t *testing.T) {
	a := &menuApp{}
	r := start(t, a.view())
	if got := attr(r.One("#m-trigger"), "data-keys"); got != "arrow_down=program arrow_up=program" {
		t.Fatalf("closed: the trigger's data-keys %q", got)
	}
	if got := attr(r.One("#m"), "data-keys"); got != "" {
		t.Fatalf("the root's data-keys %q", got)
	}
	r.Click("#m-trigger")
	// Escape is declared, but a host ignores it in data-keys: it reaches
	// the program anyway.
	if got := attr(r.One("#m-trigger"), "data-keys"); got != "arrow_down=program arrow_up=program end=program enter=program home=program space=program" {
		t.Fatalf("open: the trigger's data-keys %q", got)
	}
	if !r.Key("escape") || attr(r.One("#m-popover"), "aria-hidden") != "true" {
		t.Fatal("Escape didn't close the menu")
	}
}

// Combobox's field registers the combobox's key handler with the keys it
// declares, and the root keeps the same handler for keys from its
// buttons. A key the field's declines, the root's hears again (the double
// ask) and declines too, with nothing done either time: Escape on a
// closed combobox, Enter on one with nothing highlighted.
func TestComboboxDoubleAsk(t *testing.T) {
	a := &comboboxApp{}
	r := surftest.Start(t, surf.New("combobox", a.view(), surfui.Stylesheet()))
	r.Click("#fruit-input")
	if got := attr(r.One("#fruit-input"), "data-keys"); got == "" {
		t.Fatal("the field declares no keys")
	}
	if got := attr(r.One("#fruit"), "data-keys"); got != "" {
		t.Fatalf("the root's data-keys %q", got)
	}
	r.Key("escape") // the click opened it
	if got := attr(r.One("#fruit-popover"), "aria-hidden"); got != "true" {
		t.Fatalf("not closed: aria-hidden %q", got)
	}
	before := r.HTML()
	r.Key("escape")
	if a.value != "" || r.HTML() != before {
		t.Fatalf("Escape on a closed combobox did something: value %q", a.value)
	}
}

// A KeyScope marker a template writes is the template's, not the kit's:
// it is a fault and is dropped, the element it is on gets none of the
// scope's keys, and the kit control after it keeps them.
func TestForgedScopeMarker(t *testing.T) {
	f := &faults{T: t}
	child := templ.Join(
		templ.Raw(`<div id="forged" tabindex="0" data-hotty-keyscope="1" data-hotty-keykind="text">x</div>`),
		input.View(input.Props{Attrs: templ.Attributes{"id": "in", "aria-label": "In"}}),
	)
	view := with(surfui.KeyScope([]surfui.KeyBinding{{Key: "ctrl+s", Run: func() bool { return true }}}), child)
	r := surftest.Start(f, surf.New("t", view, surfui.Stylesheet()))
	if got := attr(r.One("#forged"), "data-keys"); got != "" {
		t.Errorf("the forged element's data-keys %q", got)
	}
	if got := attr(r.One("#in"), "data-keys"); got != "ctrl+s=program" {
		t.Errorf("the input's data-keys %q", got)
	}
	for _, sel := range []string{"[data-hotty-keyscope]", "[data-hotty-keykind]"} {
		if n := r.Find(sel); len(n) > 0 {
			t.Errorf("%s on the wire", sel)
		}
	}
	named := 0
	for _, g := range f.got {
		if strings.Contains(g, `"forged"`) {
			named++
		}
	}
	if named != 2 {
		t.Errorf("faults naming the forged element: %d of %q", named, f.got)
	}
}
