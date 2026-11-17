package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/input"
	"neuroplast.io/hotty-surf/surftest"
)

// hostRig is surftest's loop on a test host the test reads directly: a
// field's value and rev on the host, which surftest doesn't show. A reply
// that is an error (an a=field with a stale rev) fails the test.
type hostRig struct {
	t  *testing.T
	h  *hottytest.Host
	s  *surf.Surface
	in input.Reader
}

func startHost(t *testing.T, s *surf.Surface) *hostRig {
	t.Helper()
	r := &hostRig{t: t, h: hottytest.New(t, hottytest.Size(80, 24)), s: s}
	s.Attach(r.h, hottytest.DefaultCaps())
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	r.h.Write([]byte(hotty.Doc(s.Name(), doc)))
	r.h.Write([]byte(hotty.Place(s.Name(), hotty.Placement{Cols: 80, Rows: 24, Press: hotty.On, Hover: hotty.On})))
	r.settle()
	return r
}

func (r *hostRig) settle() {
	r.t.Helper()
	for i := 0; i < 100; i++ {
		busy := false
		select {
		case <-r.s.Pending():
			if err := r.s.Update(); err != nil {
				r.t.Fatal(err)
			}
			busy = true
		default:
		}
		for r.h.Buffered() > 0 {
			busy = true
			buf := make([]byte, r.h.Buffered())
			n, _ := r.h.Read(buf)
			err := r.in.Feed(buf[:n], input.Handlers{
				Event: r.s.Event,
				Reply: func(rep hotty.Reply) error {
					if err := hotty.Err(rep); err != nil {
						r.t.Errorf("the host replied: %v", err)
					}
					return nil
				},
				Key: func(k string) error { _, err := r.s.Key(k); return err },
			})
			if err != nil {
				r.t.Fatal(err)
			}
		}
		if !busy {
			return
		}
	}
	r.t.Fatal("the surface and the host are still talking after 100 rounds")
}

func (r *hostRig) click(id string) {
	r.t.Helper()
	r.h.Click(r.s.Name(), id)
	r.settle()
}

func (r *hostRig) key(k string) {
	r.t.Helper()
	r.h.Key(k)
	r.settle()
}

func (r *hostRig) typ(id, text string) {
	r.t.Helper()
	r.click(id)
	for _, c := range text {
		r.key(string(c))
	}
}

func (r *hostRig) value(id string) string {
	r.t.Helper()
	v, ok := r.h.Surface(r.s.Name()).Value(id)
	if !ok {
		r.t.Fatalf("no field %q on the host", id)
	}
	return v
}

// An input whose markup gives no value is the host's: each edit stays, and
// Input hears it.
func TestInputWithoutValueIsTheHosts(t *testing.T) {
	q := ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><input type="text"%s/><p>%s</p></div>`, on("q", surf.Input(func(s string) { q = s })), q)
	})
	r := startHost(t, surf.New("t", v, ""))
	r.typ("q", "abc")
	if q != "abc" || r.value("q") != "abc" {
		t.Fatalf("heard %q, the host has %q", q, r.value("q"))
	}
}

// An input with no handler and no value keeps what the user typed after
// it commits and something else renders.
func TestUncontrolledFieldKeepsItsText(t *testing.T) {
	n := 0
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><input type="text" id="note"/><button%s>n=%d</button></div>`, on("b", surf.Click(func() { n++ })), n)
	})
	r := startHost(t, surf.New("t", v, ""))
	r.typ("note", "abc")
	r.key("enter")
	r.click("b")
	if n != 1 || r.value("note") != "abc" {
		t.Fatalf("n = %d, the host has %q", n, r.value("note"))
	}
}

// A field taken out and put back starts at rev 0 on the host (SPEC §6.4),
// so the program's a=field after it isn't stale.
func TestFieldPutBackStartsAtRevZero(t *testing.T) {
	open, val := true, ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div><button%s>toggle</button><button%s>fill</button>`,
			on("toggle", surf.Click(func() { open = !open })), on("fill", surf.Click(func() { val = "filled" })))
		if open {
			fmt.Fprintf(w, `<input type="text"%s value="%s"/>`, on("name", surf.Input(func(s string) { val = s })), val)
		}
		fmt.Fprint(w, `</div>`)
	})
	r := startHost(t, surf.New("t", v, ""))
	r.typ("name", "abc")
	r.click("toggle")
	val = ""
	r.click("toggle")
	r.click("fill")
	if got := r.value("name"); got != "filled" {
		t.Fatalf("the host has %q", got)
	}
}

// A field the host morphs (an inner of its parent: text beside it changed)
// keeps its rev there, so the surface keeps it too.
func TestMorphedFieldKeepsItsRev(t *testing.T) {
	val := ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div id="box">typed %d <input type="text"%s value="%s"/></div><button%s>fill</button>`,
			len(val), on("name", surf.Input(func(s string) { val = s })), val, on("fill", surf.Click(func() { val = "filled" })))
	})
	r := startHost(t, surf.New("t", v, ""))
	r.typ("name", "abc")
	r.click("fill")
	if got := r.value("name"); got != "filled" {
		t.Fatalf("the host has %q", got)
	}
}

// As above, with the program setting the field by a key while the field
// has the keyboard, so no change resets the rev first.
func TestMorphedFieldKeepsItsRevUnderAKey(t *testing.T) {
	val, n := "", 0
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s>edits %d <input type="text"%s value="%s"/></div>`,
			on("page", surf.Key(func(k string) bool {
				if k == "mod+l" {
					val = "filled"
					return true
				}
				return false
			})), n, on("name", surf.Input(func(s string) { val = s; n++ })), val)
	})
	r := startHost(t, surf.New("t", v, ""))
	r.typ("name", "abc")
	r.key("ctrl+l")
	if got := r.value("name"); got != "filled" {
		t.Fatalf("the host has %q", got)
	}
}

// A native select is a live field: the program sets it through its
// options' selected, which reaches the host as a=field, even while it has
// the keyboard (where a changed default doesn't move it, SPEC §6.2).
func TestSelectIsALiveField(t *testing.T) {
	pick := "b"
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s><select%s>`, on("page", surf.Key(func(k string) bool {
			if k == "mod+r" {
				pick = "a"
				return true
			}
			return false
		})), on("s", surf.Input(func(s string) { pick = s })))
		for _, o := range []string{"a", "b", "c"} {
			sel := ""
			if o == pick {
				sel = " selected"
			}
			fmt.Fprintf(w, `<option value="%s"%s>%s</option>`, o, sel, strings.ToUpper(o))
		}
		fmt.Fprint(w, `</select></div>`)
	})
	r := startHost(t, surf.New("t", v, ""))
	if got := r.value("s"); got != "b" {
		t.Fatalf("the host starts at %q", got)
	}
	r.click("s")
	r.key("arrow_down")
	if got := r.value("s"); got != "c" || pick != "c" {
		t.Fatalf("after ↓ the host has %q, the program %q", got, pick)
	}
	r.key("ctrl+r")
	if got := r.value("s"); got != "a" {
		t.Fatalf("after the program's reset the host has %q", got)
	}
	// By a=field: the options' selected aren't sent as attributes (the test
	// host would apply those to a focused select, which SPEC §6.2 doesn't).
	if doc := r.h.Surface("t").HTML(); !strings.Contains(doc, `value="b" selected`) || strings.Contains(doc, `value="a" selected`) {
		t.Fatalf("the options' selected moved on the host: %s", doc)
	}
}

// templ keeps a space between inline siblings on one line, and writes a
// line break between them as a space: both are the text's.
func TestSpacesBetweenInlineElementsStay(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprint(w, `<div><p><b>Hello</b> <i>world</i></p><p>Press <kbd>Enter</kbd> <span>to save</span></p>`+
			"<section>\n  <p>a</p> <p>b</p>\n</section></div>")
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	h := r.HTML()
	for _, want := range []string{"Hello</b> <i", "Enter</kbd> <span"} {
		if !strings.Contains(h, want) {
			t.Errorf("no %q in %s", want, h)
		}
	}
	if strings.Contains(h, "</p> <p") || strings.Contains(h, "<section id=\"_8\">\n") {
		t.Errorf("whitespace between blocks stayed: %s", h)
	}
}

// With nothing focused, a key goes to the outermost elements with Key
// handlers, through a layout's wrappers.
func TestPageKeysUnderAWrapper(t *testing.T) {
	got := ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div class="app"><main%s><p>hi</p></main></div>`,
			on("page", surf.Key(func(k string) bool { got = k; return k == "mod+k" })))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	if !r.Key("ctrl+k") || got != "mod+k" {
		t.Fatalf("ctrl+k: heard %q", got)
	}
}
