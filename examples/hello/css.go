package main

import (
	_ "embed"

	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the kit's stylesheet, then the app's own, style/app.css. Until
// `hotty-surf build` writes that (a later leg), it is edited by hand, and
// `hotty-surf run` takes an edit as a css reload.
var CSS = ui.Stylesheet() + "\n" + appCSS
