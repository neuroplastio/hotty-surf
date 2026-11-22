package book

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/book/stories"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

func attrOf(as []html.Attribute, k string) string {
	for _, a := range as {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func paneText(a *App) string {
	var b strings.Builder
	for _, l := range a.lines().lines {
		for _, s := range l {
			b.WriteString(s.text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Every story is in the book, a later one with no step of its own.
func TestEveryStory(t *testing.T) {
	a := New()
	if len(a.all) != len(stories.BookAll()) || len(a.all) == 0 {
		t.Fatalf("the book has %d stories, package stories %d", len(a.all), len(stories.BookAll()))
	}
	for _, st := range a.all {
		a.show(st.ID)
		if a.cur == nil || a.cur.err != nil {
			t.Fatalf("%s: %v", a.ID, a.cur.err)
		}
		if len(a.lines().lines) == 0 {
			t.Errorf("%s: the pane is blank", a.ID)
		}
	}
}

// The list by keyboard, in cells (a terminal that is no host).
func TestListByKeyboard(t *testing.T) {
	a := New()
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	visible := a.model.Visible()
	if len(visible) < 2 || a.ID != visible[0].ID() {
		t.Fatalf("the book opened on %q, want the first in its list", a.ID)
	}
	// The next in the list as the book shows it, grouped, which need not
	// be the next in stories.All().
	r.Key("arrow_down")
	if a.ID != visible[1].ID() {
		t.Fatalf("arrow_down showed %q, want %q", a.ID, visible[1].ID())
	}
	r.Key("end")
	if a.ID != visible[len(visible)-1].ID() {
		t.Fatalf("end showed %q", a.ID)
	}
	if len(r.Find("#book-cells")) != 0 {
		t.Fatal("cells rendition is visible before it is selected")
	}
}

func TestFilterFromLaunch(t *testing.T) {
	a := New()
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Key("/")
	for _, k := range strings.Split("kbd", "") {
		r.Key(k)
	}
	if a.ID != "kbd/default" {
		t.Fatalf("filter opened %q", a.ID)
	}
}

func TestBookInspection(t *testing.T) {
	a := New()
	a.show("button/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#save")
	if events := a.model.Events(a.model.Sel); len(events) == 0 || !strings.Contains(events[0].Text, "save") {
		t.Fatalf("click event not recorded: %v", events)
	}
	a.clearEvents()
	if events := a.model.Events(a.model.Sel); len(events) != 0 {
		t.Fatalf("clear left events: %v", events)
	}
	a.setProp("Label")("Apply")
	r.Update()
	if !strings.Contains(r.HTML(), "Apply") {
		t.Fatal("changing the Label control did not update the preview")
	}
	if src := a.model.Sel.Sources(); len(src) < 2 || !strings.Contains(src[1].Code, "bookButton") {
		t.Fatalf("missing story source: %+v", src)
	}
	if html := a.storyHTML(); !strings.Contains(html, "Apply") {
		t.Fatalf("HTML tab has no current label: %s", html)
	}
}

func TestBookLogsStoryStateChanges(t *testing.T) {
	a := New()
	a.show("button/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#save")
	for _, event := range a.model.Events(a.model.Sel) {
		if event.Kind == "state" && strings.Contains(event.Text, "Clicks") {
			return
		}
	}
	t.Fatalf("button state change was not logged: %+v", a.model.Events(a.model.Sel))
}

func TestStoryHTMLIsComponentOnly(t *testing.T) {
	a := New()
	a.show("alert/default")
	got := a.storyHTML()
	if strings.Contains(got, "<style") || strings.Contains(got, `id="root"`) {
		t.Fatalf("HTML includes surface document: %s", got)
	}
	if !strings.Contains(got, "Heads up!") ||
		!strings.Contains(got, `<svg class="lucide`) ||
		!strings.Contains(got, " …></svg>") || strings.Contains(got, "<path") {
		t.Fatalf("HTML did not format and shorten the component: %s", got)
	}
}

func TestRenditionsShareStoryState(t *testing.T) {
	a := New()
	a.show("button/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#save")
	r.Key("r")
	if a.mode() != surfui.ModeTUI || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != surfui.ModeANSI || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != surfui.ModePlain || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != surfui.ModeHotty || !strings.Contains(r.HTML(), "Button") {
		t.Fatal("hotty mode did not retain the story")
	}
	var clicks string
	for _, c := range a.model.Sel.Controls() {
		if c.Path == "Clicks" {
			clicks = c.Value
		}
	}
	if clicks != "1" {
		t.Fatalf("click count after switching modes: %q", clicks)
	}
}

func TestBookRestore(t *testing.T) {
	a := New()
	a.show("badge/default")
	a.model.SetFilter("badge")
	a.model.SetTab("props")
	a.model.ToggleTheme()
	a.setProp("Label")("Ready")
	state := *a.Save()
	b := New()
	b.Restore(&state)
	if b.ID != "badge/default" || b.model.Filter != "badge" || b.model.Tab != "props" || b.model.Dark {
		t.Fatalf("restored book: story %q, filter %q, tab %q, dark %v", b.ID, b.model.Filter, b.model.Tab, b.model.Dark)
	}
	if got := b.model.Sel.Controls(); len(got) != 2 || got[1].Path != "Label" || got[1].Value != "Ready" {
		t.Fatalf("restored controls: %+v", got)
	}
}

func TestBookSourceIncludesComponents(t *testing.T) {
	a := New()
	a.show("alert/default")
	var names []string
	for _, src := range a.model.Sel.Sources() {
		names = append(names, src.Title)
	}
	joined := strings.Join(names, ", ")
	if !strings.Contains(joined, "alert.View") || !strings.Contains(joined, "icon.View") {
		t.Fatalf("component sources missing: %s", joined)
	}
}

// Tab through the pane: onto it from the story, through its controls by
// the empty elements round it, and out past its last.
func TestTabThroughThePane(t *testing.T) {
	a := New()
	a.show("switch/default") // two enabled switches
	before, after := a.Edge(true), a.Edge(false)

	before(true) // Tab from the story onto the element before the pane
	if !a.refocus || a.cur.focus != "airplane" {
		t.Fatalf("entering: refocus %v, the Screen on %q", a.refocus, a.cur.focus)
	}
	a.takeRefocus()
	a.PaneFocus(true)
	if !a.PaneKey("space") {
		t.Fatal("space in the pane wasn't taken")
	}
	if !strings.Contains(paneText(a), "airplane mode: on") {
		t.Fatalf("space didn't toggle the switch:\n%s", paneText(a))
	}
	after(true) // Tab in the pane
	if !a.refocus || a.cur.focus != "notify" {
		t.Fatalf("tab: refocus %v, the Screen on %q", a.refocus, a.cur.focus)
	}
	a.takeRefocus()
	after(true) // Tab past the last: the keyboard leaves
	if a.refocus || a.inPane {
		t.Fatalf("past the last: refocus %v, in the pane %v", a.refocus, a.inPane)
	}
	if strings.Contains(strings.Join(a.cur.scr.Lines(), ""), ";7m") {
		t.Fatal("the pane still draws a control reversed after the keyboard left")
	}
	before(true) // Shift+Tab back in from outside... is Tab onto the first
	a.takeRefocus()
	a.PaneFocus(true)
	before(true) // Shift+Tab in the pane, from its first: leaves
	if a.inPane {
		t.Fatal("shift+tab from the first control stayed in the pane")
	}
}

// Tab past the pane's last control goes on to the list, not to a blank
// element after the pane.
func TestTabPastThePaneGoesToTheList(t *testing.T) {
	a := New()
	a.show("switch/default")
	before, after := a.Edge(true), a.Edge(false)
	before(true)
	a.takeRefocus()
	a.PaneFocus(true)
	after(true) // to the second switch
	a.takeRefocus()
	after(true) // past it
	if !a.takeToList() {
		t.Fatal("tab past the pane's last control doesn't go to the list")
	}
}

// A story with nothing to focus takes no Tab in the pane.
func TestNoControlsNoTab(t *testing.T) {
	a := New()
	a.show("chart/default")
	a.setMode(surfui.ModeTUI)
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	if a.cur.controls {
		t.Fatal("chart/default has controls")
	}
	for _, sel := range []string{"#book-pane-before", "#book-pane", "#book-pane-after"} {
		if n := r.One(sel); attrOf(n.Attr, "tabindex") != "" {
			t.Errorf("%s takes Tab in a story with nothing to focus", sel)
		}
	}
	a.show("switch/default")
	r.Update()
	if n := r.One("#book-pane"); attrOf(n.Attr, "tabindex") != "0" {
		t.Error("the pane of a story with controls takes no Tab")
	}
}

// A key while the pane hasn't the keyboard isn't the pane's: Tab with
// nothing focused is left for run, which gives the book the keyboard.
func TestPaneKeysOnlyItsOwn(t *testing.T) {
	a := New()
	a.show("switch/default")
	if a.PaneKey("tab") || a.PaneKey("space") {
		t.Fatal("the pane took a key without the keyboard")
	}
	if strings.Contains(paneText(a), "airplane mode: on") {
		t.Fatal("space reached the pane without the keyboard")
	}
}

func TestCodeStoryPropsChangePreview(t *testing.T) {
	a := New()
	a.show("code/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	if !a.model.Sel.Set("First", "120") {
		t.Fatal("could not change the first line number")
	}
	r.Update()
	if got := a.storyHTML(); !strings.Contains(got, "120") {
		t.Fatalf("code preview did not use the edited line number: %s", got)
	}
}

func TestBookEventLogDemoClears(t *testing.T) {
	a := New()
	a.show("event-log/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	if !strings.Contains(r.HTML(), "Clicks = 1") {
		t.Fatal("event log demo did not show its entries")
	}
	r.Click("#demo-clear")
	if strings.Contains(r.HTML(), "Clicks = 1") || !strings.Contains(r.HTML(), "Nothing yet: add an event.") {
		t.Fatal("event log demo did not clear")
	}
}

func TestBookDemonstrationsHaveSource(t *testing.T) {
	a := New()
	for _, id := range []string{"code/default", "nav-list/default", "event-log/default", "icons/default", "separator/default", "kbd/default"} {
		a.show(id)
		if src := a.model.Sel.Sources(); len(src) < 2 || !strings.HasPrefix(src[1].Title, "render:") {
			t.Errorf("%s source: %+v", id, src)
		}
	}
}

func TestTabsStorySharesSelectionAndProps(t *testing.T) {
	a := New()
	a.show("tabs/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#story-tabs-tab-2")
	if got := a.storyHTML(); !strings.Contains(got, "Change your password") {
		t.Fatalf("tab selection did not update demo: %s", got)
	}
	if !a.model.Sel.Set("Tab", "team") {
		t.Fatal("could not edit the active tab prop")
	}
	r.Update()
	if got := a.storyHTML(); !strings.Contains(got, "Invite your team") {
		t.Fatalf("active prop did not update demo: %s", got)
	}
}

func TestShellStoriesUseTheirEditableState(t *testing.T) {
	a := New()
	a.show("list-item/default")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#row-0")
	if got := storyControl(a, "Selected"); got != "0" {
		t.Fatalf("list selection = %q, want 0", got)
	}
	a.show("icon-button/default")
	r.Update()
	r.Click("#ib-Layout")
	if got := storyControl(a, "Accent"); got != "Layout" {
		t.Fatalf("icon button accent = %q, want Layout", got)
	}
}

func storyControl(a *App, path string) string {
	for _, c := range a.model.Sel.Controls() {
		if c.Path == path {
			return c.Value
		}
	}
	return ""
}

func TestBookTUICUAKeys(t *testing.T) {
	a := New()
	a.show("textarea/default")
	a.setMode(surfui.ModeTUI)
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	// Focus #book-pane via the real surface (as clicking a TUI textarea does).
	a.PaneTextDrag(2, 2, 36, 3, surf.DragEvent{Start: true, Drag: hotty.Drag{X: 2, Y: 0, HasX: true, HasY: true}})
	r.Update()
	for _, k := range []string{"h", "i", "space", "w", "o", "r", "l", "d"} {
		r.Key(k)
	}
	if !r.Key("ctrl+a") {
		t.Fatalf("r.Key(ctrl+a) not taken; inPane=%v focus=%q", a.inPane, a.cur.focus)
	}
	if got := a.cur.scr.Selection(); got != "hi world" {
		t.Fatalf("after r.Key(ctrl+a), selection = %q", got)
	}
	r.Key("arrow_right")
	if !r.Key("ctrl+backspace") {
		t.Fatalf("r.Key(ctrl+backspace) not taken")
	}
	if got := paneText(a); strings.Contains(got, "world") {
		t.Fatalf("after r.Key(ctrl+backspace), pane still has world:\n%s", got)
	}
}
