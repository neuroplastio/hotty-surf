package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"time"

	tea "charm.land/bubbletea/v2"
	"neuroplast.io/hotty-go/hottytest"
)

// probe is the program under test, which also runs a test's peeks on its
// goroutine, the only one that may read the model.
type probe struct{ *model }

type peek struct {
	f    func(*model)
	done chan struct{}
}

func (p probe) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if pk, ok := msg.(peek); ok {
		pk.f(p.model)
		close(pk.done)
		return p, nil
	}
	_, cmd := p.model.Update(msg)
	return p, cmd
}

// running is the program under test, as a test sees it.
type running struct {
	p *tea.Program
}

// at runs f on the program's goroutine and waits for it.
func (r running) at(f func(*model)) {
	done := make(chan struct{})
	r.p.Send(peek{f, done})
	<-done
}

func (r running) focused() (yes bool) {
	r.at(func(m *model) { yes = m.files.Focused() })
	return yes
}

func (r running) sel() (i int) {
	r.at(func(m *model) { i = m.a.Sel })
	return i
}

func (r running) note(i int) (s string) {
	r.at(func(m *model) { s = m.a.Files[i].Note })
	return s
}

// start runs the program on h, and returns it and a function that quits it
// and waits for it to end.
func start(t *testing.T, h *hottytest.Host) (running, func()) {
	t.Helper()
	m := newModel()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	p := tea.NewProgram(probe{m}, tea.WithContext(ctx), tea.WithInput(h), tea.WithOutput(m.sess.Watch(h)),
		tea.WithoutSignalHandler(), tea.WithWindowSize(80, 16), tea.WithEnvironment([]string{"TERM=xterm-256color"}))
	m.sess.Attach(p.Send)
	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()
	return running{p}, func() {
		defer cancel()
		p.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
		select {
		case err := <-done:
			if err != nil && ctx.Err() == nil {
				t.Errorf("the program: %v", err)
			}
		case <-time.After(20 * time.Second):
			t.Fatal("the program did not end")
		}
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("never: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// text is an element's text on the host's copy of the surface, "" while
// it has none.
func text(h *hottytest.Host, id string) string {
	s := h.Surface("files")
	if s == nil {
		return ""
	}
	return s.TextOf(id)
}

// On a hotty host: the surface is placed beside the list; the user clicks
// it, types into it and tabs out of it; the program's keys move the list
// and Tab gives the surface the keyboard back.
func TestOnAHost(t *testing.T) {
	h := hottytest.New(t, hottytest.Size(80, 16))
	m, quit := start(t, h)
	defer quit()
	eventually(t, "the surface placed", func() bool {
		s := h.Surface("files")
		return s != nil && s.Placed()
	})
	if got := text(h, "name"); got != "notes.md" {
		t.Fatalf("name %q", got)
	}
	if !strings.Contains(h.Screen(), "> notes.md") {
		t.Fatalf("the list isn't drawn:\n%s", h.Screen())
	}

	// The program's keys move its list; the surface follows.
	h.Key("arrow_down")
	eventually(t, "the surface shows budget.csv", func() bool { return text(h, "name") == "budget.csv" })

	// A click on the button runs its handler; a delta shows it.
	if err := h.Click("files", "star"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a star", func() bool { return strings.Contains(text(h, "stars"), "Starred 1 times") })

	// A click in the note gives the surface the keyboard: typing is the
	// surface's, not the list's (j would move it).
	if err := h.Click("files", "note"); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		t.Fatal(err)
	}
	eventually(t, "the surface has the keyboard", func() bool { return m.focused() })
	h.Key("j")
	h.Key("j")
	eventually(t, "the note", func() bool { return m.note(1) == "jj" })
	if i := m.sel(); i != 1 {
		t.Fatalf("typing in the surface moved the list to %d", i)
	}

	// Tab past the surface's last element gives the keyboard back.
	h.Key("tab")
	h.Key("tab")
	eventually(t, "the program has the keyboard", func() bool { return !m.focused() })
	h.Key("arrow_down")
	eventually(t, "the list moved", func() bool { return text(h, "name") == "logo.svg" })

	// The program's Tab gives the surface the keyboard, at the element
	// that had it last.
	h.Key("tab")
	eventually(t, "the surface has the keyboard again", func() bool { return m.focused() })
	eventually(t, "a=focus at the button", func() bool { return lastFocus(h) == "star" })
	if errs := h.Errors(); len(errs) > 0 {
		t.Fatalf("the host refused: %v", errs)
	}
}

// In a terminal that is no host, the surface is drawn in cells beside the
// list, and the same moves work: Tab into its first field, typing in it
// with the caret shown, a click on its button, Tab out of it.
func TestInCells(t *testing.T) {
	h := hottytest.New(t, hottytest.Text(), hottytest.Size(80, 16))
	m, quit := start(t, h)
	defer quit()
	eventually(t, "the surface in cells", func() bool { return strings.Contains(h.Screen(), "notes.md") && strings.Contains(h.Screen(), "Star") })

	h.Type("\t") // the program's Tab: the surface's first field
	eventually(t, "the surface has the keyboard", func() bool { return m.focused() })
	h.Type("jj")
	eventually(t, "the note typed in cells", func() bool { return strings.Contains(h.Screen(), "jj") })
	if i := m.sel(); i != 0 {
		t.Fatalf("typing in the surface moved the list to %d", i)
	}
	var caret bool
	m.at(func(m *model) { _, _, caret = m.files.Cursor() })
	if !caret {
		t.Error("no caret in the note")
	}

	// A click on the button, where it is drawn.
	x, y := find(h.Screen(), "Star")
	if x < 0 {
		t.Fatalf("no Star in\n%s", h.Screen())
	}
	m.p.Send(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	eventually(t, "a star", func() bool { return strings.Contains(h.Screen(), "Starred 1 times") })

	// From the button, Tab goes past the last element: the list's again.
	h.Type("\t")
	eventually(t, "the program has the keyboard", func() bool { return !m.focused() })
	h.Type("j")
	eventually(t, "the list moved", func() bool { return strings.Contains(h.Screen(), "> budget.csv") })

	// Back in, at the button, which had the keyboard last: enter stars.
	h.Type("\t")
	eventually(t, "the surface has the keyboard again", func() bool { return m.focused() })
	h.Type("\r")
	eventually(t, "a star for budget.csv", func() bool { return strings.Contains(h.Screen(), "Starred 1 times") })
	if h.Surface("files") != nil {
		t.Error("a surface was sent to a terminal that is no host")
	}
}

// find is where s first shows in the screen, as a cell; -1 if nowhere.
func find(screen, s string) (x, y int) {
	for i, line := range strings.Split(screen, "\n") {
		if j := strings.Index(line, s); j >= 0 {
			return len([]rune(line[:j])), i
		}
	}
	return -1, -1
}

// lastFocus is the element the program's last a=focus named.
func lastFocus(h *hottytest.Host) string {
	cmds := h.Commands()
	for i := len(cmds) - 1; i >= 0; i-- {
		if hotty.Get(cmds[i].Control, "a") == "focus" {
			return hotty.Get(cmds[i].Control, "t")
		}
	}
	return ""
}
