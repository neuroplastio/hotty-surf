package surfrun_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// drafter is margin's composer on a host: a field of several lines whose
// text is the model's, with a key of the program's (ctrl+s) that sends it.
type drafter struct {
	text    string
	bound   bool // the field has a :value
	caret   string
	sent    []string
	changes []string
	heard   []string
}

const draftField = "r.ktext"

func (m *drafter) Render() *surfui.Node {
	f := surfui.Field().Key("text").Rows(3, 8).Autofocus().Caret(m.caret).
		OnInput(func(e surfrun.Event) { m.text = e.Value; m.heard = append(m.heard, "input "+e.Value) }).
		OnChange(func(e surfrun.Event) { m.changes = append(m.changes, e.Value) }).
		Bind("ctrl+s", "send", func() { m.sent = append(m.sent, m.text) })
	if m.bound {
		f.Value(m.text)
	}
	return surfui.Surface(f)
}

func (m *drafter) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func(surfrun.Event):
		f(ev)
	case func():
		f()
	}
}

// A multi-row field is a textarea: its value its content, the first
// newline kept, its rows and wrap and spell check on it; it asks for the
// caret.
func TestAreaDocument(t *testing.T) {
	m := &drafter{bound: true, text: "\nline <b>"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	h := surfui.HTML(s.Tree())
	for _, want := range []string{`<textarea id="r.ktext"`, `rows="3"`, `data-rows="3..8"`, `wrap="soft"`, `spellcheck="true"`,
		`data-on="`, `caret`, "\n\nline &lt;b&gt;</textarea>"} {
		if !strings.Contains(h, want) {
			t.Errorf("%s is not in\n%s", want, h)
		}
	}
	if strings.Contains(h, ` value=`) {
		t.Errorf("a textarea has no value attribute:\n%s", h)
	}
}

// Enter in a field of several lines is a newline, the host's: the surface
// commits nothing on it.
func TestAreaEnterCommitsNothing(t *testing.T) {
	m := &drafter{bound: true}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftField, Value: "a\nb", Rev: 1})
	s.Key("enter")
	s.Update()
	if len(m.changes) != 0 {
		t.Errorf("enter committed %q", m.changes)
	}
}

// An edit reaches the model before the key the program handles next.
func TestAreaEditBeforeKey(t *testing.T) {
	m := &drafter{bound: true}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftField, Value: "last words", Rev: 10})
	s.Key("ctrl+s")
	if len(m.sent) != 1 || m.sent[0] != "last words" {
		t.Errorf("ctrl+s sent %q", m.sent)
	}
}

// A value the model sets goes with the selection its caret says: the
// start; the end (the host's own); or where the user's caret was, as far
// as the new value goes. The same value again sends nothing.
func TestAreaSetValue(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want *[2]int
	}{{"start", &[2]int{0, 0}}, {"end", nil}, {"keep", &[2]int{3, 3}}, {"", &[2]int{3, 3}}} {
		m := &drafter{bound: true, caret: tc.mode, text: "abcdef"}
		s := surfrun.NewSurface(m, nil)
		s.Show()
		s.Event(hotty.Event{Kind: hotty.EventCaret, Target: draftField, Sel: &[2]int{3, 3}, Rev: 0})
		s.Update()
		m.text = "xyzw"
		f := s.Update()
		if len(f.Fields) != 1 {
			t.Fatalf("caret=%q: fields %+v", tc.mode, f.Fields)
		}
		got := f.Fields[0].Change.Sel
		if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
			t.Errorf("caret=%q: sel %v, want %v", tc.mode, got, tc.want)
		}
		if f := s.Update(); len(f.Fields) != 0 {
			t.Errorf("caret=%q: the same value again: %+v", tc.mode, f.Fields)
		}
	}
}

// keep clamps to the new text, on a character's start.
func TestAreaKeepClamps(t *testing.T) {
	m := &drafter{bound: true, text: "abcdef"}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventCaret, Target: draftField, Sel: &[2]int{4, 6}})
	m.text = "é" // two bytes
	f := s.Update()
	if sel := f.Fields[0].Change.Sel; sel == nil || *sel != [2]int{2, 2} {
		t.Errorf("sel %v, want 2 2", sel)
	}
}

// A model's value is against the rev the user left the field at, so a
// host that has an edit of theirs since refuses it (SPEC §6.4, ESTALE).
func TestAreaStaleHostEdit(t *testing.T) {
	m := &drafter{bound: true}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftField, Value: "typed", Rev: 7})
	m.text = "from the model"
	f := s.Update()
	if len(f.Fields) != 1 || f.Fields[0].Change.Rev != 7 || *f.Fields[0].Change.Value != "from the model" {
		t.Errorf("fields %+v", f.Fields)
	}
}

// A host that lost the document and shows the surface again gets the field
// back: the model's value, and the caret where the user had it, at rev 0.
func TestAreaLostDocument(t *testing.T) {
	m := &drafter{bound: true}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftField, Value: "my draft", Rev: 4, Sel: &[2]int{8, 8}})
	s.Event(hotty.Event{Kind: hotty.EventCaret, Target: draftField, Sel: &[2]int{3, 8}, Rev: 4})
	s.Update()
	s.Hide()
	f := s.Show()
	if len(f.Fields) != 1 {
		t.Fatalf("fields %+v", f.Fields)
	}
	c := f.Fields[0].Change
	if f.Fields[0].ID != draftField || c.Rev != 0 || c.Sel == nil || *c.Sel != [2]int{3, 8} {
		t.Errorf("the field back: %+v", f.Fields[0])
	}
	if c.Value != nil {
		t.Errorf("the document has the model's value already, %q sent again", *c.Value)
	}
	if !strings.Contains(surfui.HTML(s.Tree()), ">my draft</textarea>") {
		t.Errorf("the document lost the text:\n%s", surfui.HTML(s.Tree()))
	}
}

// A field the model gives no value (no :value) has only what the user
// typed: a host that lost the document gets that.
func TestAreaLostDocumentUnbound(t *testing.T) {
	m := &drafter{}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	s.Event(hotty.Event{Kind: hotty.EventInput, Target: draftField, Value: "my draft", Rev: 4, Sel: &[2]int{2, 2}})
	s.Update()
	s.Hide()
	f := s.Show()
	if len(f.Fields) != 1 || f.Fields[0].Change.Value == nil || *f.Fields[0].Change.Value != "my draft" ||
		*f.Fields[0].Change.Sel != [2]int{2, 2} || f.Fields[0].Change.Rev != 0 {
		t.Errorf("fields %+v", f.Fields)
	}
}
