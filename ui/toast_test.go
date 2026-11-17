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
		ui.Toaster(&st, ui.ToasterProps{}),
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
	surftest.Start(f, surf.New("t", ui.Toaster(&st, ui.ToasterProps{}), ui.Stylesheet()))
	if !strings.Contains(strings.Join(f.got, "\n"), `ui.Toast: no category "danger"`) {
		t.Errorf("faults: %q", f.got)
	}
}

// The toaster pauses its toasts while the pointer is over it, as toast.js's
// mouseenter does, and they go once it leaves.
func TestToasterPausesUnderThePointer(t *testing.T) {
	var st ui.ToasterState
	s := surf.New("t", page(button("show", "Show", func() { st.Show(ui.Toast{Title: "Hi", Duration: 60 * time.Millisecond}) }),
		text("note", "n"), ui.Toaster(&st, ui.ToasterProps{})), ui.Stylesheet())
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
	r := start(t, page(button("show", "Show", func() { st.Show(toast) }), ui.Toaster(&st, ui.ToasterProps{})))
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
	surftest.Start(f, surf.New("toast", ui.Toaster(&st, ui.ToasterProps{}), ui.Stylesheet()))
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
	toast := ui.Toast{ID: "job", Title: "Building", Duration: -1, OnClick: func() { pressed++ },
		Actions: []ui.ToastAction{{Label: "Stop"}}}
	s := surf.New("t", page(
		button("show", "Show", func() { st.Show(toast) }),
		button("done", "Done", func() { toast.Title = "Built"; st.Show(toast) }),
		button("remove", "Remove", func() { st.Remove("job") }),
		text("note", "n"), ui.Toaster(&st, ui.ToasterProps{OnLeave: func() { left++ }})), ui.Stylesheet())
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
	// The keyboard reaches it too: Enter on it clicks it.
	r.Key("enter")
	if pressed != 2 {
		t.Fatalf("Enter on the body: presses %d, want 2", pressed)
	}
	pressed = 1
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
// ringed. Ending the focus runs the timer on.
func TestToastFocused(t *testing.T) {
	var st ui.ToasterState
	toast := ui.Toast{ID: "n", Title: "Hi", Duration: 80 * time.Millisecond, Focused: true,
		Actions: []ui.ToastAction{{Label: "Open"}, {Label: "Later", Variant: ui.Outline}}}
	r := start(t, page(
		button("show", "Show", func() { st.Show(toast) }),
		button("away", "Away", func() { toast.Focused = false; st.Show(toast) }),
		ui.Toaster(&st, ui.ToasterProps{})))
	r.Click("#show")
	r.One("#toast-1[data-focused]")
	time.Sleep(200 * time.Millisecond)
	r.Update()
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Fatalf("focused, yet its timer closed it: aria-hidden=%s", got)
	}
	r.Click("#away")
	eventually(t, r, "gone once the focus ended", func() bool { return len(r.Find("#toast-1")) == 0 })
}

// The bar is the kit's to keep, from the app's deadline (K30): the same
// Countdown keeps its run, another starts it full, a focus holds it where
// it stands and its end runs it on from there.
func TestToastCountdown(t *testing.T) {
	t0 := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	now := t0
	defer ui.SetToastClock(func() time.Time { return now })()
	var st ui.ToasterState
	toast := ui.Toast{ID: "n", Title: "Hi", Duration: -1, Countdown: t0.Add(10 * time.Second)}
	r := start(t, page(button("show", "Show", func() { st.Show(toast) }), ui.Toaster(&st, ui.ToasterProps{})))
	bar := func(id, class, style string) {
		t.Helper()
		n := r.One("#toast-1-bar-" + id)
		if got := attr(n, "class"); got != class {
			t.Fatalf("bar %s class %q, want %q", id, got, class)
		}
		if got := attr(n, "style"); got != style {
			t.Fatalf("bar %s style %q, want %q", id, got, style)
		}
	}
	r.Click("#show")
	bar("1", "toast-bar", "--toast-bar-life:10000ms;--toast-bar-steps:100;--toast-bar-cut:0.0000;")
	now = t0.Add(2 * time.Second)
	r.Click("#show") // the same deadline: the same run
	bar("1", "toast-bar", "--toast-bar-life:10000ms;--toast-bar-steps:100;--toast-bar-cut:0.0000;")
	now = t0.Add(4 * time.Second)
	toast.Focused, toast.Countdown = true, time.Time{}
	r.Click("#show") // 60% left: still there, though focus came with no deadline (plx's)
	bar("2", "toast-bar still", "--toast-bar-cut:0.2000;")
	now = t0.Add(9 * time.Second)
	toast.Countdown = t0.Add(20 * time.Second)
	r.Click("#show") // still, whatever Countdown says
	bar("2", "toast-bar still", "--toast-bar-cut:0.2000;")
	now = t0.Add(10 * time.Second)
	toast.Focused, toast.Countdown = false, t0.Add(16*time.Second)
	r.Click("#show") // on from 60%, over what is left
	bar("3", "toast-bar", "--toast-bar-life:6000ms;--toast-bar-steps:60;--toast-bar-cut:0.2000;")
	toast.Countdown = t0.Add(time.Hour)
	r.Click("#show") // a new deadline: full again, its steps capped
	bar("4", "toast-bar", "--toast-bar-life:3590000ms;--toast-bar-steps:1200;--toast-bar-cut:0.0000;")
	toast.Countdown = time.Time{}
	r.Click("#show")
	if len(r.Find("[data-bar]")) != 0 {
		t.Fatal("a zero Countdown still draws a bar")
	}
}

// A Show that changes Duration after the old timer fired, its close
// queued, keeps the toast: only the current timer closes it.
func TestToastStaleTimer(t *testing.T) {
	var st ui.ToasterState
	r := start(t, page(
		button("show", "Show", func() { st.Show(ui.Toast{ID: "n", Title: "Hi", Duration: 30 * time.Millisecond}) }),
		button("keep", "Keep", func() {
			// The timer fires while this handler runs: its close is
			// queued behind it.
			time.Sleep(100 * time.Millisecond)
			st.Show(ui.Toast{ID: "n", Title: "Hi", Duration: -1})
		}),
		ui.Toaster(&st, ui.ToasterProps{})))
	r.Click("#show")
	r.Click("#keep")
	r.Update()
	time.Sleep(50 * time.Millisecond)
	r.Update()
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Fatalf("the old timer closed the kept toast: aria-hidden=%s", got)
	}
}

// OnClose draws the corner x, which asks and closes nothing.
func TestToastOnClose(t *testing.T) {
	var st ui.ToasterState
	asked := 0
	r := start(t, page(
		button("show", "Show", func() { st.Show(ui.Toast{ID: "n", Title: "Hi", Duration: -1, OnClose: func() { asked++ }}) }),
		ui.Toaster(&st, ui.ToasterProps{})))
	r.Click("#show")
	if got := attr(r.One("#toast-1-close"), "aria-label"); got != "Dismiss" {
		t.Fatalf("the x is named %q", got)
	}
	if n := strings.Count(r.HTML(), `aria-label="Dismiss"`) + strings.Count(r.HTML(), `>Dismiss</button>`); n != 1 {
		t.Fatalf("%d controls named Dismiss, want the x alone", n)
	}
	r.Click("#toast-1-close")
	if asked != 1 {
		t.Fatalf("OnClose ran %d times", asked)
	}
	if got := attr(r.One("#toast-1"), "aria-hidden"); got != "false" {
		t.Fatalf("the x closed the toast itself: aria-hidden=%s", got)
	}
}

// The focused look and the bar are the kit's: in Stylesheet.
func TestToastStylesInStylesheet(t *testing.T) {
	css := ui.Stylesheet()
	for _, want := range []string{".toast>.toast-bar{", "@keyframes toast-bar", ".toast[data-focused] footer>.btn:first-child", ".toast-content[data-on~=click]:hover", ".toast>.toast-close"} {
		if !strings.Contains(css, want) {
			t.Errorf("Stylesheet lacks %q", want)
		}
	}
}

// OnClose without an ID is a fault: only Remove(id) takes the toast off.
func TestToastOnCloseNeedsID(t *testing.T) {
	f := &faults{T: t}
	var st ui.ToasterState
	st.Show(ui.Toast{Title: "Hi", OnClose: func() {}})
	surftest.Start(f, surf.New("toast", ui.Toaster(&st, ui.ToasterProps{}), ui.Stylesheet()))
	if !strings.Contains(strings.Join(f.got, "\n"), "OnClose needs an ID") {
		t.Fatalf("faults %q", f.got)
	}
}
