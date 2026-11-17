package ui_test

import (
	"context"
	"io"
	"slices"
	"strings"
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

func TestComboboxKeywordsAndInputAttrs(t *testing.T) {
	var st ui.ComboboxState
	value := ""
	attrs := templ.Attributes{"aria-label": "Choose fruit"}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		child := ui.Combobox(&st, ui.ComboboxProps{ID: "fruit", Items: []ui.SelectItem{
			{Value: "a", Label: "Same", Keywords: "orchard, crimson"}, {Value: "b", Label: "Same", Keywords: "tropical"},
		}, InputAttrs: attrs, OnChange: func(v string) { value = v }})
		return with(ui.KeyScope([]ui.KeyBinding{{Key: "arrow_down", Run: func() bool { return false }}}), child).Render(ctx, w)
	})
	r := surftest.Start(t, surf.New("combobox", view, ui.Stylesheet()))
	r.One(`#fruit-input[aria-label="Choose fruit"][data-hotty-keyscope]`)
	r.Type("#fruit-input", "TROP")
	r.One("#fruit-items-1[aria-hidden=true]")
	r.One("#fruit-items-2[aria-hidden=false]")
	r.Key("arrow_down")
	r.Key("enter")
	if value != "b" {
		t.Fatalf("selected %q", value)
	}
	if len(attrs) != 1 || attrs["aria-label"] != "Choose fruit" {
		t.Fatalf("mutated attrs: %v", attrs)
	}
}

func TestComboboxInputAttrsCollision(t *testing.T) {
	f := &faults{T: t}
	var st ui.ComboboxState
	attrs := templ.Attributes{"ID": "wrong", "DATA-KEYS": "home=field", "ArIa-Expanded": "wrong", "DATA-Hotty-KeyScope": "999", "aria-label": "Choose"}
	child := ui.Combobox(&st, ui.ComboboxProps{ID: "fruit", InputAttrs: attrs})
	view := with(ui.KeyScope([]ui.KeyBinding{{Key: "arrow_down", Run: func() bool { return false }}}), child)
	r := surftest.Start(f, surf.New("combobox", view, ui.Stylesheet()))
	r.One(`#fruit-input[aria-label="Choose"]`)
	if got := attr(r.One("#fruit-input"), "data-keys"); !strings.Contains(got, "arrow_down=program") || strings.Contains(got, "home=field") {
		t.Fatalf("key bindings %q", got)
	}
	if got := attr(r.One("#fruit-input"), "data-hotty-keyscope"); got == "" || got == "999" {
		t.Fatalf("scope marker %q", got)
	}
	if len(f.got) < 4 {
		t.Fatalf("expected collisions, got %v", f.got)
	}
	if attrs["ID"] != "wrong" || attrs["DATA-KEYS"] != "home=field" || attrs["DATA-Hotty-KeyScope"] != "999" {
		t.Fatalf("mutated attrs: %v", attrs)
	}
}

func TestComboboxKeepsDisabledSelectionAndOwnsCallbackSlices(t *testing.T) {
	var st ui.ComboboxState
	values := []string{"d"}
	var reports [][]string
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Combobox(&st, ui.ComboboxProps{ID: "fruit", Items: comboboxItems, Multiple: true, Values: values, InputAttrs: templ.Attributes{"aria-label": "Fruit"}, OnValuesChange: func(v []string) { values = v; reports = append(reports, v) }}).Render(ctx, w)
	})
	r := surftest.Start(t, surf.New("combobox", view, ui.Stylesheet()))
	r.One("#fruit-items-4-2[aria-selected=true][aria-disabled=true]")
	r.Click("#fruit-input")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"d", "a"}) {
		t.Fatalf("selection %v", values)
	}
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"d", "a", "b"}) || len(reports) < 2 || !slices.Equal(reports[0], []string{"d", "a"}) {
		t.Fatalf("selection %v reports %v", values, reports)
	}
}

func TestComboboxKeepsUnknownAndRepeatedValues(t *testing.T) {
	var st ui.ComboboxState
	values := []string{"ghost", "ghost"}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Combobox(&st, ui.ComboboxProps{ID: "fruit", Items: comboboxItems, Multiple: true, Values: values, OnValuesChange: func(v []string) { values = v }}).Render(ctx, w)
	})
	r := surftest.Start(t, surf.New("combobox", view, ui.Stylesheet()))
	r.One("#fruit-chip-0")
	r.One("#fruit-chip-1")
	want(t, r, "input[type=hidden]", "value", `["ghost","ghost"]`)
	r.Click("#fruit-input")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"ghost", "ghost", "a"}) {
		t.Fatalf("values: %v", values)
	}
}
