package surfrun_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/surfrun"
)

// In the cells a menu row's icon has one cell between it and its label,
// and a row without an icon lines up with the rows that have one; with no
// row having one, no row is indented.
func TestCellsMenuIconGap(t *testing.T) {
	col := func(line, text string) int {
		i := strings.Index(line, text)
		if i < 0 {
			return -1
		}
		return len([]rune(line[:i]))
	}
	for _, d := range []string{"dense", "compact", "roomy"} {
		items := []surfrun.MenuItem{
			{ID: "a", Label: "Mute this", Icon: "bell", Enabled: true},
			{ID: "b", Label: "Delete", Enabled: true},
			{ID: "c", Label: "Rename", Icon: "pencil", Enabled: true},
		}
		lines := strings.Split(surfrun.OpenMenu(items, -1, surfrun.MenuLook{Density: d}).Cells.Text(), "\n")
		mute, del, ren := col(lines[1], "Mute this"), col(lines[2], "Delete"), col(lines[3], "Rename")
		if mute != 4 || del != mute || ren != mute {
			t.Errorf("%s: labels at %d, %d, %d, want all at 4 (border, pad, icon, gap):\n%s", d, mute, del, ren, strings.Join(lines, "\n"))
		}
		if icon := []rune(lines[1])[2]; icon == ' ' || []rune(lines[1])[3] != ' ' {
			t.Errorf("%s: no icon and a gap before the label: %q", d, lines[1])
		}

		bare := surfrun.OpenMenu(items[1:2], -1, surfrun.MenuLook{Density: d}).Cells.Text()
		if col(strings.Split(bare, "\n")[1], "Delete") != 2 {
			t.Errorf("%s: a menu with no icon is indented:\n%s", d, bare)
		}
	}
}
