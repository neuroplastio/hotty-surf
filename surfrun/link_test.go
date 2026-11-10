package surfrun_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// links is a surface of three links: one the program follows, one to an
// anchor, one to an anchor that no element has.
type links struct{ got []surfrun.Event }

func (m *links) Render() *surfui.Node {
	return surfui.Surface(surfui.Stack(
		surfui.Link("/docs/install", surfui.Str("Install")).Key("go").OnFollow("follow"),
		surfui.Link("#usage", surfui.Str("Usage")).Key("usage"),
		surfui.Link("#nowhere", surfui.Str("Nowhere")).Key("nowhere"),
		surfui.Link("#usage", surfui.Str("Taken")).Key("taken").OnFollow("follow"),
		surfui.Heading("Usage").Anchor("usage"),
	))
}

func (m *links) Do(act surfui.Action, ev surfrun.Event) { m.got = append(m.got, ev) }

// A click on a link with a follow runs it with the href as its value and
// the modifiers held; a link to a #fragment an anchor names has the next
// frame scroll to that element, whatever the scroll option says; one to a
// fragment no element has does nothing, and a follow takes the fragment
// first.
func TestFollowALink(t *testing.T) {
	m := &links{}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	id := func(key string) string { return "r.0.k" + key }

	click := func(key string, mods int) (took, due bool) {
		return s.Event(hotty.Event{Kind: hotty.EventClick, Target: id(key), Mods: mods})
	}
	if took, due := click("go", hotty.ModShift); !took || !due {
		t.Fatalf("a click on a link with a follow: took %v, due %v", took, due)
	}
	f := s.Update()
	if len(m.got) != 1 || m.got[0].Kind != "follow" || m.got[0].Value != "/docs/install" || m.got[0].Mods != hotty.ModShift {
		t.Errorf("the follow heard %+v", m.got)
	}
	if len(f.Reveals) != 0 {
		t.Errorf("a followed link revealed %v", f.Reveals)
	}

	if took, due := click("usage", 0); !took || !due {
		t.Fatalf("a click on a link to an anchor: took %v, due %v", took, due)
	}
	f = s.Update()
	if len(f.Reveals) != 1 || f.Reveals[0].Target != "usage" || f.Reveals[0].Op != "reveal" {
		t.Errorf("reveals %+v, want the anchor usage", f.Reveals)
	}
	if len(m.got) != 1 {
		t.Errorf("a link to an anchor ran the program: %+v", m.got)
	}

	click("nowhere", 0)
	if f = s.Update(); len(f.Reveals) != 0 || len(m.got) != 1 {
		t.Errorf("a fragment no element has: reveals %+v, heard %+v", f.Reveals, m.got)
	}

	click("taken", 0)
	s.Update()
	if len(m.got) != 2 || m.got[1].Value != "#usage" {
		t.Errorf("a follow takes the fragment first: %+v", m.got)
	}
}

// The runtime writes a run's link as an OSC 8 around its text, closes it
// at the row's end, and leaves it out where NoLinks says so; a href
// can't carry an escape of its own.
func TestPaintHyperlinks(t *testing.T) {
	run := func(text, href string) cells.Run {
		return cells.Run{Text: text, Width: len(text), Fg: cells.Fg, Bg: cells.Bg, Href: href}
	}
	s := cells.Screen{Cols: 12, Rows: 1, Lines: [][]cells.Run{{
		run("see ", ""), run("the", "https://x.example/a"), run(" site", "https://x.example/a"),
		run(" ", ""), run("bad", "https://x.example/\x1b]0;x\x07\x1b\\"),
	}}}
	got := surfrun.Paint(s)[0]
	want := "\x1b[0msee " +
		"\x1b]8;;https://x.example/a\x1b\\\x1b[0mthe\x1b[0m site" +
		"\x1b]8;;\x1b\\\x1b[0m " +
		"\x1b]8;;https://x.example/%1B]0;x%07%1B\\\x1b\\\x1b[0mbad" +
		"\x1b]8;;\x1b\\\x1b[0m"
	if got != want {
		t.Errorf("painted %q, want %q", got, want)
	}
	if plain := surfrun.Paint(s.NoLinks())[0]; strings.Contains(plain, "\x1b]8") {
		t.Errorf("NoLinks left a hyperlink: %q", plain)
	}
}
