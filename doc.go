// Package surf loads, checks and draws .surf templates: hotty-surf's
// layout language, in which a component's template lays out the widgets
// of package surfui, with CEL expressions (cel-lite) for values and
// handlers. A component is a Go struct: its exported fields are its
// state, its exported methods its handlers.
//
// There are two modes, from the same files. Interpreted, Load reads a
// directory of templates into a Set at run time, checked against the
// Shapes of the types they draw, and New or Component draws an Instance
// by its template as a surflive.Component; LoadDir reloads them as they
// change. Compiled, cmd/surfgen writes each template into Go at build
// time (Compiled). surfgen also writes the Shapes and the Instances, so
// nothing is looked up by reflection, and the package builds under
// TinyGo and js/wasm. A render nests at most MaxDepth levels.
//
//	//go:generate go run github.com/neuroplastio/hotty-surf/cmd/surfgen
//
//	m := surflive.New(surfInterpHello(&Hello{Name: "world"}))
//	err := surfrun.Run(ctx, m, surfrun.Options{Name: "hello"})
//
// The guide (docs/guide.md) shows a screen from its first template to its
// tests, the reference (docs/surf.md) lists every name, widget, attribute
// and directive, and the spec (docs/spec.md) has the exact rules.
package surf
