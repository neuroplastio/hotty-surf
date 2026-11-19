package ui_test

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// plx's rows in each of the three menus: a dot in the icon's
// place, a note in the keys' place, a rule between items, an on/off row
// with an aside, and a checkable row whose Leading draws its state in
// place of the kit's check. The app's markup goes where it says; the row
// is still the item: its name, its press, its aria-checked.
func TestMenuRows(t *testing.T) {
	ran := 0
	items := []ui.MenuItem{
		{Label: "work", Leading: templ.Raw(`<span id="dot" class="dot" style="color:#3a7"></span>`),
			Trailing: templ.Raw(`<span id="note">3 tabs</span>`), OnSelect: func() { ran++ }},
		{Role: "separator", Label: "ignored"},
		{Label: "Wrap", Role: "menuitemcheckbox", Checked: true,
			Content: templ.Raw(`<span id="content">Wrap <small>long lines</small></span>`)},
		{Label: "Follow", Role: "menuitemcheckbox",
			Leading: templ.Raw(`<span id="look" class="switch-look" aria-hidden="true"></span>`)},
	}
	for _, kind := range []string{"dropdown", "context", "menu"} {
		t.Run(kind, func(t *testing.T) {
			var dd ui.DropdownMenuState
			var cm ui.ContextMenuState
			var view templ.Component
			switch kind {
			case "dropdown":
				view = ui.DropdownMenu(&dd, ui.DropdownMenuProps{ID: "m", Items: items, Trigger: templ.Raw("Open")})
			case "context":
				view = with(ui.ContextMenu(&cm, ui.ContextMenuProps{ID: "m", Items: items}), templ.Raw(`<p>area</p>`))
			default:
				view = ui.Menu(ui.MenuProps{ID: "m", Items: items, Highlight: -1})
			}
			r := surftest.Start(t, surf.New("rows", page(view), ui.Stylesheet()))
			html := r.HTML()
			for _, want := range []string{`id="dot"`, `data-shortcut=""><span id="note">`, `id="note"`, `id="content"`, `aria-label="Wrap"`, `<hr role="separator" id="m-items-2"`, `data-leading="" aria-hidden="true"><span id="look"`} {
				if !strings.Contains(html, want) {
					t.Errorf("no %s in %s", want, html)
				}
			}
			if strings.Contains(html, "ignored") {
				t.Error("a separator's label was drawn")
			}
			// Wrap has the kit's check; Follow's Leading stands in its
			// place, its state in aria-checked.
			if n := strings.Count(html, "data-indicator"); n != 1 {
				t.Errorf("%d indicators, want Wrap's alone", n)
			}
			follow := r.One("#m-items-4")
			if got := attr(follow, "aria-checked"); got != "false" {
				t.Errorf("Follow's aria-checked %q", got)
			}
			r.Click("#m-items-1")
			if ran != 1 {
				t.Fatalf("a click on a row with slots ran %d", ran)
			}
			ran = 0
		})
	}
}

// A slot is a picture (K31), a slot isn't set with the field it replaces,
// and a rule goes only between items.
func TestMenuRowFaults(t *testing.T) {
	for _, c := range []struct {
		name  string
		items []ui.MenuItem
		want  string
	}{
		{"switch", []ui.MenuItem{{Label: "Wrap", Leading: ui.Switch(ui.SwitchProps{Checked: true})}}, "Leading holds an input"},
		{"button", []ui.MenuItem{{Label: "x", Trailing: templ.Raw(`<button>go</button>`)}}, "Trailing holds a button"},
		{"link", []ui.MenuItem{{Label: "x", Content: templ.Raw(`<a href="/">x</a>`)}}, "Content holds a link"},
		{"tabindex", []ui.MenuItem{{Label: "x", Content: templ.Raw(`<span tabindex="0">x</span>`)}}, "Content holds a tabindex"},
		{"icon", []ui.MenuItem{{Label: "x", Icon: "check", Leading: templ.Raw(`<i></i>`)}}, "Leading and Icon both set"},
		{"keys", []ui.MenuItem{{Label: "x", Keys: "mod+s", Trailing: templ.Raw(`<i></i>`)}}, "Trailing and Keys both set"},
		{"first", []ui.MenuItem{{Role: "separator"}, {Label: "a"}}, "separator first or last"},
		{"last", []ui.MenuItem{{Label: "a"}, {Role: "separator"}}, "separator first or last"},
		{"two", []ui.MenuItem{{Label: "a"}, {Role: "separator"}, {Role: "separator"}, {Label: "b"}}, "two separators together"},
		{"group", []ui.MenuItem{{Label: "a", Group: "One"}, {Role: "separator"}, {Label: "b", Group: "Two"}}, "at a group change"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &faults{T: t}
			surftest.Start(f, surf.New("rows", page(ui.Menu(ui.MenuProps{ID: "m", Items: c.items, Highlight: -1})), ui.Stylesheet()))
			if !strings.Contains(strings.Join(f.got, "\n"), c.want) {
				t.Fatalf("faults %q, want one with %q", f.got, c.want)
			}
		})
	}
}

// A separator inside a group keeps the group whole: one heading, one
// rule, no group rules around it.
func TestMenuSeparatorInGroup(t *testing.T) {
	items := []ui.MenuItem{
		{Label: "a", Group: "Hosts"}, {Role: "separator"},
		{Label: "b", Group: "Hosts"}, {Label: "c", Group: "Hosts"},
	}
	r := surftest.Start(t, surf.New("rows", page(ui.Menu(ui.MenuProps{ID: "m", Items: items, Highlight: -1})), ui.Stylesheet()))
	html := r.HTML()
	if n := strings.Count(html, `role="heading"`); n != 1 {
		t.Errorf("%d headings, want one Hosts", n)
	}
	if n := strings.Count(html, `<hr`); n != 1 {
		t.Errorf("%d rules, want the separator's alone", n)
	}
}
