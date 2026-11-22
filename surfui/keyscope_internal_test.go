package surfui

import (
	"bytes"
	"context"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

func TestKeyScopeInvalidNames(t *testing.T) {
	no := func() bool { return false }
	f := &rt.Frame{}
	ctx := rt.With(context.Background(), f)
	ctx = templ.WithChildren(ctx, templ.Raw("child"))
	var out bytes.Buffer
	if err := KeyScope([]KeyBinding{{Key: "tab", Run: no}, {Key: "shift+tab", Run: no}, {Key: "ctrl+", Run: no}, {Key: "enter", Run: no}, {Key: "Enter", Run: no}, {Label: "Nothing reaches it", Run: no}, {Key: "x"}}).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	if len(f.Faults) != 6 {
		t.Fatalf("faults %v, want two reserved, one invalid, one duplicate, one with neither key nor name and one with no Run", f.Faults)
	}
	if out.String() != "child" {
		t.Fatalf("scope added markup: %q", out.String())
	}
}
