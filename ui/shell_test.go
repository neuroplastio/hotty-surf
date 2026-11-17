package ui_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/ui"
)

func TestIconButtonNamesAndOwnedAttributes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attrs     templ.Attributes
		wantFault string
	}{
		{"label", templ.Attributes{"aria-label": "Hosts"}, ""},
		{"labelledby", templ.Attributes{"aria-labelledby": "label"}, ""},
		{"missing", nil, "accessible name"},
		{"owned", templ.Attributes{"aria-label": "Hosts", "type": "submit"}, "Attrs has type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := surf.New("icons", ui.IconButton(ui.IconButtonProps{Shape: ui.ShapePublic, Attrs: tc.attrs}), "")
			if _, err := s.Document(); err != nil {
				t.Fatal(err)
			}
			faults := strings.Join(rt.Peek.Faults(s), "\n")
			if tc.wantFault == "" && faults != "" || tc.wantFault != "" && !strings.Contains(faults, tc.wantFault) {
				t.Fatalf("faults: %q, want %q", faults, tc.wantFault)
			}
		})
	}
}

func TestIconButtonClickAndDisabled(t *testing.T) {
	clicks := 0
	disabled := false
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		attrs := surf.On(ctx, "icon", surf.Click(func() { clicks++ }))
		attrs["aria-label"] = "Hosts"
		if disabled {
			attrs["disabled"] = true
		}
		return ui.IconButton(ui.IconButtonProps{Shape: ui.ShapePublic, Attrs: attrs}).Render(ctx, w)
	})
	r := start(t, view)
	r.Click("#icon")
	if clicks != 1 {
		t.Fatalf("clicks = %d", clicks)
	}
	disabled = true
	r.Update()
	r.Click("#icon")
	if clicks != 1 {
		t.Fatalf("disabled button activated: %d", clicks)
	}
}

func TestSymbolsAreCallerOwnedAndAcceptCustomPath(t *testing.T) {
	names := ui.Symbols()
	if len(names) != 7 {
		t.Fatalf("symbols = %d, want 7", len(names))
	}
	original := names[0]
	names[0].Name = "changed"
	if ui.Symbols()[0] != original {
		t.Fatal("Symbols shared its slice")
	}
	var b strings.Builder
	if err := ui.Symbol(ui.Shape("M0 0L1 1"), "size-4").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `d="M0 0L1 1"`) || !strings.Contains(b.String(), `aria-hidden="true"`) {
		t.Fatalf("custom symbol: %s", b.String())
	}
}
