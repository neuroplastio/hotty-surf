package surfrun_test

import (
	"slices"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// box is a modal dialog's surface: a field, which has the keyboard when
// it is shown, and a Select; back binds Escape on the box around them.
type box struct {
	theme string
	back  bool
	ran   []string
}

func (m *box) Render() *surfui.Node {
	b := surfui.Box(
		surfui.Field().Key("f").Autofocus(),
		surfui.Select(m.theme, themes...).Key("sel").OnChange(func(e surfrun.Event) { m.theme = e.Value }),
	).Key("b")
	if m.back {
		b.Bind("escape", "Back", "Back")
	}
	return surfui.Surface(b)
}

func (m *box) Do(act surfui.Action, ev surfrun.Event) {
	switch a := act.(type) {
	case string:
		m.ran = append(m.ran, a)
	case func(surfrun.Event):
		a(ev)
	}
}

const (
	boxField  = "r.kb.kf"
	boxSelect = "r.kb.ksel"
)

// A modal dialog's host gives itself Escape (Program), as a browser's
// dialog closes on Escape whatever has the focus: with the field
// focused, Key leaves it to the host, which closes the box, and blurs
// nothing, and Keys doesn't list it, in the cells too. Without it, the
// first Escape blurs. An open list still takes its Escape, and a binding
// of Escape inside the box runs.
func TestEscapeGivenToTheHost(t *testing.T) {
	escapes := func(keys []surfrun.KeyUse) bool {
		return slices.ContainsFunc(keys, func(u surfrun.KeyUse) bool { return u.Name == "escape" })
	}

	s := surfrun.NewSurface(&box{theme: themes[0]}, nil)
	s.Show()
	if keys, _ := s.Keys(); !escapes(keys) {
		t.Errorf("not given: Keys %+v, want the Escape that blurs", keys)
	}
	if took, _ := s.Key("escape"); !took || s.Focused() != "" {
		t.Errorf("not given: escape taken %v, focused %q; want it to blur", took, s.Focused())
	}

	s = surfrun.NewSurface(&box{theme: themes[0]}, nil)
	s.Program = []string{"escape"}
	s.Show()
	if keys, _ := s.Keys(); escapes(keys) {
		t.Errorf("given: Keys %+v list escape", keys)
	}
	if took, _ := s.Key("escape"); took || s.Focused() != boxField {
		t.Errorf("given: escape taken %v, focused %q; want the host's, the field still focused", took, s.Focused())
	}

	// The Select's open list closes on Escape, and stays focused.
	focus(boxSelect)(s)
	s.Key("space")
	if f := s.Update(); f.List == nil || f.List.Closed {
		t.Fatalf("given: space left no list open, %+v", f.List)
	}
	if keys, _ := s.Keys(); !escapes(keys) {
		t.Errorf("given, the list open: Keys %+v, want the list's escape", keys)
	}
	if took, _ := s.Key("escape"); !took || s.Focused() != boxSelect {
		t.Errorf("given, the list open: escape taken %v, focused %q", took, s.Focused())
	}
	if f := s.Update(); f.List == nil || !f.List.Closed {
		t.Errorf("given, the list open: escape left the list %+v", f.List)
	}
	if took, _ := s.Key("escape"); took {
		t.Error("given, the list closed: the next escape taken, want the host's")
	}

	// A binding of Escape nearer the focus than the root runs.
	m := &box{theme: themes[0], back: true}
	s = surfrun.NewSurface(m, nil)
	s.Program = []string{"escape"}
	s.Show()
	if took, _ := s.Key("escape"); !took || !slices.Equal(m.ran, []string{"Back"}) || s.Focused() != boxField {
		t.Errorf("given, bound in the box: escape taken %v, ran %q, focused %q", took, m.ran, s.Focused())
	}

	// In the cells, the renderer's field hands Escape on to the surface.
	s = surfrun.NewSurface(&box{theme: themes[0]}, nil)
	s.Program = []string{"escape"}
	r := cells.New(s.Cells())
	r.Show()
	if keys, _ := r.Keys(); slices.ContainsFunc(keys, func(u cells.KeyUse) bool { return u.Name == "escape" }) {
		t.Errorf("cells, given: Keys %+v list escape", keys)
	}
	if took, _ := r.Key("escape"); took || r.Focused() != boxField {
		t.Errorf("cells, given: escape taken %v, focused %q; want the host's", took, r.Focused())
	}
}
