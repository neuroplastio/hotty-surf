package cells

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

var sgrRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func plain(c *Screen) []string {
	var out []string
	for _, l := range c.Lines() {
		out = append(out, strings.TrimRight(sgrRE.ReplaceAllString(l, ""), " "))
	}
	return out
}

func tipScreen(t *testing.T, doc string, cols, rows int) *Screen {
	t.Helper()
	c := New(cols, rows, nil)
	if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
		t.Fatal(err)
	}
	return c
}

// The pointer over an element with a tooltip (ui.Tooltip's data-tooltip)
// draws it beside the element, on its side, as a host's :hover does; off
// it, or off the Screen, it goes.
func TestTooltip(t *testing.T) {
	for _, tc := range []struct {
		name, side string
		pad        int // the button's column
		want       []string
	}{
		{"above", "", 10, []string{"", "        Save the file", "          [ Save ]"}},
		{"below", "bottom", 10, []string{"", "", "          [ Save ]", "        Save the file"}},
		{"left", "left", 20, []string{"", "", "      Save the file [ Save ]"}},
		{"left, no room", "left", 10, []string{"", "", "          [ Save ] Save the file"}},
		{"right", "right", 10, []string{"", "", "          [ Save ] Save the file"}},
		{"right, no room", "right", 30, []string{"", "", "                Save the file [ Save ]"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			side := ""
			if tc.side != "" {
				side = ` data-side="` + tc.side + `"`
			}
			c := tipScreen(t, fmt.Sprintf(`<div style="padding-top: 2rem; padding-left: %dch"><span data-tooltip="Save the file"%s><button>Save</button></span></div>`, tc.pad, side), 40, 6)
			before := plain(c)
			c.Move(tc.pad+1, 2, 0)
			got := plain(c)
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("row %d: %q, want %q\n%s", i, got[i], w, strings.Join(got, "\n"))
				}
			}
			c.Move(39, 5, 0)
			if strings.Join(plain(c), "\n") != strings.Join(before, "\n") {
				t.Errorf("the tooltip stayed after the pointer left:\n%s", strings.Join(plain(c), "\n"))
			}
		})
	}
}

// A tooltip above the first row goes below; one wider than the screen is
// cut, inside it; Cells names it in Basecoat's tooltip colours; Lines
// draws it reversed.
func TestTooltipEdges(t *testing.T) {
	c := tipScreen(t, `<span data-tooltip="a hint much longer than the screen is wide">x</span>`, 12, 3)
	c.Move(0, 0, 0)
	got := plain(c)
	if got[1] != " a hint mu…" && got[1] != " a hint muc…" {
		t.Fatalf("row 1 %q:\n%s", got[1], strings.Join(got, "\n"))
	}
	if !strings.Contains(c.Lines()[1], "\x1b[0;7m") {
		t.Errorf("Lines doesn't draw the tooltip reversed: %q", c.Lines()[1])
	}
	cell := c.Cells()[1][2]
	if cell.Fg != "primary-foreground" || cell.Bg != "primary" {
		t.Errorf("the tooltip's cell is %q on %q", cell.Fg, cell.Bg)
	}
}

// Dirty says when the Screen changed by itself, with no Write: a
// tooltip showing or going on a Move, and not a move within the same
// element; Lines and Cells both take what they draw as drawn.
func TestDirty(t *testing.T) {
	c := tipScreen(t, `<p>plain</p><span data-tooltip="a hint">tip me</span>`, 30, 4)
	if !c.Dirty() {
		t.Fatal("not dirty before the first draw")
	}
	c.Lines()
	steps := []struct {
		do    func()
		dirty bool
		what  string
	}{
		{func() { c.Move(1, 0, 0) }, false, "a move over no tooltip"},
		{func() { c.Move(1, 1, 0) }, true, "the tooltip shows"},
		{func() { c.Lines() }, false, "drawn by Lines"},
		{func() {
			lay := c.lay
			c.Move(2, 1, 0)
			if c.lay != lay {
				t.Error("a move within the element laid the Screen out again")
			}
		}, false, "a move within the element"},
		{func() { c.Move(25, 3, 0) }, true, "the tooltip goes"},
		{func() { c.Cells() }, false, "drawn by Cells"},
		{func() { c.Write([]byte(hotty.Doc("t", `<p>new</p>`))) }, true, "a new document"},
		{func() { c.Lines() }, false, "drawn"},
		{func() { c.Resize(30, 4) }, false, "a resize to the same size draws the same"},
		{func() { c.Resize(20, 4) }, true, "a resize"},
	}
	for _, s := range steps {
		s.do()
		if got := c.Dirty(); got != s.dirty {
			t.Errorf("%s: Dirty %v, want %v", s.what, got, s.dirty)
		}
	}
}
