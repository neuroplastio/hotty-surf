package main

import (
	_ "embed"

	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the kit's stylesheet, the app's own Tailwind utilities, then the
// nav's palette (theme.go).
var CSS = ui.Stylesheet() + "\n" + appCSS + "\n" + themeCSS

// surface is the surface the app shows on.
const surface = "nav"
