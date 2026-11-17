package ui

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// toastCSS is the toasts' bar and focused look, part of Stylesheet.
//
//go:embed toast.css
var toastCSS string

// A Toast is a short notice that goes away by itself, or when the app
// removes it.
type Toast struct {
	// ID names a toast the app updates or removes. Show with the ID of a
	// toast still shown changes that toast in place: its place in the
	// stack, its element and, unless Duration changes, its timer are kept.
	// "" is a toast only its own buttons and timer close.
	//
	// A toast a button or its timer closed is gone: Show with its ID adds
	// a new one. An app that keeps its toasts by ID drops one from its own
	// list in each action's OnSelect, and gives a lasting toast an Outline
	// action (Basecoat's cancel, whose OnSelect it hears) in place of the
	// kit's Dismiss, which tells no one.
	ID                 string
	Title, Description string
	// Category is Basecoat's toast category: "success" (the default, as
	// ""), "error", "info" or "warning". It picks the icon, and an error
	// is announced at once (role alert). surftest fails any other.
	Category string
	// Duration is how long it stays; 0 is Basecoat's default, and a
	// negative one stays until dismissed, or until the app removes it.
	Duration time.Duration
	// Actions are its footer's buttons, in order. Choosing one calls its
	// OnSelect, then closes the toast. A lasting toast without an Outline
	// action (Basecoat's cancel) also has a Dismiss button.
	Actions []ToastAction
	// OnClick runs when the toast's body, outside its buttons, is clicked.
	// The toast stays: OnClick may Remove it. The body then answers the
	// pointer as a button does (the kit's hover fill).
	OnClick func()
	// Focused draws the toast as the one the app's keyboard is on: its
	// first button ringed, its bar still, its timer paused. A toast never
	// takes the host's keyboard; the app moves Focused with its own keys
	// and runs the button's OnSelect itself. Running the button's OnSelect
	// from the app's key doesn't close the toast as a click does; Remove
	// does.
	Focused bool
	// Countdown draws a bar along the toast's top edge that runs down to
	// nothing at this time; zero draws none. It runs from full when it
	// first shows. A later Show with another Countdown starts it full
	// again; a Show with the same one keeps it running. While Focused it
	// stands still where it is, whatever Countdown says, and when the focus
	// ends it runs on from there to the Countdown it is given then. The host
	// runs it, with no render while it runs, and it doesn't stop under the
	// pointer. It is a picture only: the toast goes by Duration, or when the
	// app removes it.
	Countdown time.Time
	// OnClose, when set, draws a close control in the toast's corner (an
	// x, named Dismiss). Clicking it runs OnClose and doesn't close the
	// toast: the app removes it (see ID).
	OnClose func()
}

// ToastAction is a toast footer button. OnSelect runs before the toast
// closes; a nil OnSelect only closes it. Label must be nonempty.
type ToastAction struct {
	Label string
	// Variant is the button's look: Primary (the default) or Outline
	// (Basecoat's cancel). Others are a button's too.
	Variant  Variant
	OnSelect func()
}

// ToasterState holds the toasts shown. The zero value shows none.
type ToasterState struct {
	toasts []*shownToast
	next   int  // the next toast's number, for its id
	paused bool // toast.js's isPaused: the pointer is over the toaster
	// onLeave is the last render's ToasterProps.OnLeave.
	onLeave func()
}

// ToasterProps are a Toaster's attributes.
type ToasterProps struct {
	// OnLeave runs when the pointer leaves the toaster, after its toasts'
	// timers resume; nil runs nothing.
	OnLeave func()
	// Class is the toaster's, for the app's own.
	Class string
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
	// The bar's run, which the kit keeps: its number (in the bar's id, so
	// a new run restarts it), the fraction of the bar it starts from, how
	// long it runs, and when it began. A still bar is drawn at from.
	bar     int
	barFrom float64
	barLife time.Duration
	barAt   time.Time
}

// toastNow is the toasts' clock, read when a bar's run starts or stops;
// tests set it.
var toastNow = time.Now

// maxBarSteps caps the bar's steps, ten a second: a long countdown asks
// the host for no more.
const maxBarSteps = 1200

// The categories Basecoat's toast draws, "" being success.
var toastCategories = []string{"", "success", "error", "info", "warning"}

// toastLeave is how long a closed toast stays hidden before it goes:
// toast.css's transition (duration-300), whose end toast.js waits for.
const toastLeave = 300 * time.Millisecond

// Show adds a toast, from a handler or from Surface.Do. It goes away after
// its Duration. A toast whose ID is one shown changes that toast instead
// (see Toast.ID).
func (st *ToasterState) Show(t Toast) {
	if old := st.byID(t.ID); old != nil {
		st.update(old, t)
		return
	}
	st.next++
	shown := &shownToast{Toast: t, n: st.next, left: lifetime(t)}
	shown.runBar(1)
	st.toasts = append(st.toasts, shown)
}

// Remove closes the toast whose ID is id, as its Dismiss button does,
// running no OnSelect. An id no toast shown has does nothing. Like Show, it
// runs in a handler or in Surface.Do.
func (st *ToasterState) Remove(id string) {
	if t := st.byID(id); t != nil {
		st.close(t)
	}
}

// byID is the toast shown, and not going, whose ID is id.
func (st *ToasterState) byID(id string) *shownToast {
	if id == "" {
		return nil
	}
	for _, t := range st.toasts {
		if t.ID == id && !t.hidden {
			return t
		}
	}
	return nil
}

// update is Show on a toast shown: its new fields, its timer kept unless
// Duration changed, its bar run again when it changed or a focus ended.
func (st *ToasterState) update(old *shownToast, t Toast) {
	if t.Duration != old.Duration {
		st.stop(old)
		old.left = lifetime(t)
	}
	if t.Focused {
		st.stop(old)
	}
	switch {
	case t.Focused && !old.Focused:
		// The bar stands where it is.
		old.barFrom = old.barNow()
		old.bar++
	case t.Focused:
		// Still, whatever Countdown says.
		t.Countdown = old.Countdown
	case old.Focused:
		// The focus ended: on from where it stood.
		from := old.barFrom
		old.Toast.Countdown = t.Countdown
		old.runBar(from)
	case !t.Countdown.Equal(old.Countdown):
		old.Toast.Countdown = t.Countdown
		old.runBar(1)
	}
	old.Toast = t
	st.resume(old)
}

// runBar starts the bar's run from the fraction from, to its Countdown.
func (t *shownToast) runBar(from float64) {
	t.bar++
	t.barFrom = from
	t.barAt = toastNow()
	t.barLife = max(t.Countdown.Sub(t.barAt), 0)
}

// barNow is the fraction of the bar left now, by the run's start.
func (t *shownToast) barNow() float64 {
	if t.barLife <= 0 {
		return 0
	}
	gone := toastNow().Sub(t.barAt)
	return t.barFrom * max(0, 1-float64(gone)/float64(t.barLife))
}

// lifetime is how long t stays: its Duration, Basecoat's defaults, or -1
// until dismissed.
func lifetime(t Toast) time.Duration {
	switch {
	case t.Duration < 0:
		return -1
	case t.Duration == 0 && t.Category == "error":
		return 5000 * time.Millisecond
	case t.Duration == 0:
		return 3000 * time.Millisecond
	}
	return t.Duration
}

// Toaster is where the toasts show: Basecoat's toaster, its markup generated
// from toast.html.jinja, its behaviour toast.js's, translated to Go. A view
// places it once, last.
func Toaster(st *ToasterState, p ToasterProps) templ.Component {
	return toaster(st, p)
}

// start is initToast, at the toast's first render on a surface: its timer
// starts unless the toaster is paused.
func (st *ToasterState) start(ctx context.Context, p ToasterProps) string {
	st.onLeave = p.OnLeave
	s := surf.Of(ctx)
	if s == nil {
		return ""
	}
	for _, t := range st.toasts {
		if t.s == nil {
			t.s = s
			st.resume(t)
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
	// A timer stopped after it fired, its close already queued, is
	// stale: only the toast's current timer closes it.
	var timer *time.Timer
	timer = time.AfterFunc(t.left, func() {
		s.Do(func() {
			if t.timer == timer {
				st.close(t)
			}
		})
	})
	t.timer = timer
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

func (st *ToasterState) choose(t *shownToast, action *ToastAction) {
	if t.hidden {
		return
	}
	if action != nil && action.OnSelect != nil {
		action.OnSelect()
	}
	st.close(t)
}

// dismiss is whether t has the Dismiss button: it lasts, and no action is
// its cancel.
func (t *shownToast) dismiss() bool {
	return t.Duration < 0 && !slices.ContainsFunc(t.Actions, func(a ToastAction) bool { return a.Variant == Outline })
}

func (st *ToasterState) closeAsked(t *shownToast) {
	if !t.hidden && t.OnClose != nil {
		t.OnClose()
	}
}

func (st *ToasterState) click(t *shownToast) {
	if !t.hidden && t.OnClick != nil {
		t.OnClick()
	}
}

// barStyle is the bar's place and run, from the values its run stored
// (a render reads no clock): what is gone cut off each end evenly, run
// down over its life in tenths of a second.
func (t *shownToast) barStyle() templ.SafeCSS {
	cut := (1 - t.barFrom) / 2
	if t.Focused {
		// Still: only where it stands, so the document doesn't change with
		// the clock.
		return templ.SafeCSS("--toast-bar-cut:" + strconv.FormatFloat(cut, 'f', 4, 64))
	}
	steps := min(maxBarSteps, max(1, int(t.barLife/(100*time.Millisecond))))
	return templ.SafeCSS(fmt.Sprintf("--toast-bar-life:%dms;--toast-bar-steps:%d;--toast-bar-cut:%s",
		t.barLife.Milliseconds(), steps, strconv.FormatFloat(cut, 'f', 4, 64)))
}

func (t *shownToast) barID() string { return t.id() + "-bar-" + strconv.Itoa(t.bar) }

func (st *ToasterState) remove(t *shownToast) {
	st.toasts = slices.DeleteFunc(st.toasts, func(x *shownToast) bool { return x == t })
}

// hover is the toaster's mouseenter and mouseleave: pauseAllTimeouts and
// resumeAllTimeouts, then OnLeave.
func (st *ToasterState) hover(over bool) {
	if over == st.paused {
		return
	}
	st.paused = over
	for _, t := range st.toasts {
		if over {
			st.stop(t)
		} else {
			st.resume(t)
		}
	}
	if !over && st.onLeave != nil {
		st.onLeave()
	}
}

// stop pauses t's timer, keeping what is left of its time.
func (st *ToasterState) stop(t *shownToast) {
	if t.timer == nil {
		return
	}
	t.timer.Stop()
	t.timer = nil
	// Paused past its deadline (the timer late), the toast has 0 left and
	// closes on resume: any negative left would read as -1, "until
	// dismissed".
	t.left = max(t.left-time.Since(t.since), 0)
}

// resume runs t's timer on, unless the toaster is paused or t is focused.
func (st *ToasterState) resume(t *shownToast) {
	if st.paused || t.Focused || t.left < 0 || t.hidden || t.timer != nil || t.s == nil {
		return
	}
	if t.left > 0 {
		st.arm(t)
	} else {
		st.close(t)
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
	for _, action := range t.Actions {
		if action.Label == "" {
			rt.Fault(ctx, "ui.ToastAction: Label is required")
		}
	}
	for _, action := range t.Actions {
		checkVariant(ctx, "ToastAction", action.Variant, buttonVariants)
	}
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
