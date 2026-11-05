package cells_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// A Fit row shows its children right to left from its end, each whole,
// and leaves out those it has no room for, from the last. It is never
// narrower than its widest child, min-width: auto: what is beside it is
// cut first.
func TestFitRow(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(
			surfui.Text("pane.new"),
			surfui.Row(surfui.Text("Run"), surfui.Text("Copy"), surfui.Text("Remove")).Gap(2).Fit(),
		).Gap(2))
	}}
	for _, c := range []struct {
		cols int
		want string
	}{
		{40, "pane.new               Remove  Copy  Run"},
		{26, "pane.new         Copy  Run"},
		{13, "pane…     Run"},
	} {
		s := show(m).Draw(c.cols, 1)
		if got := strings.TrimRight(s.Text(), "\n"); got != c.want {
			t.Errorf("at %d cells: %q, want %q", c.cols, got, c.want)
		}
	}
}

// What Keep keeps stays whole: the text beside it is cut.
func TestKeep(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(surfui.Text("Toggle the zoom of the pane"), surfui.Row(surfui.Key("Ctrl"), surfui.Key("Z")).Keep()).Gap(2))
	}}
	s := show(m).Draw(24, 1)
	if got := strings.TrimRight(s.Text(), "\n "); got != "Toggle the …  [Ctrl] [Z]" {
		t.Errorf("%q", got)
	}
}

// A box whose Height was set around a scaled element, for its rows at the
// scale, holds them at 100% in cells: nothing draws below its border.
// Where it would run past the screen, it ends at the bottom, and what it
// holds scrolls where the document does, as a Scroll box's: here not.
func TestScaleInABox(t *testing.T) {
	rows := 0
	m := &model{render: func() *surfui.Node {
		var kids []*surfui.Node
		for i := range rows {
			kids = append(kids, surfui.Text("row "+strconv.Itoa(i)))
		}
		// The rows' lines at 80%, and the line's, on the screen.
		return surfui.Surface(surfui.Dialog(surfui.Stack(kids...).Scale(80)).Height(min(rows*8/10+2, 10)))
	}}
	rows = 6
	s := show(m).Draw(20, 12)
	lines := strings.Split(s.Text(), "\n")
	if !strings.Contains(lines[1], "row 0") || !strings.Contains(lines[6], "row 5") || !strings.HasPrefix(lines[7], "╰") {
		t.Errorf("6 rows at 80%% in cells:\n%s", s.Text())
	}

	rows = 20
	s = show(m).Draw(20, 12)
	lines = strings.Split(s.Text(), "\n")
	if len(lines) < 12 || !strings.HasPrefix(lines[11], "╰") || !strings.Contains(s.Text(), "┃") || strings.Contains(s.Text(), "row 10") {
		t.Errorf("20 rows on 12: the box ends at the screen's bottom, with its bar:\n%s", s.Text())
	}

	// Where the document scrolls, the page does, and the box is whole.
	r := show(m)
	r.Scroll = hotty.ScrollVertical
	s = r.Draw(20, 12)
	if strings.Contains(s.Text(), "┃ │") {
		t.Errorf("the box scrolls on a page that does:\n%s", s.Text())
	}
	if taken, _ := r.Wheel(5, 5, 10); !taken {
		t.Errorf("the wheel didn't scroll the page")
	}
	if s = r.Draw(20, 12); !strings.Contains(s.Text(), "row 19") || !strings.Contains(s.Text(), "╰") {
		t.Errorf("the page scrolled to the end:\n%s", s.Text())
	}
}

// A badge's padding is half a cell a side on a host: in a width that
// holds its text and one cell, the text is whole.
func TestBadgeInItsWidth(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(surfui.Badge("error").Width(6), surfui.Badge("warn").Width(6), surfui.Badge("errors").Width(5)))
	}}
	s := show(m).Draw(20, 1)
	if got := strings.TrimRight(s.Text(), "\n "); got != " error warn erro…" {
		t.Errorf("%q", got)
	}
}

// A path breaks after a "/", as a host's line breaker has it, in a
// wrapping text as in a paragraph, never at a "/" it starts with or
// between two; a word longer than its line with none breaks where the
// line ends.
func TestPathBreaks(t *testing.T) {
	const path = "state /srv/apps/1000/app.x1/cells-17/state/agents"
	for _, c := range []struct {
		name string
		node *surfui.Node
		want string
	}{
		{"a text", surfui.Text(path).Wrap(),
			"state /srv/apps/\n1000/app.x1/\ncells-17/state/\nagents"},
		{"a paragraph", surfui.Line(surfui.Text(path)).Wrap(),
			"state /srv/apps/\n1000/app.x1/\ncells-17/state/\nagents"},
		{"two slashes", surfui.Text("see http://example.com/a/b").Wrap(),
			"see http://\nexample.com/a/b"},
		{"no slash", surfui.Text("so supercalifragilisticexpialidocious").Wrap(),
			"so\nsupercalifragilistic\nexpialidocious"},
		{"a slash it starts with", surfui.Text("/abcdefghijklmnopqrstuvwxyz").Wrap(),
			"/abcdefghijklmnopqrs\ntuvwxyz"},
	} {
		m := &model{render: func() *surfui.Node { return surfui.Surface(c.node) }}
		s := show(m).Draw(20, 5)
		var lines []string
		for _, l := range strings.Split(s.Text(), "\n") {
			if l = strings.TrimRight(l, " "); l != "" {
				lines = append(lines, l)
			}
		}
		if got := strings.Join(lines, "\n"); got != c.want {
			t.Errorf("%s:\n%s\nwant\n%s", c.name, got, c.want)
		}
	}
}

// A palette in Compact, as plx's is written: its query and status rows
// padded 3 across and 1 down, its rows Items. A pad of one step down is
// no row, on neither side; 3 across is a cell, as an Item's own padding
// is, so every row's text starts in one column.
func TestCompactPads(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Stack(
			surfui.Row(surfui.Text("❯"), surfui.Text("sp")).Gap(2).Pad(3, 1),
			surfui.Rule(),
			surfui.List(
				surfui.Item(surfui.Text("Pane: Split right").Grow(), surfui.Text("ctrl+b %")),
				surfui.Item(surfui.Text("Pane: Close").Grow(), surfui.Text("ctrl+b x")),
			),
			surfui.Rule(),
			surfui.Row(surfui.Text("pane.split").Grow(), surfui.Text("Run")).Pad(3, 1),
		)).Density(surfui.Compact)
	}}
	s := show(m).Draw(40, 8)
	want := []string{
		" ❯ sp",
		"────────────────────────────────────────",
		" Pane: Split right             ctrl+b %",
		" Pane: Close                   ctrl+b x",
		"────────────────────────────────────────",
		" pane.split                         Run",
	}
	got := strings.Split(strings.TrimRight(s.Text(), "\n "), "\n")
	for i := range got {
		got[i] = strings.TrimRight(got[i], " ")
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A FitStart row shows its children left to right from its start, each
// whole, and leaves out those it has no room for, from the last. Its
// first is cut only where it alone is wider than the row.
func TestFitStartRow(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(
			surfui.Text("pane.new"),
			surfui.Row(surfui.Text("Run"), surfui.Text("Copy"), surfui.Text("Remove")).Gap(2).FitStart(),
		).Gap(2))
	}}
	for _, c := range []struct {
		cols int
		want string
	}{
		{40, "pane.new  Run  Copy  Remove"},
		{22, "pane.new  Run  Copy"},
		{17, "pane.new  Run"},
		{13, "pane…  Run"},
		{9, "…  Run"},
	} {
		s := show(m).Draw(c.cols, 1)
		if got := strings.TrimRight(s.Text(), "\n"); got != c.want {
			t.Errorf("at %d cells: %q, want %q", c.cols, got, c.want)
		}
	}
	// Where the row's width is set, the first is cut if it alone is wider.
	m = &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Row(
			surfui.Row(surfui.Text("Remove all"), surfui.Text("Copy")).Gap(2).FitStart().Width(6),
		))
	}}
	s := show(m).Draw(12, 1)
	if got := strings.TrimRight(s.Text(), "\n"); got != "Remov…" {
		t.Errorf("a first wider than its row: %q", got)
	}
}
