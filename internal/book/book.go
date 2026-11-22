// Package book is the storybook, `hotty-surf book`: every story of the
// kit (package stories) in a list, with hotty and cells rendition tabs for
// the chosen story. Both share one view and state.
//
// On a hotty host the cells pane is a Screen in the book's own process,
// its lines drawn in the book's surface. The pane takes the keyboard as
// one element: every key but Tab reaches the program while it has it
// (SPEC §10.2), and goes to the Screen. Tab is the host's, so the pane
// sits between two empty elements Tab reaches: Tab onto the one after it
// is Tab in the Screen, and the keyboard goes back to the pane, until the
// Screen's last control, where it stays on the empty element and the
// next Tab leaves. Shift+Tab is the same, backwards.
//
// On a terminal that is no host, run draws the whole book in surftui.
package book

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:4096 maps:64
//msgp:ignore App shown seg

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/internal/dev/hook"
	storybook "neuroplast.io/hotty-surf/internal/book/model"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/internal/book/stories"
	"neuroplast.io/hotty-surf/surfdev"
	"neuroplast.io/hotty-surf/surfapp"
	"neuroplast.io/hotty-surf/surfui"
	bookcode "neuroplast.io/hotty-surf/surfui/code"
	"neuroplast.io/hotty-surf/surfui/eventlog"
	"neuroplast.io/hotty-surf/surfui/navlist"
	uiselect "neuroplast.io/hotty-surf/surfui/select"
)

var kit = stories.Registry()

//go:embed book.css
var bookCSS string

// CSS is the book's own stylesheet and the code blocks'. The kit's is
// linked, not carried (newSurface).
var CSS = bookCSS + bookcode.Stylesheet()

// The cells pane's size. A story is at most 28rem wide (max-w-md), which
// cells takes as 56 columns.
const (
	paneCols = 96
	paneRows = 40
)

// State is what the book keeps across a live surfdev.
type State struct {
	ID   string             `msg:"id"` // the story shown
	Book storybook.DevState `msg:"book"`
	Mode uint8              `msg:"mode"`
}

// App is the book.
type App struct {
	State
	all   []stories.Story
	cur   *shown
	model *storybook.Book
	nav   navlist.NavList

	// The pane's keyboard. inPane is set while the pane has it, and on the
	// way between it and the elements round it; refocus asks the next
	// render to give it back to the pane, and toList to give it to the
	// list (Tab past the pane's last control).
	inPane     bool
	refocus    bool
	toList     bool
	startFocus bool
	wantFocus  string
	modalOpen  bool
	gripStart  [2]int
	gripOrigin [2]int
	gripping   bool
	caretTick   uint
	cols        int
	propSelects map[string]*uiselect.State
}

// own are the book's elements that take the keyboard: while one has it, a
// story's ask for the keyboard (a dialog that opens as the list lands on
// its story, a menu opened from the pane) doesn't take it (keep).
func own(id string) bool {
	return strings.HasPrefix(id, "book-nav-") || id == "book-pane" || id == "book-pane-before" || id == "book-pane-after"
}

// shown is the story on show: its view, which the preview and the pane
// both draw, and the pane's surface on its Screen.
type shown struct {
	st    stories.Story
	view  templ.Component
	pane  *surf.Surface
	scr   *surftui.Screen
	focus string // the id the Screen's keyboard is on, "" for none
	err   error  // what went wrong in the pane's last update, nil for nothing
	// controls says whether the story has anything that takes the
	// keyboard; the pane takes no Tab when it hasn't.
	controls bool
	// retried is set once a refused focus was tried again (ENOTARGET):
	// a second refusal leaves the keyboard where it is.
	retried bool
	// baseTop and baseLeft record the story's initial trimmed origin so
	// floating tooltips that appear in trimmed padding rows/cols offset the
	// pane negatively instead of shifting the underlying controls.
	hasBase           bool
	baseTop, baseLeft int
	imgs              map[string]imgMeta
}

// New is a book of every story, showing the first.
func New() *App {
	a := &App{all: stories.BookAll(), startFocus: true}
	var ss []*storybook.Story
	for _, st := range a.all {
		ss = append(ss, storybook.Wrap(st.ID, st.Doc, st.Consolidated, st.Score, st.View, st.Props))
	}
	slices.SortStableFunc(ss, func(x, y *storybook.Story) int {
		if c := storybook.GroupOrder(x.Group) - storybook.GroupOrder(y.Group); c != 0 {
			return c
		}
		if c := strings.Compare(x.Group, y.Group); c != 0 {
			return c
		}
		xbase, _, xv := strings.Cut(x.Name, " / ")
		ybase, _, yv := strings.Cut(y.Name, " / ")
		if c := strings.Compare(xbase, ybase); c != 0 {
			return c
		}
		if xv != yv {
			if xv {
				return 1
			}
			return -1
		}
		// A component's variants stay in the order its stories were
		// added (stories.BookAll), as the spike listed them.
		return 0
	})
	a.model = storybook.New(ss)
	a.nav = navlist.NavList{ID: "book-nav", Label: "Stories", FilterLabel: "Filter stories", Placeholder: "Filter stories  ( / )", Empty: "No story matches."}
	a.nav.OnSelect = a.show
	a.nav.OnFilter = a.model.SetFilter
	a.model.SetRegion(storybook.RegionList)
	if len(a.model.Stories) > 0 {
		a.show(a.model.Stories[0].ID())
	}
	a.sync()
	return a
}

// Main shows the book until ctx is done or the user quits.
func Main(ctx context.Context) error {
	a := New()
	s := newSurface("book", View(a))
	surfdev.Keep(a.Save, a.Restore)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.watchHost(runCtx, s)
	return surfapp.Run(ctx, s)
}

func (a *App) paneWidth() int {
	if a.cols > 0 {
		return a.cols
	}
	return paneCols
}

func (a *App) paneSteps() string {
	return fmt.Sprintf("%d %d", a.paneWidth(), paneRows)
}

func (a *App) syncPlacedCols(s *surf.Surface) bool {
	if s == nil {
		return false
	}
	w, _ := devhook.Area(s)
	if w <= 0 {
		return false
	}
	// Sidebar (~24ch) + preview horizontal padding (~6ch).
	cols := max(w-30, 60)
	if cols == a.cols {
		return false
	}
	a.cols = cols
	if a.cur != nil && a.cur.scr != nil {
		a.cur.scr.Resize(cols, paneRows)
	}
	return true
}

func (a *App) watchHost(ctx context.Context, s *surf.Surface) {
	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.Do(func() {
				a.syncPlacedCols(s)
				if a.mode() == surfui.ModeHotty || a.cur == nil || a.cur.scr == nil {
					return
				}
				if a.ID == "progress/default" {
					a.cur.scr.Resize(a.paneWidth(), paneRows)
				}
			})
		}
	}
}

func (a *App) Save() *State {
	a.State.Book = *a.model.Save()
	return &a.State
}

func (a *App) Restore(s *State) {
	a.Mode = s.Mode
	a.model.Load(&s.Book)
	if s.ID != "" {
		a.show(s.ID)
	}
}

// show opens the story id: a new view, with state of its own, and its
// pane.
func (a *App) show(id string) {
	for _, st := range a.all {
		if st.ID != id {
			continue
		}
		a.ID = id
		a.nav.Current = id
		story := a.model.Find(id)
		a.model.Select(story)
		view := story.HTML(a.model.Env(story))
		wrapped := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			f := rt.From(ctx)
			start := 0
			if f != nil {
				start = len(f.Order)
			}
			if err := view.Render(ctx, w); err != nil {
				return err
			}
			if f != nil {
				ids := append([]string(nil), f.Order[start:]...)
				f.AddAfter(func() { a.observeHandlers(f, story, ids) })
			}
			return nil
		})
		a.cur, a.inPane, a.refocus, a.toList = &shown{st: st, view: wrapped}, false, false, false
		return
	}
}

// observeHandlers keeps the story's own handlers intact while recording
// the events they receive, on both its host and its cells surface.
func (a *App) observeHandlers(f *rt.Frame, story *storybook.Story, ids []string) {
	for _, id := range ids {
		h := f.Get(id)
		if h == nil || strings.HasPrefix(id, "book-") {
			continue
		}
		log := func(kind, value string) { a.model.Log(story, kind, id+value) }
		if fn := h.Click; fn != nil {
			h.Click = func() { log("click", ""); fn() }
		}
		if fn := h.Input; fn != nil {
			h.Input = func(v string) { log("input", fmt.Sprintf(" = %q", v)); fn(v) }
		}
		if fn := h.Check; fn != nil {
			h.Check = func(v bool) { log("check", fmt.Sprintf(" = %t", v)); fn(v) }
		}
		if fn := h.Submit; fn != nil {
			h.Submit = func() { log("submit", ""); fn() }
		}
		for i, fn := range h.Key {
			h.Key[i] = func(k string) bool {
				log("key", fmt.Sprintf(" = %q", k))
				return fn(k)
			}
		}
	}
}

var paneDone = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// paneKeys are CUA editing keys declared on #book-pane so the host forwards
// them to PaneKey (and into surftui.Screen) rather than intercepting them.
var paneKeys = []string{
	"ctrl+a", "ctrl+c", "ctrl+x", "ctrl+v",
	"ctrl+backspace", "ctrl+delete",
	"ctrl+left", "ctrl+right", "ctrl+shift+left", "ctrl+shift+right",
	"ctrl+home", "ctrl+end", "ctrl+shift+home", "ctrl+shift+end",
	"ctrl+z", "ctrl+y", "ctrl+shift+z",
}

// start puts the story on the pane's Screen, the first time the pane is
// drawn or used: not before the book runs, whose live reload has to be
// set up before a view renders.
func (sh *shown) start(mode func() surfui.Mode) {
	if sh.pane != nil {
		return
	}
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		m := mode()
		ctx = surfui.WithMode(ctx, m)
		ctx = surfui.WithKitty(ctx, m == surfui.ModeTUI)
		return paneRoot(sh.view).Render(ctx, w)
	})
	sh.pane = surf.New("book-pane", view, "")
	// sh.pane is driven synchronously by book's own render (sh.pump), not by
	// an independent driver goroutine; mark it done so devchild's live-reload
	// handover never stalls 4s waiting on it.
	devhook.Driver(sh.pane, paneDone)
	sh.pane.Link(kit.Sheet())
	sh.scr = surftui.New(paneCols, paneRows, func(e hotty.Event) {
		switch e.Kind {
		case hotty.EventFocus:
			sh.focus = e.Target
		case hotty.EventBlur:
			sh.focus = ""
		}
		if err := sh.pane.Event(e); err != nil && sh.err == nil {
			sh.err = err
		}
	})
	sh.pane.Attach(sh.scr, hotty.Caps{})
	doc, err := sh.pane.Document()
	if err == nil {
		sh.controls = hasControl(doc)
		sh.imgs = avatarImages(doc)
		_, err = sh.scr.Write([]byte(hotty.Doc(sh.pane.Name(), doc)))
	}
	sh.err = err
}

// pump sends the pane's surface what changed in the view: its own
// handlers' changes, and the preview's.
func (sh *shown) pump(mode func() surfui.Mode) {
	sh.start(mode)
	err := sh.pane.Update()
	if err != nil && strings.Contains(err.Error(), hotty.ENOTARGET) {
		// The Screen refused the element the surface asked to focus: it
		// is gone (a dialog closing as the keyboard went back to it). The
		// surface took the move already, so it hears that it didn't
		// happen, and picks again from its tree as it is now, once, as a
		// program on a host does (surftea).
		err = sh.pane.Event(hotty.Event{Kind: hotty.EventBlur, Surface: sh.pane.Name()})
		if err == nil && !sh.retried {
			sh.retried = true
			if _, err = sh.pane.Focus(); err == nil {
				err = sh.pane.Update()
			}
		}
		if err != nil && strings.Contains(err.Error(), hotty.ENOTARGET) {
			err = nil // refused again: the keyboard stays where it is
		}
	}
	sh.err = err
}

// sync adapts the book model's groups and selection to the spike NavList.
func (a *App) sync() {
	a.nav.Groups = nil
	for _, st := range a.model.Stories {
		n := len(a.nav.Groups)
		if n == 0 || a.nav.Groups[n-1].Name != st.Group {
			a.nav.Groups = append(a.nav.Groups, navlist.Group{Name: st.Group})
			n++
		}
		a.nav.Groups[n-1].Items = append(a.nav.Groups[n-1].Items, navlist.Item{ID: st.ID(), Label: st.Name, Dot: st.Dot})
	}
	a.nav.Current, a.nav.Filter = a.ID, a.model.Filter
}

func (a *App) startingList(ctx context.Context) bool {
	if f := rt.From(ctx); f != nil && f.Focused() != "" {
		a.startFocus = false
	}
	return a.startFocus
}

func (a *App) bookKey(k string) bool {
	if k == "mod+c" || k == "ctrl+c" {
		return true
	}
	if k == "r" {
		a.stepMode(1)
		return true
	}
	if k == "escape" && a.modalOpen {
		return false
	}
	if !a.model.Key(k) {
		return false
	}
	if a.model.Sel != nil && a.model.Sel.ID() != a.ID {
		a.show(a.model.Sel.ID())
	}
	switch a.model.Wanted() {
	case storybook.RegionList:
		a.wantFocus = a.nav.ItemID(a.ID)
	case storybook.RegionFilter:
		a.wantFocus = a.nav.FilterID()
	case storybook.RegionPreview:
		a.wantFocus = "book-preview"
	case storybook.RegionTabs:
		i := 1
		for n, tab := range storybook.Tabs {
			if tab == a.model.Tab {
				i = n + 1
			}
		}
		a.wantFocus = "book-tabs-tab-" + itoa(i)
	case storybook.RegionPanel:
		a.wantFocus = "book-panel"
	}
	return true
}

func (a *App) mode() surfui.Mode {
	m := surfui.Mode(a.Mode)
	if int(m) >= len(surfui.Modes) {
		return surfui.ModeHotty
	}
	return m
}

// paneMode returns the surfui.Mode the cell-grid pane renders in (ModeTUI when the
// book itself is on ModeHotty so tests inspecting paneText work directly).
func (a *App) paneMode() surfui.Mode {
	if m := a.mode(); m != surfui.ModeHotty {
		return m
	}
	return surfui.ModeTUI
}

func (a *App) setMode(m surfui.Mode) {
	if a.mode() != m && a.inPane {
		a.PaneOutside()
	}
	a.Mode = uint8(m)
}

func (a *App) stepMode(delta int) {
	n := len(surfui.Modes)
	cur := int(a.mode())
	next := (cur + delta%n + n) % n
	a.setMode(surfui.Modes[next])
}

func (a *App) modeKey(k string) bool {
	switch k {
	case "arrow_left":
		a.stepMode(-1)
		return true
	case "arrow_right":
		a.stepMode(1)
		return true
	case "home":
		a.setMode(surfui.Modes[0])
		return true
	case "end":
		a.setMode(surfui.Modes[len(surfui.Modes)-1])
		return true
	}
	return false
}

func (a *App) focusRequest(ctx context.Context) string {
	f := rt.From(ctx)
	if f == nil {
		return ""
	}
	if a.wantFocus != "" && f.Focused() == a.wantFocus {
		a.wantFocus = ""
	}
	if a.wantFocus != "" {
		return a.wantFocus
	}
	switch {
	case strings.HasPrefix(f.Focused(), a.nav.ID+"-item-"):
		a.model.SetRegion(storybook.RegionList)
	case f.Focused() == a.nav.FilterID():
		a.model.SetRegion(storybook.RegionFilter)
	case strings.HasPrefix(f.Focused(), "book-tabs-tab-"):
		a.model.SetRegion(storybook.RegionTabs)
	case f.Focused() == "book-panel" || strings.HasPrefix(f.Focused(), "book-control-"):
		a.model.SetRegion(storybook.RegionPanel)
	case f.Focused() != "":
		a.model.SetRegion(storybook.RegionPreview)
	}
	return ""
}

func (a *App) resetStory() {
	if a.model.Sel != nil {
		a.model.Reset(a.model.Sel)
		a.show(a.ID)
	}
}

func (a *App) clearEvents() {
	if a.model.Sel != nil {
		a.model.ClearEvents(a.model.Sel)
	}
}

func noTab(attrs templ.Attributes) templ.Attributes {
	attrs["tabindex"] = "-1"
	return attrs
}

func (a *App) setProp(path string) func(string) {
	return func(value string) { a.model.Set(a.model.Sel, path, value) }
}

func (a *App) propSelect(path string) *uiselect.State {
	if a.propSelects == nil {
		a.propSelects = map[string]*uiselect.State{}
	}
	key := a.ID + ":" + path
	st := a.propSelects[key]
	if st == nil {
		st = &uiselect.State{}
		a.propSelects[key] = st
	}
	return st
}

func propSelectItems(opts []string) []uiselect.Item {
	items := make([]uiselect.Item, len(opts))
	for i, o := range opts {
		items[i] = uiselect.Item{Value: o, Label: o}
	}
	return items
}

func (a *App) entries() []eventlog.Entry {
	var out []eventlog.Entry
	for _, e := range a.model.Events(a.model.Sel) {
		kind := ""
		switch e.Kind {
		case "click", "submit", "key":
			kind = "keyword"
		case "input", "check":
			kind = "string"
		case "control":
			kind = "literal"
		}
		out = append(out, eventlog.Entry{At: e.At, Kind: e.Kind, Text: e.Text, KindClass: bookcode.TokenClass(kind)})
	}
	return out
}

func propControlID(path string) string {
	return "book-ctl-" + strings.ReplaceAll(path, ".", "-")
}

func (a *App) storyHTML() string {
	if a.model.Sel == nil {
		return ""
	}
	var b strings.Builder
	if err := a.model.Sel.HTML(nil).Render(surfui.WithMode(context.Background(), a.mode()), &b); err != nil {
		return "<!-- " + err.Error() + " -->"
	}
	return prettyHTML(b.String())
}

// PaneKey is a key the host passed the program while the pane has the
// keyboard: the Screen's, or else the story's Key handlers'.
func (a *App) PaneKey(k string) bool {
	sh := a.cur
	if sh == nil || !a.inPane {
		// Not the pane's: a key while nothing in the book has the
		// keyboard reaches every outermost Key handler (the list's too).
		return false
	}
	sh.start(a.paneMode)
	// surf.Key normalizes the platform modifier to "mod+"; surftui.Screen
	// expects physical key names ("ctrl+").
	ck := strings.Replace(k, "mod+", "ctrl+", 1)
	taken := sh.scr.Key(ck)
	if !taken {
		var err error
		taken, err = sh.pane.Key(k)
		if err != nil && sh.err == nil {
			sh.err = err
		}
	}
	sh.pump(a.paneMode)
	if taken {
		a.caretTick++
	}
	return taken
}

// PaneFocus hears the pane take the keyboard. Its losing it is heard
// where the keyboard goes: the elements round it (Edge), or a click
// elsewhere (PaneOutside). So Tab between the pane and them is no blur
// in the story, which would close what it has open.
func (a *App) PaneFocus(in bool) {
	sh := a.cur
	if !in || sh == nil {
		return
	}
	a.inPane = true
	sh.start(a.paneMode)
	sh.retried = false
	sh.scr.Focus(true)
	sh.pump(a.paneMode)
}

// PaneOutside hears the keyboard leave the pane for elsewhere: a click
// outside it, or the surface losing the keyboard.
func (a *App) PaneOutside() {
	sh := a.cur
	a.inPane = false
	if sh == nil || sh.pane == nil {
		return
	}
	sh.scr.Focus(false)
	sh.pump(a.paneMode)
}

// Edge is the keyboard on the empty element before the pane (back) or
// after it. From the pane, it is Shift+Tab (or Tab) in the Screen: when
// the Screen has a control to go to, the keyboard goes back to the pane;
// past its first (or last), it stays here, and the next Shift+Tab (or
// Tab) leaves. From outside, it enters the pane, at the control it had
// or its first (or last).
func (a *App) Edge(back bool) func(bool) {
	return func(in bool) {
		sh := a.cur
		if !in || sh == nil {
			return
		}
		sh.start(a.paneMode)
		key := "tab"
		if back == a.inPane {
			key = "shift+tab"
		}
		taken := true
		if a.inPane {
			taken = sh.scr.Key(key)
		} else {
			sh.scr.Focus(true)
			if sh.focus == "" {
				taken = sh.scr.Key(key)
			}
		}
		// Tab past the Screen's last control takes the key, and leaves
		// nothing focused.
		taken = taken && sh.focus != ""
		if !taken {
			sh.scr.Focus(false)
		}
		sh.pump(a.paneMode)
		// Tab past the pane's last control goes on to the list, where Tab
		// past the book's last element goes; Shift+Tab past its first
		// stays on "in cells", the element before it.
		a.toList = !taken && a.inPane && !back
		a.inPane, a.refocus = taken, taken
	}
}

// inert keeps the cells pane out of the keyboard sequence while a story's
// modal dialog is open outside it.
func (a *App) inert(ctx context.Context) bool {
	f := rt.From(ctx)
	a.modalOpen = f != nil && len(f.Modal) > 0
	return f != nil && len(f.Modal) > 0 && !(own(f.Focused()) && !strings.HasPrefix(f.Focused(), a.nav.ID+"-item-"))
}

// keep has the last word on what the render asks for the keyboard: a
// story's ask doesn't take it from the book's own elements (own). It runs
// after every other hook (rt.Frame.After), a dialog's included.
func (a *App) keep(ctx context.Context) {
	f := rt.From(ctx)
	if f == nil {
		return
	}
	last := func() {
		if own(f.Focused()) && f.Focus != "" && !own(f.Focus) && f.Focus != a.wantFocus {
			f.Focus = ""
		}
	}
	f.AddAfter(func() { f.AddAfter(last) })
}

// takeToList says, once, that this render gives the list the keyboard.
func (a *App) takeToList() bool {
	r := a.toList
	a.toList = false
	return r
}

// takeRefocus says, once, that this render gives the pane the keyboard.
func (a *App) takeRefocus() bool {
	r := a.refocus
	a.refocus = false
	return r
}

// seg is a run of a line's cells in one look and colour pair.
type seg struct {
	text  string
	class string
	style string
}

// hitBox is the unified cell bounding box of one interactive element in the
// trimmed cells preview, so multi-row controls (like 3-row lg buttons) get
// one unified pointer target, hover, click, and :active press effect.
type hitBox struct {
	id               string
	x, y, w, h       int
	col, row         int // untrimmed screen coordinates for Click/Move
	isText           bool
	disabled         bool
	focused          bool
	hasGrip          bool
	gripCol, gripRow int
	resizeMode       string
}

func (h hitBox) style() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf(
		"left:%dch;top:calc(%d*var(--hotty-cell-h));width:%dch;height:calc(%d*var(--hotty-cell-h))",
		h.x, h.y, h.w, h.h,
	))
}

func (h hitBox) gripStyle() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf(
		"left:%dch;top:calc(%d*var(--hotty-cell-h));width:1ch;height:var(--hotty-cell-h)",
		h.x+h.w-1, h.y+h.h-1,
	))
}

// caretPos is the blinking I-beam caret position in cell coordinates inside
// the trimmed cells preview.
type caretPos struct {
	has  bool
	x, y int
	alt  bool // toggles class on each keystroke so CSS restarts the blink (solid while typing)
}

func (c caretPos) style() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf(
		"left:%dch;top:calc(%d*var(--hotty-cell-h))",
		c.x, c.y,
	))
}

// focusRect is the bounding box in cell coordinates of the currently
// focused (Reverse) element inside the trimmed cells preview.
type focusRect struct {
	has        bool
	x, y, w, h int
}

func (f focusRect) style() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf(
		"left:%dch;top:calc(%d*var(--hotty-cell-h));width:%dch;height:calc(%d*var(--hotty-cell-h))",
		f.x, f.y, f.w, f.h,
	))
}

// paneView holds the trimmed cell lines, unified interactive hit boxes, and
// unified focus ring for the cells preview.
type imgMeta struct {
	src     string
	size    string
	inGroup bool
	online  bool
}

type imgBox struct {
	id         string
	src        string
	size       string
	inGroup    bool
	online     bool
	x, y, w, h int
}

func (im imgBox) style() templ.SafeCSS {
	d := "1.55"
	if im.inGroup {
		d = "1.45"
	} else {
		switch im.size {
		case "sm":
			d = "1.15"
		case "lg":
			d = "1.95"
		}
	}
	dia := fmt.Sprintf("calc(%s*var(--hotty-cell-h))", d)
	left := fmt.Sprintf("calc(%dch + (%dch - %s)/2)", im.x, im.w, dia)
	top := fmt.Sprintf("calc((%d + %d/2)*var(--hotty-cell-h) - %s/2)", im.y, im.h, dia)
	ring := ""
	if im.inGroup {
		ring = ";box-shadow:0 0 0 2px var(--background)"
	}
	return templ.SafeCSS(fmt.Sprintf("left:%s;top:%s;width:%s;height:%s%s", left, top, dia, dia, ring))
}

func avatarImages(doc string) map[string]imgMeta {
	if !strings.Contains(doc, "data-avatar-img") {
		return nil
	}
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return nil
	}
	var out map[string]imgMeta
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inGroup bool) {
		if n.Type == html.ElementNode {
			var id, src, sz string
			var online bool
			for _, a := range n.Attr {
				switch a.Key {
				case "id":
					id = a.Val
				case "data-avatar-img":
					src = a.Val
				case "data-size":
					sz = a.Val
				case "data-online":
					online = a.Val == "true"
				case "data-slot":
					if a.Val == "avatar-group" {
						inGroup = true
					}
				}
			}
			if id != "" && src != "" {
				if out == nil {
					out = map[string]imgMeta{}
				}
				out[id] = imgMeta{src: src, size: sz, inGroup: inGroup, online: online}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inGroup)
		}
	}
	walk(root, false)
	return out
}

type paneView struct {
	lines      [][]seg
	hits       []hitBox
	imgs       []imgBox
	focus      focusRect
	caret      caretPos
	offX, offY int // negative cell offset when an overlay (tooltip) expands into trimmed padding
}

func (pv paneView) paneStyle() templ.SafeCSS {
	if pv.offX == 0 && pv.offY == 0 {
		return ""
	}
	var parts []string
	if pv.offY != 0 {
		parts = append(parts, fmt.Sprintf("margin-top:calc(%d*var(--hotty-cell-h))", pv.offY))
	}
	if pv.offX != 0 {
		parts = append(parts, fmt.Sprintf("margin-left:%dch", pv.offX))
	}
	return templ.SafeCSS(strings.Join(parts, ";"))
}

// lines are the pane's lines as runs from Cells (with colours and
// attributes), the leading/trailing blank lines left out, plus unified hit
// boxes and focus bounding box.
func (a *App) lines() paneView {
	sh := a.cur
	if sh == nil {
		return paneView{}
	}
	sh.pump(a.paneMode)
	cols := a.paneWidth()
	if sh.scr != nil {
		sh.scr.Resize(cols, paneRows)
	}
	mode := a.paneMode()
	grid := sh.scr.Cells()
	out := make([][]seg, len(grid))
	for i, row := range grid {
		if mode == surfui.ModePlain {
			out[i] = plainRowSegs(row)
		} else {
			out[i] = rowSegs(row, false)
		}
	}
	// The blank lines at the top (at most 1 standard p-4 padding row, so
	// intentional top breathing room like pt-12 for top tooltips stays) and
	// end go; a line of spaces with a look or fill (an empty field,
	// underlined) is no blank.
	trimTop := 0
	for len(out) > 0 && trimTop < 1 && blank(out[0]) {
		out = out[1:]
		trimTop++
	}
	rowHasImg := func(r int) bool {
		if len(sh.imgs) == 0 {
			return false
		}
		for c := range grid[r] {
			if id, _ := sh.scr.At(c, r); sh.imgs[id].src != "" {
				return true
			}
		}
		return false
	}
	for len(out) > 0 && blank(out[len(out)-1]) && !rowHasImg(trimTop+len(out)-1) {
		out = out[:len(out)-1]
	}
	// Strip common leading blank columns (e.g. a story root's p-4 padding,
	// which hotty mode also zeroes via `.book-preview > .p-4 { padding: 0 }`).
	trimLeft := cols
	for r := trimTop; r < trimTop+len(out); r++ {
		row := grid[r]
		lead := 0
		for lead < len(row) {
			c := row[lead]
			if c.Attr != 0 || c.Fg != "" || c.Bg != "" || (c.Text != "" && c.Text != " ") {
				break
			}
			lead++
		}
		if lead < len(row) && lead < trimLeft {
			trimLeft = lead
		}
	}
	if trimLeft >= cols {
		trimLeft = 0
	}
	for i := range out {
		row := grid[trimTop+i][trimLeft:]
		if mode == surfui.ModePlain {
			out[i] = plainRowSegs(row)
		} else {
			out[i] = rowSegs(row, mode == surfui.ModeTUI)
		}
	}
	var pv paneView
	pv.lines = out
	if !sh.hasBase {
		sh.hasBase, sh.baseTop, sh.baseLeft = true, trimTop, trimLeft
	} else {
		if trimTop < sh.baseTop {
			pv.offY = trimTop - sh.baseTop
		}
		if trimLeft < sh.baseLeft {
			pv.offX = trimLeft - sh.baseLeft
		}
	}
	if !mode.Interactive() {
		return pv
	}
	minX, minY, maxX, maxY := 1<<30, 1<<30, -1, -1
	byID := map[string]int{}
	imgByID := map[string]int{}
	for r := trimTop; r < trimTop+len(out); r++ {
		ry := r - trimTop
		for c := trimLeft; c < len(grid[r]); c++ {
			cell := grid[r][c]
			rx := c - trimLeft
			if cell.Attr&surftui.Reverse != 0 && cell.Bg != "input/30" {
				if rx < minX {
					minX = rx
				}
				if rx > maxX {
					maxX = rx
				}
				if ry < minY {
					minY = ry
				}
				if ry > maxY {
					maxY = ry
				}
			}
			if cell.Attr&surftui.Dim != 0 && cell.Bg != "input/30" {
				continue
			}
			if id, _ := sh.scr.At(c, r); id != "" && id != "book-pane-cua" {
				if meta, isImg := sh.imgs[id]; isImg {
					if idx, ok := imgByID[id]; ok {
						im := &pv.imgs[idx]
						if rx < im.x {
							im.w += im.x - rx
							im.x = rx
						}
						if ry < im.y {
							im.h += im.y - ry
							im.y = ry
						}
						if rx+1 > im.x+im.w {
							im.w = rx + 1 - im.x
						}
						if ry+1 > im.y+im.h {
							im.h = ry + 1 - im.y
						}
					} else {
						imgByID[id] = len(pv.imgs)
						pv.imgs = append(pv.imgs, imgBox{id: id, src: meta.src, size: meta.size, inGroup: meta.inGroup, online: meta.online, x: rx, y: ry, w: 1, h: 1})
					}
					continue
				}
				if idx, ok := byID[id]; ok {
					hb := &pv.hits[idx]
					if cell.Attr&surftui.Dim != 0 {
						hb.disabled = true
					}
					if rx < hb.x {
						hb.w += hb.x - rx
						hb.x = rx
					}
					if rx >= hb.x+hb.w {
						hb.w = rx - hb.x + 1
					}
					if ry < hb.y {
						hb.h += hb.y - ry
						hb.y = ry
					}
					if ry >= hb.y+hb.h {
						hb.h = ry - hb.y + 1
					}
				} else {
					byID[id] = len(pv.hits)
					pv.hits = append(pv.hits, hitBox{
						id: id, x: rx, y: ry, w: 1, h: 1, col: c, row: r,
						disabled: cell.Attr&surftui.Dim != 0,
					})
				}
			}
		}
	}
	if col, row, ok := sh.scr.Cursor(); ok && a.inPane {
		cx, cy := col-trimLeft, row-trimTop
		if cx >= 0 && cy >= 0 && cy < len(out) {
			pv.caret = caretPos{has: true, x: cx, y: cy, alt: a.caretTick%2 == 1}
		}
	}
	for i := range pv.hits {
		hb := &pv.hits[i]
		hb.focused = a.inPane && sh.focus == hb.id
		if hb.h >= 2 {
			brRow := trimTop + hb.y + hb.h - 1
			brCol := trimLeft + hb.x + hb.w - 1
			if brRow < len(grid) && brCol < len(grid[brRow]) {
				br := grid[brRow][brCol]
				if br.Text == "◢" {
					hb.isText = true
					hb.hasGrip = true
					hb.gripCol, hb.gripRow = brCol, brRow
					hb.resizeMode = sh.scr.ResizeMode(brCol, brRow)
				} else if br.Bg == "input/30" || (pv.caret.has && hb.focused) {
					hb.isText = true
				}
			}
		}
	}
	if maxX >= minX && maxY >= minY && !pv.caret.has {
		pv.focus = focusRect{has: true, x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1}
	}
	return pv
}

// PaneClick delivers a pointer click on the cells preview to the cells
// Screen, firing press/focus/release/click on the underlying story and
// giving the pane keyboard focus when a control takes focus.
func (a *App) PaneClick(col, row int) {
	sh := a.cur
	if sh == nil || sh.scr == nil {
		return
	}
	sh.scr.Focus(true)
	sh.scr.Click(col, row)
	sh.pump(a.paneMode)
	if sh.focus != "" {
		a.inPane = true
		a.refocus = true
		a.caretTick++
	}
}

// PaneTextDrag forwards pointer press/move/release inside a TUI textarea
// (click, shift+click, double-click word, triple-click line, drag-to-select)
// with exact cell coordinates and modifiers.
func (a *App) PaneTextDrag(col, row, w, h int, d surf.DragEvent) {
	sh := a.cur
	if sh == nil || sh.scr == nil {
		return
	}
	dx := max(0, min(d.X, w-1))
	dy := max(0, min(d.Y, h-1))
	if !d.HasX {
		dx = 0
	}
	if !d.HasY {
		dy = 0
	}
	sc, sr := col+dx, row+dy
	switch {
	case d.Start:
		sh.scr.Focus(true)
		sh.scr.Press(sc, sr, d.Mods, 0)
		a.inPane = true
		a.refocus = true
		a.caretTick++
	case d.End:
		sh.scr.Release(sc, sr, d.Mods)
	default:
		sh.scr.Move(sc, sr, d.Mods)
		a.caretTick++
	}
	sh.pump(a.paneMode)
}

// PaneGripDrag forwards a drag of a textarea's corner resize grip to the
// cells Screen so the textarea resizes 1:1 with the pointer.
func (a *App) PaneGripDrag(col, row int, d surf.DragEvent) {
	sh := a.cur
	if sh == nil || sh.scr == nil {
		return
	}
	switch {
	case d.Start:
		a.gripping = true
		a.gripStart = [2]int{d.Col, d.Row}
		a.gripOrigin = [2]int{col, row}
		sh.scr.Press(col, row, 0, 0)
	case d.End:
		a.gripping = false
		sh.scr.Release(a.gripOrigin[0]+d.Col-a.gripStart[0], a.gripOrigin[1]+d.Row-a.gripStart[1], 0)
	default:
		sh.scr.Move(a.gripOrigin[0]+d.Col-a.gripStart[0], a.gripOrigin[1]+d.Row-a.gripStart[1], 0)
	}
	sh.pump(a.paneMode)
}

// PaneHover delivers pointer enter/leave on an interactive cells span to
// the cells Screen so :hover colours, underlines and tooltips update live.
func (a *App) PaneHover(col, row int, in bool) {
	sh := a.cur
	if sh == nil || sh.scr == nil {
		return
	}
	if in {
		sh.scr.Move(col, row, 0)
	} else {
		sh.scr.Move(-1, -1, 0)
	}
	sh.pump(a.paneMode)
}

func blank(l []seg) bool {
	for _, sg := range l {
		if sg.class != "" || sg.style != "" || strings.TrimSpace(sg.text) != "" {
			return false
		}
	}
	return true
}

// hasControl says whether a document has an element that takes the
// keyboard (SPEC §10.1): a button, a field, a link, a summary, or a
// tabindex of 0 or more; a disabled one or a hidden input doesn't.
func hasControl(doc string) bool {
	z := html.NewTokenizer(strings.NewReader(doc))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return false
		case html.StartTagToken, html.SelfClosingTagToken:
			name, more := z.TagName()
			attrs := map[string]string{}
			for more {
				var k, v []byte
				k, v, more = z.TagAttr()
				attrs[string(k)] = string(v)
			}
			if _, off := attrs["disabled"]; off {
				continue
			}
			if t, ok := attrs["tabindex"]; ok {
				if n, err := strconv.Atoi(t); err == nil && n >= 0 {
					return true
				}
				continue
			}
			switch {
			case bytes.Equal(name, []byte("button")), bytes.Equal(name, []byte("select")),
				bytes.Equal(name, []byte("textarea")), bytes.Equal(name, []byte("summary")):
				return true
			case bytes.Equal(name, []byte("input")):
				if !strings.EqualFold(attrs["type"], "hidden") {
					return true
				}
			case bytes.Equal(name, []byte("a")):
				if _, ok := attrs["href"]; ok {
					return true
				}
			}
		}
	}
}

// plainRowSegs extracts unstyled plain text from a row of Cells, trimmed of
// trailing spaces.
func plainRowSegs(row []surftui.Cell) []seg {
	var b strings.Builder
	for _, c := range row {
		if c.Text == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(c.Text)
		}
	}
	s := strings.TrimRight(b.String(), " ")
	if s == "" {
		return nil
	}
	return []seg{{text: s}}
}

// rowSegs groups a row of Cells into contiguous runs of equal attributes
// and colour roles, trimmed of unstyled trailing spaces. When hideGrip is
// true (interactive TUI preview), the raw ◢ cell glyph is blanked so the
// pixel-aligned .book-grip SVG overlay renders cleanly in its place.
func rowSegs(row []surftui.Cell, hideGrip bool) []seg {
	end := len(row)
	for end > 0 {
		c := row[end-1]
		if c.Attr != 0 || c.Fg != "" || c.Bg != "" || (c.Text != "" && c.Text != " ") {
			break
		}
		end--
	}
	if end == 0 {
		return nil
	}
	var out []seg
	var buf strings.Builder
	var curA surftui.Attr
	var curFg, curBg surftui.Role
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, seg{
			text:  buf.String(),
			class: cellClasses(curA, curFg, curBg),
			style: cellStyle(curA, curFg, curBg),
		})
		buf.Reset()
	}
	for i := 0; i < end; i++ {
		c := row[i]
		if c.Text == "" {
			continue // second cell of a wide character
		}
		txt := c.Text
		if hideGrip && txt == "◢" {
			txt = " "
		}
		if buf.Len() > 0 && (c.Attr != curA || c.Fg != curFg || c.Bg != curBg) {
			flush()
		}
		if buf.Len() == 0 {
			curA, curFg, curBg = c.Attr, c.Fg, c.Bg
		}
		buf.WriteString(txt)
	}
	flush()
	return out
}

func cellClasses(a surftui.Attr, fg, bg surftui.Role) string {
	var cs []string
	if a&surftui.Bold != 0 {
		cs = append(cs, "font-semibold")
	}
	if a&surftui.Dim != 0 {
		cs = append(cs, "text-muted-foreground")
	}
	if a&surftui.Italic != 0 {
		cs = append(cs, "book-italic")
	}
	if a&surftui.Underline != 0 {
		cs = append(cs, "book-underline")
	}
	if a&surftui.Strike != 0 {
		cs = append(cs, "line-through")
	}
	return strings.Join(cs, " ")
}

func cellStyle(a surftui.Attr, fg, bg surftui.Role) string {
	fgCSS := roleCSS(fg)
	bgCSS := roleCSS(bg)
	if a&surftui.Reverse != 0 && bg == "input/30" {
		fgCSS, bgCSS = bgCSS, fgCSS
		if bgCSS == "" {
			bgCSS = "var(--foreground)"
		}
	}
	var parts []string
	if a&surftui.Dim != 0 {
		base := fgCSS
		if base == "" {
			base = "var(--foreground)"
		}
		parts = append(parts, "color:color-mix(in srgb, "+base+" 45%, transparent)")
	} else if fgCSS != "" {
		parts = append(parts, "color:"+fgCSS)
	}
	if bgCSS != "" {
		parts = append(parts, "background:"+bgCSS)
	}
	return strings.Join(parts, ";")
}

func roleCSS(r surftui.Role) string {
	if r == "" {
		return ""
	}
	s := string(r)
	if s == "selected" {
		return "var(--accent)"
	}
	if name, pct, ok := strings.Cut(s, "/"); ok {
		return "color-mix(in srgb, var(--" + name + ") " + pct + "%, transparent)"
	}
	return "var(--" + s + ")"
}

// paneErr is what went wrong in the pane, "" for nothing.
func (a *App) paneErr() string {
	if a.cur == nil || a.cur.err == nil {
		return ""
	}
	return a.cur.err.Error()
}

func bstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{byte('0' + i%10)}, b...)
	}
	return string(b)
}

// newSurface is one of the book's surfaces: view, styled by the kit's sheet,
// linked rather than carried (surfui.Sheet), then by CSS.
func newSurface(name string, view templ.Component) *surf.Surface {
	s := surf.New(name, view, CSS)
	s.Link(kit.Sheet())
	return s
}
