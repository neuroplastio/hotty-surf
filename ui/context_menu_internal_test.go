package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/internal/rt"
)

// A press opens the menu at its cell less the area's, in the host's cells;
// one with no area (a host that lays nothing out) opens it under the area.
func TestContextMenuPlacement(t *testing.T) {
	for _, c := range []struct {
		area hotty.Area
		want string
	}{
		{hotty.Area{Col: 10, Row: 5, W: 30, H: 3}, "top:calc(var(--hotty-cell-h) * 2);left:calc(var(--hotty-cell-w) * 4)"},
		{hotty.Area{}, ""},
	} {
		var st ContextMenuState
		p := ContextMenuProps{ID: "m", Items: []MenuItem{{Label: "Open"}}}
		v := &contextView{st: &st, p: p, id: "m"}
		v.m = newMenuView(&st.menu, DropdownMenuProps{ID: "m", Items: p.Items})
		v.press(hotty.Cell{Col: 14, Row: 7}, c.area)
		ctx := rt.With(context.Background(), &rt.Frame{On: map[string]*rt.Handlers{}, Memo: map[string]any{}})
		var out bytes.Buffer
		if err := ContextMenu(&st, p).Render(templ.WithChildren(ctx, templ.Raw("row")), &out); err != nil {
			t.Fatal(err)
		}
		html := out.String()
		if !strings.Contains(html, `aria-hidden="false"`) {
			t.Fatalf("area %v: not open: %s", c.area, html)
		}
		placed := strings.Contains(html, "data-at")
		if c.want == "" && placed || c.want != "" && !strings.Contains(html, c.want) {
			t.Fatalf("area %v: want style %q in %s", c.area, c.want, html)
		}
	}
}
