package surf

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

// A Surface shows a view on one hotty surface: it gives the program the
// view's document to send, then writes one delta for each change, and runs
// the handlers the host's events name.
//
// The program that reads the terminal drives it: [Surface.Attach] once the
// host is known, [Surface.Document] for the program to send (hotty-go's Doc,
// or hottytea's Surface.Doc), [Surface.Event] for each event that names the
// surface, [Surface.Key] for each key while the surface has the keyboard,
// and [Surface.Update] when [Surface.Pending] fires. These calls come from
// one goroutine, the driver's. [Surface.Do] and [Surface.Pending] may be
// called from any goroutine.
type Surface struct {
	name string
	view templ.Component
	css  string

	// Do's queue, the one thing other goroutines touch.
	mu      sync.Mutex
	queue   []func()
	pending chan struct{}

	out      io.Writer
	platform hotty.Platform

	// The last frame sent: its tree under #root, the handlers its render
	// registered, every element by its id, and the ids given by place.
	prev  *html.Node
	frame *rt.Frame
	byID  map[string]*html.Node
	// keyMarks are the kit controls under a KeyScope, by id (writeKeys).
	keyMarks map[string]keyMark
	fields   []string // the live fields' ids in prev
	ids      map[string]string
	nextID   int
	faults   []string
	memo     map[string]any // the frames' Memo, kept from render to render
	n        uint64         // the renders so far: the last frame's N

	// What the host's fields hold, as far as the surface knows: what the
	// user's events said, or what it last set (SPEC §6.4).
	vals          map[string]string
	checked       map[string]bool
	revs          map[string]int
	sels          map[string][2]int
	pendingSel    map[string][2]int
	pendingNative map[string]hotty.FieldChange

	// Where the keyboard and the pointer are, as the host said: the
	// element and the ids of it and its ancestors, which Focus and Hover
	// handlers are told about as they change.
	focused string
	// lastFocus is the element that had the keyboard last, which Focus
	// gives it back to.
	lastFocus    string
	focusIn      []string
	hoverIn      []string
	hoverAt      string          // the element under the pointer, as the host last said
	pressed      bool            // the last event was a primary press on the surface (see event)
	drag         string          // the element a drag under way began on, "" for none
	lifted       bool            // that drag has moved: the element has the class dragging
	modalPressed bool            // the last press was inside the active modal's scope
	modalFocus   string          // first or last modal control to focus after an event
	asked        string          // the element the last frame's Autofocus asked for
	askDue       bool            // a Document's Autofocus, sent at the next Update
	cssDue       bool            // the stylesheet changed (a dev reload): sent at the next Update
	sheets       []Sheet         // the stylesheets documents link (Link)
	sheetSent    map[string]bool // the sheets sent on the writer since Attach, by ID
	limits       map[string]int  // the host's limits, from Attach's caps (SPEC §13)
	blurAsked    bool            // Blur ran the blur: the host's blur that follows is its echo
	sending      bool
	deferred     []hotty.Event

	r renderer
}

// New makes a surface named name (a hotty surface name, SPEC §5.1) showing
// view, styled by css: the app's stylesheet, which the build makes from the
// kit's and the app's classes, or ui.Stylesheet for a surface with no build
// of its own. It renders nothing until Document.
//
// In a dev build (the surfdev tag), New registers the surface with package
// reload, so that a live reload reaches it wherever it was made, a dialog
// opened late included. A release build has none of that.
func New(name string, view templ.Component, css string) *Surface {
	s := &Surface{name: name, view: view, css: css, pending: make(chan struct{}, 1), memo: map[string]any{}}
	s.reset()
	register(s)
	return s
}

// Name is the surface's name, as New was given it: what the host's events
// name it by.
func (s *Surface) Name() string {
	return s.name
}

// Attach says where the surface's deltas go, w, and what the host is: caps
// are its capabilities, as detection found them (hotty-go's Detector, or
// hottytea's ReadyMsg): the version it speaks, and its platform, which
// [Platform] gives a render. The zero Caps is a host of hotty's current
// version on the zero Platform (ctrl), which is what a cells.Screen is.
// Attach again when they change.
//
// Two rules for the program that drives it:
//
//   - Each Write to w is one whole hotty command. hottyterm's Out already
//     takes one command per Write; a Bubble Tea program writes through
//     hottytea's Session.Send (surftea does), never to the terminal
//     itself, or Bubble Tea's own frames land inside a delta.
//   - The program places the surface with Press and Hover on
//     (hotty.Placement's flags, SPEC §5.2): [Hover] handlers hear nothing
//     without Hover, and [Outside] misses a click on the surface's text
//     without Press. surftea's Place and run.App set both.
func (s *Surface) Attach(w io.Writer, caps hotty.Caps) {
	s.out = w
	s.platform = hotty.PlatformOf(caps)
	s.sheetSent, s.limits = nil, caps.Limits
}

// Sheet is a stylesheet that documents link as a resource (SPEC §7.1)
// rather than carry: sent once on a connection, it styles every document
// sent there after, and every surface that links it. A kit's sheet is
// ui.Sheet().
type Sheet struct {
	// ID names the resource: documents link cid:<ID>. Give the sheet a
	// new ID whenever its CSS changes (a content hash, as ui.Sheet's is),
	// so a host that kept an older sheet never styles a document with it.
	ID string
	// CSS is the sheet. Empty, the program sends the resource itself
	// (hotty.Res(ID, "text/css", …)), before the first document that links
	// it: a program with one connection for many surfaces sends it once
	// for all of them.
	CSS string
}

// Link makes the surface's documents link sheets, in order, ahead of its
// own stylesheet (New's css): each as <link rel="stylesheet"
// href="cid:ID">. A sheet with CSS is sent on the writer Attach gave, as
// a=res, once per Attach: before the next document Document returns, so a
// program that sends that document on the same writer has the sheet
// there first; after the host drops its resources, Attach again. When the
// host's caps leave no room for it (limits.resources, limits.message), the
// surface writes the CSS into the document as New's css, and records a
// fault. A sheet without CSS is the
// program's to send. Each Link replaces the sheets an earlier one gave;
// Link() with none links none. Link before the first Document; a later
// Link takes effect at the next Document.
func (s *Surface) Link(sheets ...Sheet) {
	s.sheets = append([]Sheet(nil), sheets...)
}

// Document renders the view and returns the whole document, its stylesheet
// and its body, for the program to send as the surface's document. When
// the surface links a sheet with CSS it hasn't sent since Attach, Document
// first writes it (a=res) to Attach's writer. The
// deltas that follow are written against it, so the program sends every
// Document it takes. It takes one again whenever the host may not hold the
// document as it was sent: it lost it, or it refused a delta (an error reply
// naming the surface, or an error from w's Write, as a cells.Screen
// refuses).
func (s *Surface) Document() (string, error) {
	return s.document()
}

// Event gives the surface an event from the host that names it (SPEC §9):
// it updates what it knows of the host's fields and focus, runs the handler
// the event's element has, renders, and sends what changed. An event the
// surface has no handler for changes nothing.
//
// Event may be called from inside a Write to w, while the surface is
// sending (a cells.Screen calls its events that way): the surface runs it
// once that send is done, before the call that wrote returns.
func (s *Surface) Event(e hotty.Event) error {
	return s.event(e)
}

// Focus gives the surface the keyboard, as the program's own focus moves
// onto it (Tab between panes): it sends the host a=focus for the element
// that had the keyboard last, while it is in the document, else for the
// first that takes focus, by its id (SPEC §10.1). The host sends no focus
// event for focus the program gives, so the surface takes the move as that
// event itself: its Focus handlers hear it, as they hear a Tab inside the
// surface, and the keys the host passes on reach that element's Key
// handlers. focused says whether the surface has the keyboard after it:
// true when it had it already, and false, with nothing sent, when nothing
// in it takes focus or it hasn't rendered and been attached yet. In cells
// the program gives the Screen the keyboard first (cells.Screen.Focus),
// which may already have restored the element. It runs on the driver's
// goroutine, as Event does.
func (s *Surface) Focus() (focused bool, err error) {
	return s.takeFocus()
}

// Blur takes the keyboard from the surface, as the program's own keys
// take the host's keyboard (a menu, a prompt) or its focus moves off the
// surface: it takes the move as a blur itself, so that the surface knows
// at once, and sends the host a=blur (SPEC §10.1); the blur the host sends
// after it is taken as this one and runs nothing twice. Focus handlers
// hear the element lose the keyboard, Outside handlers run (an open select
// or popover closes), and the element is kept: a later Focus gives it
// back. A field's change the host commits with the blur still reaches its
// handler, after the blur. blurred says whether the surface had the
// keyboard; with nothing focused nothing is sent. In cells the program
// takes the Screen's keyboard instead (cells.Screen.Focus(false)), which
// the surface hears as the same blur. It runs on the driver's goroutine,
// as Event does.
func (s *Surface) Blur() (blurred bool, err error) {
	return s.blur()
}

// Focused is the id of the element that has the keyboard as the surface
// knows it: the last focus the host reported, a Focus the program gave,
// or the element a render's Autofocus asked for; "" for none. A blur,
// the host's or Blur, clears it. It is the surface's view, which changes
// only with the events the program hands it, Focus, Blur and Autofocus;
// in cells, cells.Screen.Focus(false) reaches it as a blur event. The id
// may be one the surface generated. It runs on the driver's goroutine, as
// Event does.
func (s *Surface) Focused() string {
	return s.focused
}

// Overlays is the ids of the kit's overlays open in the last frame the
// surface rendered (by Document, Update, Event, Key, Focus or Blur), as
// their components' IDs: the modals (a Dialog, AlertDialog, Drawer or
// CommandDialog) outermost first, then the popup (a DropdownMenu,
// ContextMenu, Select, Combobox or Popover), of which one at most is open.
// nil is none. An overlay the app opened in its State but didn't render
// is not open. A driver that writes the surface's output through a writer
// of its own reads it after rendering and before it passes the bytes on,
// so it knows an overlay is opening before the host draws it. It is the
// same in cells. It runs on the driver's goroutine, as Event does.
func (s *Surface) Overlays() []string {
	if s.frame == nil || len(s.frame.Modal)+len(s.frame.Popups) == 0 {
		return nil
	}
	return append(slices.Clone(s.frame.Modal), s.frame.Popups...)
}

// Chars is what the focused element does with a key that types a
// character ("a", "shift+a", "1", ">"): whether the host keeps such a key
// for it, ahead of the program's bindings. It reads the last document the
// surface sent; with nothing focused it is CharsBound. It knows the
// native elements only: the kit's Select and DropdownMenu don't pick by
// characters in v0.7, so an open one is CharsBound, as a native select
// multiple is; a field composing (IME) takes every key, which it doesn't
// say. How a program routes a key with it is the keyboard model's
// (docs/keyboard-model.md, Routing). It runs on the driver's goroutine,
// as Event does.
func (s *Surface) Chars() Chars {
	return charsOf(s.byID[s.focused])
}

// OwnKeys are the keys the focused element takes itself, ahead of the
// program's bindings, other than characters (Chars says those; space is
// one only in a text field, which types it), as hotty names them, sorted.
// They are a native control's (SPEC §10.2) and a kit control's own:
//   - a text field's keymap as the host resolves it: the platform's
//     default, then the data-keys from the root down as written, so a
//     KeyScope's bindings are the program's. A move is listed with shift
//     too; a key the keymap gives the program, a prefix or a scroll is
//     not listed.
//   - a select's, a date or time field's, and a button's, link's, check
//     box's or radio button's keys, as SPEC lists them, each with shift.
//   - a kit control's own: those it gives the program on the focused
//     element itself (a list's arrows, a combobox's Enter), for its Key
//     handler to take.
//
// Tab and Shift+Tab are never in it, nor an app's Key handler's keys,
// which declare none: an app that wants a key ahead of the program's
// bindings binds it in a KeyScope. A disabled element, nothing focused
// and a surface not attached (no platform yet) have none. How a program
// routes a key with OwnKeys, Chars and ui.Bindings is the keyboard
// model's (docs/keyboard-model.md, Routing). It reads the last document
// the surface sent and its focus, as Chars does, and runs on the
// driver's goroutine.
func (s *Surface) OwnKeys() []string {
	if s.out == nil || s.frame == nil {
		return nil
	}
	return ownKeys(s.byID[s.focused], s.frame.On, s.platform)
}

// Chars is what a focused element does with a key that types a character.
type Chars uint8

const (
	// CharsBound is an element that takes no characters, or none focused:
	// such a key goes to the program's bindings, as any key does.
	CharsBound Chars = iota
	// CharsPick is a native select, which picks the next option whose
	// label starts with the character.
	CharsPick
	// CharsText is a text field (an input that edits text, a textarea,
	// readonly or not) or an editing host (contenteditable), which types
	// it (SPEC §10.2): the host keeps the key for it, and Key never hears
	// it.
	CharsText
)

// Key offers the surface a key while it has the keyboard (a key the host
// passed the program because no field in the surface took it, SPEC §10.1),
// named as hotty names keys ("enter", "arrow_down", "ctrl+s"). The [Key]
// handlers of the focused element and its ancestors are asked, innermost
// first, with the key as [Key] says; taken says whether one took it. A key
// no handler takes is the program's.
// A ui.KeyScope naming a key runs before a focused kit control's default
// and ordinary Key handlers; a decline tries enclosing scopes, then the
// default.
func (s *Surface) Key(key string) (taken bool, err error) {
	return s.key(key)
}

// Update runs the functions [Surface.Do] queued, renders the view, and
// sends what changed. A driver calls it when Pending fires; a program that
// changed the app's state on the driver's goroutine calls it itself.
func (s *Surface) Update() error {
	return s.update()
}

// Do queues fn to run on the driver's goroutine at its next Update: for a
// timer or a background job that changes the app's state. It may be called
// from any goroutine.
func (s *Surface) Do(fn func()) {
	s.mu.Lock()
	s.queue = append(s.queue, fn)
	s.mu.Unlock()
	s.wake()
}

// Pending receives when Do has queued work, so that the driver calls Update.
// It is the same channel for the surface's life.
func (s *Surface) Pending() <-chan struct{} {
	return s.pending
}

// Of is the surface rendering the view that ctx belongs to, for a component
// that keeps work beyond the render (a toast's timer reaches the loop with
// Of(ctx).Do). It is nil outside a render.
func Of(ctx context.Context) *Surface {
	if f := rt.From(ctx); f != nil {
		s, _ := f.Surface.(*Surface)
		return s
	}
	return nil
}

// Platform is the host's platform, from the capabilities Attach was given:
// the modifier its "mod" stands for and its edit shortcuts (SPEC §10.4,
// §10.2), for a render that shows a shortcut the way the host's platform
// writes it. It is the zero Platform (ctrl) outside a render.
func Platform(ctx context.Context) hotty.Platform {
	if f := rt.From(ctx); f != nil {
		return f.Platform
	}
	return hotty.Platform{}
}
