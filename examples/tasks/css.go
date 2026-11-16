package main

import (
	_ "embed"
	"encoding/json"

	"neuroplast.io/hotty-surf/ui"
)

//go:embed style/app.css
var appCSS string

// CSS is the kit's stylesheet, then the app's own Tailwind utilities.
var CSS = ui.Stylesheet() + "\n" + appCSS

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
