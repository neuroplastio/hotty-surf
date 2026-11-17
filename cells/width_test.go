package cells_test

import (
	"strings"
	"testing"

	"github.com/clipperhouse/displaywidth"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/cells"
)

// A negative padding or gap (invalid CSS, which a browser drops) draws as
// none.
func TestNegativeSpacing(t *testing.T) {
	for _, doc := range []string{
		`<div style="padding-top:-16px"><p>x</p></div>`,
		`<div class="flex flex-col" style="row-gap:-32px"><p>a</p><p>b</p></div>`,
		`<div class="py-[-16px] flex flex-col gap-y-[-2rem]"><p>a</p><p>b</p></div>`,
	} {
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("%s: panic: %v", doc, p)
				}
			}()
			c := cells.New(20, 3, nil)
			if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
				t.Fatal(err)
			}
			c.Lines()
		}()
	}
}

// A wide character in a box one cell wide stays in it: the line is as
// wide as the Screen.
func TestWideCharacterInANarrowBox(t *testing.T) {
	c := cells.New(2, 1, nil)
	c.Write([]byte(hotty.Doc("t", `<div class="flex"><span>漢</span><span>b</span></div>`)))
	plain := sgrRE.ReplaceAllString(c.Lines()[0], "")
	if w := displaywidth.String(plain); w != 2 {
		t.Fatalf("the line is %d cells on a Screen 2 wide: %q", w, plain)
	}
	if plain != "…b" {
		t.Fatalf("the wide character isn't shown as …: %q", plain)
	}
}

// An input scrolled to its caret starts at a character: the caret is
// right after the text drawn.
func TestWideCharacterScrolledInAnInput(t *testing.T) {
	c := cells.New(20, 1, nil)
	c.Write([]byte(hotty.Doc("t", `<input id="q" size="4" value="a漢字">`)))
	c.Key("tab")
	col, _, ok := c.Cursor()
	plain := sgrRE.ReplaceAllString(c.Lines()[0], "")
	if w := displaywidth.String(strings.TrimRight(plain, " ")); !ok || col != w {
		t.Fatalf("caret at %d (%v), the text %q ends at %d", col, ok, plain, w)
	}
}
