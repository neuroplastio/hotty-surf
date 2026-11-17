package ui_test

import (
	"context"
	"io"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/ui"
)

type tabsApp struct{ value string }

func (a *tabsApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(with(ui.Tabs(ui.TabsProps{
			ID: "t",
			Tabs: []ui.Tab{
				{Value: "account", Label: "Account"},
				{Value: "password", Label: "Password"},
				{Value: "billing", Label: "Billing", Disabled: true},
				{Value: "team", Label: "Team"},
			},
			Value:    a.value,
			OnChange: func(v string) { a.value = v },
		}), text("panel-text", "Panel "+a.value))).Render(ctx, w)
	})
}

func TestTabs(t *testing.T) {
	a := &tabsApp{value: "account"}
	r := start(t, a.view())
	// sel checks that tab n is the selected one and its panel the one
	// shown.
	sel := func(n string) {
		t.Helper()
		want(t, r, "#t-tab-"+n, "aria-selected", "true")
		want(t, r, "#t-tab-"+n, "tabindex", "0")
		r.One("div#t-panel-" + n + "[role=tabpanel][aria-labelledby=t-tab-" + n + "]")
		if got := len(r.Find("[role=tabpanel]")); got != 1 {
			t.Errorf("%d panels shown", got)
		}
	}
	sel("1")
	want(t, r, "#t-tab-2", "aria-selected", "false")
	want(t, r, "#t-tab-2", "tabindex", "-1")
	if !hasAttr(r.One("#t-tab-3"), "disabled") {
		t.Error("Billing isn't disabled")
	}

	r.Click("#t-tab-2")
	if a.value != "password" {
		t.Fatalf("value %q after a click on Password", a.value)
	}
	sel("2")
	r.One("p:text(Panel password)")

	// The keyboard is on Password, and follows the selection.
	for _, s := range []struct{ key, value, tab string }{
		{"arrow_right", "team", "4"}, // past the disabled tab
		{"arrow_right", "account", "1"},
		{"arrow_left", "team", "4"},
		{"home", "account", "1"},
		{"end", "team", "4"},
		{"arrow_left", "password", "2"},
	} {
		if !r.Key(s.key) {
			t.Errorf("%s: not taken", s.key)
		}
		if a.value != s.value {
			t.Fatalf("%s: value %q, not %q", s.key, a.value, s.value)
		}
		sel(s.tab)
	}
	if r.Key("arrow_down") {
		t.Error("arrow_down on a horizontal tablist: taken")
	}
}

func TestTabsUnknownValue(t *testing.T) {
	r := start(t, ui.Tabs(ui.TabsProps{ID: "t", Tabs: []ui.Tab{
		{Value: "a", Label: "A", Disabled: true},
		{Value: "b", Label: "B"},
	}, Value: "zz"}))
	// tabs.js's refresh: the first tab not disabled.
	want(t, r, "#t-tab-2", "aria-selected", "true")
	want(t, r, "#t-tab-1", "aria-selected", "false")
}

// With nothing focused, an arrow reaches the outermost Key handlers: only
// the selected tab's, so each arrow moves the selection on from it.
func TestTabsKeysWithNothingFocused(t *testing.T) {
	a := &tabsApp{value: "account"}
	r := start(t, a.view())
	for _, want := range []string{"password", "team", "account"} {
		r.Key("arrow_right")
		if a.value != want {
			t.Fatalf("value %q after arrow_right, want %q", a.value, want)
		}
	}
}

func TestTabsLineAndInnerClasses(t *testing.T) {
	view := ui.Tabs(ui.TabsProps{ID: "sample", Label: "Details", Line: true,
		ListClass: "test-list", PanelClass: "test-panel",
		Tabs: []ui.Tab{{Value: "one", Label: "One"}}, Value: "one"})
	r := start(t, view)
	want(t, r, "nav[role=tablist]", "aria-label", "Details")
	want(t, r, "nav[role=tablist]", "data-variant", "line")
	want(t, r, "nav[role=tablist]", "class", "test-list")
	want(t, r, "div[role=tabpanel]", "class", "test-panel")
}
