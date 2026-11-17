package devcmd

import (
	"fmt"
	"io"

	"neuroplast.io/hotty-surf/internal/cssclass"
	"neuroplast.io/hotty-surf/ui"
)

// Classes is `hotty-surf classes`: the classes the kit's stylesheet
// (ui.Stylesheet) defines, one a line, sorted. They are the Tailwind
// utilities the kit's own markup uses and Basecoat's, not all of
// Tailwind's; an app defines any other in its own CSS.
func Classes(out io.Writer) int {
	for _, c := range cssclass.List(ui.Stylesheet()) {
		fmt.Fprintln(out, c)
	}
	return 0
}
