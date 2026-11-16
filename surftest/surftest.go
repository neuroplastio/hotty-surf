// Package surftest drives a surface in a test, on hotty-go's test host,
// which applies the document and every delta as the SPEC says. After every
// act (a click, a key, typing) the rig checks that the host's document is the
// surface's own view, so a wrong delta fails the test where it was sent; the
// test then reads the host's document for what the user would see.
//
//	r := surftest.Start(t, surf.New("tasks", Page(a), CSS))
//	r.Click("button:text(Add task)")
//	r.Type("input[name=title]", "Write the docs")
//	r.Key("enter")
//	if len(r.Find("li:text(Write the docs)")) != 1 { … }
//
// A selector is tag#id[attr=value]:text(…), every part optional: [attr]
// alone for presence, :text(t) for an element whose whole text is t
// (spaces normalised) and :text-start(t) for one whose text starts with t.
//
// What would go wrong silently in an app fails the test: two surf.On for
// one id in a render, a ui component given a variant, size or side it
// doesn't take.
package surftest

import (
	"testing"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-surf"
)

// Start starts s on a test host and checks its first frame. A reply from the
// host that is an error fails the test.
func Start(t testing.TB, s *surf.Surface, opts ...Option) *Rig {
	panic("L0")
}

// An Option sets up the rig.
type Option struct {
	_ struct{}
}

// Size is the host's size in cells: 80×24 when not given.
func Size(cols, rows int) Option {
	panic("L0")
}

// A Rig is a surface on a test host.
type Rig struct {
	_ struct{}
}

// Click clicks the one element matching sel. It fails the test if no
// element or more than one matches.
func (r *Rig) Click(sel string) {
	panic("L0")
}

// Type gives the keyboard to the one field matching sel and types text into
// it, a key at a time.
func (r *Rig) Type(sel, text string) {
	panic("L0")
}

// Update runs what the surface's Do has queued (a load, a toast's timer),
// renders, and checks the host's document, as after any act.
func (r *Rig) Update() {
	panic("L0")
}

// Key presses a key where the keyboard is, named as hotty names keys
// ("enter", "arrow_down", "ctrl+s"), and says whether the surface took it.
func (r *Rig) Key(key string) bool {
	panic("L0")
}

// Find is the elements of the host's document that match sel, in order.
func (r *Rig) Find(sel string) []*html.Node {
	panic("L0")
}

// One is the one element matching sel. It fails the test if no element or
// more than one matches.
func (r *Rig) One(sel string) *html.Node {
	panic("L0")
}

// HTML is the host's document, without its stylesheet.
func (r *Rig) HTML() string {
	panic("L0")
}

// Lint fails the test for each rule of the kit's lint that the app's sources
// under dir break (colours outside the theme, px sizes, the root font,
// hiding the page below a breakpoint), as `surf lint` and `surf run` report
// them.
func Lint(t testing.TB, dir string) {
	panic("L0")
}
