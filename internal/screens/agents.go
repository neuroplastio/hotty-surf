package screens

import (
	"context"
	"strings"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surfrun"
)

// Agents is an agents list, a tool in a dock (S2, S3, S4): an entry per
// coding agent, each a box of its own that takes the keyboard, and a
// click, Enter or Space on one goes to its agent's pane. Its agents are
// demodata's, read when the run starts and again each time the source
// says they changed; until the first answer, nothing is drawn.
// agents.surf draws it.
type Agents struct {
	Agents  []demodata.Agent
	Loaded  bool   // the source has answered
	Refused string // why the source refused the last act, until the next
	At      int    // the entry with the keyboard

	src  *demodata.Source
	item demodata.Item      // what an entry shows: demodata's item
	stop context.CancelFunc // the watch's
}

// NewAgents is the tool on src, before its first answer.
func NewAgents(src *demodata.Source) *Agents {
	return &Agents{src: src, item: demodata.DefaultItem}
}

// Attach starts the agents' watch when the run starts, whose first answer
// is the agents then, each answer posted to the run, and stops the watch
// when it ends.
func (l *Agents) Attach(r surfrun.Runner) {
	if l.stop != nil {
		l.stop()
		l.stop = nil
	}
	if r == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.stop = cancel
	go func() {
		_ = l.src.Watch(ctx, func(now []demodata.Agent) { r.Post(func() { l.Loaded = true; l.answer(now, nil) }) })
	}()
}

// answer is the source's answer: the agents now, or why there are none.
func (l *Agents) answer(list []demodata.Agent, err error) {
	if err != nil {
		l.Refused = firstLine(err.Error())
		return
	}
	l.Agents = list
	l.At = max(min(l.At, len(list)-1), 0)
}

// GoTo goes to a's pane through the source: an agent that is no longer
// among the source's is refused, as is what the source refuses.
func (l *Agents) GoTo(a demodata.Agent) {
	l.Refused = ""
	now, err := l.src.Agents()
	if err != nil {
		l.Refused = firstLine(err.Error())
		return
	}
	found := false
	for _, b := range now {
		found = found || b.Address == a.Address
	}
	if !found {
		l.Refused = "no agent " + a.Address + " now"
		return
	}
	if err := l.src.GoTo(a.Pane); err != nil {
		l.Refused = firstLine(err.Error())
	}
}

// Enter is Enter or Space on the entry with the keyboard.
func (l *Agents) Enter() {
	if l.At < len(l.Agents) {
		l.GoTo(l.Agents[l.At])
	}
}

// Focus is entry i taking the keyboard.
func (l *Agents) Focus(i int) { l.At = i }

// Dot and Tone are a's dot and its tone: an offline agent's is a red ◌.
func (l *Agents) Dot(a demodata.Agent) string {
	if a.Offline {
		return "◌"
	}
	dot, _ := demodata.Look(a.Status)
	return dot
}

func (l *Agents) Tone(a demodata.Agent) string {
	if a.Offline {
		return "error"
	}
	_, tone := demodata.Look(a.Status)
	return tone
}

// Chips are what a's name's line shows after the name, and Lines the
// lines under it, as the style's item says: "" for a line with nothing to
// show.
func (l *Agents) Chips(a demodata.Agent) []string { return l.item.Chips(a) }
func (l *Agents) Lines(a demodata.Agent) []string { return l.item.Lines(a) }

// firstLine is s's first line.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
