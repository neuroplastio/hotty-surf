package cells_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfui"
)

func TestRevealFollowsItsItemsKeyboardFocus(t *testing.T) {
	m := &model{render: func() *surfui.Node {
		return surfui.Surface(surfui.Item(surfui.Text("Item"), surfui.Text("Secret").Reveal(), surfui.Button("Action")).Focusable(),
			surfui.Button("Elsewhere"))
	}}
	r := show(m)
	golden(t, "reveal-item-hidden", r.Draw(40, 4))
	if s := r.Draw(40, 4).Text(); strings.Contains(s, "Secret") {
		t.Fatalf("revealed before the item has focus: %q", s)
	}
	r.Key("tab")
	focused := r.Draw(40, 4)
	golden(t, "reveal-item-focused", focused)
	if s := focused.Text(); !strings.Contains(s, "Secret") {
		t.Fatalf("hidden while the item has focus: %q", s)
	}
	if len(focused.Lines) == 0 || focused.Lines[0][0].Bg != cells.Selected {
		t.Fatalf("focused item lacks the quiet selection role: %s", focused.Golden())
	}
	r.Key("tab")
	if s := r.Draw(40, 4).Text(); !strings.Contains(s, "Secret") {
		t.Fatalf("hidden while a child has focus: %q", s)
	}
	r.Key("tab")
	if s := r.Draw(40, 4).Text(); strings.Contains(s, "Secret") {
		t.Fatalf("revealed after focus leaves: %q", s)
	}
}
