package button_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	"neuroplast.io/hotty-surf/ui/button"
)

func renderHTML(t *testing.T, ctx context.Context, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func withText(c templ.Component, text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, templ.Raw(text)), w)
	})
}

func TestModeDispatch(t *testing.T) {
	c := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return button.Render(button.Props{
			Variant: ui.Destructive,
			Size:    ui.Large,
		}).Render(templ.WithChildren(ctx, templ.Raw("Delete")), w)
	})

	hottyHTML := renderHTML(t, context.Background(), c)
	if !strings.Contains(hottyHTML, `data-variant="destructive"`) || !strings.Contains(hottyHTML, `data-size="lg"`) {
		t.Fatalf("Hotty missing attributes: %s", hottyHTML)
	}
	if strings.Contains(hottyHTML, `data-mode=`) {
		t.Fatalf("Hotty should not carry data-mode: %s", hottyHTML)
	}

	tuiHTML := renderHTML(t, ui.WithMode(context.Background(), ui.ModeTUI), c)
	if !strings.Contains(tuiHTML, `data-mode="tui"`) {
		t.Fatalf("TUI missing data-mode: %s", tuiHTML)
	}

	// ModeANSI and ModePlain fall back to TUI until implemented explicitly.
	for _, m := range []ui.Mode{ui.ModeANSI, ui.ModePlain} {
		out := renderHTML(t, ui.WithMode(context.Background(), m), c)
		if !strings.Contains(out, `data-mode="tui"`) {
			t.Fatalf("%s render should fall back to TUI: %s", m, out)
		}
	}
}

func TestHottyClick(t *testing.T) {
	clicks := 0
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		b := button.Hotty(button.Props{
			ID:      "go",
			OnClick: func() { clicks++ },
		})
		return b.Render(templ.WithChildren(ctx, templ.Raw("Go")), w)
	})
	s := surf.New("btn-test", view, "")
	r := surftest.Start(t, s)
	r.Click("#go")
	if clicks != 1 {
		t.Fatalf("clicks = %d, want 1", clicks)
	}
}

func TestTUIVariantsAndSizes(t *testing.T) {
	view := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		doc := templ.Join(
			withText(button.TUI(button.Props{Variant: ui.Primary, Size: ui.XSmall, ID: "xs"}), "xs"),
			withText(button.TUI(button.Props{Variant: ui.Outline, Size: ui.Large, ID: "lg"}), "Big"),
		)
		return doc.Render(ctx, w)
	})
	html := renderHTML(t, ui.WithMode(context.Background(), ui.ModeTUI), view)
	scr := cells.New(40, 10, nil)
	if _, err := scr.Write([]byte(hotty.Doc("btn", html))); err != nil {
		t.Fatal(err)
	}
	var plain strings.Builder
	for _, row := range scr.Cells() {
		for _, c := range row {
			plain.WriteString(c.Text)
		}
		plain.WriteByte('\n')
	}
	got := plain.String()
	if !strings.Contains(got, "╭───────╮") || !strings.Contains(got, "│  Big  │") || !strings.Contains(got, "╰───────╯") {
		t.Fatalf("expected 3-row boxed lg outline button, got:\n%s", got)
	}
}
