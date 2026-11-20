package rt

import (
	"time"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/html"
)

// Peek is what a test rig, the reload child and ui's Bindings read of a
// surface that no exported method gives: package surf fills it in its
// init, so that the packages that read it need not import surf's insides
// (and surf need not export them). Production code reads Frame and Scopes
// (ui.Bindings, ui.HostMenu, ui.Pick). Snapshot, Adopt, Restyle, Document,
// Frame and Scopes run on the surface's driver goroutine: reload calls
// them through the surface's Do, or before its driver starts.
var Peek struct {
	// Tree is the tree the host was last sent, under its #root; nil before
	// the first Document.
	Tree func(s any) *html.Node
	// Focused is the id of the element the surface holds to have the
	// keyboard, "" for none: ui's differential tests compare it with a
	// browser's document.activeElement.
	Focused func(s any) string
	// Frame is the last frame rendered, nil before the first.
	Frame func(s any) *Frame
	// Scopes are the KeyScopes a key is offered to now, in the order it is
	// offered: those around the keyboard, nearest first, or with nothing
	// focused the outermost ones in render order.
	Scopes func(s any) []*KeyScope
	// FieldTakes says whether the focused element is a text field that
	// isn't a kit control under a KeyScope and whose keymap, as the host
	// resolves it from the data-keys the surface wrote, binds key to an
	// edit action (surf.ReadlineKeys' ctrl+k): the key never reaches a
	// KeyScope's binding there (ui.Bindings).
	FieldTakes func(s any, key string) bool
	// Faults are the faults of the renders since the last call, which it
	// forgets.
	Faults func(s any) []string

	// Snapshot writes the surface as the host has it, for a dev reload's
	// handover; an error before its first Document.
	Snapshot func(s any, w *msgp.Writer) error
	// Pause detaches a surface on its driver and returns a function that
	// restores its writer and platform; that function also runs on the driver.
	Pause func(s any) func()
	// Adopt takes a snapshot of the surface of the same name, from the
	// build before, as what the host has: the next Update sends a delta
	// from it, where Document would send it all again. Call it before the
	// surface's first Document; it wakes Pending.
	Adopt func(s any, r *msgp.Reader) error
	// Restyle swaps the surface's stylesheet; the next Update sends it (a
	// dev reload's css tier). It wakes Pending.
	Restyle func(s any, css string) error
	// Document is the whole document the host was last sent (the old
	// build's, read directly at a handover); "" before the first.
	Document func(s any) (string, error)
	// CSS is the surface's stylesheet, as New or the last Restyle gave it
	// (a dev reload's css tier replaces the app's part of it). It runs on
	// the driver goroutine, as Restyle does.
	CSS func(s any) string
	// Rendered is when the surface's view last rendered, the zero Time
	// before it first did. It may be called from any goroutine: the
	// reload child waits out templ's dev-mode recheck from it.
	Rendered func(s any) time.Time
	// Forget drops the surface's compiled instances (surf.Compile): a
	// dev reload's text tier, whose strings their kept markup still
	// holds. The next render renders each again.
	Forget func(s any)
	// Check turns on the check of compiled instances (surf.Compile): each
	// kept one is rendered too, and a difference from what was kept is a
	// fault. surftest turns it on, and so does a surfdev build.
	Check func(s any, on bool)
	// Instances are the surface's compiled instances after its last
	// render, each as "name key: why", why "kept", "rendered", or what
	// made it render every time.
	Instances func(s any) []string
}
