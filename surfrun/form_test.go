package surfrun_test

import (
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// signer is a form on a host: Enter in its field of one line is the
// host's (SPEC V6-03), which commits it and then submits the form; the
// field of several lines types a line break; the button submits.
type signer struct {
	log []string
}

func (m *signer) Render() *surfui.Node {
	return surfui.Surface(surfui.Form(
		surfui.Field().Key("name").Name("name").Autofocus().OnChange(func(e surfrun.Event) { m.log = append(m.log, "name "+e.Value) }),
		surfui.Field().Key("pass").Name("pass").Password().OnChange(func(e surfrun.Event) { m.log = append(m.log, "pass "+e.Value) }),
		surfui.Field().Key("notes").Name("notes").Rows(2, 4).OnChange(func(e surfrun.Event) { m.log = append(m.log, "notes "+e.Value) }),
		surfui.Button("Go").Key("go").Submit(),
	).Key("form").OnSubmit(func() { m.log = append(m.log, "submit") }))
}

func (m *signer) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func(surfrun.Event):
		f(ev)
	case func():
		f()
	}
}

func heard(t *testing.T, a *surftest.App, m *signer, want ...string) {
	t.Helper()
	var got []string
	a.Wait("the log", func() bool {
		a.Call(func() { got = slices.Clone(m.log) })
		return len(got) >= len(want)
	})
	a.Call(func() { got = slices.Clone(m.log) })
	if !slices.Equal(got, want) {
		t.Fatalf("log %q, want %q", got, want)
	}
}

func TestFormEnterOnAHost(t *testing.T) {
	m := &signer{}
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the name has the keyboard", func() bool { return a.Surface().Focused() == "r.kform.kname" })
	// The host's own Enter, not the program's: the field gives it no key.
	doc := a.Surface().HTML()
	for _, id := range []string{`id="r.kform.kname"`, `id="r.kform.kpass"`, `id="r.kform.knotes"`} {
		i := strings.Index(doc, id)
		tag := doc[i : i+strings.Index(doc[i:], ">")]
		if strings.Contains(tag, "enter=program") {
			t.Errorf("%s gives Enter to the program: %s", id, tag)
		}
	}
	if err := a.Host.Fill(a.Name, "r.kform.kname", "ada"); err != nil {
		t.Fatal(err)
	}
	a.Host.Key("enter")
	heard(t, a, m, "name ada", "submit")
}

func TestFormButtonOnAHost(t *testing.T) {
	m := &signer{}
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the name has the keyboard", func() bool { return a.Surface().Focused() == "r.kform.kname" })
	if err := a.Host.Fill(a.Name, "r.kform.kpass", "hunter22"); err != nil {
		t.Fatal(err)
	}
	if err := a.Host.Click(a.Name, "r.kform.kgo"); err != nil {
		t.Fatal(err)
	}
	heard(t, a, m, "pass hunter22", "submit")
	if typ, _ := a.Surface().Attr("r.kform.kpass", "type"); typ != "password" {
		t.Errorf("the password's type is %q", typ)
	}
}

// Enter in a field of several lines is a line break on a host, and it
// submits nothing.
func TestFormAreaEnterOnAHost(t *testing.T) {
	m := &signer{}
	a := surftest.Start(t, m, surfrun.Options{})
	a.Wait("the name has the keyboard", func() bool { return a.Surface().Focused() == "r.kform.kname" })
	if err := a.Host.Fill(a.Name, "r.kform.knotes", "one"); err != nil {
		t.Fatal(err)
	}
	a.Host.Key("enter")
	a.Host.Key("tab") // leaves it: a commit, and nothing submitted before
	heard(t, a, m, "notes one\n")
}
