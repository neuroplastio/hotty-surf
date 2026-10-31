package conformance

import (
	"embed"
	"io/fs"
	"strings"
	"testing/fstest"

	"golang.org/x/tools/txtar"
)

// Cases are the suite's cases, cases/*.txt (TestCases says their
// format), for a runtime hosted in Go to run as this package's tests do.
// They are embedded: TinyGo's wasm has no file system to read them from.
//
//go:embed cases/*.txt
var Cases embed.FS

// Templates are the fixtures' templates, the .surf files beside them,
// for a host of the interpreter to draw its own fixtures with.
//
//go:embed *.surf
var Templates embed.FS

// File is the case's file name, if it has one.
func File(ar *txtar.Archive, name string) ([]byte, bool) {
	for _, f := range ar.Files {
		if f.Name == name {
			return f.Data, true
		}
	}
	return nil, false
}

// OwnTemplates says the case brings templates of its own (NAME.surf).
func OwnTemplates(ar *txtar.Archive) bool {
	for _, f := range ar.Files {
		if strings.HasSuffix(f.Name, ".surf") {
			return true
		}
	}
	return false
}

// Overlay is the fixtures' templates with a case's own in place of those
// of the same name, and beside the rest: what a case with a load file
// loads.
func Overlay(ar *txtar.Archive) fs.FS {
	out := fstest.MapFS{}
	ents, _ := fs.ReadDir(Templates, ".")
	for _, e := range ents {
		data, err := fs.ReadFile(Templates, e.Name())
		if err != nil {
			panic(err) // embedded: it reads
		}
		out[e.Name()] = &fstest.MapFile{Data: data}
	}
	for _, f := range ar.Files {
		if strings.HasSuffix(f.Name, ".surf") {
			out[f.Name] = &fstest.MapFile{Data: f.Data}
		}
	}
	return out
}
