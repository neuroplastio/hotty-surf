package ui_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/a-h/templ"

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
