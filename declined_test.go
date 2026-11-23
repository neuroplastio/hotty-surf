package surf_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// A declared key every Key handler declines is the focused element's own
// use, as if nothing had declared it: a list that declares Space and the
// arrows, and declines them, leaves Space to a button inside it and the
// arrows to a field.
func TestDeclinedKeyIsTheElements(t *testing.T) {
	clicks, value := 0, ""
	var heard []string
	decline := func(k string) bool { heard = append(heard, k); return false }
	view := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s><button%s>b</button><input%s value=%q></div>`,
			on("list", surf.Key(decline, "space", "arrow_left")),
			on("b", surf.Click(func() { clicks++ })),
			on("f", surf.Input(func(v string) { value = v })), value)
	})
	r := surftest.Start(t, surf.New("t", view, ""))
	if got := attrOf(r.One("#b"), "data-keys"); got != "" {
		t.Fatalf("the button's data-keys %q", got)
	}
	if got := attrOf(r.One("#list"), "data-keys-inner"); got != "arrow_left=program space=program" {
		t.Fatalf("the list's data-keys %q", got)
	}
	r.Click("#b")
	clicks = 0
	if !r.Key("space") || clicks != 1 {
		t.Errorf("Space on the button: %d clicks", clicks)
	}
	r.Type("#f", "ab")
	r.Key("arrow_left")
	r.Type("#f", "x")
	if value != "axb" {
		t.Errorf("the field after arrow_left and x: %q (heard %v)", value, heard)
	}
	if len(heard) != 1 || heard[0] != "arrow_left" {
		t.Errorf("the list's handler heard %v", heard)
	}
}

// A printable key declared around a field is the field's: it types it
// because the field ignores an inherited binding for a printable key.
func TestDeclaredCharacterTypes(t *testing.T) {
	value, took := "", 0
	view := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div%s tabindex="0"><input%s value=%q></div>`,
			on("list", surf.Key(func(string) bool { took++; return true }, "j", "shift+g")),
			on("f", surf.Input(func(v string) { value = v })), value)
	})
	r := surftest.Start(t, surf.New("t", view, ""))
	if got := attrOf(r.One("#f"), "data-keys"); got != "" {
		t.Fatalf("the field's data-keys %q", got)
	}
	r.Type("#f", "jG")
	if value != "jG" || took != 0 {
		t.Errorf("typed %q, the list took %d", value, took)
	}
}
