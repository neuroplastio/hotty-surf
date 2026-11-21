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
// On a terminal that is no host, run draws the whole book in cells.
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

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/internal/devhook"
	storybook "neuroplast.io/hotty-surf/internal/book/model"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
	"neuroplast.io/hotty-surf/ui"
	bookcode "neuroplast.io/hotty-surf/ui/code"
)

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

// State is what the book keeps across a live reload.
type State struct {
	ID    string             `msg:"id"` // the story shown
	Book  storybook.DevState `msg:"book"`
	Cells bool               `msg:"cells"`
}

// App is the book.
type App struct {
	State
	all   []stories.Story
	cur   *shown
	model *storybook.Book
	nav   ui.NavList

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
	scr   *cells.Screen
	focus string // the id the Screen's keyboard is on, "" for none
	err   error  // what went wrong in the pane's last update, nil for nothing
	// controls says whether the story has anything that takes the
	// keyboard; the pane takes no Tab when it hasn't.
	controls bool
	// retried is set once a refused focus was tried again (ENOTARGET):
	// a second refusal leaves the keyboard where it is.
	retried bool
}

// New is a book of every story, showing the first.
func New() *App {
	a := &App{all: stories.BookAll(), startFocus: true}
	var ss []*storybook.Story
	for _, st := range a.all {
		ss = append(ss, storybook.Wrap(st.ID, st.Doc, st.View, st.Props))
	}
	slices.SortStableFunc(ss, func(x, y *storybook.Story) int {
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
	a.nav = ui.NavList{ID: "book-nav", Label: "Stories", FilterLabel: "Filter stories", Placeholder: "Filter stories  ( / )", Empty: "No story matches."}
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
	reload.Keep(a.Save, a.Restore)
	return run.App(ctx, s)
}

func (a *App) Save() *State {
	a.State.Book = *a.model.Save()
	return &a.State
}

func (a *App) Restore(s *State) {
	a.Cells = s.Cells
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

// start puts the story on the pane's Screen, the first time the pane is
// drawn or used: not before the book runs, whose live reload has to be
// set up before a view renders.
func (sh *shown) start() {
	if sh.pane != nil {
		return
	}
	sh.pane = surf.New("book-pane", sh.view, "")
	// sh.pane is driven synchronously by book's own render (sh.pump), not by
	// an independent driver goroutine; mark it done so devchild's live-reload
	// handover never stalls 4s waiting on it.
	devhook.Driver(sh.pane, paneDone)
	sh.pane.Link(ui.Sheet())
	sh.scr = cells.New(paneCols, paneRows, func(e hotty.Event) {
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
		_, err = sh.scr.Write([]byte(hotty.Doc(sh.pane.Name(), doc)))
	}
	sh.err = err
}

// pump sends the pane's surface what changed in the view: its own
// handlers' changes, and the preview's.
func (sh *shown) pump() {
	sh.start()
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
			a.nav.Groups = append(a.nav.Groups, ui.NavGroup{Name: st.Group})
			n++
		}
		a.nav.Groups[n-1].Items = append(a.nav.Groups[n-1].Items, ui.NavItem{ID: st.ID(), Label: st.Name})
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
	if k == "r" {
		a.setRendition(!a.Cells)
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

func (a *App) setRendition(cells bool) {
	if a.Cells != cells && a.inPane {
		a.PaneOutside()
	}
	a.Cells = cells
}

func (a *App) renditionKey(k string) bool {
	switch k {
	case "arrow_left", "arrow_right":
		a.setRendition(!a.Cells)
		return true
	case "home":
		a.setRendition(false)
		return true
	case "end":
		a.setRendition(true)
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

// controls and entries adapt the model data to the spike's kit parts.
func (a *App) controls() []ui.Control {
	var out []ui.Control
	for _, c := range a.model.Sel.Controls() {
		out = append(out, ui.Control{Name: c.Path, Kind: c.Kind, Value: c.Value, Options: c.Options})
	}
	return out
}

func (a *App) entries() []ui.LogEntry {
	var out []ui.LogEntry
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
		out = append(out, ui.LogEntry{At: e.At, Kind: e.Kind, Text: e.Text, KindClass: bookcode.TokenClass(kind)})
	}
	return out
}

func (a *App) storyHTML() string {
	if a.model.Sel == nil {
		return ""
	}
	var b strings.Builder
	if err := a.model.Sel.HTML(nil).Render(context.Background(), &b); err != nil {
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
	sh.start()
	taken := sh.scr.Key(k)
	if !taken {
		var err error
		taken, err = sh.pane.Key(k)
		if err != nil && sh.err == nil {
			sh.err = err
		}
	}
	sh.pump()
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
	sh.start()
	sh.retried = false
	sh.scr.Focus(true)
	sh.pump()
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
	sh.pump()
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
		sh.start()
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
		sh.pump()
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
	id         string
	x, y, w, h int
	col, row   int // untrimmed screen coordinates for Click/Move
}

func (h hitBox) style() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf(
		"left:%dch;top:calc(%d*var(--hotty-cell-h));width:%dch;height:calc(%d*var(--hotty-cell-h))",
		h.x, h.y, h.w, h.h,
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
type paneView struct {
	lines [][]seg
	hits  []hitBox
	focus focusRect
}

// lines are the pane's lines as runs from Cells (with colours and
// attributes), the leading/trailing blank lines left out, plus unified hit
// boxes and focus bounding box.
func (a *App) lines() paneView {
	sh := a.cur
	if sh == nil {
		return paneView{}
	}
	sh.pump()
	grid := sh.scr.Cells()
	out := make([][]seg, len(grid))
	for i, row := range grid {
		out[i] = rowSegs(row)
	}
	// The blank lines at the top and end go; a line of spaces with a look or
	// fill (an empty field, underlined) is no blank.
	trimTop := 0
	for len(out) > 0 && blank(out[0]) {
		out = out[1:]
		trimTop++
	}
	for len(out) > 0 && blank(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	var pv paneView
	pv.lines = out
	minX, minY, maxX, maxY := 1<<30, 1<<30, -1, -1
	byID := map[string]int{}
	for r := trimTop; r < trimTop+len(out); r++ {
		ry := r - trimTop
		for c, cell := range grid[r] {
			if cell.Attr&cells.Reverse != 0 {
				if c < minX {
					minX = c
				}
				if c > maxX {
					maxX = c
				}
				if ry < minY {
					minY = ry
				}
				if ry > maxY {
					maxY = ry
				}
			}
			if cell.Attr&cells.Dim != 0 {
				continue
			}
			if id, _ := sh.scr.At(c, r); id != "" {
				if idx, ok := byID[id]; ok {
					hb := &pv.hits[idx]
					if c < hb.x {
						hb.w += hb.x - c
						hb.x = c
					}
					if c >= hb.x+hb.w {
						hb.w = c - hb.x + 1
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
						id: id, x: c, y: ry, w: 1, h: 1, col: c, row: r,
					})
				}
			}
		}
	}
	if maxX >= minX && maxY >= minY {
		pv.focus = focusRect{has: true, x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1}
	}
	return pv
}

// PaneClick delivers a pointer click on the cells preview to the cells
// Screen, firing press/focus/release/click on the underlying story.
func (a *App) PaneClick(col, row int) {
	sh := a.cur
	if sh == nil || sh.scr == nil {
		return
	}
	sh.scr.Click(col, row)
	sh.pump()
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
	sh.pump()
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

// rowSegs groups a row of Cells into contiguous runs of equal attributes
// and colour roles, trimmed of unstyled trailing spaces.
func rowSegs(row []cells.Cell) []seg {
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
	var curA cells.Attr
	var curFg, curBg cells.Role
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
		if buf.Len() > 0 && (c.Attr != curA || c.Fg != curFg || c.Bg != curBg) {
			flush()
		}
		if buf.Len() == 0 {
			curA, curFg, curBg = c.Attr, c.Fg, c.Bg
		}
		buf.WriteString(c.Text)
	}
	flush()
	return out
}

func cellClasses(a cells.Attr, fg, bg cells.Role) string {
	var cs []string
	if a&cells.Bold != 0 {
		cs = append(cs, "font-semibold")
	}
	if a&cells.Dim != 0 {
		cs = append(cs, "text-muted-foreground")
	}
	if a&cells.Italic != 0 {
		cs = append(cs, "book-italic")
	}
	if a&cells.Underline != 0 {
		cs = append(cs, "book-underline")
	}
	if a&cells.Strike != 0 {
		cs = append(cs, "line-through")
	}
	return strings.Join(cs, " ")
}

func cellStyle(a cells.Attr, fg, bg cells.Role) string {
	fgCSS := roleCSS(fg)
	bgCSS := roleCSS(bg)
	var parts []string
	if fgCSS != "" {
		parts = append(parts, "color:"+fgCSS)
	}
	if bgCSS != "" {
		parts = append(parts, "background:"+bgCSS)
	}
	return strings.Join(parts, ";")
}

func roleCSS(r cells.Role) string {
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
// linked rather than carried (ui.Sheet), then by CSS.
func newSurface(name string, view templ.Component) *surf.Surface {
	s := surf.New(name, view, CSS)
	s.Link(ui.Sheet())
	return s
}
