package main

import (
	_ "embed"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
	uiselect "neuroplast.io/hotty-surf/ui/select"
)

//go:embed style/app.css
var appCSS string

// CSS is the app's own stylesheet, style/app.css. The kit's is linked,
// not carried (newSurface): sent once rather than in every document.
var CSS = appCSS

var kit = ui.NewRegistry().Register(uiselect.Register)

// newSurface is one of the app's surfaces: view, styled by the kit's sheet,
// linked rather than carried (kit.Sheet), then by CSS.
func newSurface(name string, view templ.Component) *surf.Surface {
	s := surf.New(name, view, CSS)
	s.Link(kit.Sheet())
	return s
}
