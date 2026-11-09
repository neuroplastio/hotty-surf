package conformance

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"neuroplast.io/hotty-surf/surflive"
)

// A Mode is a way a case's root is drawn: the interpreter's, or the
// compiled mode's. Root makes the root the case names (its root file,
// "" for none) with the state its state.json holds, and the log of what
// its handlers have done, which a host reads on the runtime's goroutine.
type Mode struct {
	Name string
	Root func(t testing.TB, root string, state []byte) (surflive.Component, func() []string)
}

// A Suite runs a package's own cases (the format TestCases says) in each
// of its modes on every host: the runtime and a host of its own, a
// terminal's cells and a renderer's. It is how a package that registers
// widgets (surf.Register) holds both modes, and the cells, to the same
// document for them, with its own fixtures and cases.
//
// A case is a txtar file, cases/NAME.txt in Cases. One with a load file
// or templates of its own isn't run: those are the root suite's.
type Suite struct {
	// Cases holds the files, cases/*.txt.
	Cases fs.FS
	// Dir is where Cases is on disk, which Update writes to.
	Dir string
	// Update writes each case's files as the first mode gives them (the
	// document, the ops, the log and the cells), keeping the case's
	// comment, root, scroll and state.json. A package gives it a flag.
	Update bool
	// Modes are the ways a root is drawn; the first is the one Update
	// writes from, and every case must hold in all, alike.
	Modes []Mode
}

// Run runs every case of the suite.
func (s *Suite) Run(t *testing.T) {
	t.Helper()
	files, _ := fs.Glob(s.Cases, "cases/*.txt")
	if len(files) == 0 {
		t.Fatal("no cases")
	}
	for _, file := range files {
		src, err := fs.ReadFile(s.Cases, file)
		if err != nil {
			t.Fatal(err)
		}
		ar := txtar.Parse(src)
		if _, ok := File(ar, "load"); ok || OwnTemplates(ar) {
			continue
		}
		s.Case(t, file, ar)
	}
}

// players are the hosts a case plays on: the runtime (Play), and a host
// of its own on surfrun.Surface (PlayHosted), whose files are the runtime's;
// and the cells, the runtime's (PlayCells) and a renderer's on
// surfrun.Surface (PlayCellsHosted), whose files are the same, their logs the
// hosts'.
var players = []struct {
	name string // the subtest's, after the mode's
	play player
}{{"", Play}, {"/hosted", PlayHosted}, {"/cells", PlayCells}, {"/cells/hosted", PlayCellsHosted}}

// player is Play or PlayHosted.
type player func(testing.TB, *txtar.Archive, surflive.Component, func() []string) ([]txtar.File, []string)

// Case runs the case in file, parsed as ar, in each mode on each host.
func (s *Suite) Case(t *testing.T, file string, ar *txtar.Archive) {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(file), ".txt")
	errs := map[string][]string{}
	for _, m := range s.Modes {
		for _, h := range players {
			if s.Update && h.name != "" {
				continue // the files are the runtime's
			}
			t.Run(name+"/"+m.Name+h.name, func(t *testing.T) {
				got, heard := s.play(t, h.play, ar, m)
				errs[m.Name+h.name] = heard
				if _, ok := File(ar, "errors"); ok {
					got = append(got, txtar.File{Name: "errors", Data: []byte(errorsFile(heard))})
				}
				if s.Update {
					if m.Name == s.Modes[0].Name {
						for _, name := range []string{"scroll", "root"} { // kept, before state.json
							if data, ok := File(ar, name); ok {
								got = append([]txtar.File{{Name: name, Data: data}}, got...)
							}
						}
						drawn, _ := s.play(t, PlayCells, ar, m)
						ar.Files = withCells(got, drawn)
						if err := os.WriteFile(filepath.Join(s.Dir, file), txtar.Format(ar), 0o644); err != nil {
							t.Fatal(err)
						}
					}
					return
				}
				want := map[string]string{}
				for _, f := range ar.Files {
					want[f.Name] = string(f.Data)
				}
				for _, f := range got {
					if w := want[f.Name]; string(f.Data) != w {
						t.Errorf("%s:\n got %s\nwant %s", f.Name, f.Data, w)
					}
				}
			})
		}
	}
	// The modes fail alike: the same errors, first seen in the same
	// order. (How many renders a case takes is the runtime's timing,
	// so each error counts once, as a View's stderr writes it.) A host
	// of its own fails as the runtime does.
	first := strings.Join(errs[s.Modes[0].Name], "\n")
	for _, m := range s.Modes[1:] {
		if other := strings.Join(errs[m.Name], "\n"); first != other {
			t.Errorf("%s: the modes' errors differ:\n%s:\n%s\n%s:\n%s", file, s.Modes[0].Name, first, m.Name, other)
		}
	}
	for _, m := range s.Modes {
		if r, h := strings.Join(errs[m.Name], "\n"), strings.Join(errs[m.Name+"/hosted"], "\n"); !s.Update && r != h {
			t.Errorf("%s, %s: the hosts' errors differ:\nthe runtime's:\n%s\nthe hosted:\n%s", file, m.Name, r, h)
		}
	}
}

// play runs a case in a mode on a host (Play or PlayHosted): its files as
// they come out, and the errors its renders reported.
func (s *Suite) play(t *testing.T, host player, ar *txtar.Archive, m Mode) ([]txtar.File, []string) {
	state, _ := File(ar, "state.json")
	root, _ := File(ar, "root")
	c, log := m.Root(t, strings.TrimSpace(string(root)), state)
	return host(t, ar, c, log)
}

// withCells is a case's files with the cells drawn among them: cells
// after the document, and each N.cells after its N.document.
func withCells(files, drawn []txtar.File) []txtar.File {
	cells := map[string][]byte{}
	for _, f := range drawn {
		if f.Name == "cells" || strings.HasSuffix(f.Name, ".cells") {
			cells[f.Name] = f.Data
		}
	}
	var out []txtar.File
	for _, f := range files {
		out = append(out, f)
		if name, ok := strings.CutSuffix(f.Name, "document"); ok {
			if data, ok := cells[name+"cells"]; ok {
				out = append(out, txtar.File{Name: name + "cells", Data: data})
			}
		}
	}
	return out
}

// errorsFile is an errors file of the errors heard.
func errorsFile(heard []string) string {
	if len(heard) == 0 {
		return ""
	}
	return strings.Join(heard, "\n") + "\n"
}
