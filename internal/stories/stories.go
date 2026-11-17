// Package stories is ui's stories: each component in one state, with the
// little state a story keeps of its own. ui's tests render them (their
// looks in hotty-blitz and Chromium, and by keyboard in cells), the
// coverage test holds them to ui/coverage.tsv, and the book (L3) shows
// them.
//
// A story's view writes what its controls hold as text (a <p
// class="text-muted-foreground"> line), so a test in cells can read that
// the keys did what they should.
package stories

import (
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// A Story is one component in one state.
type Story struct {
	// ID is <component>/<story>, the component as Basecoat's file names it
	// (ui/coverage.tsv): "button/default", "dropdown-menu/end-aligned".
	ID string
	// Doc is a line about what it shows.
	Doc string
	// View makes a new view, with state of its own.
	View func() templ.Component
	// Keys, pressed in cells from the first frame (Tab first, since
	// nothing has the keyboard), use the story by keyboard alone; Want is
	// text the cells' lines show afterwards. A story of a component that
	// takes no keys has neither.
	Keys []string
	Want string
	// Gap, when not "", is why Keys don't do what Want says yet: a limit
	// of hotty's protocol or of a host, written as a proposal or a gap
	// (never worked around in the kit). The test skips with it.
	Gap string
}

// Component is the story's component: its ID before the slash.
func (s Story) Component() string {
	c, _, _ := strings.Cut(s.ID, "/")
	return c
}

var all []Story

func add(s Story) {
	if slices.ContainsFunc(all, func(o Story) bool { return o.ID == s.ID }) {
		panic("stories: two stories " + s.ID)
	}
	all = append(all, s)
}

// All is every story, by ID.
func All() []Story {
	out := slices.Clone(all)
	slices.SortStableFunc(out, func(a, b Story) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// Of is the stories of a component, by ID.
func Of(component string) []Story {
	var out []Story
	for _, s := range All() {
		if s.Component() == component {
			out = append(out, s)
		}
	}
	return out
}

func onoff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
