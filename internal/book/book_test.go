package book

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
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
		if strings.TrimSpace(paneText(a)) == "" {
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
	for _, k := range strings.Split("block", "") {
		r.Key(k)
	}
	if a.ID != "input-group/block" {
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
	if a.mode() != ui.ModeTUI || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != ui.ModeANSI || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != ui.ModePlain || !strings.Contains(paneText(a), "Button") {
		t.Fatalf("mode=%v; pane=%q", a.mode(), paneText(a))
	}
	r.Key("r")
	if a.mode() != ui.ModeHotty || !strings.Contains(r.HTML(), "Button") {
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
	if !strings.Contains(joined, "ui.Alert") || !strings.Contains(joined, "ui.Icon") {
		t.Fatalf("component sources missing: %s", joined)
	}
}

func TestBookBaseStoryBeforeVariants(t *testing.T) {
	a := New()
	visible := a.model.Visible()
	base, variant := -1, -1
	for i, st := range visible {
		switch st.ID() {
		case "avatar/default":
			base = i
		case "avatar/badge":
			variant = i
		}
	}
	if base < 0 || variant <= base {
		t.Fatalf("avatar base index %d, variant index %d", base, variant)
	}
}

// Tab through the pane: onto it from the story, through its controls by
// the empty elements round it, and out past its last.
func TestTabThroughThePane(t *testing.T) {
	a := New()
	a.show("field/checkbox-group") // two check boxes
	before, after := a.Edge(true), a.Edge(false)

	before(true) // Tab from the story onto the element before the pane
	if !a.refocus || a.cur.focus != "disks" {
		t.Fatalf("entering: refocus %v, the Screen on %q", a.refocus, a.cur.focus)
	}
	a.takeRefocus()
	a.PaneFocus(true)
	if !a.PaneKey("space") {
		t.Fatal("space in the pane wasn't taken")
	}
	if !strings.Contains(paneText(a), "[x] Hard disks") {
		t.Fatalf("space didn't check the box:\n%s", paneText(a))
	}
	after(true) // Tab in the pane
	if !a.refocus || a.cur.focus != "servers" {
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
	a.show("field/checkbox-group")
	before, after := a.Edge(true), a.Edge(false)
	before(true)
	a.takeRefocus()
	a.PaneFocus(true)
	after(true) // to the second box
	a.takeRefocus()
	after(true) // past it
	if !a.takeToList() {
		t.Fatal("tab past the pane's last control doesn't go to the list")
	}
}

// A story with nothing to focus takes no Tab in the pane.
func TestNoControlsNoTab(t *testing.T) {
	a := New()
	a.show("chart/bar")
	a.setMode(ui.ModeTUI)
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	if a.cur.controls {
		t.Fatal("chart/bar has controls")
	}
	for _, sel := range []string{"#book-pane-before", "#book-pane", "#book-pane-after"} {
		if n := r.One(sel); attrOf(n.Attr, "tabindex") != "" {
			t.Errorf("%s takes Tab in a story with nothing to focus", sel)
		}
	}
	a.show("field/checkbox-group")
	r.Update()
	if n := r.One("#book-pane"); attrOf(n.Attr, "tabindex") != "0" {
		t.Error("the pane of a story with controls takes no Tab")
	}
}

// The list keeps the keyboard as it lands on a story whose dialog opens
// as it shows, and Escape there closes the dialog.
func TestListKeepsTheKeyboard(t *testing.T) {
	a := New()
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("button[aria-current=true]") // the list takes the keyboard
	a.show("dialog/default")
	r.Update()
	if !r.Key("arrow_down") || a.ID != "dialog/open" {
		t.Fatalf("arrow_down from dialog/default: on %s", a.ID)
	}
	if len(r.Find("dialog")) != 1 {
		t.Fatal("dialog/open shows no dialog")
	}
	if !r.Key("arrow_down") || a.ID == "dialog/open" {
		t.Fatalf("arrow_down on dialog/open went to its dialog: on %s", a.ID)
	}
	if !r.Key("arrow_up") || a.ID != "dialog/open" {
		t.Fatalf("arrow_up: on %s", a.ID)
	}
	if !r.Key("escape") {
		t.Fatal("escape on the list, with the story's dialog open: not taken")
	}
	if len(r.Find("dialog")) != 0 {
		t.Fatal("escape didn't close the story's dialog")
	}
}

// The pane's last line is there when it is an empty field: spaces with a
// look are no blank line.
func TestEmptyFieldLineKept(t *testing.T) {
	a := New()
	a.show("field/fieldset")
	ls := a.lines().lines
	last := ls[len(ls)-1]
	if blank(last) || !strings.Contains(fmt.Sprint(last), "book-underline") {
		t.Fatalf("the last line isn't City's empty field: %v", last)
	}
}

// A key while the pane hasn't the keyboard isn't the pane's: Tab with
// nothing focused is left for run, which gives the book the keyboard.
func TestPaneKeysOnlyItsOwn(t *testing.T) {
	a := New()
	a.show("field/checkbox-group")
	if a.PaneKey("tab") || a.PaneKey("space") {
		t.Fatal("the pane took a key without the keyboard")
	}
	if strings.Contains(paneText(a), "[x]") {
		t.Fatal("space reached the pane without the keyboard")
	}
}

func TestCodeStoryPropsChangePreview(t *testing.T) {
	a := New()
	a.show("code/go")
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
	if !strings.Contains(r.HTML(), "Popup.Open") {
		t.Fatal("event log demo did not show its entries")
	}
	r.Click("#demo-clear")
	if strings.Contains(r.HTML(), "Popup.Open") || !strings.Contains(r.HTML(), "Nothing yet: add an event.") {
		t.Fatal("event log demo did not clear")
	}
}

func TestBookDemonstrationsHaveSource(t *testing.T) {
	a := New()
	for _, id := range []string{"code/go", "nav-list/default", "controls/default", "event-log/default", "icons/default", "separator/default"} {
		a.show(id)
		if src := a.model.Sel.Sources(); len(src) < 2 || !strings.HasPrefix(src[1].Title, "render:") {
			t.Errorf("%s source: %+v", id, src)
		}
	}
}

func TestTabsDemoSharesSelectionAndProps(t *testing.T) {
	a := New()
	a.show("tabs-demo/line")
	r := surftest.Start(t, newSurface("book", View(a)), surftest.Size(120, 40))
	r.Click("#demo-tab-2")
	if got := a.storyHTML(); !strings.Contains(got, "password panel") {
		t.Fatalf("tab selection did not update demo: %s", got)
	}
	if !a.model.Sel.Set("Tabs.Active", "team") {
		t.Fatal("could not edit the active tab prop")
	}
	r.Update()
	if got := a.storyHTML(); !strings.Contains(got, "team panel") {
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

// A component's variants are listed as the spike listed them: the order
// their stories were added, not by name.
func TestVariantOrder(t *testing.T) {
	a := New()
	var got []string
	for _, s := range a.model.Stories {
		if strings.HasPrefix(s.Name, "Button") {
			got = append(got, s.Name)
		}
	}
	want := "Button, Button / destructive, Button / ghost with icon, Button / disabled, Button / every variant and size"
	if strings.Join(got, ", ") != want {
		t.Fatalf("Button's stories: %s\nwant: %s", strings.Join(got, ", "), want)
	}
}
