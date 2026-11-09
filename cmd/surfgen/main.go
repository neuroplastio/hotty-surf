// Command surfgen writes a package's surf bindings, surf_gen.go: what
// its .surf templates read and call of its Go types, with no reflection
// (gov NEIO-15), and the templates compiled into Go.
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
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir := flag.String("dir", ".", "the package's directory")
	out := flag.String("o", "surf_gen.go", "the file to write, in the directory")
	maxMiB := flag.Int64("max-memory", 2048, "the MiB of heap past which surfgen stops")
	export := flag.Bool("export", false, "write SurfPackage, for a tool that draws the package's components (a gallery)")
	flag.Parse()
	ceiling(*maxMiB << 20)
	src, err := generate(*dir, *export)
	if err != nil {
		fmt.Fprintln(os.Stderr, "surfgen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*dir, *out), src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "surfgen:", err)
		os.Exit(1)
	}
}
