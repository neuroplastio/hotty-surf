package surf_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// A palette, with made-up data: a query field under ReadlineKeys, the
// platform's keys around it, and 40 rows the query filters, one frame a
// keystroke. What a frame costs with the editing keymaps bound.
func BenchmarkPaletteKeymaps(b *testing.B) {
	for _, caps := range []hotty.Caps{{}, {Mod: "super"}} {
		b.Run("mod="+caps.Mod, func(b *testing.B) {
			query := ""
			v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
				fmt.Fprintf(w, `<div%s><input type="text"%s/><ul>`,
					on("app", surf.Bind(surf.CUAKeys())),
					on("q", surf.Bind(surf.ReadlineKeys()), surf.Input(func(s string) { query = s })))
				for i := range 40 {
					if len(query)%2 == 0 || i%2 == 0 {
						fmt.Fprintf(w, `<li><button%s>row %d</button></li>`, on(fmt.Sprintf("r%d", i), surf.Click(func() {})), i)
					}
				}
				fmt.Fprint(w, `</ul></div>`)
			})
			s := surf.New("p", v, "")
			s.Attach(io.Discard, caps)
			if _, err := s.Document(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				s.Do(func() { query = string(rune('a'+i%26)) + query[:min(len(query), 8)] })
				if err := s.Update(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
