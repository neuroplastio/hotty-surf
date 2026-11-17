package main

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"neuroplast.io/hotty-go/hottytea"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf/surftest"
)

// TestTagACommit runs the program on a test host and does what the user
// does: moves the list, tabs into the surface, tags the commit, and reads
// the tag in the list.
func TestTagACommit(t *testing.T) {
	h := hottytest.New(t, hottytest.Size(100, 30))
	sess := hottytea.New()
	p := tea.NewProgram(newModel(sess),
		tea.WithInput(h), tea.WithOutput(sess.Watch(h)),
		tea.WithWindowSize(100, 30), tea.WithoutSignals())
	sess.Attach(p.Send)
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	shows := func(s string) func() bool {
		return func() bool {
			d := h.Surface("commit")
			return d != nil && d.Placed() && strings.Contains(d.Text(), s)
		}
	}
	waitFor(t, "the first commit", shows("Cache parsed configs per directory"))

	h.Key("arrow_down")
	h.Key("arrow_down")
	waitFor(t, "the third commit", shows("Rename Sink to Writer"))

	h.Key("tab")
	waitFor(t, "the keyboard in the name field", func() bool {
		return h.Surface("commit").Focused() == "tag-name"
	})
	typeText(h, "rc2")
	h.Key("tab")
	typeText(h, "ship it")
	h.Key("enter")

	waitFor(t, "the tag in the list", func() bool {
		return strings.Contains(h.Screen(), "Mira Okafor · 11 Nov ◆ rc2")
	})
	waitFor(t, "the tag in the details", shows("rc2 — ship it"))

	// Tab past the button gives the list the keyboard back, and q quits.
	h.Key("tab")
	h.Key("tab")
	waitFor(t, "the keyboard back in the list", func() bool {
		return h.Surface("commit").Focused() == ""
	})
	h.Key("q")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("q didn't quit")
	}
}

func TestLint(t *testing.T) {
	surftest.Lint(t, ".")
}

func typeText(h *hottytest.Host, s string) {
	for _, r := range s {
		k := string(r)
		if r == ' ' {
			k = "space"
		}
		h.Key(k)
	}
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %s", what)
}
