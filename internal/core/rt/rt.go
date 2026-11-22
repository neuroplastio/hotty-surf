// Package rt is what one render registers, shared by the packages that take
// part in it: package surf makes a Frame for each render and reads it after,
// ui's components report into it, and surftest reads what went wrong. It is
// internal so that none of it is API.
package rt

import (
	"context"
	"slices"
	"strconv"
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
	// Flat are surf.Bind's bindings as its options read, in order, each
	// canonical for the frame's platform: the resolver's input (keysig).
	// The runtime lays them out as surf.Bind says where it writes the
	// element's data-keys, under the edit shortcuts in effect there: one
	// per key, a direct Action's or Unbind's over a group's, a later
	// group's over an earlier; the edit-shortcut groups' written on each
	// text field inside it (and on it, if it is one), never on another
	// element; an Unbind going back to the bindings surf.CUAKeys,
	// surf.CocoaKeys and the edit-shortcut groups read, whether or not
	// another binding holds the key on the element.
	Flat []Bind
	// Written are Flat's bindings as the surface resolved them on the
	// element when it last wrote its data-keys: the faults dropped, and
	// those that yield to a key given around it. It is the one resolver's
	// output, which data-keys, the surface's own use of a declined key
	// (EditKeys) and so a host and cells read alike.
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

// A Bind is one key bound to one of SPEC §10.2's actions: a field's edit
// action, prefix, a scroll action, default (surf.Unbind), or, from
// surf.CUAKeys and surf.CocoaKeys alone, program. Yield says it leaves alone a
// key the program is given around its element; Direct says it was given
// to surf.Bind directly, not in a group, and holds its key over the
// groups'. Base says CUAKeys or CocoaKeys made it (or an Unbind resolved it to
// their binding): bound to program it is a give-back, a key the field
// doesn't use, which isn't a key given to the program around the
// element, so a nearer binding still takes it. Edit says
// surf.CUAEditShortcuts or surf.CocoaEditShortcuts made it (Base too):
// it is written on the text fields inside its element, never on another
// element. Held says the host's edit shortcuts hold its key, so CUAKeys'
// or CocoaKeys' binding of it holds only where an edit-shortcut group
// is in effect.
type Bind struct {
	Key, Action string
	Yield       bool
	Direct      bool
	Base        bool
	Edit        bool
	Held        bool
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
	// on is each element's handlers, by its id's slot in gen, the
	// generation of slots the frame was made in (Get and Set reach
	// them): an array, so that a kept compiled instance puts its handlers
	// back by the slots it noted, with no hashing.
	on    []*Handlers
	slots *Slots
	gen   *slotGen
	count int
	autoN int
	// Order is the ids the frame registered (Set), in the order of the
	// render.
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
	// Editing are the editing keymaps' bindings for Platform: the
	// surface's, which its frames share until it is attached anew, or
	// the frame's own where nil.
	Editing *Editing
	// EditKeys are the keys of the edit-shortcut groups' tables, set the
	// first time an element in the frame binds one: where nil, the frame
	// binds none, and no field resolves them.
	EditKeys []string
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
// its secret for the kit's markers, and the slots its ids take (nil for
// slots of its own). A frame made by a surface for each render shares
// the surface's Slots, so that one render's slot is the next one's.
func NewFrame(s any, p hotty.Platform, focused string, memo map[string]any, n uint64, keyMark string, slots *Slots) *Frame {
	if slots == nil {
		slots = &Slots{}
	}
	g := slots.current()
	return &Frame{on: make([]*Handlers, len(g.ids)), slots: slots, gen: g, Platform: p, surface: s, focused: focused, memo: memo, n: n, KeyMark: keyMark}
}

// Slots number the ids a surface registers handlers under, from render
// to render: a frame holds its handlers in an array by slot, and a kept
// compiled instance notes its ids' slots once. An id keeps its slot
// until Compact, which starts a generation of slots; Epoch counts the
// Compacts, after which a noted slot is noted again.
type Slots struct {
	gen   *slotGen
	Epoch uint64
}

// A slotGen is a generation of slots: each id's, and the id in each. A
// frame reads its handlers by the generation it was made in, which an
// id given a slot after it joins in place; Compact starts another, so a
// frame made before it (an event's, read again after a handler that
// rendered) still reads its own.
type slotGen struct {
	at  map[string]int32
	ids []string
}

func (x *Slots) current() *slotGen {
	if x.gen == nil {
		x.gen = &slotGen{}
	}
	return x.gen
}

// Slot is id's slot, given one where it has none.
func (x *Slots) Slot(id string) int32 { return x.current().slot(id) }

func (g *slotGen) slot(id string) int32 {
	if i, ok := g.at[id]; ok {
		return i
	}
	if g.at == nil {
		g.at = map[string]int32{}
	}
	i := int32(len(g.ids))
	g.at[id] = i
	g.ids = append(g.ids, id)
	return i
}

// Len is how many ids have a slot.
func (x *Slots) Len() int { return len(x.current().ids) }

// Compact starts a generation of slots in which only the ids f
// registered have one, in the order of their slots, and moves f's
// handlers to it: ids that went out of use no longer take room in every
// frame. f is of the current generation; a frame made before it keeps
// reading its own.
func (x *Slots) Compact(f *Frame) {
	old := f.gen
	g := &slotGen{at: make(map[string]int32, f.count), ids: make([]string, 0, f.count)}
	on := make([]*Handlers, 0, f.count)
	for i, h := range f.on {
		if h != nil {
			g.at[old.ids[i]] = int32(len(g.ids))
			g.ids = append(g.ids, old.ids[i])
			on = append(on, h)
		}
	}
	x.gen = g
	x.Epoch++
	f.on, f.gen = on, g
}

func (f *Frame) slotsOf() *Slots {
	if f.slots == nil {
		f.slots = &Slots{}
	}
	if f.gen == nil {
		f.gen = f.slots.current()
	}
	return f.slots
}

// AutoID allocates a render-order unique id for an anonymous surf.On call
// (when the caller passes id == ""), avoiding any already-registered id.
func (f *Frame) AutoID() string {
	for {
		f.autoN++
		id := "h-" + strconv.Itoa(f.autoN)
		if f.Get(id) == nil {
			return id
		}
	}
}

// Get is the handlers registered under id in this frame, nil for none.
func (f *Frame) Get(id string) *Handlers {
	if f == nil || f.gen == nil {
		return nil
	}
	if i, ok := f.gen.at[id]; ok && int(i) < len(f.on) {
		return f.on[i]
	}
	return nil
}

// Slot is id's slot in the frame's generation, given one where it has
// none.
func (f *Frame) Slot(id string) int32 {
	f.slotsOf()
	return f.gen.slot(id)
}

// Set registers h under id, in place of what was (Order is the
// caller's).
func (f *Frame) Set(id string, h *Handlers) {
	f.slotsOf()
	f.SetAt(f.gen.slot(id), h)
}

// GetAt is the handlers in slot i, nil for none.
func (f *Frame) GetAt(i int32) *Handlers {
	if int(i) < len(f.on) {
		return f.on[i]
	}
	return nil
}

// SetAt puts h in slot i, in place of what was.
func (f *Frame) SetAt(i int32, h *Handlers) {
	if n := int(i) + 1; n > len(f.on) {
		if n <= cap(f.on) {
			f.on = f.on[:n]
		} else {
			f.on = append(f.on, make([]*Handlers, n-len(f.on))...)
		}
	}
	switch {
	case f.on[i] == nil && h != nil:
		f.count++
	case f.on[i] != nil && h == nil:
		f.count--
	}
	f.on[i] = h
}

// Len is how many ids have handlers in this frame.
func (f *Frame) Len() int {
	if f == nil {
		return 0
	}
	return f.count
}

// Each calls fn with every id that has handlers in this frame, and
// them, in the order of their slots.
func (f *Frame) Each(fn func(id string, h *Handlers)) {
	if f == nil {
		return
	}
	if f.gen == nil {
		return
	}
	for i, h := range f.on {
		if h != nil {
			fn(f.gen.ids[i], h)
		}
	}
}

// EachSlot calls fn with every slot that has handlers in this frame, in
// order, and them: Each without the ids, for a walk that compares two
// frames of one Slots by slot.
func (f *Frame) EachSlot(fn func(i int32, h *Handlers)) {
	if f == nil {
		return
	}
	for i, h := range f.on {
		if h != nil {
			fn(int32(i), h)
		}
	}
}

// Slots is the slots f's ids take.
func (f *Frame) Slots() *Slots { return f.slotsOf() }

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

// Editing are the editing keymaps' bindings for a platform, each made the
// first time an element binds it.
type Editing struct {
	// Keymaps are surf.CUAKeys', surf.CocoaKeys' and surf.ReadlineKeys'
	// bindings, in that order.
	Keymaps [3][]Bind
	// Defaults are what a text field's default keymap does with each of
	// Frame.EditKeys (hotty.Keymap.Lookup, "" for a key that reaches the
	// program), made the first time a field resolves them.
	Defaults map[string]string
}
