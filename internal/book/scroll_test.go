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
