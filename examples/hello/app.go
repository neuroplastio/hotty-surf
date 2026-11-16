package main

//go:generate msgp -marshal=false -tests=false
//msgp:newtime

import (
	"neuroplast.io/hotty-surf/ui"
)

// State is what the app keeps across a live reload.
type State struct {
	Name     string `msg:"name"`
	Greeting string `msg:"greeting"`
	Waves    int    `msg:"waves"`
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State

	greetingMenu ui.SelectState
	toasts       ui.ToasterState
}

var greetings = []ui.SelectItem{
	{Value: "Hello", Label: "Hello"},
	{Value: "Hi", Label: "Hi"},
	{Value: "Good morning", Label: "Good morning"},
}

func NewApp() *App { return &App{State: State{Greeting: "Hello"}} }

func (a *App) Save() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

func (a *App) SetName(v string)     { a.Name = v }
func (a *App) SetGreeting(v string) { a.Greeting = v }

func (a *App) Wave() {
	a.Waves++
	a.toasts.Show(ui.Toast{Title: a.Greeting + ", " + a.who() + "!"})
}

// Shortcut is the page's keys: mod+enter (ctrl, or ⌘ on macOS) waves from
// anywhere.
func (a *App) Shortcut(key string) bool {
	if key == "mod+enter" {
		a.Wave()
		return true
	}
	return false
}

func (a *App) who() string {
	if a.Name == "" {
		return "world"
	}
	return a.Name
}
