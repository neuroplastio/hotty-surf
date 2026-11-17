package surftest_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// attrs writes a's attributes, as templ's spread does.
func attrs(w io.Writer, a templ.Attributes) error {
	for k, v := range a {
		if _, err := fmt.Fprintf(w, " %s=%q", k, fmt.Sprint(v)); err != nil {
			return err
		}
	}
	return nil
}

// Enter in a form's text field, and its submit button, reach the form's
// surf.Submit, after the field's Input; a Key handler on the field never
// hears Enter.
func TestSubmit(t *testing.T) {
	var draft string
	var items []string
	keys := 0
	add := func() {
		items = append(items, draft)
		draft = ""
	}
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<form")
		attrs(w, surf.On(ctx, "f", surf.Submit(add)))
		io.WriteString(w, `><input name="title"`)
		attrs(w, surf.On(ctx, "title", surf.Input(func(v string) { draft = v }),
			surf.Key(func(string) bool { keys++; return false })))
		fmt.Fprintf(w, ` value=%q><button type="submit"`, draft)
		attrs(w, surf.On(ctx, "add"))
		io.WriteString(w, ">Add</button></form><ul>")
		for _, it := range items {
			fmt.Fprintf(w, "<li>%s</li>", templ.EscapeString(it))
		}
		_, err := io.WriteString(w, "</ul>")
		return err
	})
	r := surftest.Start(t, surf.New("todo", page, ""))
	r.Type("input[name=title]", "Buy milk")
	if !r.Key("enter") {
		t.Error("the field didn't take Enter")
	}
	if len(r.Find("li:text(Buy milk)")) != 1 {
		t.Fatalf("Enter didn't submit: %s", r.HTML())
	}
	if keys != 0 {
		t.Errorf("the field's Key handler heard %d keys", keys)
	}
	r.Type("input[name=title]", "Walk")
	r.Click("button#add")
	if len(r.Find("li:text(Walk)")) != 1 {
		t.Fatalf("the submit button didn't submit: %s", r.HTML())
	}
}

// Value and Checked read a field as the user sees it, the host's state.
func TestValueAndChecked(t *testing.T) {
	title, done := "", false
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<input name="title"`)
		attrs(w, surf.On(ctx, "title", surf.Input(func(v string) { title = v })))
		fmt.Fprintf(w, ` value=%q><input type="checkbox"`, title)
		attrs(w, surf.On(ctx, "done", surf.Check(func(c bool) { done = c })))
		if done {
			io.WriteString(w, " checked")
		}
		_, err := io.WriteString(w, ">")
		return err
	})
	r := surftest.Start(t, surf.New("todo", page, ""))
	if r.Value("input[name=title]") != "" || r.Checked("input#done") {
		t.Fatal("not as it started")
	}
	r.Type("input[name=title]", "Buy milk")
	if got := r.Value("input[name=title]"); got != "Buy milk" {
		t.Errorf("Value: %q", got)
	}
	r.Click("input#done")
	if !r.Checked("input#done") || !done {
		t.Errorf("Checked %v, the app's %v", r.Checked("input#done"), done)
	}
}

type faults struct {
	*testing.T
	got []string
}

func (f *faults) Error(args ...any) { f.got = append(f.got, fmt.Sprint(args...)) }

// A Submit on an element that isn't a form can never run: the test fails,
// naming the element.
func TestSubmitOnNoForm(t *testing.T) {
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<div")
		attrs(w, surf.On(ctx, "box", surf.Submit(func() {})))
		_, err := io.WriteString(w, `><input name="title"></div>`)
		return err
	})
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", page, ""))
	if len(f.got) != 1 || !strings.Contains(f.got[0], "surf.Submit: the element box is a <div>, not a form") {
		t.Fatalf("faults: %q", f.got)
	}
}

// A button with no text and no label is a fault (the cells fallback draws
// it as "[ · ]"); an aria-label, or text, is enough.
func TestUnlabelledButton(t *testing.T) {
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<button id="del"><svg><path d="M1 1"></path></svg></button>`+
			`<button id="ok" aria-label="Delete"><svg></svg></button><button id="go">Go</button>`)
		return err
	})
	f := &faults{T: t}
	surftest.Start(f, surf.New("t", page, ""))
	if len(f.got) != 1 || !strings.Contains(f.got[0], "a button with no text and no label (#del)") {
		t.Fatalf("faults: %q", f.got)
	}
}

// A fault that every render repeats fails the test once.
func TestFaultOnce(t *testing.T) {
	n := 0
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<form`)
		attrs(w, surf.On(ctx, "f", surf.Click(func() { n++ })))
		_, err := fmt.Fprintf(w, `><p>%d</p><button id="b"></button></form>`, n)
		return err
	})
	f := &faults{T: t}
	r := surftest.Start(f, surf.New("t", page, ""))
	r.Click("#f")
	r.Click("#f")
	if len(f.got) != 1 {
		t.Fatalf("faults: %q", f.got)
	}
}
