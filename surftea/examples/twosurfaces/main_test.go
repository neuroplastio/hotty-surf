package main

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfui"
)

func start(t *testing.T, h *hottytest.Host) (*app, func()) {
	t.Helper()
	a := newApp()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	p := tea.NewProgram(a, tea.WithContext(ctx), tea.WithInput(h), tea.WithOutput(a.s.Watch(h)),
		tea.WithoutSignalHandler(), tea.WithWindowSize(40, 12), tea.WithEnvironment([]string{"TERM=xterm-256color"}))
	a.s.Attach(p.Send)
	a.form.Attach(p.Send)
	a.source.Attach(p.Send)
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	return a, func() {
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

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

// Picking a theme in the form's list changes the source, a surface of its
// own, and the program has two surfaces and a stylesheet.
func TestTwoSurfaces(t *testing.T) {
	h := hottytest.New(t)
	_, wait := start(t, h)
	eventually(t, "both surfaces", func() bool { return h.Surface("form") != nil && h.Surface("source") != nil })
	if got := h.Surface("source").TextOf("r.ksource"); got != "theme = nord" {
		t.Errorf("the source says %q", got)
	}
	area := map[string]any{"c": 0, "r": 1, "w": 12, "h": 1}
	if err := h.Emit("form", "click", themeID, map[string]any{"area": area}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the list", func() bool { l := h.Surface("form-list"); return l != nil && l.Placed() })
	if err := h.Click("form-list", surfui.OptionID(1)); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the source changed", func() bool { return h.Surface("source").TextOf("r.ksource") == "theme = dracula" })
	eventually(t, "the list gone", func() bool { return h.Surface("form-list") == nil })
	h.Type("q")
	wait()
}

// Without a host both draw in cells, and keys reach the form.
func TestTwoSurfacesInCells(t *testing.T) {
	h := hottytest.New(t, hottytest.Text())
	_, wait := start(t, h)
	eventually(t, "the cells", func() bool {
		s := h.Screen()
		return strings.Contains(s, "Theme") && strings.Contains(s, "theme = nord")
	})
	if len(h.Surfaces()) != 0 {
		t.Error("a surface on a terminal that is no host")
	}
	h.Type("q")
	wait()
}
