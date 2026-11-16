package ui

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// A Toast is a short notice that goes away by itself.
type Toast struct {
	Title, Description string
	// Category is Basecoat's toast category: "success" (the default, as
	// ""), "error", "info" or "warning". It picks the icon, and an error
	// is announced at once (role alert). surftest fails any other.
	Category string
	// Duration is how long it stays; 0 is Basecoat's default, and a
	// negative one stays until dismissed.
	Duration time.Duration
}

// ToasterState holds the toasts shown. The zero value shows none.
type ToasterState struct {
	toasts []*shownToast
	next   int  // the next toast's number, for its id
	paused bool // toast.js's isPaused: the pointer is over the toaster
}

// shownToast is a toast on show: toast.js's state for it.
type shownToast struct {
	Toast
	n      int
	hidden bool // closeToast: aria-hidden, until its transition ends
	left   time.Duration
	since  time.Time
	timer  *time.Timer
	s      *surf.Surface // the surface its timers reach, from its first render
}

// The categories Basecoat's toast draws, "" being success.
var toastCategories = []string{"", "success", "error", "info", "warning"}

// toastLeave is how long a closed toast stays hidden before it goes:
// toast.css's transition (duration-300), whose end toast.js waits for.
const toastLeave = 300 * time.Millisecond

// Show adds a toast, from a handler or from Surface.Do. It goes away after
// its Duration.
func (st *ToasterState) Show(t Toast) {
	st.next++
	left := t.Duration
	switch {
	case left < 0:
		left = -1
	case left == 0 && t.Category == "error":
		left = 5000 * time.Millisecond
	case left == 0:
		left = 3000 * time.Millisecond
	}
	st.toasts = append(st.toasts, &shownToast{Toast: t, n: st.next, left: left})
}

// Toaster is where the toasts show: Basecoat's toaster, its markup generated
// from toast.html.jinja, its behaviour toast.js's, translated to Go. A view
// places it once, last.
func Toaster(st *ToasterState) templ.Component {
	return toaster(st)
}

// start is initToast, at the toast's first render on a surface: its timer
// starts unless the toaster is paused.
func (st *ToasterState) start(ctx context.Context) string {
	s := surf.Of(ctx)
	if s == nil {
		return ""
	}
	for _, t := range st.toasts {
		if t.s == nil {
			t.s = s
			if !st.paused {
				st.arm(t)
			}
		}
	}
	return ""
}

// arm starts t's timer for what is left of its time.
func (st *ToasterState) arm(t *shownToast) {
	if t.left < 0 || t.hidden || t.s == nil {
		return
	}
	t.since = time.Now()
	s := t.s
	t.timer = time.AfterFunc(t.left, func() { s.Do(func() { st.close(t) }) })
}

// close is closeToast: the toast hides, and goes once its transition
// would have ended.
func (st *ToasterState) close(t *shownToast) {
	if t.hidden {
		return
	}
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.hidden = true
	if t.s == nil {
		st.remove(t)
		return
	}
	s := t.s
	time.AfterFunc(toastLeave, func() { s.Do(func() { st.remove(t) }) })
}

func (st *ToasterState) remove(t *shownToast) {
	st.toasts = slices.DeleteFunc(st.toasts, func(x *shownToast) bool { return x == t })
}

// hover is the toaster's mouseenter and mouseleave: pauseAllTimeouts and
// resumeAllTimeouts.
func (st *ToasterState) hover(over bool) {
	if over == st.paused {
		return
	}
	st.paused = over
	for _, t := range st.toasts {
		if t.hidden {
			continue
		}
		switch {
		case over && t.timer != nil:
			t.timer.Stop()
			t.timer = nil
			t.left -= time.Since(t.since)
		case !over && t.left >= 0 && t.timer == nil && t.s != nil:
			if t.left > 0 {
				st.arm(t)
			} else {
				st.close(t)
			}
		}
	}
}

func (t *shownToast) category() string { return or(t.Category, "success") }

func (t *shownToast) role() string {
	if t.Category == "error" {
		return "alert"
	}
	return "status"
}

func (t *shownToast) id() string { return "toast-" + strconv.Itoa(t.n) }

// duration is data-duration, which the macro writes when a duration is
// given: in ms, or -1.
func (t *shownToast) duration() string {
	switch {
	case t.Duration < 0:
		return "-1"
	case t.Duration > 0:
		return strconv.FormatInt(t.Duration.Milliseconds(), 10)
	}
	return ""
}

func checkToast(ctx context.Context, t *shownToast) string {
	if !slices.Contains(toastCategories, t.Category) {
		rt.Fault(ctx, fmt.Sprintf("ui.Toast: no category %q (it takes %s)", t.Category, list(toastCategories[1:])))
	}
	return ""
}

// toastIcons are the macro's icons, by category.
var toastIcons = map[string]string{
	"success": `<svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="m9 12 2 2 4-4"></path></svg>`,
	"error":   `<svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="m15 9-6 6"></path><path d="m9 9 6 6"></path></svg>`,
	"info":    `<svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M12 16v-4"></path><path d="M12 8h.01"></path></svg>`,
	"warning": `<svg aria-hidden="true" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"></path><path d="M12 9v4"></path><path d="M12 17h.01"></path></svg>`,
}
