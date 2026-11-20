package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// Identity across compiled parents: an instance that moves from one to
// another, and an instance key used twice where a copy is inside a kept
// parent. Each runs a compiled surface beside a plain one.

type idRow struct{ Key, Title string }

type idGroup struct {
	Key  string
	Rows [3]idRow
	Len  int
	// Plain renders the group and its rows as plain views: the full
	// render a compiled surface must match.
	Plain bool
}

var (
	idRowC   = surf.Compile(idRowView)
	idGroupC = surf.Compile(idGroupView)
)

func idRowView(p idRow) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<div class="r"%s>%s</div>`, onAttrs(ctx, "r-"+p.Title, surf.Click(func() {})), p.Title)
		return err
	})
}

func idRowOf(r idRow, plain bool) templ.Component {
	if plain {
		return idRowView(r)
	}
	return idRowC.With(r.Key, r)
}

func idGroupView(g idGroup) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<section><h3>%s</h3>`, g.Key)
		for _, r := range g.Rows[:g.Len] {
			if err := idRowOf(r, g.Plain).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</section>`)
		return err
	})
}

func idGroupOf(g idGroup, plain bool) templ.Component {
	if plain {
		g.Plain = true
		return idGroupView(g)
	}
	return idGroupC.With(g.Key, g)
}

// faults keeps what a surface faulted, where surftest would fail.
type faults struct {
	testing.TB
	got *[]string
}

func (c faults) Error(args ...any) { *c.got = append(*c.got, fmt.Sprint(args...)) }

// idPair is one view on a compiled surface and a plain one.
type idPair struct {
	t      *testing.T
	rc, rp *surftest.Rig
	got    []string
	// dups counts the duplicate faults said since a step without one;
	// surftest says each fault once a test.
	dups int
}

func startIDPair(t *testing.T, view func(plain bool) templ.Component) *idPair {
	p := &idPair{t: t}
	p.rc = surftest.Start(faults{t, &p.got}, surf.New("t", view(false), ""))
	var plainFaults []string
	p.rp = surftest.Start(faults{t, &plainFaults}, surf.New("t", view(true), ""))
	p.same("start")
	for _, f := range p.got {
		if strings.Contains(f, "is used twice in one render") {
			p.dups++
		}
	}
	return p
}

// update renders both again and fails where the compiled host's document
// isn't the plain one's, or the compiled surface faulted anything but a
// duplicate instance key, which dup says is in the view. surftest says a
// fault once a test, so the duplicate's is looked for once a run of
// steps that have one.
func (p *idPair) update(what string, dup bool) {
	p.t.Helper()
	p.got = p.got[:0]
	p.rc.Update()
	p.rp.Update()
	p.same(what)
	for _, f := range p.got {
		switch {
		case strings.Contains(f, "is used twice in one render"):
			if !dup {
				p.t.Errorf("%s: a duplicate fault with none: %s", what, f)
			}
			p.dups++
		case strings.Contains(f, "registered twice"):
			// Both copies of a duplicated row register their id; the
			// plain surface faults it too.
		default:
			p.t.Errorf("%s: a fault: %s", what, f)
		}
	}
	if !dup {
		p.dups = 0
	}
}

// dupSaid fails where no step since the last without a duplicate said
// the duplicate's fault.
func (p *idPair) dupSaid(what string) {
	p.t.Helper()
	if p.dups == 0 {
		p.t.Errorf("%s: no fault for the instance key used twice", what)
	}
}

func (p *idPair) same(what string) {
	p.t.Helper()
	if got, want := plain(p.rc.HTML()), plain(p.rp.HTML()); got != want {
		p.t.Fatalf("%s: the document isn't a full render's:\ngot  %s\nwant %s", what, got, want)
	}
}

// A row moves from one compiled group into another, earlier or later:
// both groups change, and the host gets the row where it now is.
func TestCompiledMoveBetweenParents(t *testing.T) {
	for _, dir := range []string{"g1→g0", "g0→g1"} {
		t.Run(dir, func(t *testing.T) {
			gs := [2]idGroup{
				{Key: "g0", Rows: [3]idRow{{"a", "A"}}, Len: 1},
				{Key: "g1", Rows: [3]idRow{{"b", "B"}, {"c", "C"}}, Len: 2},
			}
			p := startIDPair(t, func(plain bool) templ.Component {
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					io.WriteString(w, `<main>`)
					for _, g := range gs {
						if err := idGroupOf(g, plain).Render(ctx, w); err != nil {
							return err
						}
					}
					_, err := io.WriteString(w, `</main>`)
					return err
				})
			})
			if dir == "g1→g0" {
				gs[0].Rows[1], gs[0].Len = gs[1].Rows[1], 2
				gs[1].Rows[1], gs[1].Len = idRow{}, 1
			} else {
				gs[1].Rows[2], gs[1].Len = gs[0].Rows[0], 3
				gs[0].Rows[0], gs[0].Len = idRow{}, 0
			}
			p.update("move", false)
			p.update("again", false)
		})
	}
}

// dupView is rows at the top and a group, in either order.
func dupView(top *[]idRow, g *idGroup, groupFirst bool) func(plain bool) templ.Component {
	return func(plain bool) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			io.WriteString(w, `<main>`)
			rows := func() error {
				for _, r := range *top {
					if err := idRowOf(r, plain).Render(ctx, w); err != nil {
						return err
					}
				}
				return nil
			}
			if !groupFirst {
				if err := rows(); err != nil {
					return err
				}
			}
			if err := idGroupOf(*g, plain).Render(ctx, w); err != nil {
				return err
			}
			if groupFirst {
				if err := rows(); err != nil {
					return err
				}
			}
			_, err := io.WriteString(w, `</main>`)
			return err
		})
	}
}

// A row at the top takes the instance key of a kept group's row: the
// group renders, its copy faults as a duplicate, and both show what a
// full render shows, with the rows before the group and after it.
func TestCompiledDupWithAKeptParent(t *testing.T) {
	for _, order := range []string{"rows first", "group first"} {
		t.Run(order, func(t *testing.T) {
			g := idGroup{Key: "g", Rows: [3]idRow{{"a", "A"}}, Len: 1}
			top := []idRow{{"x", "X"}}
			p := startIDPair(t, dupView(&top, &g, order == "group first"))
			top[0] = idRow{"a", "A2"}
			p.update("the group's row's key at the top", true)
			p.update("again", true)
			p.dupSaid("the group's row's key at the top")
			top[0] = idRow{"x", "X"}
			p.update("unique again", false)
			p.update("again", false)
			top[0] = idRow{"a", "A3"}
			p.update("twice once more", true)
		})
	}
}

// An instance key used twice inside one compiled parent: the parent
// renders each time while the key comes twice, and no row goes missing
// when the view around it changes.
func TestCompiledDupInParent(t *testing.T) {
	g := idGroup{Key: "g", Rows: [3]idRow{{"a", "A"}, {"a", "B"}}, Len: 2}
	outer := 0
	p := startIDPair(t, func(plain bool) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintf(w, "<p>%d</p>", outer)
			return idGroupOf(g, plain).Render(ctx, w)
		})
	})
	p.update("the same", true)
	outer++
	p.update("the view around changes", true)
	p.dupSaid("twice in the group")
	g.Rows[1] = idRow{"b", "B"}
	p.update("unique", false)
	outer++
	p.update("the view around changes", false)
}

// A move between two parents inside one filled instance: the instance's
// own diff takes the element out of its old parent first.

type idBoard struct {
	Cols  [2][3]string
	Lens  [2]int
	Title string
}

var idBoardC = surf.Compile(idBoardView)

func idBoardView(b idBoard) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div class="board"><h2>%s</h2>`, b.Title)
		for c := range 2 {
			io.WriteString(w, `<div class="col">`)
			for _, card := range b.Cols[c][:b.Lens[c]] {
				fmt.Fprintf(w, `<div class="card"%s>%s</div>`, onAttrs(ctx, "card-"+card, surf.Click(func() {})), card)
			}
			io.WriteString(w, `</div>`)
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

// A card with its own id moves to an earlier column of one compiled
// board, which has no instances inside: the board is the one job.
func TestCompiledMoveInsideAnInstance(t *testing.T) {
	for _, retitle := range []bool{false, true} {
		t.Run(fmt.Sprintf("retitle=%v", retitle), func(t *testing.T) {
			b := idBoard{Cols: [2][3]string{{"a"}, {"b", "c"}}, Lens: [2]int{1, 2}, Title: "Board"}
			p := startIDPair(t, func(plain bool) templ.Component {
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					io.WriteString(w, `<main>`)
					c := idBoardC.With("b", b)
					if plain {
						c = idBoardView(b)
					}
					if err := c.Render(ctx, w); err != nil {
						return err
					}
					_, err := io.WriteString(w, `</main>`)
					return err
				})
			})
			b.Cols, b.Lens = [2][3]string{{"c", "a"}, {"b"}}, [2]int{2, 1}
			if retitle {
				b.Title = "Board 2"
			}
			p.update("the card moves", false)
			p.update("again", false)
		})
	}
}

type idSect struct {
	Name   string
	Groups [2]idGroup
	Plain  bool
}

var idSectC = surf.Compile(idSectView)

func idSectView(s idSect) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<article><h2>%s</h2>`, s.Name)
		for _, g := range s.Groups {
			if err := idGroupOf(g, s.Plain).Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</article>`)
		return err
	})
}

// A row moves between two compiled groups of one compiled section while
// the section's own markup changes: the section is the one job, and the
// groups are filled inside it.
func TestCompiledMoveInsideAFilledParent(t *testing.T) {
	for _, dir := range []string{"g1→g0", "g0→g1"} {
		t.Run(dir, func(t *testing.T) {
			s := idSect{Name: "S", Groups: [2]idGroup{
				{Key: "g0", Rows: [3]idRow{{"a", "A"}}, Len: 1},
				{Key: "g1", Rows: [3]idRow{{"b", "B"}, {"c", "C"}}, Len: 2},
			}}
			p := startIDPair(t, func(plain bool) templ.Component {
				return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					if plain {
						s := s
						s.Plain = true
						return idSectView(s).Render(ctx, w)
					}
					return idSectC.With("s", s).Render(ctx, w)
				})
			})
			gs := &s.Groups
			if dir == "g1→g0" {
				gs[0].Rows[1], gs[0].Len = gs[1].Rows[1], 2
				gs[1].Rows[1], gs[1].Len = idRow{}, 1
			} else {
				gs[1].Rows[2], gs[1].Len = gs[0].Rows[0], 3
				gs[0].Rows[0], gs[0].Len = idRow{}, 0
			}
			s.Name = "S renamed"
			p.update("the row moves", false)
			p.update("again", false)
		})
	}
}
