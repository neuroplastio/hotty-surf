package cells_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// signup is a sign-up form: a name, a password, notes of several lines
// and a submit button, each field reporting its change, the form its
// submit. The program's check is the form's: it sets the errors and
// refuses.
type signup struct {
	model
	name, pass, notes string
	nameErr, passErr  string
	dense             bool
}

func newSignup() *signup {
	s := &signup{}
	s.render = func() *surfui.Node {
		form := surfui.Form(
			surfui.Field().Key("name").Name("name").Label("Name").Placeholder("Ada Lovelace").Value(s.name).Error(s.nameErr).
				OnChange(func(e surfrun.Event) { s.name = e.Value }).Autofocus(),
			surfui.Field().Key("pass").Name("pass").Label("Password").Password().Value(s.pass).Error(s.passErr).
				OnChange(func(e surfrun.Event) { s.pass = e.Value }),
			surfui.Field().Key("notes").Name("notes").Label("Notes").Rows(2, 4).Value(s.notes).
				OnChange(func(e surfrun.Event) { s.notes = e.Value }),
			surfui.Button("Sign up").Key("go").Submit(),
		).Key("form").OnSubmit(func() {
			s.nameErr, s.passErr = "", ""
			if s.name == "" {
				s.nameErr = "Name is required"
			}
			if len(s.pass) < 8 {
				s.passErr = "Use at least 8 characters"
			}
		})
		d := surfui.Compact
		if s.dense {
			d = surfui.Dense
		}
		return surfui.Surface(form).Density(d)
	}
	return s
}

func (s *signup) events() []string {
	var out []string
	for _, l := range s.log {
		out = append(out, strings.Fields(l)[0])
	}
	return out
}

// Enter in a field of one line commits it, then submits its form: the
// field's change comes before the form's submit, as a host's do (SPEC
// V6-03).
func TestFormEnterCommitsThenSubmits(t *testing.T) {
	s := newSignup()
	r := show(s)
	r.Draw(40, 20)
	typeText(r, "ada")
	r.Key("enter")
	if got, want := s.log, []string{"change r.kform.kname.0.kname ada", "submit r.kform"}; !slices.Equal(got, want) {
		t.Fatalf("log %q, want %q", got, want)
	}
	// The check refused the password and said so under it.
	screen := r.Draw(40, 20)
	if !strings.Contains(screen.Text(), "Use at least 8 characters") {
		t.Errorf("no message under the password:\n%s", screen.Text())
	}
	if strings.Contains(screen.Text(), "Name is required") {
		t.Errorf("a message under a name that is given:\n%s", screen.Text())
	}
	// Enter again with nothing new: no change to commit, the submit goes.
	r.Key("enter")
	if got := s.events(); !slices.Equal(got, []string{"change", "submit", "submit"}) {
		t.Errorf("events %q", got)
	}
}

// A field of several lines types a line break on Enter and never submits,
// in a form or not; its change comes when it is left.
func TestFormAreaNeverSubmits(t *testing.T) {
	s := newSignup()
	r := show(s)
	r.Draw(40, 20)
	r.Key("tab")
	r.Key("tab") // the notes
	typeText(r, "one\ntwo")
	if s.notes != "" || len(s.log) != 0 {
		t.Fatalf("the program heard %q: notes %q", s.log, s.notes)
	}
	r.Key("enter")
	if len(s.log) != 0 {
		t.Errorf("Enter in the notes: %q", s.log)
	}
	r.Key("tab") // leaves it: it commits, with its line breaks
	if got, want := s.log, []string{"change r.kform.knotes.0.knotes one\ntwo\n"}; !slices.Equal(got, want) {
		t.Errorf("log %q, want %q", got, want)
	}
}

// A submit button submits after its click, and after the field the click
// left has committed.
func TestFormSubmitButton(t *testing.T) {
	s := newSignup()
	r := show(s)
	r.Draw(40, 20)
	typeText(r, "ada")
	for range 3 {
		r.Key("tab")
	}
	r.Key("enter") // the button: a press
	if got := s.events(); !slices.Equal(got, []string{"change", "submit"}) {
		t.Errorf("events %q (%q)", got, s.log)
	}
}

// A button that is no Submit, in a form, submits nothing.
func TestFormPlainButtonDoesNotSubmit(t *testing.T) {
	var log []string
	m := &model{}
	m.render = func() *surfui.Node {
		return surfui.Surface(surfui.Form(
			surfui.Button("Cancel").Key("cancel").OnClick(func() { log = append(log, "cancel") }),
		).Key("form").OnSubmit(func() { log = append(log, "submit") }))
	}
	r := show(m)
	r.Draw(30, 5)
	r.Key("tab")
	r.Key("enter")
	if !slices.Equal(log, []string{"cancel"}) {
		t.Errorf("log %q", log)
	}
}

// What the form gives its program, as a host's does (SPEC §9): the named
// controls' values, in tree order, with what the user typed.
func TestFormSubmitCarriesTheFields(t *testing.T) {
	s := newSignup()
	p := &recorded{Program: surfrun.NewSurface(s, nil).Cells()}
	r := cells.New(p)
	r.Show()
	r.Draw(40, 20)
	typeText(r, "ada")
	r.Key("tab")
	typeText(r, "hunter22")
	r.Key("enter")
	var got []string
	for _, e := range p.events {
		if e.Kind == hotty.EventSubmit {
			for _, f := range e.Fields {
				got = append(got, f.Name+"="+f.Value)
			}
		}
	}
	if want := []string{"name=ada", "pass=hunter22", "notes="}; !slices.Equal(got, want) {
		t.Errorf("fields %q, want %q", got, want)
	}
}

// recorded is a program that keeps the events it is sent.
type recorded struct {
	cells.Program
	events []hotty.Event
}

func (p *recorded) Event(e hotty.Event) (bool, bool) {
	p.events = append(p.events, e)
	return p.Program.Event(e)
}

// The goldens: the form as it opens, with the password masked, its
// refusal with a message and the invalid field, and Dense, which puts each
// label before its field.
func TestFormGoldens(t *testing.T) {
	s := newSignup()
	r := show(s)
	typeText(r, "ada")
	r.Key("tab")
	typeText(r, "hunter")
	golden(t, "form-masked", r.Draw(44, 14))
	r.Key("enter")
	golden(t, "form-refused", r.Draw(44, 14))
	s = newSignup()
	s.dense = true
	r = show(s)
	typeText(r, "ada")
	r.Key("tab")
	typeText(r, "hunter")
	r.Key("enter")
	golden(t, "form-dense", r.Draw(44, 10))
}

// A password's text is never given: copy has nothing to put on the
// clipboard.
func TestFormPasswordIsNotCopied(t *testing.T) {
	s := newSignup()
	r := show(s)
	var copied []string
	r.Copy = func(text string) { copied = append(copied, text) }
	r.Draw(40, 20)
	r.Key("tab")
	typeText(r, "hunter22")
	r.Key("ctrl+a")
	r.Key("ctrl+c")
	if len(copied) != 0 {
		t.Errorf("copied %q", copied)
	}
}
