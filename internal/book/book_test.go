package book

import (
	"strings"
	"testing"

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
