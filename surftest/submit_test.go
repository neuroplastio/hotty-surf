package surftest_test

import (
	"context"
	"fmt"
	"io"
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
		io.WriteString(w, "></button></form><ul>")
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
