package conformance

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-go/hottytest"
	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
	"github.com/neuroplastio/hotty-surf/surfui"
	"golang.org/x/tools/txtar"
)

// The cells a case is drawn in, at this size (docs/spec.md, "Cells").
const (
	CellsCols = 64
	CellsRows = 32
)

// PlayCells plays a case's steps in the cells, where the terminal is no
// hotty host and the runtime draws the root's tree (surftest.StartCells):
// a click is a press and its release on the element's cells, a choose a
// click on the Select and one on its option, a fill a click on the field,
// ctrl+a and a paste of the text (Backspace for none), a key a key. The
// files are the cells drawn at the start (cells) and after each step
// (N.cells), each step's N.event, and its N.log, which is the hosts':
// the cells lead to the same calls of the model's. Golden says how the
// cells read.
func PlayCells(t testing.TB, ar *txtar.Archive, c surflive.Component, log func() []string) ([]txtar.File, []string) {
	t.Helper()
	var heard []string
	m := surflive.New(hear(c, &heard))
	opts := surfrun.Options{Name: "case"}
	if _, ok := File(ar, "scroll"); ok {
		opts.Scroll = hotty.ScrollVertical
	}
	a := surftest.StartCells(t, m, opts, hottytest.Size(CellsCols, CellsRows))
	a.Sync()
	logged := func() (out []string) {
		a.Call(func() { out = slices.Clone(log()) })
		return out
	}
	out := playCells(t, ar, a.Cells, logged, cellsUser{
		click:  func(id string) { a.Press(id) },
		choose: func(id, value string) { a.Choose(id, value) },
		fill: func(id, text string) {
			a.Press(id)
			a.Host.Key("ctrl+a")
			if text == "" {
				a.Host.Key("backspace")
			} else {
				a.Paste(text)
			}
		},
		key:  func(name string) { a.Host.Key(name) },
		sync: a.Sync,
	})
	var errs []string
	a.Call(func() { errs = append(errs, heard...) })
	return out, errs
}

// PlayCellsHosted is PlayCells with no surfrun.Runtime and no terminal: a
// cells.Renderer hosts the root's surfrun.Surface, as a host that paints its
// own cells does, and every file comes out as PlayCells'.
func PlayCellsHosted(t testing.TB, ar *txtar.Archive, c surflive.Component, log func() []string) ([]txtar.File, []string) {
	t.Helper()
	var heard []string
	return PlayCellsModel(t, ar, surflive.New(hear(c, &heard)), log, func() []string { return heard })
}

// PlayCellsModel is PlayCellsHosted for a model that is no
// surflive.Component, as PlayModel is PlayHosted's (surflua's View): heard is
// the render errors m reported so far, each once.
func PlayCellsModel(t testing.TB, ar *txtar.Archive, m surfrun.Model, log, heard func() []string) ([]txtar.File, []string) {
	t.Helper()
	rn := &posts{}
	s := surfrun.NewSurface(m, rn)
	rn.s = s
	defer s.Close()
	r := cells.New(s.Cells())
	if _, ok := File(ar, "scroll"); ok {
		s.Scroll, r.Scroll = hotty.ScrollVertical, hotty.ScrollVertical
	}
	if sz, ok := m.(surfrun.Sizer); ok {
		sz.Resize(CellsCols, CellsRows)
	}
	r.Show()
	sync := func() {
		rn.run(r)
		r.Draw(CellsCols, CellsRows)
	}
	sync()
	press := func(id string) {
		x, y, ok := r.Where(id)
		if !ok {
			t.Fatalf("%s is not on the screen:\n%s", id, r.Screen().Text())
		}
		r.Press(x, y, 0)
		r.Release(x, y, 0)
	}
	out := playCells(t, ar, r.Screen, func() []string { return slices.Clone(log()) }, cellsUser{
		click: press,
		choose: func(id, value string) {
			press(id)
			sync()
			_, options, _ := surfui.SelectOf(r.Index()[id])
			press(surfui.OptionID(slices.Index(options, value)))
		},
		fill: func(id, text string) {
			press(id)
			r.Key("ctrl+a")
			if text == "" {
				r.Key("backspace")
			} else {
				r.Paste(text)
			}
		},
		key:  func(name string) { r.Key(name) },
		sync: sync,
	})
	return out, slices.Clone(heard())
}

// posts runs what a model posts, between the steps.
type posts struct {
	fs []func()
	s  *surfrun.Surface
}

func (p *posts) Post(f func())    { p.fs = append(p.fs, f) }
func (p *posts) Quit()            {}
func (p *posts) Focus(key string) { p.s.Focus(key) }

func (p *posts) run(r *cells.Renderer) {
	for len(p.fs) > 0 {
		f := p.fs[0]
		p.fs = p.fs[1:]
		f()
		r.Update()
	}
}

// cellsUser is what the user does in the cells, and sync, which waits
// until it is done and drawn.
type cellsUser struct {
	click, key   func(string)
	choose, fill func(id, text string)
	sync         func()
}

// playCells plays the case's steps as u, and returns the files.
func playCells(t testing.TB, ar *txtar.Archive, screen func() cells.Screen, log func() []string, u cellsUser) []txtar.File {
	t.Helper()
	out := []txtar.File{{Name: "cells", Data: []byte(screen().Golden())}}
	nlog := len(log())
	for i, ev := range caseEvents(ar) {
		n := strconv.Itoa(i + 1)
		f := strings.Fields(ev)
		switch {
		case len(f) == 2 && f[0] == "click":
			u.click(f[1])
		case len(f) == 3 && f[0] == "choose":
			u.choose(f[1], f[2])
		case len(f) >= 2 && f[0] == "fill":
			u.fill(f[1], strings.Join(f[2:], " "))
		case len(f) == 2 && f[0] == "key":
			u.key(f[1])
		default:
			t.Fatalf("%s.event: %q is no event", n, ev)
		}
		u.sync()
		l := log()
		logged := l[min(nlog, len(l)):]
		nlog = len(l)
		out = append(out,
			txtar.File{Name: n + ".event", Data: []byte(ev + "\n")},
			txtar.File{Name: n + ".log", Data: []byte(strings.Join(logged, "\n") + "\n")},
			txtar.File{Name: n + ".cells", Data: []byte(screen().Golden())},
		)
	}
	return out
}

// caseEvents are the case's events, in order.
func caseEvents(ar *txtar.Archive) []string {
	var out []string
	for _, f := range ar.Files {
		if strings.HasSuffix(f.Name, ".event") {
			out = append(out, strings.TrimSpace(string(f.Data)))
		}
	}
	return out
}
