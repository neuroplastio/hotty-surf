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

func TestToastActionAndCancel(t *testing.T) {
	var st ui.ToasterState
	acted, canceled := 0, 0
	toast := ui.Toast{Title: "Deleted", Duration: -1,
		Actions: []ui.ToastAction{
			{Label: "Undo", OnSelect: func() { acted++ }},
			{Label: "Later", Variant: ui.Outline, OnSelect: func() { canceled++ }},
		},
	}
	r := start(t, page(button("show", "Show", func() { st.Show(toast) }), ui.Toaster(&st)))
	r.Click("#show")
	r.Click("#toast-1-actions-1")
	if acted != 1 || canceled != 0 {
		t.Fatalf("action=%d cancel=%d", acted, canceled)
	}
	r.One("#toast-1-actions-1[disabled]")
	r.One("#toast-1-actions-2[disabled]")
	if len(r.Find("#toast-1-dismiss")) != 0 {
		t.Fatal("an Outline action is the cancel: no Dismiss beside it")
	}
	r.Click("#show")
	r.Click("#toast-2-actions-2")
	if acted != 1 || canceled != 1 {
		t.Fatalf("action=%d cancel=%d", acted, canceled)
	}
}

func TestToastActionRequiresLabel(t *testing.T) {
	f := &faults{T: t}
	var st ui.ToasterState
	st.Show(ui.Toast{Title: "Saved", Actions: []ui.ToastAction{{}}})
	surftest.Start(f, surf.New("toast", ui.Toaster(&st), ui.Stylesheet()))
	if len(f.got) == 0 {
		t.Fatal("empty action label did not fault")
	}
}

// An app that owns its toasts' lives (plx's): an ID updates a toast in
// place and removes it, its body takes a press, and the pointer leaving
// the toaster is heard.
func TestToastOwnedByTheApp(t *testing.T) {
	var st ui.ToasterState
	pressed, left := 0, 0
	st.OnLeave = func() { left++ }
	toast := ui.Toast{ID: "job", Title: "Building", Duration: -1, OnPress: func() { pressed++ },
		Actions: []ui.ToastAction{{Label: "Stop"}}}
	s := surf.New("t", page(
		button("show", "Show", func() { st.Show(toast) }),
		button("done", "Done", func() { toast.Title = "Built"; st.Show(toast) }),
		button("remove", "Remove", func() { st.Remove("job") }),
		text("note", "n"), ui.Toaster(&st)), ui.Stylesheet())
	r := surftest.Start(t, s)
	r.Click("#show")
	r.Click("#done")
	if n := len(r.Find("div[class=toast]")); n != 1 {
		t.Fatalf("%d toasts after Show on one ID, want 1", n)
	}
	if got := r.HTML(); !strings.Contains(got, ">Built</h2>") {
		t.Fatalf("the title is not Built: want the update in place: %s", got)
	}
	r.Click("#toast-1-body")
	if pressed != 1 {
		t.Fatalf("body presses %d, want 1", pressed)
	}
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Fatalf("a body press closed the toast: aria-hidden=%s", got)
	}
	hover(t, s, r, "toast-1")
	hover(t, s, r, "note")
	if left != 1 {
		t.Fatalf("OnLeave ran %d times, want 1", left)
	}
	r.Click("#remove")
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "true" {
		t.Fatalf("after Remove aria-hidden=%s, want the toast going", got)
	}
	eventually(t, r, "gone after Remove", func() bool { return len(r.Find("#toast-1")) == 0 })
	// The ID is free again: Show adds a new toast.
	r.Click("#show")
	r.One("#toast-2")
	if pressed != 1 {
		t.Fatalf("body presses %d", pressed)
	}
}

// A focused toast is still: no timer closes it, its first button is
// ringed, its bar stands. Ending the focus runs the timer and the bar on.
func TestToastFocused(t *testing.T) {
	var st ui.ToasterState
	toast := ui.Toast{ID: "n", Title: "Hi", Duration: 80 * time.Millisecond, Focused: true,
		Left: 80 * time.Millisecond, Total: 160 * time.Millisecond,
		Actions: []ui.ToastAction{{Label: "Open"}, {Label: "Later", Variant: ui.Outline}}}
	r := start(t, page(
		button("show", "Show", func() { st.Show(toast) }),
		button("away", "Away", func() { toast.Focused = false; st.Show(toast) }),
		ui.Toaster(&st)))
	r.Click("#show")
	r.One("#toast-1[data-focused]")
	bar := r.One("#toast-1-bar-0")
	if got := attr(bar, "style"); !strings.Contains(got, "--toast-bar-life:80ms") || !strings.Contains(got, "--toast-bar-cut:0.2500") {
		t.Fatalf("bar style %q, want 80ms left, a quarter cut off each end", got)
	}
	if got := attr(bar, "class"); got != "toast-bar still" {
		t.Fatalf("bar class %q, want it still", got)
	}
	time.Sleep(200 * time.Millisecond)
	r.Update()
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Fatalf("focused, yet its timer closed it: aria-hidden=%s", got)
	}
	r.Click("#away")
	if len(r.Find("#toast-1-bar-0")) != 0 {
		t.Fatal("the focus ended, yet the bar is the same run")
	}
	bar = r.One("#toast-1-bar-1")
	if attr(bar, "class") != "toast-bar" {
		t.Fatalf("after the focus the bar is %q %q: want it running, a new run", attr(bar, "class"), attr(bar, "id"))
	}
	eventually(t, r, "gone once the focus ended", func() bool { return len(r.Find("#toast-1")) == 0 })
}

// The focused look and the bar are the kit's: in Stylesheet.
func TestToastStylesInStylesheet(t *testing.T) {
	css := ui.Stylesheet()
	for _, want := range []string{".toast>.toast-bar{", "@keyframes toast-bar", ".toast[data-focused] footer>.btn:first-child"} {
		if !strings.Contains(css, want) {
			t.Errorf("Stylesheet lacks %q", want)
		}
	}
}
