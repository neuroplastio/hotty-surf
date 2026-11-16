package surf_test

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

// view is a view written by hand: f writes its HTML, with on spreading
// surf.On's attributes.
func view(f func(ctx context.Context, w io.Writer, on func(id string, hs ...surf.Handler) string)) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		f(ctx, w, func(id string, hs ...surf.Handler) string {
			var b strings.Builder
			for _, kv := range surf.On(ctx, id, hs...).Items() {
				fmt.Fprintf(&b, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
			}
			return b.String()
		})
		return nil
	})
}

func TestClickAndInput(t *testing.T) {
	n, name := 0, ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><button%s>Added %d</button><input type="text"%s value="%s"/><p>%s</p></div>`,
			on("add", surf.Click(func() { n++ })), n, on("name", surf.Input(func(v string) { name = v })), name, name)
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	r.Click("button:text(Added 0)")
	r.Click("button:text(Added 1)")
	if n != 2 {
		t.Fatalf("n = %d", n)
	}
	r.Type("input", "hi there")
	if name != "hi there" || len(r.Find("p:text(hi there)")) != 1 {
		t.Fatalf("name = %q, %s", name, r.HTML())
	}
}

func TestKeysGoToTheFocusedElementFirst(t *testing.T) {
	var got []string
	keys := func(who, take string) surf.Handler {
		return surf.Key(func(k string) bool { got = append(got, who+":"+k); return k == take })
	}
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s><section%s><button%s>a</button></section><button%s>b</button></div>`,
			on("page", keys("page", "p")), on("menu", keys("menu", "m")), on("a", surf.Click(func() {}), keys("a", "q")), on("b", surf.Click(func() {})))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	// Nothing focused: the page's outermost element hears it.
	r.Key("x")
	if strings.Join(got, " ") != "page:x" {
		t.Fatalf("with nothing focused: %v", got)
	}
	r.Click("button:text(a)")
	got = nil
	r.Key("x")
	if strings.Join(got, " ") != "a:x menu:x page:x" {
		t.Fatalf("focused a: %v", got)
	}
	got = nil
	if !r.Key("m") {
		t.Fatal("menu's key not taken")
	}
	if strings.Join(got, " ") != "a:m menu:m" {
		t.Fatalf("taken by menu: %v", got)
	}
	r.Click("button:text(b)")
	got = nil
	r.Key("x")
	if strings.Join(got, " ") != "page:x" {
		t.Fatalf("focused b: %v", got)
	}
}

func TestModKeys(t *testing.T) {
	var got string
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s>x</div>`, on("page", surf.Key(func(k string) bool { got = k; return true })))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	if !r.Key("ctrl+s") || got != "mod+s" {
		t.Fatalf("ctrl+s reached the handler as %q", got)
	}
}

func TestFocusOutsideHover(t *testing.T) {
	var log []string
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><div%s><button%s>in</button></div><button%s>out</button><p>text</p></div>`,
			on("menu", surf.Focus(func(in bool) { log = append(log, fmt.Sprint("focus ", in)) }),
				surf.Outside(func() { log = append(log, "outside") }),
				surf.Hover(func(over bool) { log = append(log, fmt.Sprint("hover ", over)) })),
			on("in", surf.Click(func() {})), on("out", surf.Click(func() {})))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	r.Click("button:text(in)")
	if strings.Join(log, ",") != "focus true" {
		t.Fatalf("click in: %v", log)
	}
	log = nil
	r.Click("button:text(out)")
	if strings.Join(log, ",") != "outside,focus false" {
		t.Fatalf("click out: %v", log)
	}
	log = nil
	r.Click("p:text(text)") // takes no focus: a press, then a blur
	if strings.Join(log, ",") != "outside,outside" {
		t.Fatalf("click on text: %v", log)
	}
}

func TestHiddenInputIsNoField(t *testing.T) {
	v := "a"
	view := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><input type="hidden" name="h" value="%s"/><button%s>go</button></div>`, v, on("go", surf.Click(func() { v = "b" })))
	})
	r := surftest.Start(t, surf.New("t", view, ""))
	r.Click("button")
	if len(r.Find("input[value=b]")) != 1 {
		t.Fatalf("the hidden input's value is not an attribute: %s", r.HTML())
	}
}

func TestDoAndAutofocus(t *testing.T) {
	items := []string{}
	var s *surf.Surface
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		if surf.Of(ctx) != s {
			t.Error("Of is not the surface")
		}
		surf.Autofocus(ctx, "q")
		fmt.Fprintf(w, `<div><input type="text"%s/><ul>`, on("q"))
		for _, it := range items {
			fmt.Fprintf(w, "<li>%s</li>", it)
		}
		io.WriteString(w, "</ul></div>")
	})
	s = surf.New("t", v, "")
	r := surftest.Start(t, s)
	s.Do(func() { items = append(items, "one") })
	r.Update()
	if len(r.Find("li:text(one)")) != 1 {
		t.Fatal(r.HTML())
	}
	// The field has the keyboard: typing reaches it.
	r.Key("z")
	if len(r.Find("input")) != 1 {
		t.Fatal("no input")
	}
}

func TestDuplicateIDFails(t *testing.T) {
	ft := &fakeT{TB: t}
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><b%s>a</b><b%s>b</b></div>`, on("x"), on("x"))
	})
	surftest.Start(ft, surf.New("t", v, ""))
	if len(ft.errs) != 1 || !strings.Contains(ft.errs[0], "twice") {
		t.Fatalf("errors: %v", ft.errs)
	}
}

type fakeT struct {
	testing.TB
	errs []string
}

func (f *fakeT) Error(args ...any) { f.errs = append(f.errs, fmt.Sprint(args...)) }
func (f *fakeT) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}

func TestWhitespaceInPreStays(t *testing.T) {
	word := "b"
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, "<div>\n  <pre><span>a</span>\n  <span>%s</span>\n</pre>\n  <button%s>go</button>\n</div>",
			word, on("go", surf.Click(func() { word = "c" })))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	if !strings.Contains(r.HTML(), "a</span>\n  <span") {
		t.Fatalf("the pre's line break is gone: %q", r.HTML())
	}
	r.Click("button")
	if h := r.HTML(); !strings.Contains(h, "a</span>\n  <span") || !strings.Contains(h, ">c</span>\n</pre>") {
		t.Fatalf("the pre's whitespace after a change: %q", h)
	}
	if strings.Contains(r.HTML(), "</pre>\n") {
		t.Fatalf("whitespace beside elements outside a pre stays: %q", r.HTML())
	}
}
