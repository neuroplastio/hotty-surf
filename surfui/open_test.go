package surfui_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
	uibutton "neuroplast.io/hotty-surf/surfui/button"
	"neuroplast.io/hotty-surf/surfui/dialog"
	"neuroplast.io/hotty-surf/surfui/menu"
	"neuroplast.io/hotty-surf/surfui/popover"
	uiselect "neuroplast.io/hotty-surf/surfui/select"
)

// openApp has one overlay of each kind, and can drop the popover from the
// view.
type openApp struct {
	menu    menu.DropdownState
	inMenu  menu.DropdownState
	ctxMenu menu.ContextState
	sel     uiselect.State
	pop     popover.State
	dialog  bool
	noPop   bool
}

func (a *openApp) view() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		items := []menu.Item{{Label: "One", OnSelect: func() {}}}
		row := with(uibutton.View(uibutton.Props{ID: "row", OnClick: func() {}}), templ.Raw("Row"))
		parts := []templ.Component{
			templ.Raw(`<p id="outside" tabindex="0">Outside</p>`),
			menu.Dropdown(&a.menu, menu.DropdownProps{ID: "m", Trigger: templ.Raw("Menu"), Items: items}),
			with(menu.Context(&a.ctxMenu, menu.ContextProps{ID: "cm", Items: items}), row),
			uiselect.View(&a.sel, uiselect.Props{ID: "size", Placeholder: "Size", Items: []uiselect.Item{{Value: "s", Label: "Small"}}}),
		}
		if !a.noPop {
			parts = append(parts, popover.View(&a.pop, popover.Props{ID: "p", Trigger: templ.Raw("Pop")}))
		}
		parts = append(parts, with(dialog.View(dialog.Props{ID: "dlg", Title: "Dialog", Open: a.dialog,
			OnClose: func() { a.dialog = false }}),
			menu.Dropdown(&a.inMenu, menu.DropdownProps{ID: "dm", Trigger: templ.Raw("In"), Items: items})))
		return page(parts...).Render(ctx, w)
	})
}

// Overlays says which of the kit's overlays the last frame drew open: the
// modals outermost first, then the one popup.
func TestSurfaceOverlays(t *testing.T) {
	a := &openApp{}
	s := surf.New("open", a.view(), surfui.Stylesheet())
	r := surftest.Start(t, s)
	open := func(want ...string) {
		t.Helper()
		if got := s.Overlays(); !slices.Equal(got, want) || (want == nil) != (got == nil) {
			t.Fatalf("Overlays() = %q, want %q", got, want)
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
	a.pop = popover.State{}

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
func TestSurfaceOverlaysFirstDocument(t *testing.T) {
	var st popover.State
	st.Open()
	s := surf.New("open", popover.View(&st, popover.Props{ID: "p", Trigger: templ.Raw("Pop")}), surfui.Stylesheet())
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if got := s.Overlays(); !slices.Equal(got, []string{"p"}) {
		t.Fatalf("Overlays() = %q, want the popover", got)
	}
}
