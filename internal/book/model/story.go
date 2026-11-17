// Package storybook is a storybook for Go + templ components on hotty: the
// stories (a component in one state), and the book (which story is open,
// the filter, the tab, what each story did). It says nothing about how the
// book is shown: storybook/htmlbook shows it as a hotty surface, and a
// cells rendition would go beside it (README.md).
//
// A story is a few lines, next to the components or in a stories package:
//
//	storybook.Add("Basecoat", "Badge / secondary", Badge{Label: "Beta", Variant: "secondary"}, badge)
//
// where badge is a templ component (or a Go func) that takes *Badge and an
// *Env, and Badge's exported fields are the props the book shows and lets
// the user change.
package storybook

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/a-h/templ"
)

// Story is a component in one state: a name, a group, the props (a Go
// value the render reads and the component's handlers change), and the
// render.
type Story struct {
	Group string
	Name  string
	Doc   string // one line, shown beside the name

	id    string
	props any                        // *P
	reset func()                     // sets *props back to the value Add was given
	html  func(*Env) templ.Component // the HTML rendition
	// A cells rendition would add its render here, beside html.

	file string  // where Add was called: the story's own code
	line int     //
	fn   uintptr // the render, for its source
}

// registry is every story Add made, in order.
var registry []*Story

// Wrap adapts an existing v0.7 story to the book's model. Its view is
// made once per selection and shared by the preview and the cells pane.
func Wrap(id, doc string, makeView func() templ.Component, makeProps func() (any, templ.Component)) *Story {
	group, name, _ := strings.Cut(id, "/")
	family := "Kit · native/ui"
	switch group {
	case "alert", "avatar", "badge", "breadcrumb", "button", "card", "empty", "field", "input", "input-group", "item", "kbd", "label", "native-select", "progress", "scrollbar", "skeleton", "table", "textarea":
		family = "Basecoat · CSS"
	}
	title := titlePart(group)
	if name != "" && name != "default" {
		title += " / " + strings.ReplaceAll(name, "-", " ")
	}
	s := &Story{Group: family, Name: title, Doc: doc, id: id}
	var view templ.Component
	s.reset = func() {
		if makeProps != nil {
			s.props, view = makeProps()
		} else {
			view = makeView()
		}
	}
	s.reset()
	s.html = func(*Env) templ.Component { return view }
	return s
}

func titlePart(s string) string {
	r := []rune(strings.ReplaceAll(s, "-", " "))
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// Add makes a story and registers it: render shows props, and gets a
// pointer to the story's own copy, so the handlers it gives the component
// change what the next render shows. It returns the story, for Describe.
func Add[P any](group, name string, props P, render func(p *P, e *Env) templ.Component) *Story {
	_, file, line, _ := runtime.Caller(1)
	p := new(P)
	*p = props
	s := &Story{Group: group, Name: name, id: slug(group + "-" + name), props: p,
		file: file, line: line, fn: reflect.ValueOf(render).Pointer()}
	s.reset = func() { *p = props }
	s.html = func(e *Env) templ.Component { return render(p, e) }
	registry = append(registry, s)
	return s
}

// Describe sets the story's line of description.
func (s *Story) Describe(doc string) *Story { s.Doc = doc; return s }

// All are the stories Add made, in the order it made them.
func All() []*Story { return append([]*Story(nil), registry...) }

// ID is the story's name as an identifier: its group and name, in lower
// case, with dashes.
func (s *Story) ID() string { return s.id }

// Props is the story's props now: a pointer to its value.
func (s *Story) Props() any { return s.props }

// Reset puts the props back as Add had them.
func (s *Story) Reset() { s.reset() }

// HTML is the story rendered as HTML, with e for its events.
func (s *Story) HTML(e *Env) templ.Component { return s.html(e) }

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// Event is one thing a story's component did, as the Events tab lists it.
type Event struct {
	At   time.Time
	Kind string // "act": a handler the story named ran; "state": the props changed; "focus"
	Text string
}

// Env is what a story's render is given: helpers that log what the
// component did, to hand it as its handlers.
type Env struct {
	log func(kind, text string)
}

// Act is a handler that only logs name: for a component that wants a
// func() and does nothing else with it in the story.
func (e *Env) Act(name string) func() { return func() { e.Log("act", name) } }

// Do logs name, then runs fn.
func (e *Env) Do(name string, fn func()) func() {
	return func() { e.Log("act", name); fn() }
}

// Bool logs name and the value, then runs fn (if any).
func (e *Env) Bool(name string, fn func(bool)) func(bool) {
	return func(v bool) {
		e.Log("act", fmt.Sprintf("%s(%v)", name, v))
		if fn != nil {
			fn(v)
		}
	}
}

// Text logs name and the value, then runs fn (if any).
func (e *Env) Text(name string, fn func(string)) func(string) {
	return func(v string) {
		e.Log("act", fmt.Sprintf("%s(%q)", name, v))
		if fn != nil {
			fn(v)
		}
	}
}

// Log adds an event of its own to the story's.
func (e *Env) Log(kind, text string) {
	if e != nil && e.log != nil {
		e.log(kind, text)
	}
}

// Slug is s as an identifier: lower case, dashes between words.
func Slug(s string) string { return slug(s) }
