package main

import (
	_ "embed"
	"encoding/json"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the app's own stylesheet, style/app.css. The kit's is linked,
// not carried (newSurface): sent once rather than in every document.
var CSS = appCSS

// upstreamTasks is upstream's 100 tasks (shadcn/ui, MIT; data/NOTICE.md).
//
//go:embed data/tasks.json
var upstreamTasks []byte

// upstream is upstream's 100 tasks.
func upstream() ([]Task, error) {
	var ts []Task
	err := json.Unmarshal(upstreamTasks, &ts)
	return ts, err
}

// surface is the surface the app shows on.
const surface = "tasks"

// newSurface is one of the app's surfaces: view, styled by the kit's sheet,
// linked rather than carried (ui.Sheet), then by CSS.
func newSurface(name string, view templ.Component) *surf.Surface {
	s := surf.New(name, view, CSS)
	s.Link(ui.Sheet())
	return s
}
