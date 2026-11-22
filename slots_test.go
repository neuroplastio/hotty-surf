package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/surftest"
)

// A frame keeps its slots (rt.Slots): one made before a compaction reads
// its own handlers after it. A Drag handler that renders (Update) on its
// End, and the render compacts, before the event looks up the Drop in the
// frame it began with.
func TestSlotsCompactInAHandler(t *testing.T) {
	var l kLog
	n := 80
	var s *surf.Surface
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<main>`)
		for i := range n {
			id := fmt.Sprintf("c%d", i)
			fmt.Fprintf(w, `<div%s>%s</div>`, onAttrs(ctx, id, surf.Drop(func(surf.DragEvent) { l.add("drop on " + id) })), id)
		}
		fmt.Fprintf(w, `<div%s>src</div>`, onAttrs(ctx, "src", surf.Drag(func(d surf.DragEvent) {
			if d.End {
				n = 2
				if err := s.Update(); err != nil {
					t.Fatal(err)
				}
			}
		})))
		fmt.Fprintf(w, `<div%s>dst</div>`, onAttrs(ctx, "dst", surf.Drop(func(surf.DragEvent) { l.add("drop on dst") })))
		_, err := io.WriteString(w, `</main>`)
		return err
	})
	s = surf.New("t", view, "")
	r := surftest.Start(t, s)
	r.Drag("#src", "#dst")
	if e := rt.Peek.Frame(s).Slots().Epoch; e == 0 {
		t.Fatal("the render in the handler didn't compact the slots")
	}
	if len(l.lines) != 1 || l.lines[0] != "drop on dst" {
		t.Errorf("the drop went to %q", l.lines)
	}
}

// pkProps is a kept instance's: two buttons, one with a key.
type pkProps struct {
	L *kLog
}

func pkView(p pkProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		l := p.L
		_, err := fmt.Fprintf(w, `<div><button%s>keep</button><button%s>keep2</button></div>`,
			onAttrs(ctx, "k", surf.Click(func() { l.add("k") }), surf.Key(func(string) bool { l.add("key k"); return true }, "x")),
			onAttrs(ctx, "k2", surf.Click(func() { l.add("k2") })))
		return err
	})
}

var pkC = surf.Compile(pkView)

// A kept instance among ids that come and go, compiled against plain:
// the slots compact several times, and the kept instance's handlers, the
// new ids', the document and the frame are a full render's after each.
func TestSlotsCompactKept(t *testing.T) {
	type side struct {
		l      kLog
		r      *surftest.Rig
		s      *surf.Surface
		faults []string
		gen, n int
	}
	mk := func(plain bool) *side {
		x := &side{n: 40}
		view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			io.WriteString(w, `<main>`)
			for i := range x.n {
				id := fmt.Sprintf("c%d-%d", x.gen, i)
				fmt.Fprintf(w, `<button%s>%s</button>`, onAttrs(ctx, id, surf.Click(func() { x.l.add(id) })), id)
			}
			p := pkProps{L: &x.l}
			var err error
			if plain {
				err = pkView(p).Render(ctx, w)
			} else {
				err = pkC.With("k", p).Render(ctx, w)
			}
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, `</main>`)
			return err
		})
		x.s = surf.New("t", view, "")
		x.r = surftest.Start(faults{t, &x.faults}, x.s)
		return x
	}
	c, p := mk(false), mk(true)
	var last uint64
	compactions := 0
	for i := range 12 {
		for _, x := range []*side{c, p} {
			x.gen++
			if i%3 == 2 {
				x.n = 2
			} else {
				x.n = 40
			}
			x.r.Update()
			x.r.Click("#k")
			x.r.Click("#k2")
			x.r.Click(fmt.Sprintf("#c%d-1", x.gen))
			x.r.Key("x")
		}
		if e := rt.Peek.Frame(c.s).Slots().Epoch; e != last {
			last = e
			compactions++
			// The render right after a compaction, with nothing changed:
			// the same document, and the kept instance's handlers.
			for _, x := range []*side{c, p} {
				x.r.Update()
				x.r.Click("#k")
				x.r.Key("x")
			}
		}
		if got, want := strings.Join(c.l.lines, ","), strings.Join(p.l.lines, ","); got != want {
			t.Fatalf("step %d: compiled heard\n%s\nplain\n%s", i, got, want)
		}
		if got, want := plain(c.r.HTML()), plain(p.r.HTML()); got != want {
			t.Fatalf("step %d: documents differ:\n%s\n%s", i, got, want)
		}
		if got, want := frameOf(c.s), frameOf(p.s); got != want {
			t.Fatalf("step %d: frames differ\n%s\n%s", i, got, want)
		}
	}
	if compactions < 2 {
		t.Fatalf("%d compactions", compactions)
	}
	if len(c.faults)+len(p.faults) > 0 {
		t.Fatalf("faults: compiled %q, plain %q", c.faults, p.faults)
	}
}

// A surface attached to a new host after its slots compacted: the kept
// instance's handlers and a new id's are its own.
func TestSlotsCompactReattach(t *testing.T) {
	var l kLog
	gen, n := 0, 40
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<main>`)
		for i := range n {
			id := fmt.Sprintf("c%d-%d", gen, i)
			fmt.Fprintf(w, `<button%s>%s</button>`, onAttrs(ctx, id, surf.Click(func() { l.add(id) })), id)
		}
		if err := pkC.With("k", pkProps{L: &l}).Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
	s := surf.New("t", view, "")
	r := surftest.Start(t, s)
	for i := range 6 {
		gen++
		r.Update()
		if i == 3 {
			if rt.Peek.Frame(s).Slots().Epoch == 0 {
				t.Fatal("no compaction before the new host")
			}
			r = surftest.Start(t, s)
		}
	}
	r.Update()
	r.Click("#k")
	r.Key("x")
	r.Click(fmt.Sprintf("#c%d-7", gen))
	if got, want := strings.Join(l.lines, ","), fmt.Sprintf("k,key k,c%d-7", gen); got != want {
		t.Errorf("heard %s, want %s", got, want)
	}
}
