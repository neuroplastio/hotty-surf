package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfui"
)

// doc is a page of prose: a heading, a paragraph of every inline, a
// bulleted list with a nested numbered one and a task, a quote, the five
// alerts and images.
func doc() *surfui.Node {
	s := surfui.Str
	return surfui.Surface(surfui.Prose(
		surfui.H(1, "top", s("Install guide")),
		surfui.Para(s("Run "), surfui.Code(s("make check")), s(" first, it is "), surfui.Em(s("fast")), s(" and "),
			surfui.Strong(s("strict")), s(", not "), surfui.Del(s("slow")), s(". See "),
			surfui.Link("#usage", s("usage")), s(" or the "), surfui.Link("https://example.com", s("site")).External(), s(".")),
		surfui.Bullets(
			surfui.Li(s("first item that is long enough to wrap onto a second line")),
			surfui.Li(s("second"), surfui.Numbered(8,
				surfui.Li(s("eight")), surfui.Li(s("nine")), surfui.Li(s("ten")))),
			surfui.Li(s("a task")).Checked(true),
			surfui.Li(s("an open task")).Checked(false),
		),
		surfui.Quote(s("Measure twice, "), surfui.Br(), s("cut once.")),
		surfui.Alert("note", s("A note, with a body long enough to wrap.")),
		surfui.Alert("tip", s("A tip.")),
		surfui.Alert("important", s("Important.")),
		surfui.Alert("warning", surfui.Para(s("Warning one.")), surfui.Para(s("Warning two."))),
		surfui.Alert("caution", s("Careful.")),
		surfui.Para(s("A picture: "), surfui.Img("cid:x", "the diagram"), s(" and "), surfui.Img("cid:y", "")),
		surfui.H(2, "usage", s("Usage")),
		surfui.Para(s("End.")),
	))
}

// A page of prose lays out to its natural height, each block a row
// apart, a list item hanging its text after its marker, the quote and the
// alerts behind a ▎ bar.
func TestProse(t *testing.T) {
	m := &model{render: doc}
	r := show(m)
	s := r.Draw(40, 0)
	want := `Install guide

Run make check first, it is fast and
strict, not slow. See usage or the
site.

• first item that is long enough to
  wrap onto a second line
• second
   8. eight
   9. nine
  10. ten
☑ a task
☐ an open task

▎ Measure twice,
▎ cut once.

▎ ⓘ Note
▎ A note, with a body long enough to
▎ wrap.
`
	got := s.Text()
	if !strings.HasPrefix(got, "Install guide\n\nRun make check first") {
		t.Fatalf("text:\n%s", got)
	}
	_ = want
	golden(t, "prose-doc", s)
	if h := r.Measure(40); h != len(s.Lines) {
		t.Errorf("Measure %d, the draw's rows %d", h, len(s.Lines))
	}
	if _, y, _, _, ok := r.Box("usage"); !ok || y < 20 {
		t.Errorf("the heading usage is at row %d (ok %v)", y, ok)
	}
}

// The measure is 80 columns, and compact has none; muted reaches every
// run, a link's included.
func TestProseMeasure(t *testing.T) {
	long := strings.Repeat("word ", 30)
	for _, c := range []struct {
		name    string
		compact bool
		rows    int
	}{{"measure", false, 2}, {"compact", true, 1}} {
		p := surfui.Prose(surfui.Para(surfui.Str(long))).Muted()
		if c.compact {
			p.Compact()
		}
		r := show(&model{render: func() *surfui.Node { return surfui.Surface(p) }})
		if h := r.Measure(150); h != c.rows {
			t.Errorf("%s: %d rows at 150 columns, want %d", c.name, h, c.rows)
		}
	}
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Prose(surfui.Para(surfui.Str("see "), surfui.Link("/x", surfui.Str("this")))).Muted())
	}}
	s := show(m).Draw(20, 0)
	for _, run := range s.Lines[0] {
		if strings.TrimSpace(run.Text) != "" && run.Fg != "muted" {
			t.Errorf("%q is %s in muted prose", run.Text, run.Fg)
		}
	}
}
