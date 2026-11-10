package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfui"
)

// A link in a paragraph is underlined in the link tone, its runs carry its
// href, and they split where it changes; the paragraph wraps as a line
// does.
func TestLinkInALine(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Line(
			surfui.Text("See the "),
			surfui.Link("/docs/install", surfui.Text("install guide")),
			surfui.Text(" first, or "),
			surfui.Link("https://example.com/x", surfui.Str("the site")).External(),
			surfui.Text("."),
		).Wrap())
	}}
	r := show(m)
	s := r.Draw(24, 0)
	if got, want := s.Text(), "See the install guide\nfirst, or the site.\n"; got != want {
		t.Errorf("text %q, want %q", got, want)
	}
	var links []string
	for y, l := range s.Lines {
		for _, run := range l {
			if run.Href != "" {
				links = append(links, strings.TrimSpace(run.Text)+"@"+run.Href+"@"+string(run.Fg)+"@"+run.Attr.String()+"@"+string(rune('0'+y)))
			}
		}
	}
	want := []string{
		"install guide@/docs/install@accent@underline@0",
		"the site@https://example.com/x@accent@underline@1",
	}
	if strings.Join(links, "|") != strings.Join(want, "|") {
		t.Errorf("links %q, want %q", links, want)
	}
	golden(t, "link-line", s)
}

// A link in a text that wraps breaks where the text does, each part its
// own run of the link.
func TestLinkWraps(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Line(
			surfui.Text("Read "),
			surfui.Link("#usage", surfui.Str("the long usage section"), surfui.El("em", surfui.Str(" and more")).Class("u-em")),
		).Wrap())
	}}
	s := show(m).Draw(14, 0)
	if got, want := s.Text(), "Read the long\nusage section\nand more\n"; got != want {
		t.Errorf("text %q, want %q", got, want)
	}
	for y, l := range s.Lines[:3] {
		for _, run := range l {
			if strings.TrimSpace(run.Text) != "" && strings.TrimSpace(run.Text) != "Read" && run.Href != "#usage" {
				t.Errorf("row %d: %q has no href", y, run.Text)
			}
		}
	}
}

// Tab walks the links; a link that is a hyperlink to somewhere else and
// that no follow answers takes no focus, as on a host. Enter follows.
func TestLinkTabAndEnter(t *testing.T) {
	var followed []string
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(
			surfui.Link("/a", surfui.Str("alpha")).OnFollow(func() { followed = append(followed, "a") }),
			surfui.Link("https://x.example", surfui.Str("beta")).External(),
			surfui.Link("/c", surfui.Str("gamma")).OnFollow(func() { followed = append(followed, "c") }),
		))
	}}
	r := show(m)
	r.Draw(20, 0)
	r.Key("tab")
	r.Key("enter")
	r.Key("tab")
	r.Key("enter")
	if strings.Join(followed, "") != "ac" {
		t.Errorf("followed %q, want a then c: the external link takes no Tab", followed)
	}
	r.Key("shift+tab")
	r.Key("enter")
	if strings.Join(followed, "") != "aca" {
		t.Errorf("followed %q after Shift+Tab", followed)
	}
}

// The first link's focus is a fill on the link, and a press on a link's
// cells follows it, with the modifiers held.
func TestLinkPress(t *testing.T) {
	var got []string
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Line(surfui.Text("go "), surfui.Link("/x", surfui.Str("there")).OnFollow(func() { got = append(got, "x") })).Wrap())
	}}
	r := show(m)
	r.Draw(20, 0)
	r.Press(4, 0, 0)
	r.Release(4, 0, 0)
	if strings.Join(got, "") != "x" {
		t.Errorf("followed %q", got)
	}
	if r.Screen().HitAt(0, 0) != "" {
		t.Errorf("the text before the link is no link")
	}
	_ = cells.Run{}
}
