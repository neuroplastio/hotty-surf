//go:build surfdev && !js

package run

import (
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/devhook"
)

func TestDriverCompletionSignal(t *testing.T) {
	s := surf.New("test", templ.Raw("<p>test</p>"), "")
	stop := driverStarted(s)
	done := devhook.DriverDone(s)
	if done == nil {
		t.Fatal("driver was not registered")
	}
	select {
	case <-done:
		t.Fatal("running driver is marked exited")
	default:
	}
	stop()
	select {
	case <-done:
	default:
		t.Fatal("driver exit was not signalled")
	}
}
