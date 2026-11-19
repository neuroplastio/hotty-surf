// Package rt is what one render registers, shared by the packages that take
// part in it: package surf makes a Frame for each render and reads it after,
// ui's components report into it, and surftest reads what went wrong. It is
// internal so that none of it is API.
package rt

import (
	"context"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
)

// Handlers are one element's, as surf.On registered them for this render.
type Handlers struct {
	Click func()
	Input func(string)
	Check func(bool)
	Key   []func(string) bool
	// Keys are the keys the Key handlers declare, canonical for the
	// frame's platform, in the order declared: what the runtime writes
	// as the element's data-keys, each bound to program.
	Keys   []string
	KitKey bool // Key is a kit component default, rather than an app handler
	// Binds are surf.Bind's and surf.Readline's bindings, canonical for
	// the frame's platform: Readline's first, then the Binds, a later
	// one of a key replacing an earlier (the first stays where two Binds
	// disagree, a fault). The runtime writes each as key=action, but
	// where Keys declares the key.
	Binds []Bind
	// Scope is the ui.KeyScope On ran in, nil for none: the keyboard on
	// the element, or inside it, is inside the scope.
	Scope   *KeyScope
	Focus   func(bool)
	Outside func()
	Hover   func(bool)
	Submit  func()
	// Drag and Drop are surf.Drag's and surf.Drop's, as any: the event's
	// type is surf's (rt can't import surf). Secondary is surf.Secondary's.
	Drag, Drop any
	Secondary  func(hotty.Cell, hotty.Area)
}

// A Bind is one key bound to an action that isn't program (SPEC §10.2):
// a field's edit action, prefix, or a scroll action. Readline says it
// came from surf.Readline's set, which a Bind on the element overrides.
type Bind struct {
	Key, Action string
	Readline    bool
}

// EditKeys is h's bindings as a field's keymap reads them (SPEC §10.2),
// a data-keys value, then, with program, its Key handlers' keys, each
// the program's: what a host resolves on the element. Without program, a
// key the Key handlers declare is left out, so that a declined one is
// the binding's farther in, or the default's. A Readline binding of a
// key in given (the keys the chain gives the program) is left out, as
// the surface leaves it out of data-keys.
func EditKeys(h *Handlers, program bool, given []string) string {
	var b strings.Builder
	for _, x := range h.Binds {
		if program && slices.Contains(h.Keys, x.Key) || x.Readline && slices.Contains(given, x.Key) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(x.Key)
		b.WriteByte('=')
		b.WriteString(x.Action)
	}
	if p := ProgramKeys(h.Keys); program && p != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p)
	}
	return b.String()
}

// Given is the keys the Key handlers of n and the elements around it
// declare, and those they bind to prefix: the keys a host gives the
// program there, which Readline's bindings leave alone.
func Given(n *html.Node, on map[string]*Handlers) []string {
	var out []string
	for p := n; p != nil; p = p.Parent {
		if h := on[attr(p, "id")]; h != nil {
			out = append(out, h.Keys...)
			for _, b := range h.Binds {
				if b.Action == string(hotty.Prefix) {
					out = append(out, b.Key)
				}
			}
		}
	}
	return out
}

// A Frame is one render's registrations.
type Frame struct {
	// On is each element's handlers, by its id.
	On map[string]*Handlers
	// Order is the ids On registered, in the order of the render.
	Order []string
	// Focus is the element Autofocus or Refocus asked for ("" for none).
	Focus string
	// Refocus says Focus moves the keyboard though the frame before
	// asked for it too (surf.Refocus).
	Refocus bool
	// Count is the count of the click whose handler is running (SPEC §9),
	// for surf.Clicks; 0 outside one.
	Count int
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
	// Popups are the ids of the kit's popups open when this render ends
	// (a DropdownMenu, ContextMenu, Select, Combobox or Popover): at most
	// one, as one opening closes another.
	Popups []string
	// ModalOps is installed by a modal component, so applications without
	// one do not link its tree preparation and focus confinement code.
	ModalOps *ModalOps
	// KeyScopes are this frame's lexical kit overrides, indexed by the
	// private marker on each focused kit control (one-based), and
	// KeyKinds the kind of each such control ("button", "text", …).
	KeyScopes []*KeyScope
	KeyKinds  []string
	// KeyMark starts every such marker: the surface's secret, which a
	// template can't know, so a marker it writes is told from the kit's.
	KeyMark string
	// Scopes are every KeyScope the render made, in the order of the
	// render, whether or not a kit control is in it: what ui.Bindings
	// lists and ui.Pick runs.
	Scopes []*KeyScope
	// Readline is hotty.TerminalKeys as Binds for Platform, made the
	// first time an element binds it in the frame (surf.Readline).
	Readline []Bind
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
