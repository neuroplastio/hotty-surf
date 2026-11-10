package surfrun_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// treeModel flattens a tree: a caption, src (a branch) with main and
// util (a branch) with leaf, a caption, and notes. It owns what is open
// and selected: the keys and the chevron only ask, and the model draws
// the rows that show.
type treeModel struct {
	sel   string
	open  map[string]bool
	own   bool // keys="none": the program reads every key
	heard []string
}

func (m *treeModel) Render() *surfui.Node {
	l := surfui.List().Nav().Key("tree").TreeKeys(!m.own)
	add := func(key string, depth int, branch bool) {
		it := surfui.Item(surfui.Text(key)).Key(key).Depth(depth).Branch(branch).
			Selected(key == m.sel).OnClick("pick").OnActivate("open")
		if branch {
			it.Open(m.open[key]).OnToggle("toggle")
		}
		l.Add(it)
	}
	l.Add(surfui.Caption("Project").Key("cap1"))
	add("src", 0, true)
	if m.open["src"] {
		add("main", 1, false)
		add("util", 1, true)
		if m.open["util"] {
			add("leaf", 2, false)
		}
	}
	l.Add(surfui.Caption("More").Key("cap2"))
	add("notes", 0, false)
	return surfui.Surface(l)
}

func (m *treeModel) Do(act surfui.Action, ev surfrun.Event) {
	m.heard = append(m.heard, ev.Kind+" "+ev.Value)
	switch act {
	case "pick":
		m.sel = strings.TrimPrefix(ev.Target, "r.ktree.k")
	case "toggle":
		m.open[ev.Value] = !m.open[ev.Value]
	}
}

// heardNow is what the program heard since the last call.
func (m *treeModel) heardNow() string {
	s := strings.Join(m.heard, ", ")
	m.heard = nil
	return s
}

// A tree answers the keys of ARIA's tree with the handlers its items
// carry: ↑ and ↓ select by @click, across the captions; ← closes an open
// branch or goes to the parent; → opens a closed branch or goes to its
// first child; Enter is @activate; Home and End the ends.
func TestTreeKeys(t *testing.T) {
	m := &treeModel{sel: "src", open: map[string]bool{"src": true}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ktree"})
	for _, st := range []struct{ key, want string }{
		{"arrow_down", "click "},       // main
		{"arrow_down", "click "},       // util
		{"arrow_right", "toggle util"}, // closed: opens
		{"arrow_right", "click "},      // open: its first child, leaf
		{"arrow_left", "click "},       // a leaf: its parent, util
		{"arrow_left", "toggle util"},  // open: closes
		{"arrow_left", "click "},       // closed: its parent, src
		{"end", "click "},              // notes, past the caption
		{"arrow_up", "click "},         // util, across the caption
		{"enter", "activate util"},
		{"arrow_down", "click "}, // notes
		{"arrow_down", ""},       // nothing after it
	} {
		took, _ := s.Key(st.key)
		if got := m.heardNow(); got != st.want || took != (st.want != "") {
			t.Fatalf("%s at %s: heard %q (took %v), want %q", st.key, m.sel, got, took, st.want)
		}
		s.Update()
	}
	if m.sel != "notes" {
		t.Errorf("selected %q, want notes", m.sel)
	}
}

// A leaf has no ← and → beyond its parent: → on one is nothing, and a
// closed root's ← is nothing.
func TestTreeKeysAtTheEdges(t *testing.T) {
	m := &treeModel{sel: "src", open: map[string]bool{}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ktree"})
	if took, _ := s.Key("arrow_left"); took || m.heardNow() != "" {
		t.Error("← on a closed root did something")
	}
	s.Key("end")
	s.Update()
	m.heardNow()
	if took, _ := s.Key("arrow_right"); took || m.heardNow() != "" {
		t.Error("→ on a leaf did something")
	}
}

// keys="none": the tree binds nothing, so a program that reads every key
// itself hears them all, and an arrow is not taken.
func TestTreeKeysOff(t *testing.T) {
	m := &treeModel{sel: "src", open: map[string]bool{}, own: true}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventFocus, Target: "r.ktree"})
	for _, k := range []string{"arrow_down", "arrow_left", "arrow_right", "enter"} {
		if took, _ := s.Key(k); took || m.heardNow() != "" {
			t.Errorf("%s taken with the tree's keys off", k)
		}
	}
}

// A click on a branch's chevron asks the program to toggle it, its
// value the item's key; one on a leaf's, or on the row, is the row's.
func TestTreeChevron(t *testing.T) {
	m := &treeModel{sel: "src", open: map[string]bool{"src": true}}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.ktree.ksrc.kdisclosure"})
	if got := m.heardNow(); got != "toggle src" {
		t.Errorf("the chevron's click: heard %q", got)
	}
	s.Update()
	if m.open["src"] {
		t.Error("src is still open: the model flips what it keeps")
	}
	s.Update()
	s.Event(hotty.Event{Kind: hotty.EventClick, Target: "r.ktree.ksrc"})
	if got := m.heardNow(); got != "click " {
		t.Errorf("the row's click: heard %q", got)
	}
}
