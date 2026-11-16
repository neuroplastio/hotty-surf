// Command wasmcheck writes the events of a Tasks run as a JSON list of the
// bytes a host would send, one string each, for scripts/wasm-stream.sh:
// Tasks built for js by Go and by TinyGo must write the same stream for
// them (run.mjs runs a build in Node on the page glue run.App speaks to).
package main

import (
	"encoding/json"
	"os"

	"neuroplast.io/hotty-go"
)

func main() {
	on := true
	var out []string
	add := func(e hotty.Event) { e.Surface = "tasks"; out = append(out, hotty.EncodeEvent(e)) }
	click := func(id string) {
		add(hotty.Event{Kind: hotty.EventFocus, Target: id})
		add(hotty.Event{Kind: hotty.EventClick, Target: id})
	}
	pick := func(id string) {
		add(hotty.Event{Kind: hotty.EventBlur})
		add(hotty.Event{Kind: hotty.EventClick, Target: id})
	}
	for i := 0; i < 5; i++ {
		add(hotty.Event{Kind: hotty.EventInput, Target: "filter", Value: "v", Rev: 10 + 2*i})
		add(hotty.Event{Kind: hotty.EventInput, Target: "filter", Value: "", Rev: 11 + 2*i})
		click("sort-title")
		pick("sort-title-asc")
		click("facet-status")
		pick("facet-status-in-progress")
		pick("facet-status-in-progress")
		click("page-next")
		click("page-prev")
		add(hotty.Event{Kind: hotty.EventChange, Target: "r0-sel", Checked: &on})
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
