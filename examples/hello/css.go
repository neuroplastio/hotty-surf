package main

import (
	_ "embed"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the app's own stylesheet, style/app.css. The kit's is linked, not
// carried (ui.Sheet): sent once rather than in every document. Until
// `hotty-surf build` writes app.css (a later leg), it is edited by hand,
// and `hotty-surf run` takes an edit as a css reload.
var CSS = appCSS

// Surface is the app's surface: Page styled by the kit's sheet, then CSS.
func Surface(a *App) *surf.Surface {
	s := surf.New("hello", Page(a), CSS)
	s.Link(ui.Sheet())
	return s
}
