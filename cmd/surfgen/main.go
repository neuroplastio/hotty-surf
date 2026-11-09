// Command surfgen writes a package's surf bindings, surf_gen.go: what
// its .surf templates read and call of its Go types, with no reflection
// (gov NEIO-15), and the templates compiled into Go. It is
// pkg/surfgen's Main, whose doc says what it writes.
//
//	//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen
//
// A package that uses widgets registered from outside writes its own
// command over pkg/surfgen instead (docs/surf.md, "Widgets from outside").
package main

import "neuroplast.io/hotty-surf/pkg/surfgen"

func main() { surfgen.Main() }
