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

// openApp has one overlay of each kind, and can drop the popover from the
// view.
type openApp struct {
	menu    ui.DropdownMenuState
	inMenu  ui.DropdownMenuState
	ctxMenu ui.ContextMenuState
	sel     ui.SelectState
	pop     ui.PopoverState
	dialog  bool
	noPop   bool
}

func (a *openApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := []ui.MenuItem{{Label: "One", OnSelect: func() {}}}
		row := with(ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "row", surf.Click(func() {}))}), templ.Raw("Row"))
		parts := []templ.Component{
			templ.Raw(`<p id="outside" tabindex="0">Outside</p>`),
			ui.DropdownMenu(&a.menu, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Menu"), Items: items}),
			with(ui.ContextMenu(&a.ctxMenu, ui.ContextMenuProps{ID: "cm", Items: items}), row),
			ui.Select(&a.sel, ui.SelectProps{ID: "size", Placeholder: "Size", Items: []ui.SelectItem{{Value: "s", Label: "Small"}}}),
		}
		if !a.noPop {
			parts = append(parts, ui.Popover(&a.pop, ui.PopoverProps{ID: "p", Trigger: templ.Raw("Pop")}))
		}
		parts = append(parts, with(ui.Dialog(ui.DialogProps{ID: "dlg", Title: "Dialog", Open: a.dialog,
			OnClose: func() { a.dialog = false }}),
			ui.DropdownMenu(&a.inMenu, ui.DropdownMenuProps{ID: "dm", Trigger: templ.Raw("In"), Items: items})))
		return page(parts...).Render(ctx, w)
	})
}

// Open says which of the kit's overlays the last frame drew open: the
// modals outermost first, then the one popup.
func TestSurfaceOpen(t *testing.T) {
	a := &openApp{}
	s := surf.New("open", a.view(), ui.Stylesheet())
	r := surftest.Start(t, s)
	open := func(want ...string) {
		t.Helper()
		if got := s.Open(); !slices.Equal(got, want) || (want == nil) != (got == nil) {
			t.Fatalf("Open() = %q, want %q", got, want)
		}
	}
	open()
	r.Click("#m-trigger")
	open("m")
	r.Key("escape")
	open()
	r.Click("#row")
	r.Secondary("#row")
	open("cm")
	r.Click("#size-trigger") // one opening closes the other
	open("size")
	r.Click("#outside")
	open()
	r.Click("#p-trigger")
	open("p")
	a.noPop = true
	r.Update() // gone from the view, open nowhere
	open()
	a.noPop = false
	a.pop = ui.PopoverState{}

	// A popup in a modal: the modal first.
	a.dialog = true
	r.Update()
	open("dlg")
	r.Click("#dm-trigger")
	open("dlg", "dm")
	r.Key("escape") // the menu's Escape
	open("dlg")
	a.dialog = false
	r.Update()
	open()
}

// A popup its State opened before the first document is open in it.
func TestSurfaceOpenFirstDocument(t *testing.T) {
	var st ui.PopoverState
	st.Open()
	s := surf.New("open", ui.Popover(&st, ui.PopoverProps{ID: "p", Trigger: templ.Raw("Pop")}), ui.Stylesheet())
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if got := s.Open(); !slices.Equal(got, []string{"p"}) {
		t.Fatalf("Open() = %q, want the popover", got)
	}
}
