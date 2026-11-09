package screens

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surflive"
)

// The agents tool's parts, by id.
var (
	agList    = id("list")
	agEmpty   = id("empty")
	agRefused = id("refused")
)

// agEntry is a part of agent address's entry: its row's dot (0), emoji
// (1) and lines (2: the name's line, "name", then l0, l1… as the item
// numbers them: the topic, then the status).
func agEntry(address string, parts ...any) string {
	return id(append([]any{"list", address}, parts...)...)
}

// openAgents is the tool in a left dock, 29 cells wide, on the mock.
func openAgents(t *testing.T, m mode) (*app, *Agents) {
	t.Helper()
	src := demodata.Mock()
	l := NewAgents(src)
	return start(t, m.agents(l), src, false, 29, 43), l
}

// entries are the addresses of the entries the host shows, in order.
func entries(a *app) []string {
	var out []string
	html := a.html()
	prefix := `id="` + agList + ".k"
	for {
		i := strings.Index(html, prefix)
		if i < 0 {
			return out
		}
		html = html[i+len(prefix):]
		end := strings.IndexByte(html, '"')
		if !strings.Contains(html[:end], ".") {
			out = append(out, html[:end])
		}
		html = html[end:]
	}
}

// Until the source answers, nothing is drawn.
func TestAgentsLoading(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		l := NewAgents(demodata.Mock())
		doc := Document(surflive.New(m.agents(l)), "")
		body := doc[strings.Index(doc, "<body"):]
		if strings.Contains(body, "u-list") || strings.Contains(body, "No agents") {
			t.Errorf("drawn before the first answer: %s", body)
		}
	})
}

// The entries, in the source's order, each its dot in its tone, its
// emoji, its name in bold with ✋ while its hand is up and its host
// after it, and a muted line for the topic and the status; the first has
// the keyboard.
func TestAgentsOpens(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openAgents(t, m)
		want := []string{"claude-1", "claude-2", safeID("devbox:codex-1"), "pi-1", "opencode-1",
			"claude-3", safeID("devbox:codex-2"), "claude-4", "pi-2", "opencode-2"}
		if got := entries(a); !slices.Equal(got, want) {
			t.Errorf("entries %q, want %q", got, want)
		}
		if f := a.Surface().Focused(); f != agEntry("claude-1") {
			t.Errorf("focus on %q, want the first entry", f)
		}
		for _, c := range []struct{ addr, dot, tone, emoji string }{
			{"claude-1", "●", "u-tone-warn", "🦀"},
			{"claude-2", "◉", "u-tone-error", "🐙"},
			{"pi-1", "●", "u-tone-info", "🦊"},
			{"opencode-1", "○", "u-tone-muted", "🐝"},
			{"claude-3", "✗", "u-tone-error", "🦉"},
		} {
			if got := a.text(agEntry(c.addr, 0, 0)); got != c.dot {
				t.Errorf("%s's dot %q", c.addr, got)
			}
			if !a.Has(agEntry(c.addr, 0, 0), c.tone) {
				t.Errorf("%s's dot is not %s", c.addr, c.tone)
			}
			if got := a.text(agEntry(c.addr, 0, 1)); got != c.emoji {
				t.Errorf("%s's emoji %q", c.addr, got)
			}
		}
		if got := a.text(agEntry("claude-1", 0, 2)); got != "claude-1Fix the flaky parser testworking" {
			t.Errorf("claude-1's lines %q", got)
		}
		if got := a.text(agEntry("claude-2", 0, 2, "name", "hand")); got != "✋" {
			t.Errorf("claude-2's hand %q", got)
		}
		if got := a.text(agEntry("opencode-1", 0, 2)); got != "opencode-1idle" {
			t.Errorf("opencode-1, with no topic: %q", got)
		}
		if got := a.text(agEntry("devbox:codex-2", 0, 2, "name", "host")); got != "devbox" {
			t.Errorf("codex-2's host %q", got)
		}
		if got := a.text(agEntry("devbox:codex-2", 0, 0)); got != "◌" || !a.Has(agEntry("devbox:codex-2", 0, 0), "u-tone-error") {
			t.Errorf("codex-2, offline: its dot %q", got)
		}
		if got := a.text(agEntry("devbox:codex-2", 0, 2, "l1")); got != "devbox is not answering" {
			t.Errorf("codex-2, offline: its status line %q", got)
		}
		if !a.Has(agEntry("claude-1", 0, 2, "name", 0), "u-bold") || !a.Has(agEntry("claude-1", 0, 2, "l1"), "u-tone-muted") {
			t.Error("the name is not bold, or the lines not muted")
		}
	})
}

// A click, Enter or Space on an entry goes to its agent's pane through
// the source. What the source refuses is said under the entries, in the
// error tone, until the next act.
func TestAgentsGoTo(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openAgents(t, m)
		a.keys("enter")
		a.acted("go p-1")
		a.click(agEntry("claude-3", 0, 2, "l0")) // any part of the entry
		a.acted("go p-1", "go p-6")
		a.Wait("claude-3 has the keyboard", func() bool { return a.Surface().Focused() == agEntry("claude-3") })
		a.keys("tab")
		a.Wait("codex-2 has the keyboard", func() bool { return a.Surface().Focused() == agEntry("devbox:codex-2") })
		a.keys("space")
		a.acted("go p-1", "go p-6", "go p-7")
		a.Wait("the refusal", func() bool { return a.text(agRefused) == "devbox is not answering" && a.Has(agRefused, "u-tone-error") })
		a.click(agEntry("claude-4"))
		a.acted("go p-1", "go p-6", "go p-7", "go p-8")
		a.Wait("the refusal gone", func() bool { return !strings.Contains(a.html(), agRefused) })
	})
}

// An agent the source no longer has is refused, in the error tone under
// the entries, until the next act, and nothing is asked of the source.
func TestAgentsRefused(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, l := openAgents(t, m)
		a.Call(func() {
			l.answer(append(slices.Clone(l.Agents), demodata.Agent{Address: "gone-1", Name: "gone-1", Status: "idle", Pane: "p-99"}), nil)
		})
		a.Wait("the stale entry", func() bool { return strings.Contains(a.html(), agEntry("gone-1")) })
		a.click(agEntry("gone-1"))
		a.Wait("the refusal", func() bool { return a.text(agRefused) == "no agent gone-1 now" && a.Has(agRefused, "u-tone-error") })
		if acts := a.src.Acts(); len(acts) != 0 {
			t.Errorf("acts %q", acts)
		}
		a.click(agEntry("pi-1"))
		a.acted("go p-4")
		a.Wait("the refusal gone", func() bool { return !strings.Contains(a.html(), agRefused) })
	})
}

// With no agents, the caption says how one appears.
func TestAgentsEmpty(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, l := openAgents(t, m)
		a.Call(func() { l.answer(nil, nil) })
		a.Wait("the caption", func() bool {
			return a.text(agEmpty) == "No agents yet. An agent appears when its harness reports from its pane."
		})
		if !a.Has(agEmpty, "u-small") || !a.Has(agEmpty, "u-tone-muted") {
			t.Error("the caption is not a muted caption")
		}
	})
}

// An agent that changes in the source redraws its entry, and nothing else
// does: a few ops (S4, F10).
func TestAgentsChangeIsAFewOps(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, l := openAgents(t, m)
		n := len(a.Ops())
		a.Call(func() {
			now := slices.Clone(l.Agents)
			now[3].Status = "working"
			l.answer(now, nil)
		})
		a.Wait("pi-1 working", func() bool { return a.text(agEntry("pi-1", 0, 2, "l1")) == "working" })
		ops := a.opsSince(n)
		if len(ops) == 0 || len(ops) > 4 {
			t.Errorf("%d ops for a status: %+v", len(ops), ops)
		}
		for _, op := range ops {
			if !strings.HasPrefix(op.Target, agEntry("pi-1")) {
				t.Errorf("an op on %s", op.Target)
			}
		}
	})
}

// The cells UI (F7): the same entries as lines, the arrows move the
// keyboard between them, and Enter goes to the agent.
func TestAgentsCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, l := openAgents(t, m)
		a.cells()
		a.screenHas("claude-1")
		a.screenHas("Fix the flaky")
		a.keys("arrow_down", "arrow_down", "arrow_down")
		a.until("pi-1 has the keyboard", func() bool { return l.At == 3 })
		a.keys("enter")
		a.acted("go p-4")
	})
}
