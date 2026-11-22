package cells

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

func TestKitTextareaResizeAndCaret(t *testing.T) {
	c := New(60, 16, nil)
	doc := `<textarea id="msg" data-slot="textarea" class="textarea" rows="3" cols="20" placeholder="Write here"></textarea>`
	if _, err := c.Write([]byte(hotty.Doc("ta", doc))); err != nil {
		t.Fatal(err)
	}
	// Bottom-right corner of 20x3 textarea is (19, 2) with the resize grip.
	grid := c.Cells()
	if got := grid[2][19].Text; got != "◢" {
		t.Fatalf("grip at (19,2) = %q, want ◢", got)
	}
	// Drag grip right by 6 cols and down by 2 rows -> 26x5.
	if !c.Press(19, 2, 0, 0) {
		t.Fatal("Press on grip was not taken")
	}
	c.Move(25, 4, 0)
	c.Release(25, 4, 0)
	grid = c.Cells()
	if got := grid[4][25].Text; got != "◢" {
		t.Fatalf("after resize, grip at (25,4) = %q, want ◢", got)
	}
	// Focus and type: caret starts at inner left pad (col 1) and advances.
	c.Focus(true)
	c.Click(2, 1)
	c.Key("h")
	c.Key("i")
	col, row, ok := c.Cursor()
	if !ok || col != 3 || row != 0 {
		t.Fatalf("caret = (%d,%d,%v), want (3,0,true)", col, row, ok)
	}
	if !strings.Contains(strings.Join(plain(c), "\n"), " hi") {
		t.Fatalf("typed text missing:\n%s", strings.Join(plain(c), "\n"))
	}
}

func TestTextareaMouseSelectionAndCtrlA(t *testing.T) {
	c := New(60, 16, nil)
	doc := `<textarea id="msg" data-slot="textarea" class="textarea" rows="3" cols="30">hello world
second line</textarea>`
	if _, err := c.Write([]byte(hotty.Doc("ta", doc))); err != nil {
		t.Fatal(err)
	}
	c.Focus(true)

	// Single click on 'w' of "world" (padX=1, "hello "=6 -> col 7).
	c.Press(7, 0, 0, 0)
	c.Release(7, 0, 0)
	if col, row, ok := c.Cursor(); !ok || col != 7 || row != 0 {
		t.Fatalf("single click caret = (%d,%d,%v), want (7,0,true)", col, row, ok)
	}
	if got := c.Selection(); got != "" {
		t.Fatalf("single click selected %q, want empty", got)
	}

	// Shift+click at col 12 (end of "world") selects "world".
	c.Press(12, 0, hotty.ModShift, 0)
	c.Release(12, 0, hotty.ModShift)
	if got := c.Selection(); got != "world" {
		t.Fatalf("shift+click selection = %q, want %q", got, "world")
	}

	// Double-click on "hello" (col 2) selects "hello".
	c.clicks = clickRun{}
	c.Press(2, 0, 0, 0)
	c.Release(2, 0, 0)
	c.Press(2, 0, 0, 0)
	c.Release(2, 0, 0)
	if got := c.Selection(); got != "hello" {
		t.Fatalf("double-click selection = %q, want %q", got, "hello")
	}

	// Triple-click on row 0 selects entire first line "hello world".
	c.Press(2, 0, 0, 0)
	c.Release(2, 0, 0)
	if got := c.Selection(); got != "hello world" {
		t.Fatalf("triple-click selection = %q, want %q", got, "hello world")
	}

	// Drag from col 1 to col 6 selects "hello".
	c.clicks = clickRun{}
	c.Press(1, 0, 0, 0)
	c.Move(6, 0, 0)
	c.Release(6, 0, 0)
	if got := c.Selection(); got != "hello" {
		t.Fatalf("drag selection = %q, want %q", got, "hello")
	}

	// Ctrl+A selects all and highlights cells with primary bg, including
	// selected newlines (blank lines show a 1-cell selection indicator).
	if !c.Key("ctrl+a") {
		t.Fatal("ctrl+a was not taken")
	}
	if got := c.Selection(); got != "hello world\nsecond line" {
		t.Fatalf("ctrl+a selection = %q", got)
	}
	grid := c.Cells()
	if cell := grid[0][1]; cell.Bg != "primary" || cell.Fg != "primary-foreground" {
		t.Fatalf("selected cell roles = (%q,%q), want (primary-foreground,primary)", cell.Fg, cell.Bg)
	}
	// Selected newline at end of row 0 (padX=1, "hello world"=11 -> col 12).
	if cell := grid[0][12]; cell.Bg != "primary" {
		t.Fatalf("selected newline indicator at (12,0) bg = %q, want primary", cell.Bg)
	}

	// Ctrl+Backspace deletes the previous word (CUAKeys).
	c.clicks = clickRun{}
	c.Press(12, 0, 0, 0)
	c.Release(12, 0, 0)
	if !c.Key("ctrl+backspace") {
		t.Fatal("ctrl+backspace was not taken")
	}
	if got := strings.Join(plain(c), "\n"); strings.Contains(got, "world") {
		t.Fatalf("ctrl+backspace did not delete word:\n%s", got)
	}
}

func TestTextareaResizeConstraints(t *testing.T) {
	c := New(80, 20, nil)
	doc := `<textarea id="msg" data-slot="textarea" class="textarea" rows="3" cols="24" data-min-cols="16" data-max-cols="30" data-min-rows="2" data-max-rows="5"></textarea>`
	if _, err := c.Write([]byte(hotty.Doc("ta", doc))); err != nil {
		t.Fatal(err)
	}
	// Drag far beyond max (to col 60, row 15) -> clamped to 30x5 (grip at 29, 4).
	if !c.Press(23, 2, 0, 0) {
		t.Fatal("Press on grip was not taken")
	}
	c.Move(60, 15, 0)
	c.Release(60, 15, 0)
	if got := c.Cells()[4][29].Text; got != "◢" {
		t.Fatalf("max clamp grip at (29,4) = %q, want ◢", got)
	}
	// Drag far below min (to col 2, row 0) -> clamped to 16x2 (grip at 15, 1).
	if !c.Press(29, 4, 0, 0) {
		t.Fatal("Press on grip was not taken")
	}
	c.Move(2, 0, 0)
	c.Release(2, 0, 0)
	if got := c.Cells()[1][15].Text; got != "◢" {
		t.Fatalf("min clamp grip at (15,1) = %q, want ◢", got)
	}
}

func TestTextareaTUIScrollbar(t *testing.T) {
	c := New(40, 10, nil)
	doc := `<textarea id="msg" data-slot="textarea" class="textarea" rows="4" cols="20">line1
line2
line3
line4
line5
line6
line7
line8</textarea>`
	if _, err := c.Write([]byte(hotty.Doc("ta", doc))); err != nil {
		t.Fatal(err)
	}
	// Overflowing 8 lines in 4 rows: rightmost col (19) has ▴ at row 0, solid
	// reverse thumb at row 1, ▾ at row 2, and ◢ grip at row 3.
	grid := c.Cells()
	if got := grid[3][19].Text; got != "◢" {
		t.Fatalf("grip at (19,3) = %q, want ◢", got)
	}
	if grid[0][19].Text != "▴" || grid[2][19].Text != "▾" {
		t.Fatalf("expected ▴/▾ arrows at rows 0 and 2; got %q and %q", grid[0][19].Text, grid[2][19].Text)
	}
	if grid[1][19].Attr&Reverse == 0 {
		t.Fatalf("expected reverse-video thumb at (19,1); got %+v", grid[1][19])
	}
	// Clicking ▴ scrolls up toward line1; clicking ▾ scrolls back down.
	if !c.Press(19, 0, 0, 0) {
		t.Fatal("Press on ▴ arrow was not taken")
	}
	c.Release(19, 0, 0)
	if !c.Press(19, 2, 0, 0) {
		t.Fatal("Press on ▾ arrow was not taken")
	}
	c.Release(19, 2, 0)
	// Wheel up scrolls the textarea toward line1, then wheel down scrolls back.
	if !c.Wheel(5, 1, -2) {
		t.Fatal("Wheel up on overflowing textarea was not taken")
	}
	if !c.Wheel(5, 1, 2) {
		t.Fatal("Wheel down on overflowing textarea was not taken")
	}
}
