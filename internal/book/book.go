// Package book is the storybook, `hotty-surf book`: every story of the
// kit (package stories) in a list, the chosen one drawn by the host, and
// beside it the same story drawn in cells, as a terminal that is no hotty
// host draws it. Both are one view with one state, so what is done in
// one shows in the other.
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
// On a terminal that is no host, the whole book is drawn in cells (run's
// fallback), the preview is already the cells rendition, and the pane is
// left out: it has the hidden class, which cells reads, and the book's
// stylesheet shows it to a host.
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
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	bookcode "neuroplast.io/hotty-surf/internal/book/highlight"
	storybook "neuroplast.io/hotty-surf/internal/book/model"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/reload"
	"neuroplast.io/hotty-surf/run"
	"neuroplast.io/hotty-surf/ui"
)

//go:embed book.css
var bookCSS string

// CSS is the book's stylesheet: the kit's, and the book's own.
var CSS = ui.Stylesheet() + bookCSS + bookcode.CSS

// The cells pane's size. A story is at most 28rem wide (max-w-md), which
// cells takes as 56 columns.
const (
	paneCols = 56
	paneRows = 40
	// listRows is how many stories the list shows around the chosen one.
	listRows = 24
)

// State is what the book keeps across a live reload.
type State struct {
	ID   string             `msg:"id"` // the story shown
	Book storybook.DevState `msg:"book"`
}

// App is the book.
type App struct {
	State
	all   []stories.Story
	cur   *shown
	model *storybook.Book

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
var own = map[string]bool{"book-list": true, "book-filter": true, "book-pane": true, "book-pane-before": true, "book-pane-after": true}

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
	a := &App{all: stories.All(), startFocus: true}
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
		return strings.Compare(x.Name, y.Name)
	})
	a.model = storybook.New(ss)
	a.model.SetRegion(storybook.RegionList)
	if len(a.all) > 0 {
		a.show(a.all[0].ID)
	}
	return a
}

// Main shows the book until ctx is done or the user quits.
func Main(ctx context.Context) error {
	a := New()
	s := surf.New("book", View(a), CSS)
	reload.Keep(a.Save, a.Restore)
	return run.App(ctx, s)
}

func (a *App) Save() *State {
	a.State.Book = *a.model.Save()
	return &a.State
}

func (a *App) Restore(s *State) {
	a.model.Load(&s.Book)
	if s.ID != "" {
		a.show(s.ID)
	}
}

// index is the place of the story shown in the list, -1 for none.
func (a *App) index() int {
	return a.storyIndex(a.ID)
}

func (a *App) storyIndex(id string) int {
	for i, st := range a.all {
		if st.ID == id {
			return i
		}
	}
	return -1
}

// show opens the story id: a new view, with state of its own, and its
// pane.
func (a *App) show(id string) {
	for _, st := range a.all {
		if st.ID != id {
			continue
		}
		a.ID = id
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
				f.After = append(f.After, func() { a.observeHandlers(f, story, ids) })
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
		h := f.On[id]
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

// start puts the story on the pane's Screen, the first time the pane is
// drawn or used: not before the book runs, whose live reload has to be
// set up before a view renders.
func (sh *shown) start() {
	if sh.pane != nil {
		return
	}
	sh.pane = surf.New("book-pane", sh.view, ui.Stylesheet())
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

// ListKey moves through the list and handles the book shortcuts.
func (a *App) ListKey(k string) bool {
	if k == "escape" && a.modalOpen {
		return false
	}
	if k == "/" || k == "enter" || k == "escape" || k == "t" || k == "n" || k == "N" || k == "shift+n" {
		return a.bookKey(k)
	}
	visible := a.model.Visible()
	i, n := 0, len(visible)
	for j, st := range visible {
		if st.ID() == a.ID {
			i = j
			break
		}
	}
	if n == 0 {
		return false
	}
	switch k {
	case "arrow_down":
		i++
	case "arrow_up":
		i--
	case "page_down":
		i += listRows / 2
	case "page_up":
		i -= listRows / 2
	case "home":
		i = 0
	case "end":
		i = n - 1
	default:
		return false
	}
	i = max(0, min(n-1, i))
	if visible[i].ID() != a.ID {
		a.show(visible[i].ID())
	}
	return true
}

// ListFocus hears the list take the keyboard or lose it. It does nothing:
// listening has the host tell the surface where the keyboard is
// (rt.Frame.Focused), which keep reads.
func (a *App) ListFocus(bool) {}

// Pick shows the story id: a click in the list.
func (a *App) Pick(id string) func() { return func() { a.show(id) } }

func (a *App) filter(v string) {
	a.model.SetFilter(v)
	if a.model.Sel != nil && a.model.Sel.ID() != a.ID {
		a.show(a.model.Sel.ID())
	}
}

func (a *App) storyLabel(id string) string {
	if st := a.model.Find(id); st != nil {
		return st.Name
	}
	return id
}

func (a *App) groupHeading(i, from int) string {
	visible := a.model.Visible()
	if i < 0 || i >= len(visible) {
		return ""
	}
	if i == from || i == 0 || visible[i-1].Group != visible[i].Group {
		return visible[i].Group
	}
	return ""
}

func (a *App) startingList(ctx context.Context) bool {
	if f := rt.From(ctx); f != nil && f.Focused != "" {
		a.startFocus = false
	}
	return a.startFocus
}

func (a *App) bookKey(k string) bool {
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
		a.wantFocus = "book-list"
	case storybook.RegionFilter:
		a.wantFocus = "book-filter"
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

func (a *App) focusRequest(ctx context.Context) string {
	f := rt.From(ctx)
	if f == nil {
		return ""
	}
	if a.wantFocus != "" && f.Focused == a.wantFocus {
		a.wantFocus = ""
	}
	if a.wantFocus != "" {
		return a.wantFocus
	}
	switch {
	case f.Focused == "book-list":
		a.model.SetRegion(storybook.RegionList)
	case f.Focused == "book-filter":
		a.model.SetRegion(storybook.RegionFilter)
	case strings.HasPrefix(f.Focused, "book-tabs-tab-"):
		a.model.SetRegion(storybook.RegionTabs)
	case f.Focused == "book-panel" || strings.HasPrefix(f.Focused, "book-control-"):
		a.model.SetRegion(storybook.RegionPanel)
	case f.Focused != "":
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

func (a *App) setBool(path string) func(bool) {
	return func(value bool) { a.model.Set(a.model.Sel, path, strconv.FormatBool(value)) }
}

func (a *App) storyHTML() string {
	if a.cur == nil {
		return ""
	}
	a.cur.start()
	if a.cur.err != nil {
		return a.cur.err.Error()
	}
	doc, err := rt.Peek.Document(a.cur.pane)
	if err != nil {
		return err.Error()
	}
	return prettyHTML(doc)
}

// window is the stories the list shows: listRows of them round the one
// shown, and how many are above and below it.
func (a *App) window() (from int, sts []stories.Story, below int) {
	visible := a.model.Visible()
	for _, v := range visible {
		for _, st := range a.all {
			if st.ID == v.ID() {
				sts = append(sts, st)
				break
			}
		}
	}
	i := 0
	for n, st := range sts {
		if st.ID == a.ID {
			i = n
			break
		}
	}
	from = max(0, min(i-listRows/2, len(sts)-listRows))
	to := min(len(sts), from+listRows)
	return from, sts[from:to], len(sts) - to
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
	return f != nil && len(f.Modal) > 0 && !(own[f.Focused] && f.Focused != "book-list")
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
		if own[f.Focused] && f.Focus != "" && !own[f.Focus] && f.Focus != a.wantFocus {
			f.Focus = ""
		}
	}
	f.After = append(f.After, func() { f.After = append(f.After, last) })
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

// seg is a run of a line's cells in one look.
type seg struct {
	text  string
	class string
}

var sgrRE = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// lines are the pane's lines as runs, the blank ones at the end left out.
func (a *App) lines() [][]seg {
	sh := a.cur
	if sh == nil {
		return nil
	}
	sh.pump()
	ls := sh.scr.Lines()
	out := make([][]seg, len(ls))
	for i, l := range ls {
		out[i] = segs(l)
	}
	// The blank lines at the end go; a line of spaces with a look (an
	// empty field, underlined) is no blank.
	for len(out) > 0 && blank(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return out
}

func blank(l []seg) bool {
	for _, sg := range l {
		if sg.class != "" || strings.TrimSpace(sg.text) != "" {
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

// segs is a line with SGR as runs, each with the classes of its look.
func segs(l string) []seg {
	var out []seg
	class := ""
	for len(l) > 0 {
		loc := sgrRE.FindStringSubmatchIndex(l)
		if loc == nil {
			out = append(out, seg{l, class})
			break
		}
		if loc[0] > 0 {
			out = append(out, seg{l[:loc[0]], class})
		}
		class = looks(l[loc[2]:loc[3]])
		l = l[loc[1]:]
	}
	return out
}

// looks is the classes of an SGR's parameters.
func looks(params string) string {
	var cs []string
	for _, p := range strings.Split(params, ";") {
		switch p {
		case "1":
			cs = append(cs, "font-semibold")
		case "2":
			cs = append(cs, "text-muted-foreground")
		case "3":
			cs = append(cs, "book-italic")
		case "4":
			cs = append(cs, "book-underline")
		case "7":
			cs = append(cs, "book-rev")
		}
	}
	return strings.Join(cs, " ")
}

// paneErr is what went wrong in the pane, "" for nothing.
func (a *App) paneErr() string {
	if a.cur == nil || a.cur.err == nil {
		return ""
	}
	return a.cur.err.Error()
}

// keysLine is the story's Keys and Want, as the gate presses them.
func keysLine(st stories.Story) string {
	if len(st.Keys) == 0 {
		return ""
	}
	s := "Keys: " + strings.Join(st.Keys, " ") + " → " + st.Want
	if st.Gap != "" {
		s += " (gap: " + st.Gap + ")"
	}
	return s
}

func bstr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func optionID(i int) string { return "book-s-" + itoa(i) }

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
