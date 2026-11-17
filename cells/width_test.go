package cells_test

import (
	"testing"

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
