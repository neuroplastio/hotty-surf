package cells_test

import (
	"strconv"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// composer is margin's composer: a field of several lines over a row of
// buttons, in a stack that traps Tab. Its program hears every edit
// (@input) and the field's height (@height); what sends is its own.
type composer struct {
	model
	text    string
	caret   string
	heights []string
	sent    []string // the text the program had when ctrl+s came
	rows    [2]int
	input   bool
}

func newComposer(text string) *composer {
	c := &composer{text: text, caret: "keep", rows: [2]int{2, 5}, input: true}
	c.render = func() *surfui.Node {
		f := surfui.Field().Key("text").Rows(c.rows[0], c.rows[1]).Value(c.text).Caret(c.caret).Autofocus().
			Placeholder("Write a comment").
			OnHeight(func(e surfrun.Event) { c.heights = append(c.heights, strconv.Itoa(e.Rows)) }).
			Bind("ctrl+s", "send", func() { c.sent = append(c.sent, c.text) })
		if c.input {
			f.OnInput(func(e surfrun.Event) { c.text = e.Value })
		} else {
			f.OnChange(func(e surfrun.Event) { c.text = e.Value })
		}
		return surfui.Surface(surfui.Stack(f, surfui.Row(
			surfui.Button("Save").Key("save").OnClick(func() {}),
			surfui.Button("Keep").Key("keep").OnClick(func() {}),
		)).Trap()).Density(surfui.Compact)
	}
	return c
}

func lines(s cells.Screen) []string { return strings.Split(s.Text(), "\n") }

func typeText(r *cells.Renderer, text string) {
	for _, ch := range text {
		switch ch {
		case '\n':
			r.Key("enter")
		case ' ':
			r.Key("space")
		default:
			r.Key(string(ch))
		}
	}
}

// A field of several lines wraps its text at its width, shows the rows it
// has, and puts the caret at its row and column.
func TestAreaWrapsAndCaret(t *testing.T) {
	c := newComposer("one two three four five six")
	r := show(c)
	s := r.Draw(14, 8)
	got := lines(s)
	// 12 cells of text, a cell of room each side; the caret keeps its start.
	for i, want := range []string{" one two ", " three four ", " five six"} {
		if i < len(got) && strings.TrimRight(got[i], " ") != strings.TrimRight(want, " ") {
			t.Errorf("row %d: %q, want %q", i, got[i], want)
		}
	}
	if !s.Caret.Visible || s.Caret.X != 1 || s.Caret.Y != 0 {
		t.Errorf("caret at the start: %+v", s.Caret)
	}
	r.Key("ctrl+end")
	s = r.Draw(14, 8)
	if s.Caret.X != 1+len("five six") || s.Caret.Y != 2 {
		t.Errorf("caret at the end: %+v\n%s", s.Caret, s.Text())
	}
}

// Enter and Shift+Enter type a newline; what sends is the program's.
func TestAreaEnter(t *testing.T) {
	c := newComposer("")
	r := show(c)
	r.Draw(30, 8)
	typeText(r, "ab\ncd")
	r.Key("shift+enter")
	typeText(r, "e")
	if c.text != "ab\ncd\ne" {
		t.Errorf("text %q", c.text)
	}
	s := r.Draw(30, 8)
	if s.Caret.Y != 2 || s.Caret.X != 2 {
		t.Errorf("caret %+v", s.Caret)
	}
	r.Key("ctrl+s")
	if len(c.sent) != 1 || c.sent[0] != "ab\ncd\ne" {
		t.Errorf("sent %q", c.sent)
	}
}

// Up and Down go by visual row and keep the column; Home and End go to
// the row's ends.
func TestAreaRows(t *testing.T) {
	c := newComposer("aaaa bbbb cccc\nd")
	r := show(c)
	r.Draw(8, 8) // 6 cells: aaaa / bbbb / cccc / d
	r.Key("ctrl+end")
	r.Key("arrow_up") // the column of d (1) on cccc
	s := r.Draw(8, 8)
	if s.Caret.Y != 2 || s.Caret.X != 2 {
		t.Errorf("up from d: %+v\n%s", s.Caret, s.Text())
	}
	r.Key("home")
	if s = r.Draw(8, 8); s.Caret.X != 1 || s.Caret.Y != 2 {
		t.Errorf("home: %+v", s.Caret)
	}
	r.Key("end")
	if s = r.Draw(8, 8); s.Caret.Y != 2 {
		t.Errorf("end: %+v", s.Caret)
	}
}

// A selection spans rows, drawn on the selection's fill, and typing over
// it replaces it.
func TestAreaSelection(t *testing.T) {
	c := newComposer("aaaa bbbb cccc")
	r := show(c)
	r.Draw(8, 8)
	r.Key("shift+arrow_down")
	r.Key("shift+arrow_down")
	s := r.Draw(8, 8)
	sel := 0
	for _, row := range s.Lines {
		for _, run := range row {
			if run.Bg == cells.Sel {
				sel += len([]rune(run.Text))
			}
		}
	}
	if sel != 10 {
		t.Errorf("%d cells selected across rows, want 10:\n%s", sel, s.Golden())
	}
	typeText(r, "x")
	if c.text != "xcccc" {
		t.Errorf("typed over the selection: %q", c.text)
	}
}

// A paste of many lines is kept whole: LF, tabs as spaces, CRLF as LF, one
// step to undo.
func TestAreaPaste(t *testing.T) {
	c := newComposer("")
	r := show(c)
	r.Draw(30, 8)
	r.Paste("one\r\ntwo\tx\nthree\n")
	if c.text != "one\ntwo    x\nthree\n" {
		t.Errorf("paste: %q", c.text)
	}
	r.Key("ctrl+z")
	if c.text != "" {
		t.Errorf("undo: %q", c.text)
	}
	r.Key("ctrl+shift+z")
}

// @height tells the program the rows the field has as it grows, from when
// it first shows, and the field is that tall.
func TestAreaHeight(t *testing.T) {
	c := newComposer("")
	r := show(c)
	s := r.Draw(20, 12)
	if len(c.heights) != 1 || c.heights[0] != "2" {
		t.Fatalf("heights %q", c.heights)
	}
	buttons := func(s cells.Screen) int {
		for y, l := range lines(s) {
			if strings.Contains(l, "Save") {
				return y
			}
		}
		return -1
	}
	if y := buttons(s); y != 2 {
		t.Errorf("buttons at row %d under a field of 2 rows:\n%s", y, s.Text())
	}
	typeText(r, "a\nb\nc")
	s = r.Draw(20, 12)
	if got := strings.Join(c.heights, " "); got != "2 3" {
		t.Errorf("heights %q", got)
	}
	if y := buttons(s); y != 3 {
		t.Errorf("buttons at row %d under a field of 3 rows:\n%s", y, s.Text())
	}
	typeText(r, "\nd\ne\nf\ng")
	s = r.Draw(20, 12)
	if got := strings.Join(c.heights, " "); got != "2 3 5" {
		t.Errorf("heights %q (most 5)", got)
	}
	if y := buttons(s); y != 5 {
		t.Errorf("buttons at row %d under a field of at most 5 rows:\n%s", y, s.Text())
	}
	// Past its most it scrolls, the caret's row in view.
	if s.Caret.Y != 4 || !strings.Contains(s.Text(), "g") || strings.Contains(s.Text(), "a\n") {
		t.Errorf("scrolled:\n%s\ncaret %+v", s.Text(), s.Caret)
	}
}

// Tab and Shift+Tab stay in a trap: from the last button to the field.
func TestTrap(t *testing.T) {
	c := newComposer("x")
	r := show(c)
	r.Draw(20, 8)
	if r.Focused() != "r.0.ktext" {
		t.Fatalf("focused %q", r.Focused())
	}
	var order []string
	for range 4 {
		r.Key("tab")
		order = append(order, r.Focused())
	}
	if strings.Join(order, " ") != "r.0.1.ksave r.0.1.kkeep r.0.ktext r.0.1.ksave" {
		t.Errorf("tab: %q", order)
	}
	r.Key("shift+tab")
	r.Key("shift+tab")
	if r.Focused() != "r.0.1.kkeep" {
		t.Errorf("shift+tab: %q", r.Focused())
	}
}

// Never lose words.
//
// An edit reaches the model before any key the program handles next: with
// @input it is heard already; a field that only commits is committed first.
func TestAreaEditBeforeKey(t *testing.T) {
	for _, input := range []bool{true, false} {
		c := newComposer("")
		c.input = input
		r := show(c)
		r.Draw(30, 8)
		typeText(r, "last words")
		r.Key("ctrl+s")
		if len(c.sent) != 1 || c.sent[0] != "last words" {
			t.Errorf("input=%v: ctrl+s sent %q", input, c.sent)
		}
		var order []string
		for _, l := range c.log {
			order = append(order, strings.Fields(l)[0])
		}
		last := order[len(order)-1]
		if last != "key" && last != "change" && last != "input" {
			t.Errorf("input=%v: log %q", input, c.log)
		}
	}
}

// A host that lost the document gets the value back, and a program that
// sets the same value again moves nothing: not the caret, not the
// selection, not the history.
func TestAreaSameValueMovesNothing(t *testing.T) {
	c := newComposer("hello world")
	r := show(c)
	r.Draw(30, 8)
	r.Key("ctrl+end")
	typeText(r, "!")
	r.Key("arrow_left")
	r.Key("shift+arrow_left")
	before := r.Draw(30, 8)
	for range 3 {
		r.Update() // the model renders the same :value again
	}
	after := r.Draw(30, 8)
	if before.Golden() != after.Golden() {
		t.Errorf("same value moved something:\n%s\n%s", before.Golden(), after.Golden())
	}
	r.Key("ctrl+z") // the history is whole
	if c.text != "hello world" {
		t.Errorf("undo after re-render: %q", c.text)
	}
}

// A value the program sets goes where caret says: keep (clamped), end, start.
func TestAreaCaretModes(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want int
	}{{"keep", 3}, {"end", 12}, {"start", 0}} {
		c := newComposer("abcdef")
		c.caret = tc.mode
		r := show(c)
		r.Draw(30, 8)
		r.Key("arrow_right")
		r.Key("arrow_right")
		r.Key("arrow_right")
		c.text = "abcdef ghijk"
		r.Update()
		s := r.Draw(30, 8)
		if s.Caret.X-1 != tc.want {
			t.Errorf("caret=%s: column %d, want %d\n%s", tc.mode, s.Caret.X-1, tc.want, s.Text())
		}
	}
}

// injecting is a program that also sends the a=field commands set.
type injecting struct {
	cells.Program
	set []cells.FieldSet
}

func (p *injecting) Update() cells.Frame {
	f := p.Program.Update()
	f.Fields = append(f.Fields, p.set...)
	p.set = nil
	return f
}

// A value for a field the user has edited since is stale and dropped
// (SPEC §6.4); one against the rev they left it at is put in, with its
// selection where the program said.
func TestAreaStale(t *testing.T) {
	c := newComposer("")
	p := &injecting{Program: surfrun.NewSurface(c, nil).Cells()}
	r := cells.New(p)
	r.Show()
	r.Draw(30, 8)
	typeText(r, "mine") // four edits: rev 4
	put := func(rev int, v string, sel *[2]int) {
		p.set = []cells.FieldSet{{ID: "r.0.ktext", Change: hotty.FieldChange{Rev: rev, Value: &v, Sel: sel}}}
		r.Update()
	}
	put(0, "old", nil)
	if s := r.Draw(30, 8); !strings.Contains(s.Text(), "mine") || strings.Contains(s.Text(), "old") {
		t.Errorf("stale value:\n%s", s.Text())
	}
	put(4, "newer", &[2]int{1, 3})
	s := r.Draw(30, 8)
	if !strings.Contains(s.Text(), "newer") || s.Caret.X != 1+3 {
		t.Errorf("current value, selection 1..3:\n%s\ncaret %+v", s.Text(), s.Caret)
	}
	sel := 0
	for _, row := range s.Lines {
		for _, run := range row {
			if run.Bg == cells.Sel {
				sel += len([]rune(run.Text))
			}
		}
	}
	if sel != 2 {
		t.Errorf("%d cells selected, want 2 (ew)", sel)
	}
}

// The goldens: a field of several lines empty with its placeholder, with a
// short comment and its caret, and with a text past its most rows, scrolled
// to the caret, and a selection across rows.
func TestAreaGolden(t *testing.T) {
	c := newComposer("")
	r := show(c)
	golden(t, "area-empty", r.Draw(40, 8))

	c = newComposer("Looks good to me, one nit on the name.")
	r = show(c)
	r.Key("ctrl+end")
	golden(t, "area-short", r.Draw(40, 8))

	c = newComposer("The loop keeps the lock while it retries, so a second run waits behind it.\n\nFix: take the lock after the first attempt, not before; and put the retry's reason in the error, or no one will know why it waited. Then add the test.")
	c.rows = [2]int{2, 4}
	r = show(c)
	r.Key("ctrl+end")
	golden(t, "area-overflow", r.Draw(40, 8))

	r.Key("ctrl+home")
	r.Key("shift+arrow_down")
	r.Key("shift+arrow_down")
	r.Key("shift+arrow_right")
	golden(t, "area-selection", r.Draw(40, 8))
}

// A press in a field of several lines puts the caret under it: on the row
// pressed, at the cell, and after the row's text for a press beyond it.
func TestAreaPressPlacesCaret(t *testing.T) {
	c := newComposer("one two three four five six")
	r := show(c)
	r.Draw(14, 8)
	// The second row is " three four ": its text starts a cell in.
	r.Press(4, 1, 0)
	r.Release(4, 1, 0)
	s := r.Draw(14, 8)
	if !s.Caret.Visible || s.Caret.X != 4 || s.Caret.Y != 1 {
		t.Fatalf("caret after the press: %+v", s.Caret)
	}
	typeText(r, "x")
	if c.text != "one two thrxee four five six" {
		t.Errorf("text %q", c.text)
	}
	// Beyond the last row's text: its end.
	r.Press(12, 2, 0)
	r.Release(12, 2, 0)
	typeText(r, "y")
	if c.text != "one two thrxee four five sixy" {
		t.Errorf("text %q", c.text)
	}
}

// The wheel over a field of several lines scrolls its rows and leaves the
// caret where it is; the next draw does not bring the caret's row back.
func TestAreaWheelScrollsRows(t *testing.T) {
	text := "l0\nl1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9"
	c := newComposer(text)
	r := show(c)
	r.Draw(14, 8)
	if taken, redraw := r.Wheel(3, 1, 1); !taken || !redraw {
		t.Fatalf("wheel: taken %v, redraw %v", taken, redraw)
	}
	for range 2 {
		s := r.Draw(14, 8)
		if got := strings.TrimSpace(lines(s)[0]); got != "l3" {
			t.Fatalf("first row %q after the wheel, want l3\n%s", got, s.Text())
		}
	}
	// Typing brings the caret back into view.
	typeText(r, "x")
	if got := strings.TrimSpace(lines(r.Draw(14, 8))[0]); got != "xl0" {
		t.Errorf("first row %q after typing, want xl0", got)
	}
	// Past the last row the wheel stops taking it.
	for range 6 {
		r.Wheel(3, 1, 1)
	}
	if taken, _ := r.Wheel(3, 1, 1); taken {
		t.Errorf("the wheel at the field's end still taken")
	}
}
