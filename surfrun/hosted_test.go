package surfrun_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// These run models on surftest.StartHosted's host, which is no surfrun.Runtime:
// it relays the documents to the test host itself through a surfrun.Surface,
// and reads the host's events and keys from its input, with no terminal.

// hosts are the two ways to run a model on a test host.
var hosts = []struct {
	name  string
	start func(testing.TB, surfrun.Model, surfrun.Options) *surftest.App
}{
	{"runtime", func(t testing.TB, m surfrun.Model, o surfrun.Options) *surftest.App { return surftest.Start(t, m, o) }},
	{"hosted", func(t testing.TB, m surfrun.Model, o surfrun.Options) *surftest.App {
		return surftest.StartHosted(t, m, o)
	}},
}

// Hosted, a field's Enter commits it once, and leaving it after commits
// nothing more.
func TestHostedFieldCommitsOnEnter(t *testing.T) {
	m := &committer{value: "80"}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	a.Wait("the field has the keyboard", func() bool { return a.Surface().Focused() == commitField })
	fill(t, a, "120")
	a.Host.Key("enter")
	commits(t, a, m, "120")
	a.Host.Key("tab")
	a.Wait("the button has the keyboard", func() bool { return a.Surface().Focused() == commitButton })
	a.Call(func() { m.ticks++ })
	a.Wait("the count", func() bool { return a.Surface().TextOf(commitCount) == "1" })
	commits(t, a, m, "120")
}

// Hosted, a Cycle's keys and clicks step it.
func TestHostedCycle(t *testing.T) {
	m := &cycler{value: "auto"}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	a.Wait("the Cycle has the keyboard", func() bool { return a.Surface().Focused() == cycleID })
	for _, step := range []struct{ key, want string }{
		{"arrow_right", "cells"}, {"arrow_left", "auto"}, {"end", "hotty"}, {"h", "hotty"}, {"c", "cells"},
	} {
		a.Host.Key(step.key)
		shows(t, a, m, step.want)
	}
	if err := a.Host.Click(a.Name, moreID); err != nil {
		t.Fatal(err)
	}
	shows(t, a, m, "hotty")
}

// Hosted, a click opens the Select's list in the host's own surface,
// where its click says the Select is: the list's tree, under its anchor.
// Its keys move in it, and a click on an option picks it.
func TestHostedSelectList(t *testing.T) {
	m := &chooser{value: "neuroplastio"}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	a.Wait("the Select has the keyboard", func() bool { return a.Surface().Focused() == selectID })
	area := map[string]any{"c": 10, "r": 3, "w": 30, "h": 2}
	if err := a.Host.Emit(a.Name, "click", selectID, map[string]any{"area": area}); err != nil {
		t.Fatal(err)
	}
	a.Wait("the list open", func() bool { return open(a) })
	if col, row := a.List().At(); col != 10 || row != 5 {
		t.Errorf("the list at %d,%d, want under its anchor, 10,5", col, row)
	}
	if p := a.List().Placement(); p.Z <= 0 || !a.List().Press() || p.Cols != 30 || p.Rows != len(themes)+1 {
		t.Errorf("the list's placement %+v, presses %v", p, a.List().Press())
	}
	for i, th := range themes {
		if got := a.List().TextOf(surfui.OptionID(i)); got != th {
			t.Errorf("option %d is %q, want %q", i, got, th)
		}
	}
	if v, _ := a.List().Attr(surfui.OptionID(3), "aria-selected"); v != "true" || hi(a) != 3 {
		t.Errorf("neuroplastio is not marked picked and the keys' (%q, %d)", v, hi(a))
	}
	a.Host.Key("arrow_down")
	a.Wait("nord, the keys'", func() bool { return hi(a) == 4 })
	if err := a.Host.Click(a.ListName, surfui.OptionID(1)); err != nil {
		t.Fatal(err)
	}
	changed(t, a, m, "dracula")
	a.Wait("the list closed", func() bool { return !open(a) })
	a.Wait("the Select has the keyboard again", func() bool { return a.Surface().Focused() == selectID })
	a.Pick(selectID, "nord")
	changed(t, a, m, "dracula", "nord")
}

// Hosted, a key goes up the focus chain to a binding; Escape blurs;
// a binding to Quit ends the run.
func TestHostedKeyClimbs(t *testing.T) {
	m := &binds{}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	const field = "r.kb.kl.0.kf"
	a.Wait("the field has the keyboard", func() bool { return a.Surface().Focused() == field })
	heard := func(n int) func() bool {
		return func() bool { var k int; a.Call(func() { k = len(m.heard) }); return k == n }
	}
	a.Host.Key("arrow_down")
	a.Wait("the list's ↓", heard(1))
	a.Host.Key("ctrl+k")
	a.Wait("the root's mod+k", heard(2))
	a.Host.Key("escape")
	a.Wait("nothing focused", func() bool { return a.Surface().Focused() == "" })
	a.Host.Key("z")
	a.Wait("the root's z, from the home", heard(3))
	a.Call(func() {
		if want := "list ↓,root mod+k,root z"; strings.Join(m.heard, ",") != want {
			t.Errorf("heard %q, want %s", m.heard, want)
		}
	})
}

// poster is a model that works off its goroutine: attached, it posts
// what it hears into its runner, and it stops when detached.
type poster struct {
	heard   string
	r       surfrun.Runner
	stop    context.CancelFunc
	stopped chan struct{}
}

func (m *poster) Render() *surfui.Node {
	return surfui.Surface(surfui.Text(m.heard).Key("t"), surfui.Button("Close").Key("c").OnClick(func() { m.r.Quit() }))
}

func (m *poster) Do(act surfui.Action, ev surfrun.Event) { act.(func())() }

func (m *poster) Attach(r surfrun.Runner) {
	m.r = r
	if r == nil {
		m.stop()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.stop = cancel
	go func() {
		defer close(m.stopped)
		for i := 1; ; i++ {
			n := strings.Repeat("!", i)
			r.Post(func() { m.heard = "news" + n })
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()
}

// A model attached to its runner, the runtime or a host of its own,
// posts into it from its goroutine, and its posts are drawn; a handler
// ends the run with Quit, and the model hears the end.
func TestAttachPostsIntoAnyHost(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			m := &poster{stopped: make(chan struct{})}
			a := h.start(t, m, surfrun.Options{})
			a.Wait("a post drawn", func() bool { return strings.HasPrefix(a.Surface().TextOf("r.kt"), "news!") })
			a.Wait("another", func() bool { return strings.HasPrefix(a.Surface().TextOf("r.kt"), "news!!!") })
			if err := a.Host.Click(a.Name, "r.kc"); err != nil {
				t.Fatal(err)
			}
			select {
			case <-m.stopped:
			case <-time.After(surftest.Patience):
				t.Fatal("the model did not hear the run end")
			}
		})
	}
}

// A host of its own tells a Sizer its surface's size before the first
// render, as the runtime does.
func TestHostedSizerHearsTheSize(t *testing.T) {
	m := &sized{}
	a := surftest.StartHosted(t, m, surfrun.Options{})
	var sizes [][2]int
	a.Call(func() { sizes = m.sizes })
	if len(sizes) != 1 || sizes[0] != [2]int{80, 24} {
		t.Errorf("sizes %v, want the host's 80×24 once", sizes)
	}
}

// The runtime is a Runner, as a host of its own is.
var _ surfrun.Runner = (*surfrun.Runtime)(nil)

// Both hosts send the same ops for the same acts: the document, a
// selection the arrows move (with a reveal where it scrolls), and a pick.
func TestHostsSendTheSameOps(t *testing.T) {
	ops := map[string][]hotty.Op{}
	for _, h := range hosts {
		m := &rows{sel: 30}
		a := h.start(t, m, surfrun.Options{Scroll: hotty.ScrollVertical})
		a.Wait("the list has the keyboard", func() bool { return a.Surface().Focused() == rowsList })
		a.Host.Key("arrow_down")
		a.Wait("row 31", func() bool { return a.Has(rowsList+".k31", "is-sel") })
		a.Host.Key("arrow_up")
		a.Host.Key("arrow_up")
		a.Wait("row 29", func() bool { return a.Has(rowsList+".k29", "is-sel") })
		a.Call(func() {})
		ops[h.name] = a.Ops()
	}
	r, o := ops["runtime"], ops["hosted"]
	if len(r) == 0 || len(r) != len(o) {
		t.Fatalf("the runtime sent %d ops, the hosted %d", len(r), len(o))
	}
	for i := range r {
		if r[i].Op != o[i].Op || r[i].Target != o[i].Target || r[i].Key != o[i].Key {
			t.Errorf("op %d: the runtime's %+v, the hosted %+v", i, r[i], o[i])
		}
	}
}
