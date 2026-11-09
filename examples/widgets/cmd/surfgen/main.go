// Command surfgen is surfgen with this example's widgets linked: a
// package whose templates use widgets registered from outside needs one,
// since the registrations are Go values that the packages' init runs
// (docs/surf.md, "Widgets from outside").
package main

import (
	_ "neuroplast.io/hotty-surf/examples/widgets/gauge"
	"neuroplast.io/hotty-surf/pkg/surfgen"
)

func main() { surfgen.Main() }
