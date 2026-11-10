package surftea

import (
	"runtime"
	"sync"
	"weak"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytea"

	"neuroplast.io/hotty-surf/surfui"
)

// The stylesheet is a resource of the host's session, which every surface's
// document links (surfui.Document), so a Session needs it once however many
// Hosts it has. sheets remembers, by Session, which have sent it. A Session
// is held weakly: a program's Session ends with the program, and its entry
// goes with it.
var sheets struct {
	mu   sync.Mutex
	sent map[weak.Pointer[hottytea.Session]]bool
}

// sendSheet sends surf's stylesheet through s, once: css is the sheet
// (the first Host to ask decides its hues). A Session that is not a host
// yet (Mode is not Native) sends nothing, and the sheet is still owed.
func sendSheet(s *hottytea.Session, css func() string) {
	if s.Mode != hottytea.Native {
		return
	}
	key := weak.Make(s)
	sheets.mu.Lock()
	if sheets.sent == nil {
		sheets.sent = map[weak.Pointer[hottytea.Session]]bool{}
	}
	if sheets.sent[key] {
		sheets.mu.Unlock()
		return
	}
	sheets.sent[key] = true
	sheets.mu.Unlock()
	runtime.AddCleanup(s, func(k weak.Pointer[hottytea.Session]) {
		sheets.mu.Lock()
		delete(sheets.sent, k)
		sheets.mu.Unlock()
	}, key)
	s.Send(hotty.Res(surfui.StyleID, "text/css", []byte(css())))
}

// Relayout is for a hottytea.RelayoutMsg: when the host lost its documents
// (Lost), it may have lost the stylesheet too, so the next Surface sends it
// again, ahead of the documents Layout then sends again. Call it once per
// message, whatever the number of Hosts.
func Relayout(s *hottytea.Session, msg hottytea.RelayoutMsg) {
	if !msg.Lost {
		return
	}
	sheets.mu.Lock()
	delete(sheets.sent, weak.Make(s))
	sheets.mu.Unlock()
}
