// Package rt is what one render registers, shared by the packages that take
// part in it: package surf makes a Frame for each render and reads it after,
// ui's components report into it, and surftest reads what went wrong. It is
// internal so that none of it is API.
package rt

import (
	"context"

	"neuroplast.io/hotty-go"
)

// Handlers are one element's, as surf.On registered them for this render.
type Handlers struct {
	Click   func()
	Input   func(string)
	Check   func(bool)
	Key     []func(string) bool
	Focus   func(bool)
	Outside func()
	Hover   func(bool)
}

// A Frame is one render's registrations.
type Frame struct {
	// On is each element's handlers, by its id.
	On map[string]*Handlers
	// Order is the ids On registered, in the order of the render.
	Order []string
	// Focus is the element Autofocus asked for ("" for none).
	Focus string
	// Platform is the host's, for a render that shows keys.
	Platform hotty.Platform
	// Surface is the *surf.Surface rendering, as any (rt can't import surf).
	Surface any
	// Faults are what a test fails on and an app gets away with: an id
	// registered twice, a value a component doesn't take.
	Faults []string
	// Focused is the element the keyboard is on as the surface knows it
	// when the render starts ("" for none): where a component's keys come
	// from, and where a dialog gives the keyboard back to.
	Focused string
	// Memo is the surface's, the same map from render to render: what a
	// component that has no State remembers, by a key it makes from its
	// id (a Dialog: that it is open, and where the keyboard was).
	Memo map[string]any
}

type key struct{}

// With is ctx carrying f.
func With(ctx context.Context, f *Frame) context.Context {
	return context.WithValue(ctx, key{}, f)
}

// From is the frame ctx carries, nil outside a render.
func From(ctx context.Context) *Frame {
	f, _ := ctx.Value(key{}).(*Frame)
	return f
}

// Fault records what a test should fail on; outside a render it is
// dropped.
func Fault(ctx context.Context, msg string) {
	if f := From(ctx); f != nil {
		f.Faults = append(f.Faults, msg)
	}
}
