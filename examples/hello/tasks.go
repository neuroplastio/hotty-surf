package main

import (
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-surf/surfrun"
)

// Tasks is the guide's second screen (tasks.surf): a list of tasks with
// a filter, a selection the arrows move, a field that adds a task, and a
// panel that keeps its own state (Help).
type Tasks struct {
	Items  []*Task
	Sel    string // the selected task's id
	Filter string // "all" or "open"
	Draft  string // what the field holds
	Help   *HelpPanel

	made int            // unexported: no template sees it
	rt   surfrun.Runner // the run, to end it
}

// Task is a data struct: a template reads its exported fields.
type Task struct {
	ID   string
	Text string
	Done bool
}

// NewTasks is a list of tasks with these texts, the first selected.
func NewTasks(texts ...string) *Tasks {
	s := &Tasks{Filter: "all", Help: &HelpPanel{}}
	for _, t := range texts {
		s.add(t)
	}
	if len(s.Items) > 0 {
		s.Sel = s.Items[0].ID
	}
	return s
}

func (s *Tasks) add(text string) *Task {
	s.made++
	t := &Task{ID: "t" + strconv.Itoa(s.made), Text: text}
	s.Items = append(s.Items, t)
	return t
}

// Shown are the tasks the filter lets through.
func (s *Tasks) Shown() []*Task {
	var out []*Task
	for _, t := range s.Items {
		if s.Filter != "open" || !t.Done {
			out = append(out, t)
		}
	}
	return out
}

// Left is how many tasks are not done.
func (s *Tasks) Left() int {
	n := 0
	for _, t := range s.Items {
		if !t.Done {
			n++
		}
	}
	return n
}

// Filters are the filter's options.
func (s *Tasks) Filters() []string { return []string{"all", "open"} }

// Select selects a task: a row's click.
func (s *Tasks) Select(id string) { s.Sel = id }

// Toggle marks a task done, or not done: a row's switch.
func (s *Tasks) Toggle(t *Task) { t.Done = !t.Done }

// ToggleSel toggles the selected task: the list's enter.
func (s *Tasks) ToggleSel() {
	for _, t := range s.Items {
		if t.ID == s.Sel {
			t.Done = !t.Done
		}
	}
}

// Next and Prev move the selection through the shown tasks.
func (s *Tasks) Next() { s.move(1) }
func (s *Tasks) Prev() { s.move(-1) }

func (s *Tasks) move(by int) {
	shown := s.Shown()
	for i, t := range shown {
		if t.ID == s.Sel {
			s.Sel = shown[max(0, min(len(shown)-1, i+by))].ID
			return
		}
	}
	if len(shown) > 0 {
		s.Sel = shown[0].ID
	}
}

// SetFilter is the select's change: f is the option picked.
func (s *Tasks) SetFilter(f string) { s.Filter = f }

// SetDraft is the field's input: text is what it holds.
func (s *Tasks) SetDraft(text string) { s.Draft = text }

// AddDraft adds the field's text as a task, selects it, and empties the
// field: the field's enter.
func (s *Tasks) AddDraft() {
	if strings.TrimSpace(s.Draft) == "" {
		return
	}
	s.Sel = s.add(strings.TrimSpace(s.Draft)).ID
	s.Draft = ""
}

// ClearDone takes the tasks that are done away.
func (s *Tasks) ClearDone() {
	var keep []*Task
	for _, t := range s.Items {
		if !t.Done {
			keep = append(keep, t)
		}
	}
	s.Items = keep
}

// Attach is the runtime's (surfrun.Attacher): Tasks keeps the run, to end
// it. It hears nil when the run ends.
func (s *Tasks) Attach(r surfrun.Runner) { s.rt = r }

// Quit ends the run: escape.
func (s *Tasks) Quit() {
	if s.rt != nil {
		s.rt.Quit()
	}
}

// HelpPanel is a stateful child: a field of Tasks, with a template of its
// own (help-panel.surf), its own state and its own handlers.
type HelpPanel struct {
	Open bool
}

// Flip shows or hides the keys.
func (p *HelpPanel) Flip() { p.Open = !p.Open }

// String is the list as text, for a log (fmt.Stringer). No template
// calls it: string is CEL's own function, so surfgen makes it a handler
// only, and says so in surf_gen.go.
func (s *Tasks) String() string {
	return strconv.Itoa(len(s.Items)) + " tasks, " +
		strconv.Itoa(s.Left()) + " left"
}
