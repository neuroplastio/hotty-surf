package ui_test

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// plx's rows (G-menu-rows) in each of the three menus: a dot and a switch
// in the icon's place, a label with an aside, a note in the keys' place,
// and a bare rule between items. The app's markup goes where it says; the
// row is still the item: its name, its press.
func TestMenuRows(t *testing.T) {
	ran := 0
	items := []ui.MenuItem{
		{Label: "work", Leading: templ.Raw(`<span id="dot" class="dot" style="color:#3a7"></span>`),
			Trailing: templ.Raw(`<span id="note">3 tabs</span>`), OnSelect: func() { ran++ }},
		{Separator: true, Label: "ignored"},
		{Label: "Wrap", Leading: ui.Switch(ui.SwitchProps{Checked: true}),
			Content: templ.Raw(`<span id="content">Wrap <small>long lines</small></span>`)},
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
			for _, want := range []string{`id="dot"`, `data-shortcut=""><span id="note">`, `id="note"`, `role="switch"`, `id="content"`, `aria-label="Wrap"`, `<hr role="separator" id="m-items-2"`} {
				if !strings.Contains(html, want) {
					t.Errorf("no %s in %s", want, html)
				}
			}
			if strings.Contains(html, "ignored") {
				t.Error("a Separator's label was drawn")
			}
			r.Click("#m-items-1")
			if ran != 1 {
				t.Fatalf("a click on a row with slots ran %d", ran)
			}
			ran = 0
		})
	}
}
