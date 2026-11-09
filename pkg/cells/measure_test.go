package cells_test

import (
	"strconv"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfui"
)

// thread is a page of comments, each a Stack with a wrapping body, keyed
// as an :each would: comments > c<i> > body.
func thread(n int) *model {
	return &model{render: func() *surfui.Node {
		var rows []*surfui.Node
		for i := range n {
			rows = append(rows, surfui.Stack(
				surfui.Text("author "+strconv.Itoa(i)).Key("by"),
				surfui.Line(
					surfui.Text("one two three four five six seven eight nine ten").Key("lead"),
					surfui.Text("eleven twelve thirteen fourteen fifteen sixteen").Key("tail"),
				).Class("u-wrap").Key("body"),
			).Key("c"+strconv.Itoa(i)))
		}
		return surfui.Surface(surfui.Heading("Thread"), surfui.Stack(rows...).Key("comments"))
	}}
}

// Draw at rows 0 is as tall as the content, and Measure says how tall.
func TestMeasureAndFullDraw(t *testing.T) {
	r := show(thread(5))
	for _, cols := range []int{20, 40, 80} {
		h := r.Measure(cols)
		s := r.Draw(cols, 0)
		if s.Rows != h || len(s.Lines) != h || s.Cols != cols {
			t.Errorf("at %d: Measure %d, Draw %dx%d with %d lines", cols, h, s.Cols, s.Rows, len(s.Lines))
		}
		// The same lines as a screen that was tall enough all along.
		if want := show(thread(5)).Draw(cols, h); s.Text() != want.Text() {
			t.Errorf("at %d: full draw differs from a draw at %d rows:\n%s\n%s", cols, h, s.Text(), want.Text())
		}
	}
	if a, b := r.Measure(20), r.Measure(80); a <= b {
		t.Errorf("narrower is not taller: %d at 20, %d at 80", a, b)
	}
}

// Box finds any element, not only those a press reaches, in document
// coordinates over the whole height: where the full draw shows it.
func TestBox(t *testing.T) {
	r := show(thread(30))
	s := r.Draw(40, 0)
	if _, _, _, _, ok := r.Box("nowhere"); ok {
		t.Error("Box found an element that isn't there")
	}
	x, y, w, h, ok := r.BoxOf("comments", "c29", "by")
	if !ok || x != 0 || w < 9 || h != 1 {
		t.Fatalf("c29's author: %d,%d %dx%d %v", x, y, w, h, ok)
	}
	if line := strings.Split(s.Text(), "\n")[y]; !strings.HasPrefix(line, "author 29") {
		t.Errorf("row %d of the full draw is %q", y, line)
	}
	// By id and by key path are the same box.
	for id, n := range r.Index() {
		if surfui.KeyOf(n) == "c3" {
			bx, by, bw, bh, ok := r.Box(id)
			cx, cy, cw, ch, ok2 := r.BoxOf("comments", "c3")
			if !ok || !ok2 || bx != cx || by != cy || bw != cw || bh != ch {
				t.Errorf("Box(%q) = %d,%d %dx%d, BoxOf = %d,%d %dx%d", id, bx, by, bw, bh, cx, cy, cw, ch)
			}
		}
	}
	if _, _, _, _, ok := r.BoxOf("comments", "c3", "nope"); ok {
		t.Error("BoxOf found a key that isn't there")
	}
	// Document coordinates don't move with the scroll.
	r.Draw(40, 5)
	if _, y2, _, _, _ := r.BoxOf("comments", "c29", "by"); y2 != y {
		t.Errorf("c29 at row %d after a short draw, %d in the full one", y2, y)
	}
}

// An element a wrap split over rows has a box that covers them, and
// Rows says its first and last.
func TestRows(t *testing.T) {
	r := show(thread(1))
	r.Draw(24, 0)
	first, last, ok := r.RowsOf("comments", "c0", "body")
	if !ok || last <= first {
		t.Fatalf("body at 24 cells: rows %d to %d, %v", first, last, ok)
	}
	_, y, _, h, _ := r.BoxOf("comments", "c0", "body")
	if y != first || y+h-1 != last {
		t.Errorf("the box is rows %d to %d, Rows says %d to %d", y, y+h-1, first, last)
	}
	// The lead span wraps too, and ends on the row the tail starts on or before.
	lf, ll, _ := r.RowsOf("comments", "c0", "body", "lead")
	tf, tl, _ := r.RowsOf("comments", "c0", "body", "tail")
	if lf != first || ll < lf || tf < ll || tl != last {
		t.Errorf("lead %d-%d, tail %d-%d, body %d-%d", lf, ll, tf, tl, first, last)
	}
}

// An unchanged tree at the same width is laid out once.
func TestLayoutCached(t *testing.T) {
	m := thread(20)
	r := show(m)
	a := r.Draw(40, 0)
	b := r.Draw(40, 0)
	if a.Gen != b.Gen || a.Text() != b.Text() {
		t.Error("a second draw of the same tree differs")
	}
	if n := testing.AllocsPerRun(20, func() { r.Draw(40, 0) }); n > 4 {
		t.Errorf("a cached draw allocates %v times", n)
	}
	// A new tree is drawn anew.
	m.render = func() *surfui.Node { return surfui.Surface(surfui.Text("other")) }
	r.Update()
	if s := r.Draw(40, 0); s.Rows != 1 || !strings.HasPrefix(s.Text(), "other") {
		t.Errorf("after the tree changed: %q", s.Text())
	}
}

// A draw kept for the same again is not kept past input: scrolling,
// which the layout doesn't depend on, still shows.
func TestCacheDoesNotOutliveInput(t *testing.T) {
	r := show(thread(30))
	r.Scroll = hotty.ScrollVertical
	a := r.Draw(40, 10)
	if b := r.Draw(40, 10); b.Gen != a.Gen {
		t.Fatal("the same draw changed")
	}
	r.Wheel(1, 1, 3)
	b := r.Draw(40, 10)
	if b.Gen == a.Gen || b.Text() == a.Text() {
		t.Errorf("a wheel left the screen as it was:\n%s", b.Text())
	}
	// Another size, then back: each is its own.
	if c := r.Draw(30, 10); c.Cols != 30 {
		t.Errorf("30 cells asked, %d drawn", c.Cols)
	}
	if d := r.Draw(40, 10); d.Text() != b.Text() || d.Cols != 40 {
		t.Errorf("back at 40 cells:\n%s", d.Text())
	}
	// And Box still says where the elements are, over the whole height.
	if _, y, _, _, ok := r.BoxOf("comments", "c29"); !ok || y < 10 {
		t.Errorf("c29 at row %d (%v)", y, ok)
	}
}
