package main

import _ "embed"

// CSS is the app's stylesheet: the kit's and the app's classes, in
// style/app.css. Until `hotty-surf build` writes it (a later leg), it is
// edited by hand, and `hotty-surf run` takes an edit as a css reload.
//
//go:embed style/app.css
var CSS string
