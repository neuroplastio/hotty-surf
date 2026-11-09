package surfrun_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// clicks is a surface with no cells UI of its own: a field and a button,
// whose clicks it counts.
type clicks struct {
	n    int
	name string
}

func (m *clicks) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Row(surfui.Text("Name"), surfui.Field().Value(m.name).OnChange("name")).Gap(1),
		surfui.Row(surfui.Button("Count").OnClick("count"), surfui.Text(fmt.Sprintf("%d clicks", m.n))).Gap(1),
	).Bind("q", "Quit", surfrun.Quit{})
}

func (m *clicks) Do(act surfui.Action, ev surfrun.Event) {
	switch {
	case act == "name":
		m.name = ev.Value
	case ev.Kind == "click":
		m.n++
	}
}

// cellsRun runs m on a terminal that is no hotty host, and returns the
// terminal, the runtime, and stop, which ends the run and returns all it
// wrote.
func cellsRun(t *testing.T, m surfrun.Model) (h *hottytest.Host, r *surfrun.Runtime, stop func() string) {
	t.Helper()
	h = hottytest.New(t, hottytest.Text(), hottytest.Size(40, 6))
	term := hottyterm.New(h, h, "t", h.TermSize, &hottyterm.Known{})
	var out strings.Builder
	r = surfrun.New(term, m, surfrun.Options{Name: "app", Stream: &out})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	stop = func() string {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("the run: %v", err)
		}
		_ = h.Close()
		return out.String()
	}
	return h, r, stop
}

// waitScreen waits until the terminal's screen holds want.
func waitScreen(t *testing.T, h *hottytest.Host, want string) {
	t.Helper()
	for end := time.Now().Add(30 * time.Second); !strings.Contains(h.Screen(), want); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("the screen never showed %q:\n%s", want, h.Screen())
		}
	}
}

// A model with no cells UI of its own, in a terminal that is no host, is
// drawn from its tree, and takes the keys and the mouse there: a click on
// its button, typing in its field and Tab out of it, which commits.
func TestCellsFromTheTree(t *testing.T) {
	m := &clicks{name: "bob"}
	h, r, stop := cellsRun(t, m)
	waitScreen(t, h, "0 clicks")
	if s := h.Screen(); !strings.Contains(s, "Name  bob") || !strings.Contains(s, "Count") {
		t.Errorf("the screen:\n%s", s)
	}
	// The button's cells, by the renderer, then a press and a release
	// there in SGR's mouse encoding, 1-based.
	var x, y int
	r.Call(func() { x, y, _ = r.Renderer().Where("r.1.0") })
	h.Type(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", x+1, y+1, x+1, y+1))
	waitScreen(t, h, "1 clicks")
	r.Call(func() { x, y, _ = r.Renderer().Where("r.0.1") })
	h.Type(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", x+1, y+1, x+1, y+1))
	h.Type("\x7f\x7f\x7fann\t")
	waitScreen(t, h, "Name  ann")
	var name string
	r.Call(func() { name = m.name })
	if name != "ann" {
		t.Errorf("the field committed %q", name)
	}
	if s := h.Output(); !strings.Contains(s, "\x1b[?1000h\x1b[?1006h") {
		t.Error("the mouse is not reported, with a button to press")
	}
	out := stop()
	if on, off := strings.LastIndex(out, "\x1b[?1000h"), strings.LastIndex(out, "\x1b[?1000l"); off < on {
		t.Errorf("the mouse left on at the end: on at %d, off at %d", on, off)
	}
	if strings.Contains(out, "no cells UI") {
		t.Error("the notice of no cells UI")
	}
}

// After Fail, a model with no cells UI is drawn from its tree, and after
// Recover it is on the host again, the mouse and the caret given back.
func TestFailToTheTree(t *testing.T) {
	h := hottytest.New(t, hottytest.Size(40, 6))
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	m := &clicks{name: "bob"}
	var out strings.Builder
	r := surfrun.New(term, m, surfrun.Options{Name: "app", Stream: &out})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	defer func() { cancel(); <-done; _ = h.Close() }()
	name := term.Surface("app")
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(2 * time.Millisecond) {
		if s := h.Surface(name); s != nil && s.Placed() {
			break
		}
		if time.Now().After(end) {
			t.Fatal("never placed")
		}
	}
	r.Call(r.Fail)
	waitScreen(t, h, "0 clicks")
	var drawn bool
	r.Call(func() { drawn = r.Renderer() != nil })
	if !drawn {
		t.Fatal("no renderer after Fail")
	}
	r.Call(r.Recover)
	r.Call(func() { drawn = r.Renderer() != nil })
	if drawn {
		t.Error("a renderer still, back on the host")
	}
	var s string
	r.Call(func() { s = out.String() })
	if on, off := strings.LastIndex(s, "\x1b[?1000h"), strings.LastIndex(s, "\x1b[?1000l"); off < on {
		t.Errorf("back on the host with the mouse reported: on at %d, off at %d", on, off)
	}
}

// While an element's menu is open the runtime asks its terminal for all
// the pointer's motion (1003), and the bare motion lights the row under
// it; when the menu closes it goes back to the motion with a button down
// (1002), and the end turns 1003 off too.
func TestCellsMenuAsksForAllMotion(t *testing.T) {
	m := &ctx{rows: []string{"a", "b", "plain"}, chosen: "red"}
	h, r, stop := cellsRun(t, m)
	waitScreen(t, h, "plain")
	var y int
	r.Call(func() { _, y, _ = r.Renderer().Where(rowA) })
	if strings.Contains(h.Output(), "\x1b[?1003h") {
		t.Error("all motion is asked for with no menu open")
	}
	h.Type(fmt.Sprintf("\x1b[<2;%d;%dM\x1b[<2;%d;%dm", 30, y+1, 30, y+1))
	waitScreen(t, h, "Open in")
	if !strings.Contains(h.Output(), "\x1b[?1003h") {
		t.Fatal("all motion is not asked for with a menu open")
	}
	// The pointer moves, no button down (SGR 35), onto "Open in".
	var cx, cy int
	for yy, line := range strings.Split(h.Screen(), "\n") {
		if i := strings.Index(line, "Open in"); i >= 0 {
			cx, cy = len([]rune(line[:i])), yy
		}
	}
	h.Type(fmt.Sprintf("\x1b[<35;%d;%dM", cx+1, cy+1))
	lit := func() string {
		var got string
		r.Call(func() {
			for _, line := range r.Renderer().Screen().Lines {
				for _, run := range line {
					if run.Bg == cells.Sel && strings.TrimSpace(run.Text) != "" {
						got = strings.TrimSpace(run.Text)
					}
				}
			}
		})
		return got
	}
	for end := time.Now().Add(30 * time.Second); !strings.HasPrefix(lit(), "Open in"); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("the row under the pointer is not lit: %q", lit())
		}
	}
	// A click elsewhere closes the menu: 1003 goes off, 1002 is back.
	h.Type(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", 1, 1, 1, 1))
	for end := time.Now().Add(30 * time.Second); !strings.Contains(h.Output(), "\x1b[?1003l\x1b[?1002h"); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("1003 is never dropped after the menu closed:\n%q", h.Output())
		}
	}
	out := stop()
	if on, off := strings.LastIndex(out, "\x1b[?1003h"), strings.LastIndex(out, "\x1b[?1003l"); off < on {
		t.Errorf("all motion left on at the end: on at %d, off at %d", on, off)
	}
}
