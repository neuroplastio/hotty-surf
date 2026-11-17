package main

import _ "embed"

//go:embed style/app.css
var appCSS string

// CSS is the app's own stylesheet, style/app.css. The kit's is linked, not
// carried (main's s.Link(ui.Sheet())): sent once rather than in every
// document.
var CSS = appCSS
