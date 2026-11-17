package main

import (
	"testing"

	"neuroplast.io/hotty-surf/surftest"
)

// TestTasks adds a task, ticks it off, looks at the done ones and clears
// them, as a user would, checking the count at each step.
func TestTasks(t *testing.T) {
	r := surftest.Start(t, newSurface("todo", Page(NewApp())))
	r.One("#left:text(4 tasks left)")

	r.Type("#draft", "Buy milk")
	r.Key("enter")
	r.One("label:text(Buy milk)")
	r.One("#left:text(5 tasks left)")
	if v := r.Value("#draft"); v != "" {
		t.Errorf("the field holds %q after adding, want it empty", v)
	}

	r.Click("#task-7")
	if !r.Checked("#task-7") {
		t.Error("Buy milk isn't ticked")
	}
	r.One("#left:text(4 tasks left)")

	r.Click("button:text(Done 3)")
	r.One("label:text(Buy milk)")
	if n := len(r.Find("input[type=checkbox]")); n != 3 {
		t.Errorf("Done shows %d tasks, want 3", n)
	}
	r.One("#left:text(4 tasks left)")

	r.Click("button:text(Clear done)")
	r.One("h3:text(Nothing here)")
	r.Click("button:text(All 4)")
	if n := len(r.Find("input[type=checkbox]")); n != 4 {
		t.Errorf("All shows %d tasks, want 4", n)
	}
	r.One("#left:text(4 tasks left)")

	r.Click("button[aria-label=Delete Call Grandma]")
	r.One("#left:text(3 tasks left)")
}

// TestLint: the app keeps the kit's rules (package lint).
func TestLint(t *testing.T) { surftest.Lint(t, ".") }
