// Package surfgen is the generator behind cmd/surfgen. It writes a
// package's surf bindings, surf_gen.go: what its .surf templates read and
// call of its Go types, with no reflection (gov NEIO-15), and the
// templates compiled into Go.
//
//	//go:generate go run neuroplast.io/hotty-surf/cmd/surfgen
//
// in a package whose directory holds .surf files. A template named for a
// type (agent-row.surf for AgentRow, notices.surf for notices) is that
// component's; any other is a partial. For each component, and each
// struct its exported fields and methods reach, it writes:
//
//   - surfShapeT, its Shape, and surfShapes, all of them;
//   - surfT, an Instance (a component) or an Object (a struct) over a *T;
//   - surfInterpT(c *T) surflive.Component: c drawn by the embedded
//     templates, interpreted;
//   - surfCompiledT(c *T) surflive.Component: c drawn by its templates
//     compiled into Go (docs/spec.md, "Modes").
//
// With -export it writes SurfPackage() surf.Package too: the templates,
// the shapes and each component's Instance, exported for a tool that
// draws the package's components with templates of its own beside
// them (cmd/surfgallery draws screens' stories so).
//
// It checks the templates against the types as it writes them, and
// fails where they don't compile. Past -max-memory MiB of heap (2048) it
// stops itself, since go generate gives it no cap.
//
// A package whose templates use a widget registered from outside
// (surf.Register) can't use cmd/surfgen as it is, which links none: the
// registrations are Go values, run by the packages' init. It writes its
// own command, a main that links the registering packages and calls
// Main (docs/surf.md, "Widgets from outside"):
//
//	package main
//
//	import (
//		_ "example.com/gauge"
//		"neuroplast.io/hotty-surf/pkg/surfgen"
//	)
//
//	func main() { surfgen.Main() }
//
// The generated code imports each widget's Pkg and calls its Func.
package surfgen

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Main is surfgen's command line: it reads the flags (-dir, -o,
// -max-memory, -export) and writes the package's bindings, or exits with
// the error. The widgets registered by the packages the program links
// (surf.Register) are the vocabulary the templates are checked against.
func Main() {
	dir := flag.String("dir", ".", "the package's directory")
	out := flag.String("o", "surf_gen.go", "the file to write, in the directory")
	maxMiB := flag.Int64("max-memory", 2048, "the MiB of heap past which surfgen stops")
	export := flag.Bool("export", false, "write SurfPackage, for a tool that draws the package's components (a gallery)")
	flag.Parse()
	ceiling(*maxMiB << 20)
	src, err := Generate(*dir, *export)
	if err != nil {
		fmt.Fprintln(os.Stderr, "surfgen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*dir, *out), src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "surfgen:", err)
		os.Exit(1)
	}
}
