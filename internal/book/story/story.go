// Package story defines the Story descriptor and explicit Registry used by
// per-component ui/<component>/story packages and consumed by the storybook
// and test suites.
package story

import (
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// Story describes one component showcase in the storybook and test suites.
type Story struct {
	// ID is <component>/<story>, e.g. "kbd/default", "button/default".
	ID string
	// Doc is a one-line description of what the story shows.
	Doc string
	// View makes a fresh component view with its own state.
	View func() templ.Component
	// Props, when non-nil, creates editable props and their bound view for
	// the book's Controls panel.
	Props func() (any, templ.Component)
	// Keys, pressed in cells from the first frame, exercise the story by
	// keyboard alone; Want is text expected in the cells output afterwards.
	Keys []string
	Want string
	// Gap, when non-empty, explains why Keys do not yet produce Want.
	Gap string
	// Consolidated marks stories belonging to a self-contained ui/<component> package.
	// Unconsolidated (legacy) stories show a red dot in the storybook sidebar.
	Consolidated bool
	// Score is the maintainer-assigned 1–5 star rating (0 = unscored, shown with a yellow dot).
	Score int
	// BookOnly marks stories shown in the storybook that are not tracked in ui/coverage.tsv.
	BookOnly bool
}

// Component returns the story's component name (the part of ID before '/').
func (s Story) Component() string {
	c, _, _ := strings.Cut(s.ID, "/")
	return c
}

// Registry collects stories registered explicitly by component story packages.
type Registry struct {
	stories []Story
}

// NewRegistry returns an empty story Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register invokes each registration function against r and returns r.
func (r *Registry) Register(fns ...func(*Registry)) *Registry {
	for _, fn := range fns {
		if fn != nil {
			fn(r)
		}
	}
	return r
}

// Add appends s to the registry, panicking if a story with the same ID exists.
func (r *Registry) Add(s Story) {
	if slices.ContainsFunc(r.stories, func(o Story) bool { return o.ID == s.ID }) {
		panic("story: duplicate story " + s.ID)
	}
	r.stories = append(r.stories, s)
}

// All returns every non-BookOnly registered story sorted by ID.
func (r *Registry) All() []Story {
	var out []Story
	for _, s := range r.stories {
		if !s.BookOnly {
			out = append(out, s)
		}
	}
	slices.SortStableFunc(out, func(a, b Story) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// Ordered returns every registered story (including BookOnly) in registration order.
func (r *Registry) Ordered() []Story {
	return slices.Clone(r.stories)
}

// Of returns all stories belonging to component, sorted by ID.
func (r *Registry) Of(component string) []Story {
	var out []Story
	for _, s := range r.All() {
		if s.Component() == component {
			out = append(out, s)
		}
	}
	return out
}
