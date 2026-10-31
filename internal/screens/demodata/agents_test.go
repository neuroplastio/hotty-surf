package demodata

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"
)

// watch runs s.Watch with changed until done is called, and waits for it
// to be listening.
func watch(t *testing.T, s *Source, changed func([]Agent)) (done func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ended := make(chan struct{})
	go func() {
		_ = s.Watch(ctx, changed)
		close(ended)
	}()
	for deadline := time.Now().Add(5 * time.Second); ; {
		s.mu.Lock()
		n := len(s.watches)
		s.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the watch never started")
		}
		time.Sleep(time.Millisecond)
	}
	return func() { cancel(); <-ended }
}

// The roster: an address per agent, its host's before its name, and a
// pane of its own.
func TestRoster(t *testing.T) {
	as, err := Mock().Agents()
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 10 || as[0].Address != "claude-1" || as[2].Address != "devbox:codex-1" {
		t.Fatalf("roster %+v", as[:3])
	}
	panes := map[string]bool{}
	for _, a := range as {
		if panes[a.Pane] {
			t.Errorf("pane %s twice", a.Pane)
		}
		panes[a.Pane] = true
	}
	if !as[1].Attention || !as[6].Offline || as[4].Topic != "" {
		t.Error("the roster lost its blocked, offline or quiet agent")
	}
}

func TestLook(t *testing.T) {
	for _, c := range []struct{ status, dot, tone string }{
		{"working", "●", "warn"}, {"blocked", "◉", "error"}, {"error", "✗", "error"},
		{"done", "●", "info"}, {"idle", "○", "muted"}, {"", "·", "muted"}, {"thinking", "●", ""},
	} {
		if dot, tone := Look(c.status); dot != c.dot || tone != c.tone {
			t.Errorf("Look(%q) = %s %s, want %s %s", c.status, dot, tone, c.dot, c.tone)
		}
	}
}

// An item's lines are its refs, those with nothing to show left out; a
// line with nothing at all is "".
func TestItem(t *testing.T) {
	a := Agent{Name: "pi-1", Kind: "pi", Status: "done", Workspace: "docs"}
	if got := DefaultItem.Badges(a); len(got) != 0 {
		t.Errorf("badges %q", got)
	}
	if got := DefaultItem.Lines(a); !slices.Equal(got, []string{"", "done"}) {
		t.Errorf("lines %q", got)
	}
	it := Item{{"@kind", "@host"}, {"@workspace", "@pane", "@status"}}
	if got := it.Badges(a); !slices.Equal(got, []string{"pi"}) {
		t.Errorf("badges %q", got)
	}
	if got := it.Lines(a); !slices.Equal(got, []string{"docs · done"}) {
		t.Errorf("lines %q", got)
	}
	off := Agent{Name: "codex-2", Host: "devbox", Status: "working", Topic: "Speed up the test suite", Offline: true}
	if got := DefaultItem.Lines(off); !slices.Equal(got, []string{"Speed up the test suite", "devbox is not answering"}) {
		t.Errorf("an offline agent's lines %q", got)
	}
}

// A Watch hears the agents as it starts; a report changes an agent's
// status, and every Watch hears the agents now; one for an agent that
// isn't there is refused.
func TestReport(t *testing.T) {
	s := Mock()
	heard := make(chan []Agent, 1)
	done := watch(t, s, func(as []Agent) { heard <- as })
	if as := <-heard; len(as) != 10 || as[3].Status != "done" {
		t.Fatalf("heard first %+v", as)
	}
	if err := s.Report("pi-1", "working"); err != nil {
		t.Fatal(err)
	}
	if as := <-heard; as[3].Name != "pi-1" || as[3].Status != "working" {
		t.Errorf("heard %+v", as[3])
	}
	if err := s.Report("nobody", "idle"); err == nil {
		t.Error("a report for an agent that isn't there")
	}
	done()
	if err := s.Report("pi-1", "idle"); err != nil {
		t.Fatal(err)
	}
	select {
	case as := <-heard:
		t.Errorf("heard after the watch ended: %+v", as[3])
	default:
	}
}

// Reports from two goroutines at once: a Watch's last call has the last
// change, and its calls come one at a time.
func TestReportsAtOnce(t *testing.T) {
	s := Mock()
	var mu sync.Mutex
	var last []Agent
	inside := 0
	done := watch(t, s, func(as []Agent) {
		mu.Lock()
		inside++
		if inside > 1 {
			t.Error("two calls at once")
		}
		mu.Unlock()
		time.Sleep(10 * time.Microsecond)
		mu.Lock()
		last, inside = as, inside-1
		mu.Unlock()
	})
	defer done()
	var wg sync.WaitGroup
	for _, name := range []string{"pi-1", "claude-1"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				_ = s.Report(name, []string{"working", "done"}[i%2])
			}
		}()
	}
	wg.Wait()
	_ = s.Report("pi-1", "idle")
	want, _ := s.Agents()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
		mu.Lock()
		ok := slices.Equal(last, want)
		mu.Unlock()
		if ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the last call has %+v, want %+v", last, want)
		}
	}
}
