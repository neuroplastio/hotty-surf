package main

import (
	_ "embed"

	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the kit's stylesheet, then the app's own (style/app.css), which
// hotty-surf run swaps in as the file is saved.
var CSS = ui.Stylesheet() + "\n" + appCSS
