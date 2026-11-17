package cells_test

import (
	"context"
	"io"
	"strings"
	"testing"
)

// The kit's components that its stylesheet lays out (kit.go).
func TestKit(t *testing.T) {
	for _, c := range []struct {
		name, html, want string
	}{
		{"a field is a column", `<div class="field"><label for="u">Name</label><input id="u" class="input" value="ada"></div>`,
			"Name\nada"},
		{"a horizontal field is a row, its label growing", `<div class="field" data-orientation="horizontal"><input type="checkbox" id="c"><label for="c">Disks</label></div>`,
			"[ ] Disks"},
		{"an input group's addons go by data-align", `<div class="input-group"><input id="q" size="4" value="go"><span data-align="inline-end">K</span><span data-align="inline-start">S</span></div>`,
			"S go"},
		{"a block addon goes above", `<div class="input-group"><input id="a" size="4" value="42"><span data-align="block-start">Amount</span></div>`,
			"Amount\n42"},
		{"a breadcrumb is a row with separators", `<nav class="breadcrumb"><ol><li><a href="#">Home</a></li><li role="presentation" aria-hidden="true"><svg></svg></li><li><span>Here</span></li></ol></nav>`,
			"Home › Here"},
		{"a progress bar from aria-valuenow", `<div role="progressbar" aria-valuenow="25"></div>`,
			"██████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░"},
		{"a skeleton from its size", `<div class="skeleton" style="width: 2rem; height: 2rem"></div>`,
			"░░░░\n░░░░"},
		{"a chart's bars", `<div class="chart"><svg><rect data-value="10"></rect><rect data-value="5"></rect></svg></div>`,
			"████████████████████████████████████████ 10\n████████████████████ 5"},
		{"a chart's lines", `<div class="chart"><svg><polyline data-values="0 7 14"></polyline></svg></div>`,
			"▁ ▅ █"},
		{"a legend's swatches", `<div class="chart-legend"><span class="chart-legend-item"><span class="chart-legend-indicator"></span> Desktop</span><span class="chart-legend-item"><span class="chart-legend-indicator"></span> Mobile</span></div>`,
			"■ Desktop  ■ Mobile"},
		{"an avatar group's avatars apart", `<div class="avatar-group"><span class="avatar">AL</span><span class="avatar">GH</span></div>`,
			"AL GH"},
		{"a row wraps with flex-wrap", `<div class="flex flex-wrap gap-2"><button>one</button><button>two</button><button>three</button><button>four</button><button>fifty-five-six</button></div>`,
			"[ one ] [ two ] [ three ] [ four ]\n[ fifty-five-six ]"},
		{"a Command's empty text", `<div class="command"><div role="menu" data-empty="Nothing."><div role="menuitem" aria-hidden="true">Copy</div></div></div>`,
			"Nothing."},
		{"a Command's items left by its filter", `<div class="command"><div role="menu"><div role="group"><span>Edit</span><div role="menuitem" aria-hidden="true">Copy</div></div><div role="menuitem">Paste</div></div></div>`,
			"Paste"},
		{"a Combobox with none left", `<div class="combobox"><div role="listbox"><div role="option" data-value="a" aria-hidden="true">A</div></div></div>`,
			"No results found"},
		{"a popover on top goes above its trigger", `<div class="popover"><button>Open</button><div data-popover data-side="top" aria-hidden="false">Tip</div></div>`,
			"Tip\n[ Open ]"},
		{"a popover inline-end goes beside its trigger", `<div class="popover"><button>Open</button><div data-popover data-side="inline-end" aria-hidden="false">Tip</div></div>`,
			"[ Open ] Tip"},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := view(func(ctx context.Context, w io.Writer, on on) { io.WriteString(w, c.html) })
			r := start(t, 50, 6, v)
			got := r.text()
			if !strings.Contains(got, c.want) {
				t.Errorf("got\n%s\nwant it to show\n%s", got, c.want)
			}
		})
	}
}
