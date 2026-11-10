package conformance

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
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
	at := func(id string, ty int) (x, y int) {
		var ok bool
		a.Call(func() {
			if r := a.Run.Renderer(); r != nil {
				x, y, ok = dragCell(r, id, ty)
			}
		})
		if !ok {
			t.Fatalf("%s is not on the screen at step %d:\n%s", id, ty, a.Cells().Text())
		}
		return x, y
	}
	sgr := func(id string, ty int, b int, final string) {
		x, y := at(id, ty)
		a.Host.Type(fmt.Sprintf("\x1b[<%d;%d;%d%s", b, x+1, y+1, final))
	}
	atX := func(id string, step int) (x, y int) {
		var ok bool
		a.Call(func() {
			if r := a.Run.Renderer(); r != nil {
				x, y, ok = r.WhereStepX(id, step)
			}
		})
		if !ok {
			t.Fatalf("%s is not on the screen at x step %d:\n%s", id, step, a.Cells().Text())
		}
		return x, y
	}
	sgrX := func(id string, step int, b int, final string) {
		x, y := atX(id, step)
		a.Host.Type(fmt.Sprintf("\x1b[<%d;%d;%d%s", b, x+1, y+1, final))
	}
	out := playCells(t, ar, a.Cells, logged, cellsUser{
		press:     func(id string) { sgr(id, -1, 0, "M") },
		over:      func(id string, ty int) { sgr(id, ty, 32, "M") },
		release:   func(id string, ty int) { sgr(id, ty, 0, "m") },
		pressAt:   func(id string, x int) { sgrX(id, x, 0, "M") },
		moveAt:    func(id string, x int) { sgrX(id, x, 32, "M") },
		releaseAt: func(id string, x int) { sgrX(id, x, 0, "m") },
		click:     func(id string) { a.Press(id) },
		choose:    func(id, value string) { a.Choose(id, value) },
		fill: func(id, text string) {
			a.Press(id)
			a.Host.Key("ctrl+a")
			if text == "" {
				a.Host.Key("backspace")
			} else {
				a.Paste(text)
			}
		},
		key: func(name string) {
			if name == "escape" {
				// Escape as CSI u (kitty's): a lone ESC waits for what
				// follows it, and Sync's F12 would follow at once.
				a.Host.Type("\x1b[27u")
				return
			}
			a.Host.Key(name)
		},
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
	at := func(id string, ty int) (x, y int) {
		x, y, ok := dragCell(r, id, ty)
		if !ok {
			t.Fatalf("%s is not on the screen at step %d:\n%s", id, ty, r.Screen().Text())
		}
		return x, y
	}
	atX := func(id string, step int) (x, y int) {
		x, y, ok := r.WhereStepX(id, step)
		if !ok {
			t.Fatalf("%s is not on the screen at x step %d:\n%s", id, step, r.Screen().Text())
		}
		return x, y
	}
	out := playCells(t, ar, r.Screen, func() []string { return slices.Clone(log()) }, cellsUser{
		pressAt:   func(id string, x int) { cx, cy := atX(id, x); r.Press(cx, cy, 0) },
		moveAt:    func(id string, x int) { cx, cy := atX(id, x); r.Move(cx, cy, 0) },
		releaseAt: func(id string, x int) { cx, cy := atX(id, x); r.Release(cx, cy, 0) },
		press: func(id string) {
			x, y := at(id, -1)
			r.Press(x, y, 0)
		},
		over: func(id string, ty int) {
			x, y := at(id, ty)
			r.Move(x, y, 0)
		},
		release: func(id string, ty int) {
			x, y := at(id, ty)
			r.Release(x, y, 0)
		},
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
// until it is done and drawn. A drag is a press, the pointer's moves
// over and its release, each on an element, at a step of a place to drop
// on (-1 for none).
type cellsUser struct {
	click, key    func(string)
	choose, fill  func(id, text string)
	press         func(id string)
	over, release func(id string, ty int)
	// pressAt, moveAt and releaseAt are a drag of an element with steps of
	// its own (a slider's), the pointer at step x of it.
	pressAt, moveAt, releaseAt func(id string, x int)
	sync                       func()
}

// dragCell is where a drag's step puts the pointer in the cells: a cell of
// the element id at step ty of its data-steps, or with ty -1 a cell where
// a press on it reaches it.
func dragCell(r *cells.Renderer, id string, ty int) (x, y int, ok bool) {
	if ty < 0 {
		return r.Where(id)
	}
	return r.WhereStep(id, ty)
}

// playCells plays the case's steps as u, and returns the files.
func playCells(t testing.TB, ar *txtar.Archive, screen func() cells.Screen, log func() []string, u cellsUser) []txtar.File {
	t.Helper()
	out := []txtar.File{{Name: "cells", Data: []byte(screen().Golden())}}
	nlog := len(log())
	for i, ev := range caseEvents(ar) {
		n := strconv.Itoa(i + 1)
		for _, cmd := range strings.Split(ev, ";") {
			f := strings.Fields(cmd)
			ty := -1
			if len(f) == 3 && (f[0] == "over" || f[0] == "release") {
				v, err := strconv.Atoi(f[2])
				if err != nil {
					t.Fatalf("%s.event %q: %v", n, ev, err)
				}
				ty = v
			}
			switch {
			case len(f) == 2 && f[0] == "click":
				u.click(f[1])
			case len(f) == 2 && f[0] == "doubleclick": // two clicks within the double click's time
				u.click(f[1])
				u.sync() // the first click's update is drawn: the element is where it is now
				u.click(f[1])
			case len(f) == 3 && f[0] == "choose":
				u.choose(f[1], f[2])
			case len(f) >= 2 && f[0] == "fill":
				u.fill(f[1], strings.Join(f[2:], " "))
			case len(f) == 2 && f[0] == "key":
				u.key(f[1])
			case len(f) == 2 && f[0] == "press":
				u.press(f[1])
			case (len(f) == 2 || len(f) == 3) && f[0] == "over":
				u.over(f[1], ty)
			case (len(f) == 2 || len(f) == 3) && f[0] == "release":
				u.release(f[1], ty)
			case len(f) == 3 && (f[0] == "pressat" || f[0] == "moveat" || f[0] == "releaseat"):
				x, err := strconv.Atoi(f[2])
				if err != nil {
					t.Fatalf("%s.event %q: %v", n, ev, err)
				}
				switch f[0] {
				case "pressat":
					u.pressAt(f[1], x)
				case "moveat":
					u.moveAt(f[1], x)
				default:
					u.releaseAt(f[1], x)
				}
			default:
				t.Fatalf("%s.event: %q is no event", n, ev)
			}
			u.sync()
		}
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
