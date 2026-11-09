// Package widgets is a screen drawn with widgets registered from outside
// the framework (examples/widgets/gauge): x-gauge, x-callout and
// x-tagline, in the interpreted and the compiled mode alike. Its
// generator is this example's own command, which links the widgets
// (cmd/surfgen); docs/surf.md's "Widgets from outside" tells the rest.
package widgets

//go:generate go run ./cmd/surfgen

// Dash is the screen: a gauge, a callout and a tagline. Its template is
// dash.surf.
type Dash struct {
	Title string
	Level int
	Note  string
	Log   []string
}

// Raise fills the gauge ten percent more, to a full one.
func (d *Dash) Raise() {
	d.Level = min(d.Level+10, 100)
	d.Log = append(d.Log, "raise")
}

// Pick logs a click on the callout.
func (d *Dash) Pick(id string) { d.Log = append(d.Log, "pick "+id) }
