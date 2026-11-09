package screens

import (
	"context"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// The render cost of each screen, both ways: interpreted and compiled
// (make bench; not in check, which doesn't time the machine). Each way
// starts from the same state: the screen opened on demodata's mock, on a
// host of the size its tests use, and the same acts played through the
// host. The run then ends, and the bench has the model to itself.
// vault/knowledge/measures.md has the numbers.

// benchWays are the ways a screen is drawn: the modes.
func benchWays(*testing.B) []struct {
	name string
	m    Mode
} {
	return []struct {
		name string
		m    Mode
	}{{string(Interpreted), Interpreted}, {string(Compiled), Compiled}}
}

// benchRun is a screen running on a host, while the bench brings it to a
// state.
type benchRun struct {
	h    *hottytest.Host
	name string
	r    *surfrun.Runtime
	sc   Screen
}

// A benchAct is what the host or the source does to a running screen.
type benchAct func(o *benchRun) error

func benchKey(k string) benchAct { return func(o *benchRun) error { o.h.Key(k); return nil } }
func benchClick(id string) benchAct {
	return func(o *benchRun) error { return o.h.Click(o.name, id) }
}
func benchTick(o *benchRun) error { o.r.Post(o.sc.Tick.(func())); return nil }

// benchScreens are the screens, each with its host's size, the acts that
// bring it to where its one change starts, and the change: what the
// screen's own test calls one change (F10's "a few ops" tests). The
// agents' change is the source's: the mock reports pi-1 working before
// the screen opens, as TestAgentsChangeIsAFewOps answers it.
var benchScreens = []struct {
	name       string
	cols, rows int
	before     []benchAct
	change     benchAct
	report     [2]string
}{
	{name: "notices", cols: 100, rows: 30, change: benchTick},
	{name: "settings", cols: 160, rows: 43, before: []benchAct{benchClick(setNavItem("tools"))},
		change: benchClick(setRow("tools", 0, "dock.controls", "label", "control"))},
	{name: "palette", cols: 64, rows: 44, change: benchKey("arrow_down")},
	{name: "agents", cols: 29, rows: 43, report: [2]string{"pi-1", "working"}},
}

// benchAt is the screen name, drawn in mode m, in a state: opened on
// the mock, which first reports report's status where it names an
// agent, on a host cols by rows, with acts played through the host, each
// waited for. The run has ended when it returns: no goroutine but the
// bench's touches the model.
func benchAt(b *testing.B, m Mode, name string, cols, rows int, report [2]string, acts ...benchAct) surfrun.Model {
	b.Helper()
	src := demodata.Mock()
	if report[0] != "" {
		if err := src.Report(report[0], report[1]); err != nil {
			b.Fatal(err)
		}
	}
	sc, ok := On(name, src, m)
	if !ok {
		b.Fatalf("no screen %s", name)
	}
	h := hottytest.New(b, hottytest.Size(cols, rows))
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	opts := surfrun.Options{Name: "p"}
	if sc.Scroll {
		opts.Scroll = hotty.Axes(1)
	}
	o := &benchRun{h: h, r: surfrun.New(term, sc.Model, opts), sc: sc}
	o.name = term.Surface("p")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- o.r.Loop(ctx) }()
	stop := func() {
		cancel()
		_ = h.Close()
		<-done
	}
	html := func() string {
		if s := h.Surface(o.name); s != nil {
			return s.HTML()
		}
		return ""
	}
	// until waits for ok, then for the document to stand for 20ms.
	until := func(what string, ok func() bool) {
		b.Helper()
		for end := time.Now().Add(surftest.Patience); !ok(); time.Sleep(2 * time.Millisecond) {
			if time.Now().After(end) {
				stop()
				b.Fatalf("%s: waited %v for %s", name, surftest.Patience, what)
			}
		}
		for d := html(); ; d = html() {
			time.Sleep(20 * time.Millisecond)
			if html() == d {
				return
			}
		}
	}
	until("the keyboard in its place", func() bool {
		s := h.Surface(o.name)
		return s != nil && s.Placed() && s.Focused() != ""
	})
	for i, act := range acts {
		was := html()
		if err := act(o); err != nil {
			stop()
			b.Fatal(err)
		}
		until("act "+string(rune('1'+i))+"'s change", func() bool { return html() != was })
	}
	stop()
	return sc.Model
}

// BenchmarkTree is the tree alone, of the screen as it opens: the part
// of a render that differs by the way it is drawn. The ids, the HTML and
// the diff are surfui's, on trees that are the same but for the order of
// their classes and attributes.
func BenchmarkTree(b *testing.B) {
	for _, s := range benchScreens {
		for _, w := range benchWays(b) {
			b.Run(s.name+"/"+w.name, func(b *testing.B) {
				m := benchAt(b, w.m, s.name, s.cols, s.rows, [2]string{})
				b.ReportAllocs()
				for b.Loop() {
					m.Render()
				}
			})
		}
	}
}

// BenchmarkFirst is a screen's first render: its tree, its ids and the
// document's HTML, which the runtime sends as the screen opens.
func BenchmarkFirst(b *testing.B) {
	for _, s := range benchScreens {
		for _, w := range benchWays(b) {
			b.Run(s.name+"/"+w.name, func(b *testing.B) {
				m := benchAt(b, w.m, s.name, s.cols, s.rows, [2]string{})
				var n int
				b.ReportAllocs()
				for b.Loop() {
					t := m.Render()
					surfui.Resolve(t, "r")
					n = len(surfui.Document(t))
				}
				b.ReportMetric(float64(n), "doc-bytes")
			})
		}
	}
}

// BenchmarkUpdate is an update after the screen's one change: the tree
// rendered again, its ids, and the diff into ops, as the runtime does
// after a handler. Two copies of the screen stand on either side of the
// change, and the bench renders each in turn and diffs it against the
// other's last tree: an update a loop, the change and back, with no
// handler timed.
func BenchmarkUpdate(b *testing.B) {
	for _, s := range benchScreens {
		for _, w := range benchWays(b) {
			b.Run(s.name+"/"+w.name, func(b *testing.B) {
				after := s.before
				if s.change != nil {
					after = append(append([]benchAct(nil), s.before...), s.change)
				}
				ms := [2]surfrun.Model{
					benchAt(b, w.m, s.name, s.cols, s.rows, [2]string{}, s.before...),
					benchAt(b, w.m, s.name, s.cols, s.rows, s.report, after...),
				}
				prev := ms[0].Render()
				surfui.Resolve(prev, "r")
				ops, i := 0, 0
				b.ReportAllocs()
				for b.Loop() {
					i++
					t := ms[i%2].Render()
					surfui.Resolve(t, "r")
					ops += len(surfui.Diff(prev, t))
					prev = t
				}
				if ops == 0 {
					b.Fatal("the change diffs to no ops")
				}
				b.ReportMetric(float64(ops)/float64(i), "ops/op")
			})
		}
	}
}
