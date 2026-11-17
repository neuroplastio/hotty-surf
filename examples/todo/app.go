package main

import (
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/ui"
)

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App

// Task is one line of the list.
type Task struct {
	ID    int    `msg:"id"`
	Title string `msg:"title"`
	Done  bool   `msg:"done"`
}

// State is what the app keeps across a live reload (hotty-surf run).
type State struct {
	Tasks  []Task `msg:"tasks"`
	NextID int    `msg:"next_id"`
	Filter string `msg:"filter"` // "all", "active" or "done"
	Draft  string `msg:"draft"`  // the new task's field, as typed
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State
}

// Tabs is the filters, each with how many tasks it shows.
func (a *App) Tabs() []ui.Tab {
	return []ui.Tab{
		{Value: "all", Label: "All " + strconv.Itoa(len(a.Tasks))},
		{Value: "active", Label: "Active " + strconv.Itoa(a.Left())},
		{Value: "done", Label: "Done " + strconv.Itoa(a.DoneCount())},
	}
}

func NewApp() *App {
	a := &App{State: State{Filter: "all"}}
	for i, title := range []string{
		"Water the plants",
		"Book the dentist",
		"Pay the electricity bill",
		"Return the library books",
		"Call Grandma",
		"Fix the squeaky door",
	} {
		a.Draft = title
		a.Add()
		a.Tasks[i].Done = i == 1 || i == 3
	}
	return a
}

func (a *App) Save() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

func (a *App) SetDraft(v string)  { a.Draft = v }
func (a *App) SetFilter(v string) { a.Filter = v }

// Add adds the field's text as a task, and clears the field: the form's
// Submit, so Enter in the field adds as the button does.
func (a *App) Add() {
	title := strings.TrimSpace(a.Draft)
	if title == "" {
		return
	}
	a.NextID++
	a.Tasks = append(a.Tasks, Task{ID: a.NextID, Title: title})
	a.Draft = ""
}

func (a *App) SetDone(id int, done bool) {
	for i := range a.Tasks {
		if a.Tasks[i].ID == id {
			a.Tasks[i].Done = done
		}
	}
}

func (a *App) Delete(id int) {
	a.Tasks = keep(a.Tasks, func(t Task) bool { return t.ID != id })
}

// ClearDone deletes every done task.
func (a *App) ClearDone() {
	a.Tasks = keep(a.Tasks, func(t Task) bool { return !t.Done })
}

// Shown is the tasks the filter lets through.
func (a *App) Shown() []Task {
	return keep(a.Tasks, func(t Task) bool {
		return a.Filter == "all" || t.Done == (a.Filter == "done")
	})
}

func (a *App) Left() int { return len(a.Tasks) - a.DoneCount() }

func (a *App) DoneCount() int {
	return len(keep(a.Tasks, func(t Task) bool { return t.Done }))
}

// LeftText is "1 task left", "3 tasks left".
func (a *App) LeftText() string {
	if n := a.Left(); n != 1 {
		return strconv.Itoa(n) + " tasks left"
	}
	return "1 task left"
}

func keep(ts []Task, ok func(Task) bool) []Task {
	var out []Task
	for _, t := range ts {
		if ok(t) {
			out = append(out, t)
		}
	}
	return out
}

// done is the class a done task's title takes.
func done(t Task) string {
	if t.Done {
		return "done"
	}
	return ""
}

func labelled(a templ.Attributes, label string) templ.Attributes {
	a["aria-label"] = label
	return a
}
