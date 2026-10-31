// Package surflive draws components that keep their own state, as go-app's
// components and Phoenix LiveView's live components do (candidate A of
// gov R-23).
//
// A component is a struct: its fields are its state, its methods its
// handlers, and a method draws it. A child component is a field of its
// parent's, or one of a slice or a map of them, so it keeps its state for
// as long as its parent keeps it (F1, F2). What the parent tells it at a
// render, its props, are the arguments of its draw method. An app adds a
// state it needs by adding a field.
//
// Handlers are closures, most of them method values (`row.restart`), so
// what an event carries is whatever the closure holds, typed (F3). After a
// handler runs, the app draws again from its root, and the runtime's diff
// sends what changed as ops: a tick that changes one agent sends that
// row's few ops, not a document.
package surflive

import (
	"time"

	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Component is what an app shows: its root, and any child that draws
// with nothing from its parent but the Ctx.
type Component interface {
	Render(c *Ctx) *surfui.Node
}

// App is a root component run as the runtime's model.
type App struct {
	root Component
	rt   surfrun.Runner
}

// New is the runtime's model for root. If root also draws in cells
// (surfrun.Cells), the model does too, from the same state (F7).
func New(root Component) surfrun.Model {
	a := &App{root: root}
	if c, ok := root.(surfrun.Cells); ok {
		return cellsApp{a, c}
	}
	return a
}

// Render draws the app from its root.
func (a *App) Render() *surfui.Node { return a.root.Render(&Ctx{app: a}) }

// Do runs a handler: a func(), or a func(surfrun.Event) that reads the event.
func (a *App) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func():
		f()
	case func(surfrun.Event):
		f(ev)
	}
}

// Attach is the runner's (surfrun.Runtime, or a host of its own), so that a
// timer can post back to it. A root that is a surfrun.Attacher hears it too:
// it learns the runner when the run starts, to start work of its own that
// posts its results (Runner.Post), such as a watch on its data, or to end
// the run from a handler (Runner.Quit); and it hears Attach(nil) when the
// run ends, to stop that work.
func (a *App) Attach(r surfrun.Runner) {
	a.rt = r
	if at, ok := a.root.(surfrun.Attacher); ok {
		at.Attach(r)
	}
}

// Resize tells the root the surface's size in cells, if it lays out by
// it (surfrun.Sizer): a list that shows as many rows as fit.
func (a *App) Resize(cols, rows int) {
	if s, ok := a.root.(surfrun.Sizer); ok {
		s.Resize(cols, rows)
	}
}

// KeyStyle is how the root draws its keys, if it says (surfrun.KeyStyler):
// the zero Style, which says none, where it doesn't.
func (a *App) KeyStyle() surfkeys.Style {
	if k, ok := a.root.(surfrun.KeyStyler); ok {
		return k.KeyStyle()
	}
	return surfkeys.Style{}
}

type cellsApp struct {
	*App
	surfrun.Cells
}

// Ctx is what a render hands its components: what they need of the app
// beyond their own state. A handler may keep the Ctx of the render that
// made it.
type Ctx struct {
	app   *App
	depth int
}

// Depth is how many levels deep the component c is handed to is drawn: 0
// for the root, and for a component a template embeds (surf's <embed>),
// the level of the element that embeds it. A component drawn by a
// template goes on from there, so that the render as a whole nests no
// deeper than surf.MaxDepth.
func (c *Ctx) Depth() int {
	if c == nil {
		return 0
	}
	return c.depth
}

// At is c for a component drawn d levels deep: what a render hands a
// component it embeds.
func (c *Ctx) At(d int) *Ctx {
	if c == nil {
		return &Ctx{depth: d}
	}
	return &Ctx{app: c.app, depth: d}
}

// After runs fn on the app's goroutine once d has passed, and then draws
// the app again: a state that ends by itself, such as a row's
// "restarting…" or a key cap's press. Outside a run it does nothing.
func (c *Ctx) After(d time.Duration, fn func()) {
	if c == nil || c.app == nil || c.app.rt == nil {
		return
	}
	rt := c.app.rt
	time.AfterFunc(d, func() { rt.Post(fn) })
}
