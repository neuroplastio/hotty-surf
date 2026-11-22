package ui_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// sizes are a Select's items: two alone, a disabled one, and a group.
var sizes = []ui.SelectItem{
	{Value: "s", Label: "Small"},
	{Value: "m", Label: "Medium"},
	{Value: "x", Label: "Huge", Disabled: true},
	{Value: "l", Label: "Large", Group: "Big"},
	{Value: "xl", Label: "Extra large", Group: "Big"},
}

type selectApp struct {
	st      ui.SelectState
	value   string
	changes []string
}

func (a *selectApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(
			ui.Select(&a.st, ui.SelectProps{
				ID: "size", Items: sizes, Value: a.value, Placeholder: "Pick a size",
				OnChange: func(v string) { a.value = v; a.changes = append(a.changes, v) },
			}),
			button("other", "Other", func() {}),
			text("note", "A note"),
		).Render(ctx, w)
	})
}

func TestSelectMarkup(t *testing.T) {
	a := &selectApp{}
	r := start(t, a.view())
	// The macro's ids: the separator counts, a group's options are under
	// its index.
	for _, id := range []string{"size-items-1", "size-items-2", "size-items-3", "size-items-5-1", "size-items-5-2"} {
		r.One("div#" + id + "[role=option]")
	}
	r.One("div[role=group][aria-labelledby=group-label-size-items-5]")
	r.One("div#group-label-size-items-5[role=heading]:text(Big)")
	r.One("hr[role=separator]")
	want(t, r, "#size-items-3", "aria-disabled", "true")
	want(t, r, "#size-trigger", "aria-expanded", "false")
	want(t, r, "#size-popover", "aria-hidden", "true")
	want(t, r, "span:text(Pick a size)", "class", "truncate text-muted-foreground")
	want(t, r, "input[type=hidden]", "name", "size-value")
}

func TestSelectKeys(t *testing.T) {
	a := &selectApp{}
	r := start(t, a.view())
	r.Click("#size-trigger")
	want(t, r, "#size-trigger", "aria-expanded", "true")
	want(t, r, "#size-popover", "aria-hidden", "false")
	if got := active(r, "#size-listbox"); got != "" {
		t.Errorf("opened with %q active, not none", got)
	}
	steps := []struct{ key, active string }{
		{"arrow_down", "size-items-1"},
		{"arrow_down", "size-items-2"},
		{"arrow_down", "size-items-5-1"}, // past the disabled option
		{"arrow_down", "size-items-5-2"},
		{"arrow_down", "size-items-5-2"}, // the last stays
		{"home", "size-items-1"},
		{"arrow_up", "size-items-1"}, // the first stays
		{"end", "size-items-5-2"},
		{"arrow_up", "size-items-5-1"},
	}
	for _, s := range steps {
		if !r.Key(s.key) {
			t.Errorf("%s: not taken", s.key)
		}
		if got := active(r, "#size-listbox"); got != s.active {
			t.Errorf("%s: %q active, not %q", s.key, got, s.active)
		}
		want(t, r, "#size-trigger", "aria-activedescendant", s.active)
	}
	r.Key("enter")
	if !slices.Equal(a.changes, []string{"l"}) {
		t.Fatalf("changes %v, not [l]", a.changes)
	}
	want(t, r, "#size-trigger", "aria-expanded", "false")
	want(t, r, "#size-popover", "aria-hidden", "true")
	want(t, r, "span:text(Large)", "class", "truncate")
	want(t, r, "#size-items-5-1", "aria-selected", "true")

	// The keyboard is still on the trigger: an arrow opens the list on
	// the chosen option, and Escape closes it.
	if !r.Key("arrow_down") {
		t.Fatal("arrow_down on the closed select: not taken")
	}
	if got := active(r, "#size-listbox"); got != "size-items-5-1" {
		t.Errorf("reopened with %q active, not the chosen", got)
	}
	r.Key("escape")
	want(t, r, "#size-popover", "aria-hidden", "true")
	if r.Key("escape") {
		t.Error("escape on the closed select: taken")
	}
	// Enter on the closed trigger is the button's click.
	r.Key("enter")
	want(t, r, "#size-popover", "aria-hidden", "false")
	r.Key("enter") // with Large active: no change
	if len(a.changes) != 1 {
		t.Errorf("enter on the chosen option changed it: %v", a.changes)
	}
}

func TestSelectClick(t *testing.T) {
	a := &selectApp{}
	r := start(t, a.view())
	r.Click("#size-trigger")
	r.Click("#size-items-2")
	if !slices.Equal(a.changes, []string{"m"}) {
		t.Fatalf("changes %v, not [m]", a.changes)
	}
	want(t, r, "#size-popover", "aria-hidden", "true")
	r.One("span:text(Medium)")
	// The option took the keyboard off the surface; the trigger has it
	// back.
	if !r.Key("arrow_down") {
		t.Fatal("the keyboard isn't back on the trigger")
	}
	want(t, r, "#size-popover", "aria-hidden", "false")

	// A disabled option does nothing.
	r.Click("#size-items-3")
	if len(a.changes) != 1 {
		t.Errorf("a disabled option changed it: %v", a.changes)
	}
}

func TestSelectOutside(t *testing.T) {
	a := &selectApp{}
	r := start(t, a.view())
	r.Click("#size-trigger")
	r.Click("#note") // text: a press outside, and a blur
	want(t, r, "#size-popover", "aria-hidden", "true")
	r.Click("#size-trigger")
	r.Click("#other") // a button: a press outside, the keyboard goes there
	want(t, r, "#size-popover", "aria-hidden", "true")
	if len(a.changes) != 0 {
		t.Errorf("a click outside changed it: %v", a.changes)
	}
	// The keyboard stayed where the click put it: the arrows aren't the
	// select's.
	if r.Key("arrow_down") {
		t.Error("arrow_down on the other button: taken")
	}
	want(t, r, "#size-popover", "aria-hidden", "true")
}

func TestSelectDisabled(t *testing.T) {
	var st ui.SelectState
	r := start(t, ui.Select(&st, ui.SelectProps{ID: "s", Items: sizes, Value: "m", Disabled: true}))
	if !hasAttr(r.One("#s-trigger"), "disabled") {
		t.Error("the trigger isn't disabled")
	}
	r.One("span:text(Medium)")
	want(t, r, "input[type=hidden]", "value", "m")
}

// The pointer highlights the option it is over, as select.js's mousemove
// does, and leaving the list puts the highlight back on the chosen.
func TestSelectHover(t *testing.T) {
	a := &selectApp{value: "m"}
	s := surf.New("t", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#size-trigger")
	if got := active(r, "#size-listbox"); got != "size-items-2" {
		t.Fatalf("open: %q active", got)
	}
	hover(t, s, r, "size-items-1")
	if got := active(r, "#size-listbox"); got != "size-items-1" {
		t.Errorf("hover over Small: %q active", got)
	}
	hover(t, s, r, "note")
	if got := active(r, "#size-listbox"); got != "size-items-2" {
		t.Errorf("pointer left the list: %q active, not the chosen", got)
	}
}

// A choice's ask for the trigger is for the next frame only. A Select whose
// OnChange hides it, shown again later by a shortcut while the user types
// elsewhere, doesn't take the keyboard back.
func TestSelectAskExpires(t *testing.T) {
	var st ui.SelectState
	shown, typed := true, ""
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := surf.On(ctx, "app", surf.Key(func(k string) bool {
			if k == "mod+b" {
				shown = true
				return true
			}
			return false
		}))
		io.WriteString(w, `<main id="`+a["id"].(string)+`">`)
		if shown {
			if err := ui.Select(&st, ui.SelectProps{ID: "s", Items: sizes, Placeholder: "Size",
				OnChange: func(string) { shown = false }}).Render(ctx, w); err != nil {
				return err
			}
		}
		if err := field("editor", typed, func(v string) { typed = v }).Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
	r := start(t, view)
	r.Click("#s-trigger")
	r.Click("#s-items-1") // hides the Select
	r.Type("#editor", "ab")
	r.Key("ctrl+b") // the Select comes back
	if !shown {
		t.Fatal("the shortcut didn't show the Select")
	}
	r.Key("c")
	if typed != "abc" {
		t.Errorf("the editor holds %q: the Select's return took the keyboard to its trigger", typed)
	}
}

func TestSelectMultiple(t *testing.T) {
	var st ui.SelectState
	values := []string{}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Select(&st, ui.SelectProps{ID: "sizes", Items: sizes, Multiple: true, Values: values, Placeholder: "Pick sizes", OnValuesChange: func(v []string) { values = v }}).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#sizes-trigger")
	r.Key("arrow_down")
	r.Key("enter")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"s", "m"}) {
		t.Fatalf("values: %v", values)
	}
	want(t, r, "#sizes-trigger", "aria-expanded", "true")
	r.One("#sizes-listbox[aria-multiselectable=true]")
	r.One("#sizes-items-1[aria-selected=true]")
	r.One("#sizes-items-2[aria-selected=true]")
	r.Click("#sizes-items-1")
	if !slices.Equal(values, []string{"m"}) {
		t.Fatalf("toggle: %v", values)
	}
	want(t, r, "#sizes-trigger", "aria-expanded", "true")
	r.Key("escape")
	want(t, r, "#sizes-trigger", "aria-expanded", "false")
}

func TestSelectMultipleCloseOnSelect(t *testing.T) {
	var st ui.SelectState
	values := []string{}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Select(&st, ui.SelectProps{ID: "sizes", Items: sizes, Multiple: true, Values: values, CloseOnSelect: true, Placeholder: "Pick sizes", OnValuesChange: func(v []string) { values = v }}).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#sizes-trigger")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"s"}) {
		t.Fatalf("values: %v", values)
	}
	want(t, r, "#sizes-trigger", "aria-expanded", "false")
}

func TestSelectAlignEnd(t *testing.T) {
	var st ui.SelectState
	r := start(t, ui.Select(&st, ui.SelectProps{ID: "edge", Items: sizes, Placeholder: "Pick", Align: "end"}))
	r.One("#edge-popover[data-align=end]")
}

func TestSelectMultipleKeepsDisabledSelection(t *testing.T) {
	var st ui.SelectState
	values := []string{"x"}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Select(&st, ui.SelectProps{ID: "sizes", Items: sizes, Multiple: true, Values: values, Placeholder: "Pick sizes", OnValuesChange: func(v []string) { values = v }}).Render(ctx, w)
	})
	r := start(t, view)
	r.One("#sizes-items-3[aria-selected=true][aria-disabled=true]")
	r.One("span:text(Huge)")
	want(t, r, "input[type=hidden]", "value", `["x"]`)
	r.Click("#sizes-trigger")
	r.Key("arrow_down")
	r.Key("enter")
	if !slices.Equal(values, []string{"s", "x"}) {
		t.Fatalf("values after choosing enabled option: %v", values)
	}
	r.One("#sizes-items-3[aria-selected=true][aria-disabled=true]")
	r.One("span:text(Small, Huge)")
	want(t, r, "input[type=hidden]", "value", `["s","x"]`)
}

func TestSelectMultipleValuesAreInItemOrderAndFresh(t *testing.T) {
	var st ui.SelectState
	values := []string{}
	var reports [][]string
	items := []ui.SelectItem{{Value: "a", Label: "Same"}, {Value: "b", Label: "Same", Group: "Other"}}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Select(&st, ui.SelectProps{ID: "select", Items: items, Multiple: true, Values: values, Placeholder: "Pick", OnValuesChange: func(v []string) { values = v; reports = append(reports, v) }}).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#select-trigger")
	r.Key("end")
	r.Key("enter")
	r.Key("home")
	r.Key("enter")
	if !slices.Equal(values, []string{"a", "b"}) || len(reports) != 2 || !slices.Equal(reports[0], []string{"b"}) {
		t.Fatalf("values %v reports %v", values, reports)
	}
	r.Key("enter")
	if !slices.Equal(values, []string{"b"}) || !slices.Equal(reports[1], []string{"a", "b"}) {
		t.Fatalf("toggle values %v reports %v", values, reports)
	}
}

func TestSelectRejectsAmbiguousValues(t *testing.T) {
	f := &faults{T: t}
	var st ui.SelectState
	p := ui.SelectProps{ID: "select", Items: []ui.SelectItem{{Value: "a", Label: "One"}, {Value: "a", Label: "Two"}}, Multiple: true, Values: []string{"missing"}}
	surftest.Start(f, surf.New("select", ui.Select(&st, p), ui.Stylesheet()))
	if len(f.got) < 2 {
		t.Fatalf("expected duplicate and absent value faults: %v", f.got)
	}
}

func TestSelectMultipleEscapesHiddenValues(t *testing.T) {
	var st ui.SelectState
	value := "a\"b\n"
	r := start(t, ui.Select(&st, ui.SelectProps{ID: "select", Items: []ui.SelectItem{{Value: value, Label: "A"}}, Multiple: true, Values: []string{value}}))
	want(t, r, "input[type=hidden]", "value", `["a\"b\n"]`)
}
