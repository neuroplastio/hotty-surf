package ui_test

import (
	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
	"testing"
)

func TestTooltipMarkup(t *testing.T) {
	view := with(ui.Tooltip(ui.TooltipProps{Text: "More", Side: "bottom", Align: "end", Arrow: true}), templ.Raw(`<button>Details</button>`))
	r := surftest.Start(t, surf.New("tooltip", view, ui.Stylesheet()))
	r.One(`[data-tooltip="More"][data-side="bottom"][data-align="end"][data-arrow]`)
}
