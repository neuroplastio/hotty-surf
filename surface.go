package surf

import (
	"context"
	"io"
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
	prev   *html.Node
	frame  *rt.Frame
	byID   map[string]*html.Node
	fields []string // the live fields' ids in prev
	ids    map[string]string
	nextID int
	faults []string

	// What the host's fields hold, as far as the surface knows: what the
	// user's events said, or what it last set (SPEC §6.4).
	vals    map[string]string
	checked map[string]bool
	revs    map[string]int

	// Where the keyboard and the pointer are, as the host said: the
	// element and the ids of it and its ancestors, which Focus and Hover
	// handlers are told about as they change.
	focused  string
	focusIn  []string
	hoverIn  []string
	asked    string // the element the last frame's Autofocus asked for
	askDue   bool   // a Document's Autofocus, sent at the next Update
	cssDue   bool   // the stylesheet changed (a dev reload): sent at the next Update
	sending  bool
	deferred []hotty.Event

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
	s := &Surface{name: name, view: view, css: css, pending: make(chan struct{}, 1)}
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
}

// Document renders the view and returns the whole document, its stylesheet
// and its body, for the program to send as the surface's document. The
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

// Key offers the surface a key while it has the keyboard (a key the host
// passed the program because no field in the surface took it, SPEC §10.1),
// named as hotty names keys ("enter", "arrow_down", "ctrl+s"). The [Key]
// handlers of the focused element and its ancestors are asked, innermost
// first, with the key as [Key] says; taken says whether one took it. A key
// no handler takes is the program's.
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
