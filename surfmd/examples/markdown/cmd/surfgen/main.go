// Command surfgen is surfgen with surfmd linked: a package whose templates
// draw <surf-markdown> needs one, since the registration is a Go value that
// the package's init runs (hotty-surf's docs/surf.md, "Widgets from
// outside").
package main

import (
	_ "neuroplast.io/hotty-surf/surfmd"

	"neuroplast.io/hotty-surf/pkg/surfgen"
)

func main() { surfgen.Main() }
