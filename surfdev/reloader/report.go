package reloader

//go:generate msgp -marshal=false -tests=false -file report.go -o report_gen.go

//msgp:newtime
//msgp:limit arrays:65536 maps:65536

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// Finding is one lint finding reported by the reloader.
type Finding struct {
	File string `msg:"file"`
	Line int    `msg:"line"`
	Col  int    `msg:"col"`
	Rule string `msg:"rule"`
	Text string `msg:"text"`
}

// A Report is the outcome of a save. It has msgp's codec (EncodeMsg,
// DecodeMsg), a map keyed by the names in its tags, with Took in
// nanoseconds, so that `hotty-surf ctl reload` and a dev parent such as
// plx carry the same Report on their wires (gov P-3).
type Report struct {
	Tier string        `msg:"tier"` // "text", "css" or "go"
	Took time.Duration `msg:"took"` // from the save to every process showing it
	// Build is what kept the save from the app, when the build failed:
	// the compiler's output. The old processes still run. It is "" when
	// the build succeeded.
	Build string `msg:"build"`
	// Failed is what went wrong for a process, by the key it gave
	// surfdev.Serve: restart's error, a key that didn't come back in time, a
	// process that "exited before handover", a handover's own error, or a
	// new process that started without the state handed over, which no
	// longer decoded.
	Failed map[string]string `msg:"failed"`
	Lint   []Finding         `msg:"lint"`
}

// Text is r as `hotty-surf ctl reload` prints it, so that another dev
// parent prints the same: a first line with the tier and what the save
// took to the millisecond ("ok: go in 312ms"), or the build's output under
// "build failed"; then a line for each failed process, sorted by key; then
// the lint's findings, one "file:line:col: rule: text" line each.
func Text(r *Report) string {
	var b strings.Builder
	took := r.Took.Round(time.Millisecond)
	switch {
	case r.Build != "":
		fmt.Fprintf(&b, "build failed (%s tier; the app runs the last build):\n%s\n", r.Tier, strings.TrimRight(r.Build, "\n"))
	case r.Tier == "":
		b.WriteString("ok: nothing reloaded yet\n")
	default:
		fmt.Fprintf(&b, "ok: %s in %v\n", r.Tier, took)
	}
	for _, k := range slices.Sorted(maps.Keys(r.Failed)) {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.TrimRight(r.Failed[k], "\n"))
	}
	if n := len(r.Lint); n > 0 {
		s := "s"
		if n == 1 {
			s = ""
		}
		fmt.Fprintf(&b, "lint: %d finding%s\n", n, s)
		for _, f := range r.Lint {
			fmt.Fprintf(&b, "  %s:%d:%d: %s: %s\n", f.File, f.Line, f.Col, f.Rule, f.Text)
		}
	}
	return b.String()
}
