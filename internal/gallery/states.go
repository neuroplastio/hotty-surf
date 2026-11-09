package gallery

import (
	"neuroplast.io/hotty-surf/internal/screens"
	"neuroplast.io/hotty-surf/internal/screens/demodata"
)

// states are the stories files' Go state, by file: the value a story of
// it is drawn in, made new for each story, by its name. A partial's
// story is drawn in it, its fields read and its methods called by the
// partial (palette-row's icons() and groupMarks(row)), and its bound
// props evaluated in it (:row="rows[0]"); a screen's story is it. Every
// one is on demodata's made-up data. Each must be a component of a
// package the gallery has (packs), and is never attached to the run: a
// palette's Enter would end it.
var states = map[string]func(story string) any{
	"foot-hint":     func(string) any { return screens.NewPalette(demodata.Mock()) },
	"key-caps":      func(string) any { return screens.NewPalette(demodata.Mock()) },
	"palette-row":   func(string) any { return screens.NewPalette(demodata.Mock()) },
	"notice-row":    func(string) any { return screens.NewNotices() },
	"setting-group": func(string) any { return screens.NewSettings(demodata.Mock()) },
	"setting-row":   func(string) any { return screens.NewSettings(demodata.Mock()) },
	"palette":       palette,
	"settings":      settings,
	"notices":       notices,
	"agents":        agents,
}

func palette(story string) any {
	p := screens.NewPalette(demodata.Mock())
	switch story {
	case "a query":
		p.SetQuery("split")
	case "no match":
		p.SetQuery("zzzz")
	}
	return p
}

func settings(story string) any {
	s := screens.NewSettings(demodata.Mock())
	if story == "a search" {
		s.Search("border")
	}
	return s
}

func notices(story string) any {
	n := screens.NewNotices()
	if story == "errors" {
		n.SetFilter("Errors")
	}
	return n
}

// agents is the agents tool with each story's entries: the row, drawn
// through its screen (internal/screens/agents.surf has it inline).
func agents(story string) any {
	l := screens.NewAgents(demodata.Mock())
	l.Loaded = true
	switch story {
	case "working":
		l.Agents = []demodata.Agent{
			{Address: "claude-1", Name: "claude-1", Status: "working", Topic: "the gallery's canvas", Icon: "🦀", Kind: "claude", Model: "opus", Workspace: "main", Pane: "p-3"},
			{Address: "codex-1", Name: "codex-1", Status: "done", Topic: "a benchmark", Icon: "🐙", Kind: "codex", Workspace: "main", Pane: "p-4"},
			{Address: "pi-1", Name: "pi-1", Status: "idle", Kind: "pi", Workspace: "notes", Pane: "p-9"},
		}
	case "attention and offline":
		l.Agents = []demodata.Agent{
			{Address: "claude-2", Name: "claude-2", Status: "blocked", Topic: "waits for a review", Attention: true, Kind: "claude", Workspace: "main", Pane: "p-5"},
			{Address: "devbox:codex-1", Name: "codex-1", Host: "devbox", Status: "error", Offline: true, Kind: "codex", Workspace: "remote", Pane: "p-1"},
		}
	case "refused":
		l.Agents = []demodata.Agent{{Address: "claude-1", Name: "claude-1", Status: "working", Kind: "claude", Pane: "p-3"}}
		l.Refused = "no pane p-3"
	}
	return l
}
