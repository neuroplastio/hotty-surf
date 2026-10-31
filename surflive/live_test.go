package surflive_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/hotty-go/hottyterm"
	"github.com/neuroplastio/hotty-go/hottytest"

	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

type agent struct {
	Name, Status string
}

// rows is a list whose rows each hold a button, and which keeps its
// selection: the shape of S3.
type rows struct {
	agents    []agent
	sel       int
	restarted []string
}

func (l *rows) Render(c *surflive.Ctx) *surfui.Node {
	list := surfui.List().Focusable().Autofocus().
		Bind("arrow_down", "Next", func() { l.sel = min(l.sel+1, len(l.agents)-1) }).
		Bind("arrow_up", "Previous", func() { l.sel = max(l.sel-1, 0) })
	for i, a := range l.agents {
		list.Add(surfui.Item(
			surfui.Text(a.Name).Bold().Grow(),
			surfui.Text(a.Status).Muted(),
			surfui.Button("Restart").OnClick(func() { l.restarted = append(l.restarted, a.Name) }),
		).Key(a.Name).Selected(i == l.sel).OnClick(func() { l.sel = i }))
	}
	return surfui.Surface(list)
}

// r is the run start made last: a test reads the model's state on its
// goroutine, with r.Call.
var r *surfrun.Runtime

func start(t *testing.T, m surfrun.Model) (*hottytest.Host, string, func()) {
	t.Helper()
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	r = surfrun.New(term, m, surfrun.Options{Name: "app"})
	go func() { done <- r.Loop(ctx) }()
	name := term.Surface("app")
	wait(t, func() bool { s := h.Surface(name); return s != nil && s.Placed() })
	return h, name, func() {
		cancel()
		_ = h.Close()
		<-done
	}
}

// wait waits until ok holds, calling it on the run's goroutine.
func wait(t *testing.T, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(30 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		var held bool
		r.Call(func() { held = ok() })
		if held {
			return
		}
	}
	t.Fatal("timed out")
}

func TestRowButtonActsOnItsRow(t *testing.T) {
	l := &rows{agents: []agent{{"alpha", "idle"}, {"beta", "working"}, {"gamma", "done"}}}
	h, name, stop := start(t, surflive.New(l))
	defer stop()
	s := h.Surface(name)
	if got := s.Text(); !strings.Contains(got, "beta") {
		t.Fatalf("text %q", got)
	}
	if err := h.Click(name, "r.0.kbeta.2"); err != nil {
		t.Fatal(err)
	}
	wait(t, func() bool { return len(l.restarted) == 1 && l.restarted[0] == "beta" && l.sel == 0 })
	if err := h.Click(name, "r.0.kgamma"); err != nil {
		t.Fatal(err)
	}
	wait(t, func() bool {
		c, _ := s.Attr("r.0.kgamma", "class")
		return strings.Contains(c, "is-sel")
	})
	if c, _ := s.Attr("r.0.kalpha", "class"); strings.Contains(c, "is-sel") {
		t.Fatalf("alpha still selected: %q", c)
	}
}

func TestKeysMoveTheSelection(t *testing.T) {
	l := &rows{agents: []agent{{"alpha", "idle"}, {"beta", "working"}}}
	h, name, stop := start(t, surflive.New(l))
	defer stop()
	h.Key("arrow_down")
	wait(t, func() bool { return l.sel == 1 })
	wait(t, func() bool {
		c, _ := h.Surface(name).Attr("r.0.kbeta", "class")
		return strings.Contains(c, "is-sel")
	})
}

// A status that changes is a text op on its span, not a document.
func TestChangeIsADelta(t *testing.T) {
	l := &rows{agents: []agent{{"alpha", "idle"}, {"beta", "working"}}}
	old := l.Render(nil)
	surfui.Resolve(old, "r")
	l.agents[1].Status = "done"
	next := l.Render(nil)
	surfui.Resolve(next, "r")
	ops := surfui.Diff(old, next)
	if len(ops) != 1 || ops[0].Op != "text" || ops[0].Target != "r.0.kbeta.1" || *ops[0].Value != "done" {
		t.Fatalf("ops %+v", ops)
	}
	// A new row goes after the one before it.
	l.agents = append(l.agents[:1], agent{"new", "idle"}, l.agents[1])
	next2 := l.Render(nil)
	surfui.Resolve(next2, "r")
	ops = surfui.Diff(next, next2)
	if len(ops) != 1 || ops[0].Op != "after" || ops[0].Target != "r.0.kalpha" {
		t.Fatalf("ops %+v", ops)
	}
}

// timed is a button that says "on" for a moment after a click.
type timed struct{ on bool }

func (m *timed) Render(c *surflive.Ctx) *surfui.Node {
	label := "off"
	if m.on {
		label = "on"
	}
	return surfui.Surface(surfui.Button(label).OnClick(func() {
		m.on = true
		c.After(20*time.Millisecond, func() { m.on = false })
	}))
}

// After runs its function on the run's goroutine, and the app draws
// again: a state that ends by itself.
func TestAfter(t *testing.T) {
	h, name, stop := start(t, surflive.New(&timed{}))
	defer stop()
	if err := h.Click(name, "r.0"); err != nil {
		t.Fatal(err)
	}
	wait(t, func() bool { return h.Surface(name).TextOf("r.0") == "on" })
	wait(t, func() bool { return h.Surface(name).TextOf("r.0") == "off" })
}

// watched is a root that starts a watch of its own when the run starts:
// its goroutine posts what it hears, and it stops when the run ends.
type watched struct {
	heard   string
	stopped chan struct{}
	stop    func()
	rt      surfrun.Runner
}

func (m *watched) Attach(r surfrun.Runner) {
	m.rt = r
	if r == nil {
		m.stop()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.stop = cancel
	go func() {
		r.Post(func() { m.heard = "news" })
		<-ctx.Done()
		close(m.stopped)
	}()
}

func (m *watched) Render(c *surflive.Ctx) *surfui.Node {
	return surfui.Surface(surfui.Text(m.heard), surfui.Button("Close").OnClick(func() { m.rt.Quit() }))
}

// A root that is a surfrun.Attacher learns the runtime: what its goroutine
// posts is drawn, a handler can end the run, and it hears the end.
func TestRootAttaches(t *testing.T) {
	m := &watched{stopped: make(chan struct{})}
	h, name, stop := start(t, surflive.New(m))
	defer stop()
	wait(t, func() bool { return h.Surface(name).TextOf("r.0") == "news" })
	if err := h.Click(name, "r.1"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-m.stopped:
	case <-time.After(30 * time.Second):
		t.Fatal("the root did not hear the run end")
	}
}
