package main

import (
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/ui"
)

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App

// Settings are the values the form edits.
type Settings struct {
	Name    string `msg:"name"`
	Email   string `msg:"email"`
	Bio     string `msg:"bio"`
	EmailMe bool   `msg:"email_me"`
	Push    bool   `msg:"push"`
	Digest  string `msg:"digest"` // daily, weekly or never
	Theme   string `msg:"theme"`  // light, dark or system
	Compact bool   `msg:"compact"`
}

// State is what the app keeps across a live reload (hotty-surf run).
type State struct {
	Form  Settings `msg:"form"`  // as the user has it
	Saved Settings `msg:"saved"` // as it was saved last
	Tried bool     `msg:"tried"` // a save failed: the errors show
}

// App is the app's state and its handlers. The view (page.templ) reads it.
type App struct {
	State
	toasts ui.ToasterState
}

var digests = []ui.SelectItem{
	{Value: "daily", Label: "Daily"},
	{Value: "weekly", Label: "Weekly"},
	{Value: "never", Label: "Never"},
}

var themes = []ui.SelectItem{
	{Value: "light", Label: "Light"},
	{Value: "dark", Label: "Dark"},
	{Value: "system", Label: "System"},
}

func NewApp() *App {
	s := Settings{
		Name:    "Robin Vale",
		Email:   "robin@example.com",
		Bio:     "Maps rivers on weekends.",
		EmailMe: true,
		Digest:  "weekly",
		Theme:   "system",
		Compact: true,
	}
	return &App{State: State{Form: s, Saved: s}}
}

func (a *App) Keep() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

func (a *App) SetName(v string)   { a.Form.Name = v }
func (a *App) SetEmail(v string)  { a.Form.Email = v }
func (a *App) SetBio(v string)    { a.Form.Bio = v }
func (a *App) SetEmailMe(on bool) { a.Form.EmailMe = on }
func (a *App) SetPush(on bool)    { a.Form.Push = on }
func (a *App) SetDigest(v string) { a.Form.Digest = v }
func (a *App) SetCompact(on bool) { a.Form.Compact = on }
func (a *App) SetTheme(v string) func(bool) {
	return func(on bool) {
		if on {
			a.Form.Theme = v
		}
	}
}

// NameErr and EmailErr are the fields' errors, shown once a save has
// failed, and from then on as the user fixes them.
func (a *App) NameErr() string {
	if a.Tried && strings.TrimSpace(a.Form.Name) == "" {
		return "Enter your name."
	}
	return ""
}

func (a *App) EmailErr() string {
	if a.Tried && !validEmail(a.Form.Email) {
		return "Enter an email like name@example.com."
	}
	return ""
}

// Save saves the form, unless a field is wrong: then it shows the errors
// and keeps what was saved.
func (a *App) Save() {
	a.Tried = true
	if n := a.errors(); n > 0 {
		a.toasts.Show(ui.Toast{
			Category:    "error",
			Title:       "Not saved",
			Description: "Fix the " + plural(n, "marked field") + " first.",
		})
		return
	}
	a.Tried = false
	a.Saved = a.Form
	a.toasts.Show(ui.Toast{Title: "Settings saved"})
}

// Reset puts back what was saved last.
func (a *App) Reset() {
	a.Form = a.Saved
	a.Tried = false
}

// Shortcut is the page's keys: mod+s saves from anywhere.
func (a *App) Shortcut(key string) bool {
	if key == "mod+s" {
		a.Save()
		return true
	}
	return false
}

func (a *App) Changed() bool { return a.Form != a.Saved }

func (a *App) errors() int {
	n := 0
	for _, e := range []string{a.NameErr(), a.EmailErr()} {
		if e != "" {
			n++
		}
	}
	return n
}

// validEmail is a light check: something@something.something, no spaces.
func validEmail(s string) bool {
	at := strings.LastIndexByte(s, '@')
	if at < 1 || strings.ContainsAny(s, " \t\n") {
		return false
	}
	dot := strings.LastIndexByte(s[at+1:], '.')
	return dot > 0 && dot < len(s[at+1:])-1
}

func plural(n int, w string) string {
	if n == 1 {
		return "1 " + w
	}
	return strconv.Itoa(n) + " " + w + "s"
}
