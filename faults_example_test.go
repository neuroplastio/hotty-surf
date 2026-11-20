package surf_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// Stock is a compiled row's props: a pointer, against the rules, so a
// change behind it isn't seen.
type Stock struct {
	Name  string
	Count *int
}

var stockRow = surf.Compile(StockView)

// StockView draws the count it reads behind the pointer.
func StockView(p Stock) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<p>%s: %d</p>`, p.Name, *p.Count)
		return err
	})
}

// With the check on, a kept row that reads what isn't its props is a
// fault.
func ExampleSurface_CheckCompiled() {
	pears := 3
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return stockRow.With("pears", Stock{Name: "pears", Count: &pears}).Render(ctx, w)
	})
	s := surf.New("stock", view, "")
	s.CheckCompiled(true)
	s.OnFault(func(f string) { fmt.Println(f) })
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		return
	}
	pears = 4 // behind the pointer: the props are the same, and the row is kept
	if err := s.Update(); err != nil {
		return
	}
	// Output:
	// surf.Compile: neuroplast.io/hotty-surf_test.StockView, instance key "pears": rendered again with the same props its markup differs: it reads what isn't its props (a ctx value, a pointer's target, a kit component's state)
}

// OnFault hears what the surface rendered around, each fault once until
// the next Document.
func ExampleSurface_OnFault() {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		save := surf.On(ctx, "save", surf.Click(func() {}))
		surf.On(ctx, "save", surf.Click(func() {}))
		_, err := fmt.Fprintf(w, `<button id=%q data-on=%q>Save</button>`, save["id"], save["data-on"])
		return err
	})
	s := surf.New("form", view, "")
	s.OnFault(func(f string) { fmt.Println("fault:", f) })
	s.Attach(io.Discard, hotty.Caps{})
	if _, err := s.Document(); err != nil {
		return
	}
	for range 3 {
		if err := s.Update(); err != nil {
			return
		}
	}
	// Output:
	// fault: surf.On: the id save is registered twice in one render
}

// collect is a test whose errors are kept, not failed.
type collect struct {
	testing.TB
	got *[]string
}

func (c collect) Error(args ...any) { *c.got = append(*c.got, fmt.Sprint(args...)) }

// A rig and an app's fn both hear a fault.
func TestOnFaultBesideARig(t *testing.T) {
	s := surf.New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		surf.On(ctx, "x")
		surf.On(ctx, "x")
		_, err := io.WriteString(w, `<p>x</p>`)
		return err
	}), "")
	var heard, failed []string
	s.OnFault(func(f string) { heard = append(heard, f) })
	surftest.Start(collect{t, &failed}, s)
	if len(heard) != 1 || len(failed) != 1 || heard[0] != failed[0] {
		t.Errorf("fn heard %q, the rig %q: want the one fault each", heard, failed)
	}
}
