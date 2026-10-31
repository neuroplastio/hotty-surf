package surfui_test

import (
	"testing"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// TakesFocus is what a host gives the keyboard to (SPEC §10.1).
func TestTakesFocus(t *testing.T) {
	for _, c := range []struct {
		what string
		n    *surfui.Node
		want bool
	}{
		{"a field", surfui.Field(), true},
		{"a button", surfui.Button("Go"), true},
		{"a disabled button", surfui.Button("Go").Attr("disabled", ""), false},
		{"a Select", surfui.Select("a", "a", "b"), true},
		{"a Cycle", surfui.Cycle("a", "a", "b"), true},
		{"a text", surfui.Text("hi"), false},
		{"a box", surfui.Box(), false},
		{"a box with a tabindex", surfui.Box().Attr("tabindex", "0"), true},
		{"a box out of the tab order", surfui.Box().Attr("tabindex", "-1"), false},
		{"a hidden input", surfui.El("input").Attr("type", "hidden"), false},
		{"a link", surfui.El("a").Attr("href", "#x"), true},
		{"a hyperlink", surfui.El("a").Attr("href", "https://x").Attr("target", "_blank"), false},
		{"an anchor with no href", surfui.El("a"), false},
		{"an editing host", surfui.El("div").Attr("contenteditable", ""), true},
	} {
		if got := surfui.TakesFocus(c.n); got != c.want {
			t.Errorf("%s: TakesFocus %v, want %v", c.what, got, c.want)
		}
	}
}
