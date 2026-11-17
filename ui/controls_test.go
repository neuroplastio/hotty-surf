package ui_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/ui"
)

func TestControlsEditAndEventLogClear(t *testing.T) {
	values := []ui.Control{{Name: "Label", Kind: "string", Value: ""}, {Name: "Enabled", Kind: "bool", Value: "false"}}
	entries := []ui.LogEntry{{At: time.Unix(1, 0), Kind: "click", Text: "older"}, {At: time.Unix(2, 0), Kind: "input", Text: "newer"}}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return templ.Join(ui.Controls(ui.ControlsProps{ID: "test", Controls: values, OnChange: func(name, value string) {
			for i := range values {
				if values[i].Name == name {
					values[i].Value = value
				}
			}
		}}), ui.EventLog(ui.EventLogProps{ID: "events", Entries: entries, OnClear: func() { entries = nil }, Empty: "No events"})).Render(ctx, w)
	})
	r := start(t, view)
	r.Type("#test-ctl-Label", "New")
	if values[0].Value != "New" {
		t.Fatalf("text edit: %q", values[0].Value)
	}
	r.Click("#test-ctl-Enabled")
	if values[1].Value != "true" {
		t.Fatalf("switch edit: %q", values[1].Value)
	}
	page := r.HTML()
	if strings.Index(page, "newer") > strings.Index(page, "older") {
		t.Fatal("event log was not newest first")
	}
	r.Click("#events-clear")
	if len(entries) != 0 || !strings.Contains(r.HTML(), "No events") {
		t.Fatal("event log did not clear")
	}
}
