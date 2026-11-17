package surf_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/surftest"
)

// Clicks hears each click's count, and 1 for Enter; a row selects on a
// click and runs on a double click.
func TestClicks(t *testing.T) {
	var heard []int
	selected, ran := "", ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		for _, r := range []string{"a", "b"} {
			fmt.Fprintf(w, `<button%s>%s</button>`, on("row-"+r, surf.Clicks(func(n int) {
				heard = append(heard, n)
				selected = r
				if n >= 2 {
					ran = r
				}
			})), r)
		}
	})
	s := surf.New("t", v, "")
	r := surftest.Start(t, s)
	if on := attrOf(r.One("#row-a"), "data-on"); on != "click" {
		t.Fatalf("data-on %q", on)
	}
	for _, n := range []int{1, 2} {
		if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventClick, Target: "row-b", Count: n}); err != nil {
			t.Fatal(err)
		}
	}
	if fmt.Sprint(heard) != "[1 2]" || selected != "b" || ran != "b" {
		t.Fatalf("heard %v, selected %q, ran %q after a double click on b", heard, selected, ran)
	}
	heard, ran = nil, ""
	r.Click("#row-a")
	r.Key("enter")
	if fmt.Sprint(heard) != "[1 1]" || ran != "" {
		t.Fatalf("heard %v, ran %q after a click and Enter", heard, ran)
	}
}

// A cells.Screen counts a double click, and Clicks hears it.
func TestClicksInCells(t *testing.T) {
	var heard []int
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<button%s>Row</button>`, on("row", surf.Clicks(func(n int) { heard = append(heard, n) })))
	})
	s := surf.New("t", v, "")
	scr := cells.New(20, 3, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Error(err)
		}
	})
	s.Attach(scr, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scr.Write([]byte(hotty.Doc("t", doc))); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	col := strings.Index(scr.Lines()[0], "Row")
	scr.Click(col, 0)
	scr.Click(col, 0)
	if fmt.Sprint(heard) != "[1 2]" {
		t.Fatalf("heard %v from a double click in cells", heard)
	}
}

// Click and Clicks on one element are a fault.
func TestClickAndClicks(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<button%s>Go</button>`, on("go", surf.Click(func() {}), surf.Clicks(func(int) {})))
	})
	f := &faultT{T: t}
	surftest.Start(f, surf.New("t", v, ""))
	if !strings.Contains(strings.Join(f.got, "\n"), "two click handlers") {
		t.Fatalf("faults %v", f.got)
	}
}

// Refocus takes the keyboard back to the query in the same frame after it
// left for a stop, though the frame before asked for the query too;
// Autofocus doesn't.
func TestRefocus(t *testing.T) {
	for _, again := range []bool{false, true} {
		t.Run(fmt.Sprint("refocus=", again), func(t *testing.T) {
			back := false
			v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
				fmt.Fprintf(w, `<input%s>`, on("line", surf.Input(func(string) {})))
				fmt.Fprintf(w, `<button%s>stop</button>`, on("stop", surf.Focus(func(in bool) {
					if in {
						back = true // the stop says "tab" to the program, then gives the keyboard back
					}
				})))
				switch {
				case back && again:
					back = false
					surf.Refocus(ctx, "line")
				default:
					surf.Autofocus(ctx, "line")
				}
			})
			s := surf.New("t", v, "")
			var out bytes.Buffer
			s.Attach(&out, hotty.Caps{})
			if _, err := s.Document(); err != nil {
				t.Fatal(err)
			}
			if err := s.Update(); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err := s.Event(hotty.Event{Surface: "t", Kind: hotty.EventFocus, Target: "stop"}); err != nil {
				t.Fatal(err)
			}
			sent := strings.Contains(out.String(), hotty.Focus("t", "line"))
			if sent != again || (s.Focused() == "line") != again {
				t.Fatalf("a=focus for line sent %v, focused %q", sent, s.Focused())
			}
		})
	}
}

// faultT keeps the faults surftest reports rather than failing on them.
type faultT struct {
	*testing.T
	got []string
}

func (f *faultT) Error(args ...any) { f.got = append(f.got, fmt.Sprint(args...)) }
