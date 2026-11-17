package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"
)

// running is the program under test, as a test sees it: its reads run on
// the program's loop, the only goroutine that may read its state.
type running struct{ p *program }

func (r running) at(f func(p *program)) {
	done := make(chan struct{})
	r.p.calls <- func() { f(r.p); close(done) }
	<-done
}

func (r running) inPane() (yes bool) { r.at(func(p *program) { yes = p.inPane }); return yes }
func (r running) sel() (i int)       { r.at(func(p *program) { i = p.a.Sel }); return i }
func (r running) note(i int) (s string) {
	r.at(func(p *program) { s = p.a.Files[i].Note })
	return s
}

// start runs the program on h, and returns it and a function that ends it.
func start(t *testing.T, h *hottytest.Host) (running, func()) {
	t.Helper()
	term := hottyterm.New(h, h, "pane", h.TermSize, nil)
	p := newProgram(term, NewApp())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.run(ctx) }()
	return running{p}, func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("the program: %v", err)
		}
		term.Close()
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

func text(h *hottytest.Host, name, id string) string {
	s := h.Surface(name)
	if s == nil {
		return ""
	}
	return s.TextOf(id)
}

// On a hotty host: the surface is placed beside the list; the user clicks
// it, types into it and tabs out of it; the program's keys move the list,
// and its Tab gives the surface the keyboard back, at the element that had
// it last.
func TestOnAHost(t *testing.T) {
	h := hottytest.New(t, hottytest.Size(80, 16))
	r, stop := start(t, h)
	defer stop()
	name := r.p.s.Name()
	eventually(t, "the surface placed", func() bool {
		s := h.Surface(name)
		return s != nil && s.Placed()
	})
	if got := text(h, name, "name"); got != "notes.md" {
		t.Fatalf("name %q", got)
	}
	eventually(t, "the list drawn", func() bool { return strings.Contains(h.Screen(), "> notes.md") })

	h.Key("arrow_down")
	eventually(t, "the surface shows budget.csv", func() bool { return text(h, name, "name") == "budget.csv" })

	if err := h.Click(name, "star"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a star", func() bool { return strings.Contains(text(h, name, "stars"), "Starred 1 times") })

	if err := h.Click(name, "note"); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		t.Fatal(err)
	}
	eventually(t, "the surface has the keyboard", r.inPane)
	h.Key("j")
	h.Key("j")
	eventually(t, "the note", func() bool { return r.note(1) == "jj" })
	if i := r.sel(); i != 1 {
		t.Fatalf("typing in the surface moved the list to %d", i)
	}

	h.Key("tab")
	h.Key("tab")
	eventually(t, "the program has the keyboard", func() bool { return !r.inPane() })
	h.Key("arrow_down")
	eventually(t, "the list moved", func() bool { return text(h, name, "name") == "logo.svg" })

	h.Key("tab")
	eventually(t, "the surface has the keyboard again", r.inPane)
	eventually(t, "a=focus at the button", func() bool { return lastFocus(h) == "star" })
	h.Key("enter") // the host passes it to the button: a star for logo.svg
	eventually(t, "a star for logo.svg", func() bool { return strings.Contains(text(h, name, "stars"), "Starred 1 times") })
	if errs := h.Errors(); len(errs) > 0 {
		t.Fatalf("the host refused: %v", errs)
	}
}

// In a terminal that is no host, the program draws the surface in cells
// beside the list, and the same moves work.
func TestInCells(t *testing.T) {
	h := hottytest.New(t, hottytest.Text(), hottytest.Size(80, 16))
	r, stop := start(t, h)
	defer stop()
	eventually(t, "the surface in cells", func() bool {
		return strings.Contains(h.Screen(), "notes.md") && strings.Contains(h.Screen(), "Star")
	})

	h.Type("\t") // the program's Tab: the surface's first field
	eventually(t, "the surface has the keyboard", r.inPane)
	h.Type("jj")
	eventually(t, "the note typed in cells", func() bool { return strings.Contains(h.Screen(), "jj") })
	if i := r.sel(); i != 0 {
		t.Fatalf("typing in the surface moved the list to %d", i)
	}
	var caret bool
	r.at(func(p *program) { _, _, caret = p.scr.Cursor() })
	if !caret {
		t.Error("no caret in the note")
	}

	click(h, "Star")
	eventually(t, "a star", func() bool { return strings.Contains(h.Screen(), "Starred 1 times") })

	h.Type("\t") // past the button, the last element: the list's again
	eventually(t, "the program has the keyboard", func() bool { return !r.inPane() })
	h.Type("j")
	eventually(t, "the list moved", func() bool { return strings.Contains(h.Screen(), "> budget.csv") })

	h.Type("\t") // back in, at the button, which had the keyboard last
	eventually(t, "the surface has the keyboard again", r.inPane)
	h.Type("\r")
	eventually(t, "a star for budget.csv", func() bool { return strings.Contains(h.Screen(), "Starred 1 times") })

	click(h, "main.go") // a click on the list gives the keyboard back
	eventually(t, "the list's keyboard, main.go picked", func() bool { return !r.inPane() && r.sel() == 3 })
	if h.Surface(r.p.s.Name()) != nil {
		t.Error("a surface was sent to a terminal that is no host")
	}
}

// click clicks where s first shows on the screen, as a terminal reports a
// left click (SGR).
func click(h *hottytest.Host, s string) {
	for y, line := range strings.Split(h.Screen(), "\n") {
		if j := strings.Index(line, s); j >= 0 {
			x := len([]rune(line[:j]))
			h.Type(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", x+1, y+1, x+1, y+1))
			return
		}
	}
	panic("no " + s + " on the screen:\n" + h.Screen())
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
