//go:build unix

package surfrun

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/neuroplastio/hotty-go/hottyterm"
)

// onResize tells t its size (Term.Resized) at each SIGWINCH, until t
// closes. Sizes is made first, so a resize before Loop takes the channel
// is kept for it: a window tiled just after it opened is the usual one.
func onResize(t *hottyterm.Term) {
	t.Sizes()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	done := make(chan struct{})
	t.OnClose(func() error {
		signal.Stop(sig)
		close(done)
		return nil
	})
	go func() {
		for {
			select {
			case <-sig:
				t.Resized(t.Size())
			case <-done:
				return
			}
		}
	}()
}
