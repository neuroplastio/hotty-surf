//go:build js

package surfrun

import (
	"fmt"

	"neuroplast.io/hotty-go/hottyterm"
)

// open is Run's terminal, which js/wasm has none of: no /dev/tty, so
// hottyterm has no Open there.
func open(string) (*hottyterm.Term, error) {
	return nil, fmt.Errorf("%w: js/wasm has no /dev/tty: make a Term with hottyterm.New, and run it with surfrun.New", hottyterm.ErrNoTerminal)
}
