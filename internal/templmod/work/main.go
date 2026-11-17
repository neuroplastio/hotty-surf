// Command work makes the reloader's go.work for this repository, with
// mods/templ applied, in the directory it is given, and prints its path:
// the gate vets and lints internal/devchild's surftemplmod file with
// GOWORK set to it, since that file compiles only against the patched
// templ.
package main

import (
	"context"
	"fmt"
	"os"

	"neuroplast.io/hotty-surf/internal/templmod"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: work <out dir>")
		os.Exit(2)
	}
	w, err := templmod.Workspace(context.Background(), ".", os.Environ(), os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "work:", err)
		os.Exit(1)
	}
	fmt.Println(w)
}
