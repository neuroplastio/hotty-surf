package surftea_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftea"
	"neuroplast.io/hotty-surf/surfui"
)

// panel is a model with a count, a button that adds one, a Select, and a
// row with a menu: what a surface in a program shows.
type panel struct {
	rn    surfrun.Runner
	auto  string // the key of the element with the keyboard to begin with
	value string
	count int
	picks []string
}

func (m *panel) Attach(r surfrun.Runner) { m.rn = r }

const (
	countID  = "r.kn"
	addID    = "r.kadd"
	selectID = "r.ksel"
	rowID    = "r.krow"
)

func (m *panel) Render() *surfui.Node {
	add := surfui.Button("Add").Key("add").OnClick(func(surfrun.Event) { m.count++ })
	sel := surfui.Select(m.value, "red", "green", "blue").Key("sel").OnChange(func(e surfrun.Event) { m.value = e.Value })
	if m.auto == "add" {
		add.Autofocus()
	} else {
		sel.Autofocus()
	}
	row := surfui.Row(surfui.Text("menu")).Key("row").Menu(
		surfui.MenuItem{ID: "open", Label: "Open", Act: func(e surfrun.Event) { m.picks = append(m.picks, e.Value) }},
		surfui.MenuItem{ID: "drop", Label: "Drop", Act: func(e surfrun.Event) { m.picks = append(m.picks, e.Value) }})
	return surfui.Surface(surfui.Text(fmt.Sprintf("%s %d", m.value, m.count)).Key("n"), add, sel, row).Density(surfui.Compact).
		Bind("x", "Add", func(surfrun.Event) { m.count++ })
}

func (m *panel) Do(act surfui.Action, ev surfrun.Event) { act.(func(surfrun.Event))(ev) }

// probe runs f on the program's goroutine.
type probe struct {
	f    func()
	done chan struct{}
}

// app is a Bubble Tea program with two surfaces: left, above, and right.
type app struct {
	s           *hottytea.Session
	left, right *surftea.Host
	lm, rm      *panel
	keys        []string // the keys neither host took
	w, h        int
	frame       string
}

func (a *app) Init() tea.Cmd { return a.s.Detect() }

func (a *app) hosts() []*surftea.Host { return []*surftea.Host{a.left, a.right} }

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	msg, cmd := a.s.Update(msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
	case hottytea.ReadyMsg:
		for _, h := range a.hosts() {
			h.Ready(msg.Caps)
		}
	case hottytea.RelayoutMsg:
		surftea.Relayout(a.s, msg)
	case hottytea.EventMsg:
		for _, h := range a.hosts() {
			if h.Event(a.s, msg.Event) {
				break
			}
		}
	case surftea.PostMsg:
		msg.Host.Posted(a.s, msg)
	case probe:
		msg.f()
		close(msg.done)
	case tea.KeyPressMsg:
		name := hottytea.KeyName(msg.Key())
		if name == "q" {
			return a, tea.Sequence(a.s.Close(), tea.Quit)
		}
		took := false
		if a.s.Mode == hottytea.Native {
			took = a.left.Key(a.s, name)
		} else {
			took = a.left.CellsKey(name)
		}
		if !took {
			a.keys = append(a.keys, name)
		}
	}
	return a, tea.Batch(cmd, a.draw())
}

var (
	leftRect  = hottytea.Rect{X: 2, Y: 1, W: 30, H: 6}
	rightRect = hottytea.Rect{X: 2, Y: 8, W: 30, H: 4}
	screen    = hottytea.Rect{W: 40, H: 14}
)

func (a *app) draw() tea.Cmd {
	if a.s.Mode != hottytea.Native {
		a.frame = strings.Join(a.left.Lines(leftRect.W, leftRect.H), "\n")
		return nil
	}
	a.frame = "two surfaces"
	want := a.left.Surfaces(a.s, leftRect, screen)
	want = append(want, a.right.Surfaces(a.s, rightRect, screen)...)
	a.s.Layout(want)
	for _, h := range a.hosts() {
		h.AfterLayout(a.s)
	}
	return a.s.Flush()
}

func (a *app) View() tea.View {
	v := tea.NewView(a.frame)
	v.AltScreen = true
	return v
}

// run starts the program on a host.
func run(t *testing.T, h *hottytest.Host) (*app, *tea.Program, func()) {
	t.Helper()
	a := &app{s: hottytea.New(), lm: &panel{value: "red"}, rm: &panel{value: "red", auto: "add"}}
	a.left = surftea.New("left", a.lm, nil, surftea.Options{NoQuit: true})
	a.right = surftea.New("right", a.rm, nil, surftea.Options{NoQuit: true})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	p := tea.NewProgram(a, tea.WithContext(ctx), tea.WithInput(h), tea.WithOutput(a.s.Watch(h)),
		tea.WithoutSignalHandler(), tea.WithWindowSize(screen.W, screen.H), tea.WithEnvironment([]string{"TERM=xterm-256color"}))
	a.s.Attach(p.Send)
	a.left.Attach(p.Send)
	a.right.Attach(p.Send)
	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()
	t.Cleanup(func() { p.Quit() })
	return a, p, func() {
		defer cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("the program: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the program did not end")
		}
	}
}

// do runs f on the program's goroutine, and waits for it.
func do(t *testing.T, p interface{ Send(tea.Msg) }, f func()) {
	t.Helper()
	pr := probe{f, make(chan struct{})}
	p.Send(pr)
	select {
	case <-pr.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the program did not run the probe")
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}
