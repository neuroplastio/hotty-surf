// Package rt is what one render registers, shared by the packages that take
// part in it: package surf makes a Frame for each render and reads it after,
// ui's components report into it, and surftest reads what went wrong. It is
// internal so that none of it is API.
package rt

import (
	"context"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
)

// Handlers are one element's, as surf.On registered them for this render.
type Handlers struct {
	Click   func()
	Input   func(string)
	Check   func(bool)
	Key     []func(string) bool
	KitKey  bool // Key is a kit component default, rather than an app handler
	Focus   func(bool)
	Outside func()
	Hover   func(bool)
	Submit  func()
	Drag    func(hotty.Event)
}

// A Frame is one render's registrations.
type Frame struct {
	// On is each element's handlers, by its id.
	On map[string]*Handlers
	// Order is the ids On registered, in the order of the render.
	Order []string
	// Focus is the element Autofocus asked for ("" for none).
	Focus string
	// Reveal names an element to scroll into view after this frame.
	Reveal string
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
	// N is the surface's render count: 1 for its first render, one more
	// for each after. A component's ask made in an event is for the frame
	// after the one whose handler ran (N+1), and expires with it.
	N uint64
	// After are run in order once the view has rendered, before the
	// frame is read: a component's last word on what the frame asks
	// (Focus), given what the whole view asked. A hook may add hooks.
	After []func()
	// Modal are the ids of the modal elements open in this render (a
	// Dialog), outermost first. The last is the active modal; the runtime
	// confines interaction and focus to it.
	Modal []string
	// ModalOps is installed by a modal component, so applications without
	// one do not link its tree preparation and focus confinement code.
	ModalOps *ModalOps
	// KeyScopes are this frame's lexical kit overrides, indexed by the
	// private marker on each focused kit control (one-based).
	KeyScopes []*KeyScope
	// Scopes are every KeyScope the render made, in the order of the
	// render, whether or not a kit control is in it: what ui.Bindings
	// lists and ui.Pick runs.
	Scopes []*KeyScope
}

// ModalOps performs the tree and focus work of an active modal.
type ModalOps struct {
	Prepare func(*html.Node, string)
	Scope   func(*html.Node, map[string]*html.Node, string) *html.Node
	Applies func(*html.Node, map[string]*html.Node, string, string) bool
	Target  func(map[string]*html.Node, string, bool) string
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
