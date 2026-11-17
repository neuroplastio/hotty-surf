package main

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App

import (
	"strconv"
	"strings"
	"time"

	"neuroplast.io/hotty-surf/ui"
)

// The levels, lowest first. A line's Level is its index here.
var levels = []string{"debug", "info", "warn", "error"}

var levelItems = []ui.SelectItem{
	{Value: "debug", Label: "Debug and up"},
	{Value: "info", Label: "Info and up"},
	{Value: "warn", Label: "Warn and up"},
	{Value: "error", Label: "Errors only"},
}

// keep is how many lines the app holds; older ones drop off the top.
const keep = 2000

// window is how many lines the view draws at most, the newest of them at
// its bottom. The list clips those its height has no room for.
const window = 120

// Line is one line of the service's log.
type Line struct {
	Seq   int       `msg:"seq"`
	Time  time.Time `msg:"time"`
	Level int       `msg:"level"`
	Text  string    `msg:"text"`
}

// State is what the app keeps across a live reload.
type State struct {
	Lines  []Line `msg:"lines"`
	Next   int    `msg:"next"`   // the next line's Seq
	Min    int    `msg:"min"`    // the lowest level shown
	Query  string `msg:"query"`  // the search, as typed
	Follow bool   `msg:"follow"` // the view keeps to the newest line
	Paused bool   `msg:"paused"` // new lines are dropped
	// Anchor is the Seq of the line at the view's bottom while it doesn't
	// follow.
	Anchor int `msg:"anchor"`
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State

	levelMenu ui.SelectState
}

func NewApp() *App { return &App{State: State{Min: 1, Follow: true}} }

func (a *App) Save() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

// Add takes a line from the service, unless the view is paused.
func (a *App) Add(l Line) {
	if a.Paused {
		return
	}
	l.Seq = a.Next
	a.Next++
	a.Lines = append(a.Lines, l)
	if n := len(a.Lines) - keep; n > 0 {
		a.Lines = append(a.Lines[:0], a.Lines[n:]...)
	}
}

func (a *App) SetQuery(q string) { a.Query = q }

func (a *App) SetMin(v string) {
	for i, l := range levels {
		if l == v {
			a.Min = i
		}
	}
}

func (a *App) SetFollow(on bool) {
	if !on && a.Follow {
		a.Anchor = a.Next - 1
	}
	a.Follow = on
}

func (a *App) TogglePause() { a.Paused = !a.Paused }

func (a *App) Clear() {
	a.Lines = nil
	a.Anchor = a.Next - 1
}

func (a *App) Newest() { a.SetFollow(true) }

// Keys scroll the list: up and Page Up go back and stop following, down
// and Page Down come forward, End (or coming down to the newest line)
// follows again.
func (a *App) Keys(key string) bool {
	switch key {
	case "arrow_up":
		a.scroll(-1)
	case "page_up":
		a.scroll(-10)
	case "arrow_down":
		a.scroll(1)
	case "page_down":
		a.scroll(10)
	case "home":
		a.scroll(-keep)
	case "end":
		a.SetFollow(true)
	default:
		return false
	}
	return true
}

func (a *App) scroll(by int) {
	m := a.matching()
	if len(m) == 0 {
		return
	}
	end := len(m) - 1
	if !a.Follow {
		end = lastAtOrBefore(m, a.Anchor)
	}
	end += by
	if end >= len(m)-1 {
		a.Follow = true
		return
	}
	a.Follow = false
	a.Anchor = m[max(end, 0)].Seq
}

// matching is the lines the level and the search let through.
func (a *App) matching() []Line {
	q := strings.ToLower(strings.TrimSpace(a.Query))
	var out []Line
	for _, l := range a.Lines {
		if l.Level >= a.Min && (q == "" || strings.Contains(strings.ToLower(l.Text), q)) {
			out = append(out, l)
		}
	}
	return out
}

// Shown is what the view draws, newest first, and how many matching lines
// came after its newest (none while following).
func (a *App) Shown() (lines []Line, newer int) {
	m := a.matching()
	end := len(m)
	if !a.Follow {
		end = lastAtOrBefore(m, a.Anchor) + 1
	}
	for i := end - 1; i >= max(end-window, 0); i-- {
		lines = append(lines, m[i])
	}
	return lines, len(m) - end
}

// lastAtOrBefore is the index of the last line in ls up to Seq seq, -1
// for none.
func lastAtOrBefore(ls []Line, seq int) int {
	i := len(ls) - 1
	for i >= 0 && ls[i].Seq > seq {
		i--
	}
	return i
}

func (a *App) errors() int {
	n := 0
	for _, l := range a.Lines {
		if l.Level == 3 {
			n++
		}
	}
	return n
}

// part is a piece of a line's text, Match where the search found it.
type part struct {
	Text  string
	Match bool
}

// parts splits text around the search's matches, ignoring case.
func (a *App) parts(text string) []part {
	q := strings.ToLower(strings.TrimSpace(a.Query))
	if q == "" {
		return []part{{Text: text}}
	}
	var out []part
	low := strings.ToLower(text)
	for {
		i := strings.Index(low, q)
		if i < 0 {
			break
		}
		if i > 0 {
			out = append(out, part{Text: text[:i]})
		}
		out = append(out, part{Text: text[i : i+len(q)], Match: true})
		text, low = text[i+len(q):], low[i+len(q):]
	}
	if text != "" {
		out = append(out, part{Text: text})
	}
	return out
}

func levelVariant(l int) ui.Variant {
	switch l {
	case 0:
		return ui.Ghost
	case 1:
		return ui.Secondary
	case 2:
		return ui.Outline
	}
	return ui.Destructive
}

func plural(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return strconv.Itoa(n) + " " + what + "s"
}
