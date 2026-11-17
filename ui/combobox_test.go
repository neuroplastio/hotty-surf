package ui_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

var comboboxItems = []ui.SelectItem{
	{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"},
	{Value: "c", Label: "Cherry", Group: "More"}, {Value: "d", Label: "Date", Group: "More", Disabled: true},
}

type comboboxApp struct {
	st       ui.ComboboxState
	value    string
	values   []string
	multiple bool
	clear    bool
}

func (a *comboboxApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(ui.Combobox(&a.st, ui.ComboboxProps{ID: "fruit", Items: comboboxItems, Value: a.value, Values: a.values,
			Multiple: a.multiple, Clear: a.clear, Placeholder: "Choose fruit", OnChange: func(v string) { a.value = v },
			OnValuesChange: func(v []string) { a.values = v }}), button("other", "Other", func() {})).Render(ctx, w)
	})
}

func TestComboboxSearchAndChoose(t *testing.T) {
	a := &comboboxApp{}
	s := surf.New("combobox", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Type("#fruit-input", "ba")
	if got := attr(r.One("#fruit-items-1"), "aria-hidden"); got != "true" {
		t.Fatalf("Apple hidden=%q", got)
	}
	if got := attr(r.One("#fruit-items-2"), "aria-hidden"); got != "false" {
		t.Fatalf("Banana hidden=%q", got)
	}
	r.Key("arrow_down")
	if got := attr(r.One("#fruit-input"), "aria-activedescendant"); got != "fruit-items-2" {
		t.Fatalf("active=%q", got)
	}
	r.Key("enter")
	if a.value != "b" {
		t.Fatalf("chosen=%q", a.value)
	}
	if got := r.Value("#fruit-input"); got != "Banana" {
		t.Fatalf("field=%q", got)
	}
	if got := attr(r.One("#fruit-popover"), "aria-hidden"); got != "true" {
		t.Fatalf("list hidden=%q", got)
	}
	if got := rt.Peek.Focused(s); got != "fruit-input" {
		t.Fatalf("focus=%q", got)
	}
}

func TestComboboxMultiple(t *testing.T) {
	a := &comboboxApp{multiple: true}
	r := surftest.Start(t, surf.New("combobox", a.view(), ui.Stylesheet()))
	r.Click("#fruit-input")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(a.values, []string{"a"}) {
		t.Fatalf("first selection %v", a.values)
	}
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(a.values, []string{"a", "b"}) {
		t.Fatalf("second selection %v", a.values)
	}
	r.One(`button[aria-label="Remove Apple"]`)
	r.Key("backspace")
	if !slices.Equal(a.values, []string{"a"}) {
		t.Fatalf("backspace selection %v", a.values)
	}
	r.Click("#fruit-chip-0")
	if len(a.values) != 0 {
		t.Fatalf("remove selection %v", a.values)
	}
}

func TestComboboxClearAndOutside(t *testing.T) {
	a := &comboboxApp{value: "a", clear: true}
	r := surftest.Start(t, surf.New("combobox", a.view(), ui.Stylesheet()))
	r.Click("#fruit-clear")
	if a.value != "" {
		t.Fatalf("clear value=%q", a.value)
	}
	r.Click("#fruit-input")
	r.Click("#other")
	if got := attr(r.One("#fruit-popover"), "aria-hidden"); got != "true" {
		t.Fatalf("outside hidden=%q", got)
	}
}

func TestComboboxProgrammaticValue(t *testing.T) {
	a := &comboboxApp{}
	r := surftest.Start(t, surf.New("combobox", a.view(), ui.Stylesheet()))
	r.Type("#fruit-input", "ch")
	a.value = "b"
	r.Update()
	if got := r.Value("#fruit-input"); got != "Banana" {
		t.Fatalf("programmatic value shows %q, want Banana", got)
	}
}
