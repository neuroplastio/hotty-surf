package surftest_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	uiselect "neuroplast.io/hotty-surf/ui/select"
)

// faultsOf starts view and returns the faults its first render records.
func faultsOf(t *testing.T, view templ.Component) []string {
	t.Helper()
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", view, ui.Stylesheet()))
	return f.got
}

func raw(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// A button HTML names some other way than text or aria-label is named:
// by a <label for> it, a <label> around it, an image's alt.
func TestNamedButtons(t *testing.T) {
	for name, markup := range map[string]string{
		"label for":      `<label for="go">Go</label><button id="go"><svg></svg></button>`,
		"label around":   `<label>Go <button id="go"><svg></svg></button></label>`,
		"an image's alt": `<button id="go"><img src="data:," alt="Ada"></button>`,
		"sr-only text":   `<button id="go"><svg></svg><span class="sr-only">More</span></button>`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := faultsOf(t, raw(markup)); len(got) != 0 {
				t.Errorf("faults: %q", got)
			}
		})
	}
}

// The fault names where the author finds a button with no id of its own,
// and says what a kit component with no Attrs can do.
func TestUnnamedButtons(t *testing.T) {
	var sel uiselect.State
	var menu ui.DropdownMenuState
	for _, c := range []struct {
		name string
		view templ.Component
		want string
	}{
		{"in an app's element", raw(`<ul id="todo-list"><li><button><svg></svg></button></li></ul>`), "(in #todo-list)"},
		{"a Select with nothing chosen", uiselect.View(&sel, uiselect.Props{ID: "size",
			Items: []uiselect.Item{{Value: "s", Label: "Small"}}}), "takes a Placeholder"},
		{"a menu with an icon alone", ui.DropdownMenu(&menu, ui.DropdownMenuProps{ID: "more",
			Trigger: ui.Icon("ellipsis", ""), Items: []ui.MenuItem{{Label: "Edit"}}}), `<span class="sr-only">`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := faultsOf(t, c.view)
			if len(got) != 1 || !strings.Contains(got[0], "a button with no text and no label") || !strings.Contains(got[0], c.want) {
				t.Errorf("faults: %q, want one with %q", got, c.want)
			}
		})
	}
}
