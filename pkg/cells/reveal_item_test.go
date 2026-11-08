package cells_test

import (
	"strings"
	"testing"

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
	golden(t, "reveal-item-focused", r.Draw(40, 4))
	if s := r.Draw(40, 4).Text(); !strings.Contains(s, "Secret") {
		t.Fatalf("hidden while the item has focus: %q", s)
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
