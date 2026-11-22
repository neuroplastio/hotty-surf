package surftui

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// TestDeltaOffersRetainWhatAParseDoes: a log that appends a row each
// step, beside a status of 60 spans written again (an inner), keeps no
// more with offers than a Screen that parses: a row's copy keeps its own
// fragment, not the step's status.
func TestDeltaOffersRetainWhatAParseDoes(t *testing.T) {
	steps := 300
	if testing.Short() {
		steps = 100
	}
	markup := func(i int) string {
		var b strings.Builder
		fmt.Fprintf(&b, `<div id="st">step %d`, i)
		for k := range 60 {
			fmt.Fprintf(&b, `<span class="text-muted-foreground font-bold">s%d-%d</span> `, k, i)
		}
		b.WriteString(`</div><div id="log">`)
		for r := 0; r <= i; r++ {
			fmt.Fprintf(&b, `<div id="r%d">row %d</div>`, r, r)
		}
		b.WriteString(`</div>`)
		return b.String()
	}
	heap := func() uint64 {
		var m runtime.MemStats
		runtime.GC()
		runtime.GC()
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	retained := func(offer bool) uint64 {
		cur := markup(0)
		before := heap()
		o := newOnScreen(t, "v", changing(&cur), "", 80, 20, offer)
		o.document(t)
		for i := 1; i < steps; i++ {
			cur = markup(i)
			if err := o.s.Update(); err != nil || o.stale {
				t.Fatal("the step was refused", err)
			}
		}
		after := heap()
		runtime.KeepAlive(o)
		return after - min(after, before)
	}
	parsed, offered := retained(false), retained(true)
	t.Logf("%d steps keep %d KB parsed, %d KB offered", steps, parsed/1024, offered/1024)
	if offered > parsed*3/2+64<<10 {
		t.Errorf("with offers, %d steps keep %d KB, where a Screen that parses keeps %d KB", steps, offered/1024, parsed/1024)
	}
}

type fillRow struct {
	ID, Title, Tag string
	N, Items       int
	Mixed, Table   bool
}

func fillRowView(p fillRow) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		var b strings.Builder
		if p.Table {
			fmt.Fprintf(&b, `<tr class="row"><td>%s</td><td>`, templ.EscapeString(p.Title))
		} else {
			fmt.Fprintf(&b, `<div class="flex gap-1"><span>%s</span>`, templ.EscapeString(p.Title))
		}
		if p.Mixed {
			fmt.Fprintf(&b, `<span>n=<b>%d</b> of <i>x</i></span>`, p.N)
		} else {
			fmt.Fprintf(&b, `<%s>%d</%s>`, p.Tag, p.N, p.Tag)
		}
		fmt.Fprintf(&b, `<span id="%s-l" class="flex">`, p.ID)
		for i := range p.Items {
			fmt.Fprintf(&b, `<em id="%s-%d">%d</em>`, p.ID, (i*7+p.N)%p.Items, i)
		}
		b.WriteString(`</span>`)
		if p.Table {
			b.WriteString(`</td></tr>`)
		} else {
			b.WriteString(`</div>`)
		}
		_, err := io.WriteString(w, b.String())
		return err
	})
}

var fillRowC = surf.Compile(fillRowView)

// TestDeltaOffersThroughFill: compiled rows, in a column and in a tbody,
// changed at random: their deltas come from fill, whose fragments are
// parsed in place (parseIn), and are offered, taken and held to the
// oracle and to a Screen that parses, node for node and cell for cell.
func TestDeltaOffersThroughFill(t *testing.T) {
	frags, taken := 0, 0
	for _, table := range []bool{false, true} {
		for run := range 20 {
			r := rand.New(rand.NewPCG(uint64(run), 77))
			rows := make([]fillRow, 6)
			for i := range rows {
				rows[i] = fillRow{ID: fmt.Sprintf("k%d", i), Title: fmt.Sprintf("t%d", i), Tag: "b", Items: 2, Table: table}
			}
			view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				open, end := `<div class="flex flex-col">`, `</div>`
				if table {
					open, end = `<table><tbody>`, `</tbody></table>`
				}
				io.WriteString(w, open)
				for _, p := range rows {
					if err := fillRowC.With(p.ID, p).Render(ctx, w); err != nil {
						return err
					}
				}
				_, err := io.WriteString(w, end)
				return err
			})
			got := newOnScreen(t, "v", view, "", 60, 12, true)
			h := got.checked(t)
			want := newOnScreen(t, "v", view, "", 60, 12, false)
			got.document(t)
			want.document(t)
			for step := range 30 {
				i := r.IntN(len(rows))
				switch r.IntN(5) {
				case 0:
					rows[i].Mixed = !rows[i].Mixed
				case 1:
					rows[i].Tag = []string{"b", "i", "u", "code"}[r.IntN(4)]
				case 2:
					rows[i].Items = 1 + r.IntN(5)
				case 3:
					rows[i].N = r.IntN(9)
				default:
					rows[i].Title = fmt.Sprintf("t%d<%d>", i, r.IntN(9))
				}
				for _, o := range []*onScreen{got, want} {
					if err := o.s.Update(); err != nil {
						t.Fatal(err)
					}
					if o.stale {
						t.Errorf("table=%v run %d step %d: a delta was refused", table, run, step)
						o.document(t)
					}
				}
				taken += h.taken()
				what := fmt.Sprintf("table=%v run %d step %d", table, run, step)
				if len(h.diverged) > 0 {
					t.Errorf("%s: a markup parses as another tree than its nodes: %s", what, h.diverged[0])
					break
				}
				if d := sameTrees(childrenOf(got.c.body), childrenOf(want.c.body), false); d != "" {
					t.Errorf("%s: the trees differ: %s", what, d)
					break
				}
				if !sameCells(t, what, got.c.Cells(), want.c.Cells()) {
					break
				}
			}
			frags += h.frags
			rt.HostOf = hostOf
		}
	}
	t.Logf("fill offered %d fragments; the Screen's trees took %d of their nodes", frags, taken)
	if frags == 0 || taken == 0 {
		t.Error("no fill's fragment was offered and taken")
	}
}
