package conformance

import (
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfui"
)

var update = flag.Bool("update", false, "write each case's document, ops and log as the interpreter gives them (Cases, for the next build)")

// mode is a way a root is drawn, for each fixture.
type mode struct {
	name  string
	view  func(*Board) surflive.Component
	form  func(*Form) surflive.Component
	frame func(*Frame) surflive.Component
	shelf func(*Shelf) surflive.Component
	deck  func(*Deck) surflive.Component
	crew  func(*Crew) surflive.Component
	draft func(*Draft) surflive.Component
	links func(*Links) surflive.Component
	prose func(*Prose) surflive.Component
}

// modes are the ways a root is drawn: each case holds in both, alike.
var modes = []mode{
	{"interpreted", surfInterpBoard, surfInterpForm, surfInterpFrame, surfInterpShelf, surfInterpDeck, surfInterpCrew, surfInterpDraft, surfInterpLinks, surfInterpProse},
	{"compiled", surfCompiledBoard, surfCompiledForm, surfCompiledFrame, surfCompiledShelf, surfCompiledDeck, surfCompiledCrew, surfCompiledDraft, surfCompiledLinks, surfCompiledProse},
}

// suite runs the cases in the fixtures' two modes.
var suite = Suite{Cases: Cases, Dir: ".", Modes: []Mode{
	{"interpreted", func(t testing.TB, root string, state []byte) (surflive.Component, func() []string) {
		return fixtureRoot(t, root, state, modes[0])
	}},
	{"compiled", func(t testing.TB, root string, state []byte) (surflive.Component, func() []string) {
		return fixtureRoot(t, root, state, modes[1])
	}},
}}

// fixtureRoot is a case's root in a mode, a Board or the fixture it
// names: its state read from state.
func fixtureRoot(t testing.TB, root string, state []byte, m mode) (surflive.Component, func() []string) {
	read := func(v any) {
		if err := json.Unmarshal(state, v); err != nil {
			t.Fatal(err)
		}
	}
	switch root {
	case "", "Board":
		b := &Board{}
		read(b)
		return m.view(b), func() []string { return b.Log }
	case "Form":
		f := &Form{}
		read(f)
		return m.form(f), func() []string { return f.Log }
	case "Frame":
		f := &Frame{}
		read(f)
		return m.frame(f), func() []string { return f.Log }
	case "Shelf":
		s := &Shelf{}
		read(s)
		return m.shelf(s), func() []string { return s.Log }
	case "Deck":
		d := &Deck{}
		read(d)
		return m.deck(d), func() []string { return d.Log }
	case "Crew":
		c := &Crew{}
		read(c)
		return m.crew(c), func() []string { return c.Log }
	case "Draft":
		d := &Draft{}
		read(d)
		return m.draft(d), func() []string { return d.Log }
	case "Links":
		l := &Links{}
		read(l)
		return m.links(l), func() []string { return l.Log }
	case "Prose":
		p := &Prose{}
		read(p)
		return m.prose(p), func() []string { return p.Log }
	}
	t.Fatalf("root: no fixture %s", root)
	return nil, nil
}

// A case is a txtar file (cases/*.txt): its comment says what it shows;
// state.json is the root, a Board, or the fixture its root file names
// (Form, Frame, Shelf); a scroll file runs it in a run whose document
// scrolls; document is what the host shows, and cells what the cells
// show (PlayCells); then each step N has N.event (`click ID`, `choose ID
// VALUE` (a select's option, picked as the user picks it), `fill ID
// TEXT`, `key NAME`; a drag's `press ID`, its start, `over ID [STEP]`,
// the pointer onto ID at step STEP of a place to drop on split in steps,
// and `release ID [STEP]`; several, split by `;`, are one step), and what
// follows it: N.log (what the handlers did,
// on a host and in the cells alike), N.ops (the ops the runtime sent),
// N.document and N.cells. Where it has an errors file, that is the render
// errors the host hears, one a line, each once, as first heard.
//
// A case with a load file brings templates (NAME.surf) that join the
// fixtures', or take the place of one of the same name, and don't load:
// load is the error, a line each, as Load gives it. The compiled mode's
// load is surfgen's, which pkg/surfgen's TestConformanceLoad checks
// against the same file. A typed file says why a host whose parameters
// are all dyn (surflua's) doesn't run the case.
//
// A case may instead bring its own templates and no load file, as
// NAME.surf files, the first of them the root's. Its state.json is then
// the root, read as a host reads JSON into its own values: a case for
// hosts with one number type (Shape.OneNumber; surflua's tests run them
// as a Lua host), which Go's modes skip. Its errors, where it has the
// file, are the render errors the host hears, one a line, each once, as
// first heard.
func TestCases(t *testing.T) {
	suite.Update = *update
	files, _ := fs.Glob(Cases, "cases/*.txt")
	if len(files) == 0 {
		t.Fatal("no cases")
	}
	for _, file := range files {
		src, err := fs.ReadFile(Cases, file)
		if err != nil {
			t.Fatal(err)
		}
		ar := txtar.Parse(src)
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		if _, ok := File(ar, "load"); ok {
			t.Run(name+"/interpreted", func(t *testing.T) { testLoad(t, file, ar) })
			continue
		}
		if OwnTemplates(ar) {
			t.Run(name, func(t *testing.T) {
				t.Skip("its own templates, drawn for its state as a host with one number type reads it: not a Go host's")
			})
			continue
		}
		suite.Case(t, file, ar)
	}
}

// testLoad loads a case's templates with the fixtures', which fails with
// its load file's errors.
func testLoad(t *testing.T, file string, ar *txtar.Archive) {
	want, _ := File(ar, "load")
	_, err := surf.Load(Overlay(ar), surfShapes)
	got := "<nil>\n"
	if err != nil {
		got = err.Error() + "\n"
	}
	if *update {
		for i := range ar.Files {
			if ar.Files[i].Name == "load" {
				ar.Files[i].Data = []byte(got)
			}
		}
		if err := os.WriteFile(file, txtar.Format(ar), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got != string(want) {
		t.Errorf("load:\n got %s\nwant %s", got, want)
	}
}

// A render goes as deep as surf.MaxDepth and no deeper, in both modes:
// deep.txt's trees are drawn whole, and toodeep.txt's, a branch deeper,
// with the element past the limit left out and ErrTooDeep reported
// there. At the limit, the tree's walks run too, the deepest text
// changed for Diff. make stack runs it under TinyGo on its 64 KB stack,
// where a pause after each render checks the goroutine's stack canary:
// an overflow fails there.
func TestTooDeep(t *testing.T) {
	tooDeep := "lab-tree.surf:2:1: nested past 32 levels (surf.MaxDepth): left out"
	for _, c := range []struct {
		file string
		want []string
	}{
		{"cases/deep.txt", nil},
		{"cases/toodeep.txt", []string{tooDeep, tooDeep}},
	} {
		src, err := fs.ReadFile(Cases, c.file)
		if err != nil {
			t.Fatal(err)
		}
		var state []byte
		for _, f := range txtar.Parse(src).Files {
			if f.Name == "state.json" {
				state = f.Data
			}
		}
		for _, m := range modes {
			b := &Board{}
			if err := json.Unmarshal(state, b); err != nil {
				t.Fatal(err)
			}
			var deep []string
			on := func(err error) {
				if errors.Is(err, surf.ErrTooDeep) {
					deep = append(deep, err.Error())
				}
			}
			v := m.view(b)
			switch v := v.(type) {
			case *surf.View:
				v.OnError = on
			case *surf.CompiledView:
				v.OnError = on
			}
			old := v.Render(nil)
			runtime.Gosched()
			if got, want := strings.Join(deep, "\n"), strings.Join(c.want, "\n"); got != want {
				t.Errorf("%s, %s: %q, want %q", c.file, m.name, got, want)
			}
			if c.want != nil {
				continue
			}
			for _, tr := range b.Trees { // the deepest branch's text, renamed
				leaf := tr
				for leaf.Next != nil || len(leaf.Kids) > 0 {
					if leaf.Next != nil {
						leaf = leaf.Next
					} else {
						leaf = leaf.Kids[0]
					}
				}
				leaf.Name += "!"
			}
			next := v.Render(nil)
			surfui.Resolve(old, "r")
			surfui.Resolve(next, "r")
			if ops := surfui.Diff(old, next); len(ops) != 2 {
				t.Errorf("%s, %s: the deepest texts' diff is %d ops, not 2", c.file, m.name, len(ops))
			}
			if doc := surfui.Document(surfui.Clone(next)); !strings.Contains(doc, "b5!") || !strings.Contains(surfui.TextOf(next), "c15!") {
				t.Errorf("%s, %s: the deepest texts are not drawn", c.file, m.name)
			}
			runtime.Gosched()
		}
	}
}
