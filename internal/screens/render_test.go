package screens

import (
	"testing"

	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// No expression fails as a screen renders, in either mode, in the states
// the tests reach and a few more: a mode would leave the part out and say
// so on stderr only (docs/surf.md, "Expressions"), which no other test
// sees. And the two modes draw the same document, ids and all. (A screen
// with a cells view is wrapped, so its View's OnError is out of reach:
// this renders it by its templates, each mode's way.)
func TestRendersClean(t *testing.T) {
	check := func(what string, c any) {
		t.Helper()
		var inst surf.Instance
		var compiled func(e *surf.Errs) []*surfui.Node
		switch c := c.(type) {
		case *Notices:
			inst, compiled = surfNotices{c}, func(e *surf.Errs) []*surfui.Node { return surfRenderNotices(e, nil, c, nil) }
		case *Settings:
			inst, compiled = surfSettings{c}, func(e *surf.Errs) []*surfui.Node { return surfRenderSettings(e, nil, c, nil) }
		case *Palette:
			inst, compiled = surfPalette{c}, func(e *surf.Errs) []*surfui.Node { return surfRenderPalette(e, nil, c, nil) }
		case *Agents:
			inst, compiled = surfAgents{c}, func(e *surf.Errs) []*surfui.Node { return surfRenderAgents(e, nil, c, nil) }
		}
		n, errs := surfSet().Render(inst)
		if n == nil || len(errs) > 0 {
			t.Errorf("%s, interpreted: %v", what, errs)
		}
		var e surf.Errs
		out := compiled(&e)
		if len(out) != 1 || len(e.Errors()) > 0 {
			t.Errorf("%s, compiled: %d nodes, %v", what, len(out), e.Errors())
		}
		if n != nil && len(out) == 1 {
			surfui.Resolve(n, "r")
			surfui.Resolve(out[0], "r")
			if a, b := surfui.HTML(n), surfui.HTML(out[0]); a != b {
				t.Errorf("%s: the modes draw apart:\ninterpreted %s\ncompiled    %s", what, a, b)
			}
		}
	}

	n := NewNotices()
	check("notices", n)
	for _, f := range noticeFilters {
		n.SetFilter(f)
		check("notices, "+f, n)
	}
	n.SetFilter("Errors")
	n.Dismiss(n.Selected())
	n.Dismiss(n.Selected())
	check("notices, none shown", n)
	n.Tick()
	check("notices, a tick", n)

	s := NewSettings(demodata.Mock())
	for i := range s.Secs {
		s.Pick(i)
		for _, st := range s.Shown() {
			s.Drafts[st.Name] = "typed"
		}
		check("settings, "+s.Title(), s)
	}
	s.Search("palette")
	check("settings, a search", s)
	s.Search("qqqq")
	check("settings, no match", s)

	p := NewPalette(demodata.Mock())
	check("palette", p)
	for _, q := range []string{"split", "@", "@claude", "theme ", "pane.split ", "pane.split down ", "qqqq", "palette scale"} {
		p.SetQuery(q)
		check("palette, "+q, p)
		p.TogglePanel()
		check("palette, "+q+", the panel", p)
		p.TogglePanel()
	}
	p.setLine("config.set palette.style.scale 30")
	p.Enter()
	check("palette, refused", p)
	p.Resize(64, 24)
	check("palette, 24 rows", p)

	src := demodata.Mock()
	l := NewAgents(src)
	check("agents, before the answer", l)
	list, _ := src.Agents()
	l.Loaded = true
	l.answer(append(list, demodata.Agent{Address: "devbox:x-1", Name: "x-1", Host: "devbox", Offline: true, Attention: true}), nil)
	check("agents", l)
	l.Refused = "no agent gone-1 now"
	check("agents, refused", l)
	l.answer(nil, nil)
	check("agents, none", l)
}
