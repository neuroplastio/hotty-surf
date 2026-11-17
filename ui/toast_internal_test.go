package ui

import (
	"testing"
	"time"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
)

// A toast paused past its deadline, its timer late (Stop still succeeds),
// closes when the pointer leaves: it has no Dismiss to stay for.
func TestToastPausedPastItsDeadline(t *testing.T) {
	var st ToasterState
	st.Show(Toast{Title: "Hi", Duration: time.Hour})
	tt := st.toasts[0]
	tt.s = surf.New("t", templ.NopComponent, "")
	st.arm(tt)
	tt.since = time.Now().Add(-time.Hour - time.Millisecond)
	st.hover(true)
	st.hover(false)
	if tt.timer == nil && !tt.hidden {
		t.Errorf("after pause and resume past its deadline: no timer, not closed, left %v: it stays for ever", tt.left)
	}
}
