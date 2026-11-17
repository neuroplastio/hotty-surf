package book

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

func TestStoryListRevealsEndInCells(t *testing.T) {
	a := New()
	s := surf.New("book", View(a), CSS)
	c := cells.New(120, 30, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Error(err)
		}
	})
	s.Attach(c, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Write([]byte(hotty.Doc("book", doc))); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Pending():
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
	default:
	}
	if !c.Key("end") {
		if taken, err := s.Key("end"); err != nil || !taken {
			t.Fatalf("end: taken=%v err=%v", taken, err)
		}
	}
	last := a.model.Stories[len(a.model.Stories)-1]
	if a.ID != last.ID() {
		t.Fatalf("selected %q, want %q", a.ID, last.ID())
	}
	if lines := strings.Join(c.Lines(), "\n"); !strings.Contains(lines, last.Name) {
		t.Fatalf("selected story %q not visible:\n%s", last.Name, lines)
	}
}

func TestInspectorScrollKeepsTabsVisibleInCells(t *testing.T) {
	a := New()
	s := surf.New("book", View(a), CSS)
	c := cells.New(120, 50, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Error(err)
		}
	})
	s.Attach(c, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Write([]byte(hotty.Doc("book", doc))); err != nil {
		t.Fatal(err)
	}
	lines := c.Lines()
	for row, line := range lines {
		if strings.Contains(line, "Source") && strings.Contains(line, "Props") {
			if row+2 >= len(lines) {
				t.Fatalf("no visible inspector below tabs: %v", lines)
			}
			if !c.Wheel(5, row+2, 5) {
				t.Fatalf("inspector did not scroll at row %d:\n%s", row+2, strings.Join(lines, "\n"))
			}
			after := c.Lines()
			if !strings.Contains(after[row], "Source") || !strings.Contains(after[row], "Props") {
				t.Fatalf("inspector tabs moved from row %d:\n%s", row, strings.Join(after, "\n"))
			}
			if strings.Join(lines[row+1:], "\n") == strings.Join(after[row+1:], "\n") {
				t.Fatal("inspector content did not move")
			}
			return
		}
	}
	t.Fatalf("inspector tabs not visible:\n%s", strings.Join(lines, "\n"))
}
