package ui_test

import (
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// eventually runs the surface's queued work (its timers') until ok, or
// fails the test after a few seconds.
func eventually(t *testing.T, r *surftest.Rig, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		r.Update()
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("still not %s", what)
}

func TestToaster(t *testing.T) {
	var st ui.ToasterState
	show := func(t ui.Toast) func() { return func() { st.Show(t) } }
	r := start(t, page(
		button("saved", "Save", show(ui.Toast{Title: "Saved", Description: "All of it."})),
		button("failed", "Fail", show(ui.Toast{Title: "Failed", Category: "error", Duration: 30 * time.Millisecond})),
		button("sticky", "Stay", show(ui.Toast{Title: "Stays", Category: "info", Duration: -1})),
		ui.Toaster(&st),
	))
	r.One("div#toaster[class=toaster]")
	if len(r.Find("[role=status]")) != 0 {
		t.Fatal("a toast before any was shown")
	}

	r.Click("#saved")
	toast := r.One("div#toast-1")
	for k, v := range map[string]string{"role": "status", "data-category": "success", "aria-hidden": "false", "aria-atomic": "true"} {
		if got := attr(toast, k); got != v {
			t.Errorf("the toast's %s is %q, not %q", k, got, v)
		}
	}
	if hasAttr(toast, "data-duration") {
		t.Error("a toast of the default duration has data-duration")
	}
	r.One("h2:text(Saved)")
	r.One("p:text(All of it.)")
	if len(r.Find("svg[aria-hidden=true]")) != 1 {
		t.Error("no icon")
	}

	// An error is announced at once, and goes after its duration.
	r.Click("#failed")
	want(t, r, "#toast-2", "role", "alert")
	want(t, r, "#toast-2", "data-duration", "30")
	eventually(t, r, "hidden", func() bool { return attr(r.One("#toast-2"), "aria-hidden") == "true" })
	eventually(t, r, "gone", func() bool { return len(r.Find("#toast-2")) == 0 })

	// One that stays is dismissed.
	r.Click("#sticky")
	want(t, r, "#toast-3", "data-duration", "-1")
	r.Click("#toast-3-dismiss")
	want(t, r, "#toast-3", "aria-hidden", "true")
	eventually(t, r, "gone", func() bool { return len(r.Find("#toast-3")) == 0 })

	// The first goes after Basecoat's 3 s.
	eventually(t, r, "the first gone", func() bool { return len(r.Find("#toast-1")) == 0 })
}

func TestToastFaults(t *testing.T) {
	f := &faults{T: t}
	var st ui.ToasterState
	st.Show(ui.Toast{Title: "A", Category: "danger"})
	surftest.Start(f, surf.New("t", ui.Toaster(&st), ui.Stylesheet()))
	if !strings.Contains(strings.Join(f.got, "\n"), `ui.Toast: no category "danger"`) {
		t.Errorf("faults: %q", f.got)
	}
}

// The toaster pauses its toasts while the pointer is over it, as toast.js's
// mouseenter does, and they go once it leaves.
func TestToasterPausesUnderThePointer(t *testing.T) {
	var st ui.ToasterState
	s := surf.New("t", page(button("show", "Show", func() { st.Show(ui.Toast{Title: "Hi", Duration: 60 * time.Millisecond}) }),
		text("note", "n"), ui.Toaster(&st)), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#show")
	hover(t, s, r, "toast-1")
	time.Sleep(200 * time.Millisecond)
	r.Update()
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Errorf("paused under the pointer, yet aria-hidden=%s", got)
	}
	hover(t, s, r, "note")
	eventually(t, r, "gone after the pointer left", func() bool { return len(r.Find("#toast-1")) == 0 })
}
