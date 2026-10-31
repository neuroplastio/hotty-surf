package demodata

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Agent is one entry of the agents list: a coding agent at work in a
// pane.
type Agent struct {
	// Address is how the list names it: "claude-1" on this machine,
	// "devbox:codex-1" on another host.
	Address string
	// Name is its name ("claude-1").
	Name string
	// Host is "" on this machine, else the host's name.
	Host string
	// Status is a word: working, blocked, done, idle and error are the
	// ones Look knows.
	Status string
	// Topic is what it is at, "" while it has said nothing.
	Topic string
	// Icon is its pane's emoji, "" for none.
	Icon string
	// Attention is whether it waits for the person (its hand is up).
	Attention bool
	// Offline is whether its host has stopped answering. The entry stays.
	Offline bool
	// Kind is its harness ("claude"), Model its model, Workspace its
	// workspace's name and Pane its pane's id.
	Kind, Model, Workspace, Pane string
}

// Hand is the mark an entry wears while its agent waits for the person.
const Hand = "✋"

// Look is a status's dot and its tone, in the vocabulary's names:
// working a dot in warn, blocked a ringed dot in error, error a cross in
// error, done a dot in info, idle a hollow dot muted, and no status a
// middle dot muted. Any other word is a dot in the text's colour (tone
// "").
func Look(status string) (dot, tone string) {
	switch status {
	case "working":
		return "●", "warn"
	case "blocked":
		return "◉", "error"
	case "error":
		return "✗", "error"
	case "done":
		return "●", "info"
	case "idle":
		return "○", "muted"
	case "":
		return "·", "muted"
	}
	return "●", ""
}

// Item is what an agent's entry shows. Its first line holds the badges
// after the name, and each line after it is a line under the name. A
// line is a list of refs: @topic, @model, @kind, @status, @workspace,
// @host or @pane. An offline agent's @status is that its host is not
// answering, not the last status it sent.
type Item [][]string

// DefaultItem is the item the demo's agents list draws: no badges after
// the name, then the topic, then the status.
var DefaultItem = Item{{}, {"@topic"}, {"@status"}}

// Badges are what the name's line shows after the name, those with
// nothing to show left out.
func (it Item) Badges(a Agent) []string {
	if len(it) == 0 {
		return nil
	}
	return show(a, it[0])
}

// Lines are the lines under the name, one per line of the item after the
// first, its refs joined by " · ". A line with nothing to show is "".
func (it Item) Lines(a Agent) []string {
	if len(it) < 2 {
		return nil
	}
	out := make([]string, len(it)-1)
	for i, refs := range it[1:] {
		out[i] = strings.Join(show(a, refs), " · ")
	}
	return out
}

// show is what refs show of a, in order, those with nothing to show
// left out.
func show(a Agent, refs []string) []string {
	var out []string
	for _, ref := range refs {
		var v string
		switch ref {
		case "@topic":
			v = a.Topic
		case "@model":
			v = a.Model
		case "@kind":
			v = a.Kind
		case "@status":
			v = a.Status
			if a.Offline {
				v = a.Host + " is not answering"
			}
		case "@workspace":
			v = a.Workspace
		case "@host":
			v = a.Host
		case "@pane":
			v = a.Pane
		}
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// roster is the made-up agents, in the list's order: four harnesses,
// two agents on another host (one of them offline), two waiting for the
// person, and each status Look knows.
func roster() []Agent {
	return addressed([]Agent{
		{Name: "claude-1", Kind: "claude", Model: "opus-5.5", Workspace: "api", Pane: "p-1",
			Icon: "🦀", Status: "working", Topic: "Fix the flaky parser test"},
		{Name: "claude-2", Kind: "claude", Model: "opus-5.5", Workspace: "api", Pane: "p-2",
			Icon: "🐙", Status: "blocked", Topic: "Redesign the toolbar", Attention: true},
		{Name: "codex-1", Host: "devbox", Kind: "codex", Model: "gpt-6", Workspace: "main", Pane: "p-3",
			Status: "working", Topic: "Rebuild the search index"},
		{Name: "pi-1", Kind: "pi", Model: "sonnet-5.5", Workspace: "docs", Pane: "p-4",
			Icon: "🦊", Status: "done", Topic: "Write the release notes"},
		{Name: "opencode-1", Kind: "opencode", Model: "kimi-k2.5", Workspace: "docs", Pane: "p-5",
			Icon: "🐝", Status: "idle"},
		{Name: "claude-3", Kind: "claude", Model: "opus-5.5", Workspace: "render", Pane: "p-6",
			Icon: "🦉", Status: "error", Topic: "Bisect the login regression", Attention: true},
		{Name: "codex-2", Host: "devbox", Kind: "codex", Model: "gpt-6", Workspace: "ci", Pane: "p-7",
			Status: "working", Topic: "Speed up the test suite", Offline: true},
		{Name: "claude-4", Kind: "claude", Model: "sonnet-5.5", Workspace: "site", Pane: "p-8",
			Icon: "🌿", Status: "working", Topic: "Group the settings by name"},
		{Name: "pi-2", Kind: "pi", Model: "opus-5.5", Workspace: "api", Pane: "p-9",
			Icon: "🐢", Status: "idle", Topic: "Review the keymap tables"},
		{Name: "opencode-2", Kind: "opencode", Model: "kimi-k2.5", Workspace: "render", Pane: "p-10",
			Icon: "🧪", Status: "done", Topic: "Add fuzz tests for the tokenizer"},
	})
}

// addressed is as, each agent's Address set from its host and name.
func addressed(as []Agent) []Agent {
	for i := range as {
		as[i].Address = as[i].Name
		if as[i].Host != "" {
			as[i].Address = as[i].Host + ":" + as[i].Name
		}
	}
	return as
}

// Agents are the agents now, a fresh copy each call.
func (s *Source) Agents() ([]Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.agents), nil
}

// Watch calls changed with the agents now, and again each time they
// change (Report, Close), until ctx ends; it returns nil then. Its calls
// come one at a time, on Watch's goroutine, each with the agents as they
// are as it is made: changes that come quickly may give one call, but
// the last call has the last change. changed may call the Source.
func (s *Source) Watch(ctx context.Context, changed func([]Agent)) error {
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	s.mu.Lock()
	n := s.nextWatch
	s.nextWatch++
	s.watches[n] = wake
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.watches, n)
		s.mu.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-wake:
			s.mu.Lock()
			now := slices.Clone(s.agents)
			s.mu.Unlock()
			changed(now)
		}
	}
}

// Report sets the status of the agent called name, as its harness would
// report it, and every Watch hears it.
func (s *Source) Report(name, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.agents, func(a Agent) bool { return a.Name == name })
	if i < 0 {
		return fmt.Errorf("no agent %s", name)
	}
	s.agents[i].Status = status
	s.tell()
	return nil
}
