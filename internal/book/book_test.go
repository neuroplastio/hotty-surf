package book

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surftest"
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
	for _, l := range a.lines() {
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
	if len(a.all) != len(stories.All()) || len(a.all) == 0 {
		t.Fatalf("the book has %d stories, package stories %d", len(a.all), len(stories.All()))
	}
	for range a.all {
		if a.cur == nil || a.cur.err != nil {
			t.Fatalf("%s: %v", a.ID, a.cur.err)
		}
		if strings.TrimSpace(paneText(a)) == "" {
			t.Errorf("%s: the pane is blank", a.ID)
		}
		a.ListKey("arrow_down")
	}
}

// The list by keyboard, in cells (a terminal that is no host).
func TestListByKeyboard(t *testing.T) {
	a := New()
	r := surftest.Start(t, surf.New("book", View(a), CSS), surftest.Size(120, 40))
	first := a.ID
	r.Key("tab") // onto the list
	r.Key("arrow_down")
	if a.ID == first || a.ID != a.all[1].ID {
		t.Fatalf("arrow_down showed %q, want %q", a.ID, a.all[1].ID)
	}
	r.Key("end")
	if a.ID != a.all[len(a.all)-1].ID {
		t.Fatalf("end showed %q", a.ID)
	}
	pane := r.One("section#book-cells")
	if c := attrOf(pane.Attr, "class"); !strings.Contains(" "+c+" ", " hidden ") {
		t.Fatalf("the pane isn't hidden from cells: class %q", c)
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
	r := surftest.Start(t, surf.New("book", View(a), CSS), surftest.Size(120, 40))
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

// Type-to-find: letters jump to the first story that starts with them; the
// same letter again goes to the next; a pause starts anew.
func TestTypeToFind(t *testing.T) {
	a := New()
	at := time.Unix(0, 0)
	a.now = func() time.Time { return at }
	press := func(ks ...string) {
		for _, k := range ks {
			a.ListKey(k)
			at = at.Add(100 * time.Millisecond)
		}
	}
	press("d", "i", "a")
	if a.ID != "dialog/default" {
		t.Fatalf("dia: %q", a.ID)
	}
	press("l", "o", "g", "/", "o")
	if a.ID != "dialog/open" {
		t.Fatalf("dialog/o: %q", a.ID)
	}
	at = at.Add(2 * time.Second)
	press("t")
	first := a.ID
	if !strings.HasPrefix(first, "t") {
		t.Fatalf("t after a pause: %q", first)
	}
	press("t")
	if a.ID == first || !strings.HasPrefix(a.ID, "t") {
		t.Fatalf("t again: %q after %q", a.ID, first)
	}
	at = at.Add(2 * time.Second)
	press("v", "a", "r", "i", "a", "n", "t")
	if !strings.Contains(a.ID, "variant") {
		t.Fatalf("variant (in the middle of an ID): %q", a.ID)
	}
}

// The list keeps the keyboard as it lands on a story whose dialog opens
// as it shows, and Escape there closes the dialog.
func TestListKeepsTheKeyboard(t *testing.T) {
	a := New()
	r := surftest.Start(t, surf.New("book", View(a), CSS), surftest.Size(120, 40))
	r.Click("#book-list") // the list takes the keyboard
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
	ls := a.lines()
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
