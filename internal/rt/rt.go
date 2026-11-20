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
	// Binds are surf.Bind's bindings, canonical for the frame's
	// platform, one per key, layered as surf.Bind says (a direct Action's
	// or Unbind's over a group's, a later group's over an earlier). The runtime
	// writes each as key=action, but where Keys declares the key, or
	// one yields to a key given around it.
	Binds []Bind
	// Written are Binds as the surface resolved them on the element when
	// it last wrote its data-keys: the faults dropped, and those that
	// yield to a key given around it. It is the one resolver's output,
	// which data-keys, the surface's own use of a declined key (EditKeys)
	// and so a host and cells read alike.
	Written []Bind
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
// a field's edit action, prefix, a scroll action, or default (surf.Unbind).
// Yield says it leaves alone a key the program is given around its
// element; Direct says it was given to surf.Bind directly, not in a group,
// and holds its key over the groups'.
type Bind struct {
	Key, Action string
	Yield       bool
	Direct      bool
}

// EditKeys is h's resolved bindings (Written) as a field's keymap reads
// them (SPEC §10.2), a data-keys value, then, with program, its Key
// handlers' keys, each the program's: what a host resolves on the
// element. Without program, a key the Key handlers declare is its
// binding's, so that a declined one is what the host does with no
// declaration of it.
func EditKeys(h *Handlers, program bool) string {
	var b strings.Builder
	for _, x := range h.Written {
		if program && slices.Contains(h.Keys, x.Key) {
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
	// Reveal names an element to scroll into view after this frame.
	Reveal string
	// Platform is the host's, for a render that shows keys.
	Platform hotty.Platform
	// surface is the *surf.Surface rendering, as any (rt can't import
	// surf): Surface reads it, and Owner for the runtime itself.
	surface any
	// Faults are what a test fails on and an app gets away with: an id
	// registered twice, a value a component doesn't take.
	Faults []string
	// focused, memo, n and after are what a render reads of its surface,
	// each through its method (Focused, Memo, N, AddAfter), which marks
	// the instance rendering as one that must render again (Instance).
	focused string
	memo    map[string]any
	n       uint64
	after   []func()
	// In is the compiled instance rendering (surf.Compile), nil outside
	// one.
	In *Instance
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
	// first time an element binds surf.Readline in the frame.
	Readline []Bind
	// Replays are what the render registered on a component around it
	// (Register), in order: a kept compiled instance runs its own again.
	Replays []func(context.Context)
	// Stops are the elements whose tabindex is 0, not the -1 the kit
	// wrote: a list's roving tab stop, which the runtime writes as it
	// writes data-keys, so that an item's markup is its props' alone.
	Stops []string
	// Shadow says the render is a check's (surf.Compile): what it
	// registers goes nowhere.
	Shadow bool
}

// Register runs fn, which registers what a component reads around it
// (a list item, with its list), and notes it in the frame: a compiled
// instance kept is not rendered, and runs it again where it renders,
// with the context there.
func Register(ctx context.Context, fn func(context.Context)) {
	f := From(ctx)
	if f == nil {
		fn(ctx)
		return
	}
	if !f.Shadow {
		fn(ctx)
	}
	f.Replays = append(f.Replays, fn)
}

// ModalOps performs the tree and focus work of an active modal.
type ModalOps struct {
	Prepare func(*html.Node, string)
	Scope   func(*html.Node, map[string]*html.Node, string) *html.Node
	Applies func(*html.Node, map[string]*html.Node, string, string) bool
	Target  func(map[string]*html.Node, string, bool) string
}

// NewFrame is a render's frame: of the surface s (a *surf.Surface), its
// platform, the element it has focused, its memo, the render's count,
// its secret for the kit's markers, and how many ids it may register.
func NewFrame(s any, p hotty.Platform, focused string, memo map[string]any, n uint64, keyMark string, ids int) *Frame {
	return &Frame{On: make(map[string]*Handlers, ids), Platform: p, surface: s, focused: focused, memo: memo, n: n, KeyMark: keyMark}
}

// Surface is the *surf.Surface rendering, as any (rt can't import surf):
// what surf.Of gives a view. A component reading it renders every time.
func (f *Frame) Surface() any {
	f.Read("surf.Of")
	return f.surface
}

// Owner is the *surf.Surface rendering, for the runtime's own use: a read
// that marks nothing.
func (f *Frame) Owner() any { return f.surface }

// Focused is the element the keyboard is on as the surface knows it when
// the render starts ("" for none): where a component's keys come from,
// and where a dialog gives the keyboard back to.
func (f *Frame) Focused() string {
	f.Read("the focused element")
	return f.focused
}

// Memo is the surface's, the same map from render to render: what a
// component that has no State remembers, by a key it makes from its id
// (a Dialog: that it is open, and where the keyboard was).
func (f *Frame) Memo() map[string]any {
	f.Read("the surface's memo")
	return f.memo
}

// N is the surface's render count: 1 for its first render, one more for
// each after. A component's ask made in an event is for the frame after
// the one whose handler ran (N+1), and expires with it.
func (f *Frame) N() uint64 {
	f.Read("the render count")
	return f.n
}

// AddAfter adds fn to what runs, in order, once the view has rendered,
// before the frame is read: a component's last word on what the frame
// asks (Focus), given what the whole view asked. A hook may add hooks.
// It closes over this frame, so a compiled instance adding one renders
// every time.
func (f *Frame) AddAfter(fn func()) {
	f.Read("an After hook")
	f.after = append(f.after, fn)
}

// RunAfter runs the hooks AddAfter added, the ones they add too.
func (f *Frame) RunAfter() {
	for i := 0; i < len(f.after); i++ {
		f.after[i]()
	}
}

// An Instance is a compiled component's instance as it renders
// (surf.Compile). Volatile says why it must render every time, "" for
// an instance whose output is its props' alone; Up is the instance it
// renders in. Deps are the context values it read (Read), each kept only
// while the context gives it the same.
type Instance struct {
	Volatile string
	Up       *Instance
	Deps     []Dep
}

// A Dep is a context value an instance read: its key, and the value.
type Dep struct{ Key, Val any }

// Read marks the instance rendering, and every one it renders in, as one
// that must render every time, for why: it read what isn't its props.
func (f *Frame) Read(why string) {
	if f == nil {
		return
	}
	for in := f.In; in != nil && in.Volatile == ""; in = in.Up {
		in.Volatile = why
	}
}

// Volatile is Read, of ctx's frame.
func Volatile(ctx context.Context, why string) {
	From(ctx).Read(why)
}

// Value is ctx.Value(k), noted on the instance rendering and every one it
// renders in: one is kept only while its context gives the same value
// (==, so v must be comparable).
func Value(ctx context.Context, k any) any {
	v := ctx.Value(k)
	if f := From(ctx); f != nil {
		for in := f.In; in != nil; in = in.Up {
			in.Deps = append(in.Deps, Dep{Key: k, Val: v})
		}
	}
	return v
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
