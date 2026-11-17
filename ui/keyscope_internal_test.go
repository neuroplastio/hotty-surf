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
	if err := KeyScope([]string{"tab", "shift+tab", "ctrl+", "enter", "Enter"}, func(string) bool { return false }).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	if len(f.Faults) != 4 {
		t.Fatalf("faults %v, want two reserved, one invalid and one duplicate", f.Faults)
	}
	if out.String() != "child" {
		t.Fatalf("scope added markup: %q", out.String())
	}
}
