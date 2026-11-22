// Package surf runs a hotty app written in Go: its state is a Go value, its
// view is templ components (HTML with the kit's classes: Basecoat's, and
// the Tailwind utilities the kit uses, not all of Tailwind's, which
// `hotty-surf classes` lists), and after every event the view is rendered
// again, compared with what the host has, and the difference goes to the
// host as one delta.
//
// A [Surface] is one view on one hotty surface. It owns no terminal: it
// writes hotty commands to the writer it is attached to and takes the
// host's events from whoever reads the terminal. So the same Surface is a
// whole app under package surfapp, a part of another program's screen (a Bubble
// Tea program through surftea, a raw-mode program through hotty-go's
// hottyterm), a test under surftest, and a picture in character cells under
// package surftui.
//
// Handlers are Go closures that the view registers as it renders, on an
// element it names with [On]:
//
//	<button { surf.On(ctx, "save", surf.Click(a.Save))... }>Save</button>
//
// The name becomes the element's id; when the host reports an event on it,
// the Surface calls the closure the last render registered, then renders
// again. Nothing in the host runs app code.
//
// Keymap options for [Bind] live in package surfkey, and the UI kit and its
// components live under package surfui.
package surf
