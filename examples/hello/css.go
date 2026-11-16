package main

import _ "embed"

// CSS is the app's stylesheet: the kit's and the app's classes, built by
// `hotty-surf build` into style/app.css.
//
//go:embed style/app.css
var CSS string
