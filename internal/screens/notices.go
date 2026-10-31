package screens

import (
	"strconv"
	"time"
)

// Level is a notice's severity. Its name is its badge's tone.
type Level string

// The levels.
const (
	LevelError Level = "error"
	LevelWarn  Level = "warn"
	LevelInfo  Level = "info"
)

// Notice is one notice. It is a row of the list, and its own state is
// Unread, which opening it clears.
type Notice struct {
	ID     string
	Level  Level
	Source string
	Text   string
	Age    time.Duration // how long ago it came
	Unread bool
}

// The filter's values, in the order f steps through them, and the level
// each shows ("" for all).
var (
	noticeFilters = []string{"All", "Errors", "Warnings", "Info"}
	filterLevel   = map[string]Level{"Errors": LevelError, "Warnings": LevelWarn, "Info": LevelInfo}
)

// arrivals are the notices that come on the clock, in turn.
var arrivals = []Notice{
	{Level: LevelInfo, Source: "agents", Text: "pi-1 finished its task"},
	{Level: LevelWarn, Source: "remote", Text: "devbox is slow to answer"},
	{Level: LevelError, Source: "panes", Text: "Pane p-7 exited with status 2"},
}

// Notices is a notices list (F9's fourth screen): the notices newest
// first, a filter by level, a selection, and a status line saying what
// the last act did. notices.surf draws it, a notice-row.surf a row.
type Notices struct {
	Rows   []*Notice
	Filter string
	Sel    string // the selected notice's ID
	Status string
	Came   int // how many have arrived, for the next one's ID
}

// NewNotices is the screen with its eight notices, n1 selected.
func NewNotices() *Notices {
	s := &Notices{Filter: "All"}
	for _, n := range []Notice{
		{"n1", LevelError, "panes", "Pane p-3 exited with status 1", 2 * time.Minute, true},
		{"n2", LevelWarn, "agents", "claude-2 is waiting for you", 5 * time.Minute, true},
		{"n3", LevelInfo, "build", "hotty-blitz 0.5.3 built", 12 * time.Minute, false},
		{"n4", LevelInfo, "config", "Theme switched to dusk", 20 * time.Minute, false},
		{"n5", LevelWarn, "panes", "Workspace main has 14 panes", 31 * time.Minute, true},
		{"n6", LevelError, "remote", "devbox stopped answering", time.Hour, false},
		{"n7", LevelInfo, "agents", "codex-1 finished its task", 2 * time.Hour, false},
		{"n8", LevelInfo, "build", "Update 26.10.29 installed", 3 * time.Hour, false},
	} {
		s.Rows = append(s.Rows, &n)
	}
	s.Sel = s.Rows[0].ID
	return s
}

// Filters are the filter's values.
func (s *Notices) Filters() []string { return noticeFilters }

// Shown are the notices the filter lets through, newest first.
func (s *Notices) Shown() []*Notice {
	lv := filterLevel[s.Filter]
	var out []*Notice
	for _, n := range s.Rows {
		if lv == "" || n.Level == lv {
			out = append(out, n)
		}
	}
	return out
}

// Cur is the selection's index among the shown rows: the selected
// notice's, or the first row's when it isn't shown.
func (s *Notices) Cur() int {
	for i, n := range s.Shown() {
		if n.ID == s.Sel {
			return i
		}
	}
	return 0
}

// Selected is the selected row's notice, nil when none is shown.
func (s *Notices) Selected() *Notice {
	if rows := s.Shown(); len(rows) > 0 {
		return rows[s.Cur()]
	}
	return nil
}

// Unread is how many notices are unread.
func (s *Notices) Unread() int {
	k := 0
	for _, n := range s.Rows {
		if n.Unread {
			k++
		}
	}
	return k
}

// Age says how long ago: now under a minute, then minutes, then hours.
func (s *Notices) Age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	return strconv.Itoa(int(d/time.Hour)) + "h"
}

// Select selects the notice id: a click on its row.
func (s *Notices) Select(id string) { s.Sel = id }

// Move steps the selection by one, stopping at the ends.
func (s *Notices) Move(by int) {
	if rows := s.Shown(); len(rows) > 0 {
		s.Sel = rows[max(0, min(s.Cur()+by, len(rows)-1))].ID
	}
}

// Open reads n, and says so.
func (s *Notices) Open(n *Notice) {
	if n == nil {
		return
	}
	n.Unread = false
	s.Status = "Opened: " + n.Text
}

// Dismiss takes n away. If it was the selection, the selection stays at
// its index, or goes to the last row.
func (s *Notices) Dismiss(n *Notice) {
	if n == nil {
		return
	}
	rows, at := s.Shown(), s.Cur()
	for i, r := range s.Rows {
		if r == n {
			s.Rows = append(s.Rows[:i], s.Rows[i+1:]...)
			break
		}
	}
	s.Status = "Dismissed: " + n.Text
	if rows[at] != n {
		return // another row's Dismiss: the selection stays on its notice
	}
	if rows = s.Shown(); len(rows) > 0 {
		s.Sel = rows[min(at, len(rows)-1)].ID
	} else {
		s.Sel = ""
	}
}

// SetFilter shows only f's level, keeping the selection on its notice if
// that is still shown, else on the first row.
func (s *Notices) SetFilter(f string) {
	s.Filter = f
	rows := s.Shown()
	for _, n := range rows {
		if n.ID == s.Sel {
			return
		}
	}
	s.Sel = ""
	if len(rows) > 0 {
		s.Sel = rows[0].ID
	}
}

// NextFilter moves the filter to its next value, round to All.
func (s *Notices) NextFilter() {
	for i, f := range noticeFilters {
		if f == s.Filter {
			s.SetFilter(noticeFilters[(i+1)%len(noticeFilters)])
			return
		}
	}
	s.SetFilter(noticeFilters[0])
}

// Tick is the clock's, every 3 seconds: every age grows by 3 seconds, and
// the next arrival comes at the top, unread.
func (s *Notices) Tick() {
	for _, n := range s.Rows {
		n.Age += 3 * time.Second
	}
	n := arrivals[s.Came%len(arrivals)]
	s.Came++
	n.ID, n.Age, n.Unread = "new"+strconv.Itoa(s.Came), 0, true
	s.Rows = append([]*Notice{&n}, s.Rows...)
}
