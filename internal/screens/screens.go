// Package screens is four demo apps in .surf (docs/surf.md): a command
// palette, a settings page, an agents list and a notices list, on the
// made-up data of internal/screens/demodata. Each is a Go struct, its state, whose
// methods are its handlers, and .surf templates, its layout; each also
// draws in cells (F7), in Go, from the same state.
package screens

//go:generate go run ../../cmd/surfgen -export

import (
	"time"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surfkeys"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// Mode is how the screens' templates are drawn.
type Mode string

// The modes.
const (
	// Interpreted draws them by the embedded templates, interpreted
	// (surfInterpT).
	Interpreted Mode = "interpreted"
	// Compiled draws them by the templates compiled into Go
	// (surfCompiledT).
	Compiled Mode = "compiled"
)

// Names are the screens' names, in order.
var Names = []string{"palette", "settings", "agents", "notices"}

// Screen is one screen: a fresh model of it, on fresh data, and what the
// runtime runs for it.
type Screen struct {
	Name  string
	Model surfrun.Model
	// Tick, if set, is the model's action the runtime runs every
	// TickEvery: the notices' clock.
	Tick      surfui.Action
	TickEvery time.Duration
	// Scroll, if set, lets the document scroll (settings).
	Scroll bool
}

// New is the screen name, fresh, on demodata.Mock, drawn in mode m; ok is
// false for a name that isn't one of Names.
func New(name string, m Mode) (s Screen, ok bool) { return On(name, demodata.Mock(), m) }

// On is the screen name, fresh, on src, drawn in mode m; ok is false for
// a name that isn't one of Names.
func On(name string, src *demodata.Source, m Mode) (s Screen, ok bool) {
	switch name {
	case "agents":
		return Screen{Name: name, Model: draw(m, NewAgents(src), surfInterpAgents, surfCompiledAgents)}, true
	case "palette":
		return Screen{Name: name, Model: draw(m, NewPalette(src), surfInterpPalette, surfCompiledPalette)}, true
	case "settings":
		return Screen{Name: name, Model: draw(m, NewSettings(src), surfInterpSettings, surfCompiledSettings), Scroll: true}, true
	case "notices":
		n := NewNotices()
		return Screen{Name: name, Model: draw(m, n, surfInterpNotices, surfCompiledNotices), Tick: n.Tick, TickEvery: 3 * time.Second}, true
	}
	return Screen{}, false
}

// draw is c's model, drawn by its templates in mode m: interp and
// compiled are its surfInterpT and surfCompiledT.
func draw[T any](m Mode, c *T, interp, compiled func(*T) surflive.Component) surfrun.Model {
	if m == Compiled {
		return surflive.New(compiled(c))
	}
	return surflive.New(interp(c))
}

// Document is a model's tree now, as a document `hotty render` and a
// browser open, the stylesheet inline: at density d where it is set,
// whatever the model's (surfrun.Options.Density), its keys drawn in the
// model's style (surfrun.KeyStyler) as a surface draws them.
func Document(m surfrun.Model, d surfui.Density) string {
	if s, ok := m.(surfrun.DensitySetter); ok && d != "" {
		s.SetDensity(d)
	}
	root := m.Render()
	st := surfkeys.Style{}
	if k, ok := m.(surfrun.KeyStyler); ok {
		st = k.KeyStyle()
	}
	surfkeys.Expand(root, st, true)
	if d != "" {
		root.Density(d)
	}
	surfui.Resolve(root, "r")
	return surfui.StandaloneDocument(root, "")
}
