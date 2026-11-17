package main

import "neuroplast.io/hotty-surf/internal/spikeui"

// AgentKind is the program an agent runs in a session, if one does.
type AgentKind string

const (
	NoAgent    AgentKind = ""
	ClaudeCode AgentKind = "claude-code"
	Codex      AgentKind = "codex"
)

// AgentState is what the agent is doing.
type AgentState string

const (
	NoState AgentState = ""
	Working AgentState = "working"
	Idle    AgentState = "idle"
	Done    AgentState = "done"
)

// Session is one session, as the nav lists it. Whether it is busy is
// its layout's: a pane an agent is working in.
type Session struct {
	ID        string
	Name      string
	Workspace string
	Agent     AgentKind
	State     AgentState
	Activity  string // the agent's last activity, if it said one
	Layout    *spikeui.PaneLayout
}

// Data is what the nav is told: the sessions, which one is active (the
// one shown), and which the cursor is on.
type Data struct {
	Sessions []Session
	Active   string
	Selected string
}

// Status is a session's third line: its last activity if it has one, else
// the agent and what it is doing.
func Status(s Session) string {
	switch {
	case s.Activity != "":
		return s.Activity
	case s.Agent != NoAgent && s.State != NoState:
		return string(s.Agent) + " · " + string(s.State)
	}
	return ""
}

func pane(busy bool, share float64) *spikeui.PaneLayout {
	return &spikeui.PaneLayout{Busy: busy, Share: share}
}

func split(d spikeui.SplitDir, share float64, kids ...*spikeui.PaneLayout) *spikeui.PaneLayout {
	return &spikeui.PaneLayout{Dir: d, Share: share, Kids: kids}
}

// Mock is five made-up sessions, the first active and the third selected.
func Mock() Data {
	return Data{
		Active:   "s1",
		Selected: "s3",
		Sessions: []Session{
			{ID: "s1", Name: "api", Workspace: "acme", Agent: ClaudeCode, State: Working,
				Layout: split(spikeui.SplitRow, 0, pane(true, 0), pane(true, 0))},
			{ID: "s2", Name: "docs", Workspace: "acme",
				Layout: pane(false, 0)},
			{ID: "s3", Name: "web", Workspace: "acme", Agent: ClaudeCode, State: Working,
				Activity: "Reading the parser's tests and their fixtures",
				Layout:   split(spikeui.SplitRow, 0, split(spikeui.SplitCol, 0.5, pane(true, 0), pane(true, 0)), pane(false, 0.25), pane(false, 0.25))},
			{ID: "s4", Name: "infra", Workspace: "acme", Agent: Codex, State: Idle,
				Layout: pane(false, 0)},
			{ID: "s5", Name: "notes", Workspace: "home",
				Layout: split(spikeui.SplitCol, 0, pane(false, 0), pane(false, 0))},
		},
	}
}
