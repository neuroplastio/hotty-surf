package input

import (
	"fmt"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestSGRAuxiliaryButtons(t *testing.T) {
	var decoder uv.EventDecoder
	for _, tc := range []struct{ sgr, dom int }{{1, 1}, {2, 2}, {128, 3}, {129, 4}, {130, 6}, {131, 7}} {
		for _, suffix := range []byte{'M', 'm'} {
			report := fmt.Appendf(nil, "\x1b[<%d;2;3%c", tc.sgr, suffix)
			length, event := decoder.Decode(report)
			var button uv.MouseButton
			switch e := event.(type) {
			case uv.MouseClickEvent:
				if suffix != 'M' {
					t.Fatalf("release decoded as press: %v", e)
				}
				button = e.Button
			case uv.MouseReleaseEvent:
				if suffix != 'm' {
					t.Fatalf("press decoded as release: %v", e)
				}
				button = e.Button
			default:
				t.Fatalf("unexpected SGR event %T %v", event, event)
			}
			dom, ok := MouseButton(button)
			if length != len(report) || !ok || dom != tc.dom {
				t.Errorf("SGR %d %c decoded as DOM %d (%v), want %d", tc.sgr, suffix, dom, ok, tc.dom)
			}
		}
	}
}
