package ui

import (
	"bytes"
	"context"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/internal/rt"
)

func TestKeyScopeInvalidNames(t *testing.T) {
	f := &rt.Frame{On: map[string]*rt.Handlers{}}
	ctx := rt.With(context.Background(), f)
	ctx = templ.WithChildren(ctx, templ.Raw("child"))
	var out bytes.Buffer
	if err := KeyScope([]KeyBinding{{Key: "tab"}, {Key: "shift+tab"}, {Key: "ctrl+"}, {Key: "enter"}, {Key: "Enter"}, {Label: "Nothing reaches it"}}, func(KeyBinding) bool { return false }).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	if len(f.Faults) != 5 {
		t.Fatalf("faults %v, want two reserved, one invalid, one duplicate and one with neither key nor name", f.Faults)
	}
	if out.String() != "child" {
		t.Fatalf("scope added markup: %q", out.String())
	}
}
