//go:build !js

package surfrun

import "github.com/neuroplastio/hotty-go/hottyterm"

// open is Run's terminal: the process's, /dev/tty. Its size follows the
// window's (onResize): hottyterm.Open leaves SIGWINCH to the program, and
// Loop re-places the surface on each size Sizes delivers.
func open(name string) (*hottyterm.Term, error) {
	t, err := hottyterm.Open(name)
	if err != nil {
		return nil, err
	}
	onResize(t)
	return t, nil
}
