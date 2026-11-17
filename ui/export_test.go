package ui

import "time"

// SetToastClock sets the toasts' clock for a test, and returns what puts
// it back.
func SetToastClock(now func() time.Time) (restore func()) {
	old := toastNow
	toastNow = now
	return func() { toastNow = old }
}
