package screens

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
)

// Settings is a terminal workspace's settings page (S1, S7), on
// demodata's made-up settings, as the look draws it (look/settings): the sections in a sidebar on the
// chrome, each with its icon; the section's title and the search; and its
// settings under their groups' captions, in one stack that scrolls, each
// with its control. Its data and its saves are demodata's: a change saves
// at once, a field's when the user commits it (Enter, or leaving it), and
// the page reads the settings again after it.
// settings.surf draws it, a setting-group.surf a group, a setting-row.surf
// a setting.
type Settings struct {
	Secs   []demodata.Section
	Sec    int // the section shown, but during a search
	Query  string
	Status string            // after the heading: what the last save did
	Drafts map[string]string // what is typed in a field and not saved, by name
	At     int               // the cells' setting with the keyboard, -1 for the search

	src *demodata.Source
}

// NewSettings is the page on src's settings, General shown.
func NewSettings(src *demodata.Source) *Settings {
	s := &Settings{src: src, Drafts: map[string]string{}, At: -1}
	s.reload()
	return s
}

// reload reads the settings again: the values in force and where they
// come from.
func (s *Settings) reload() {
	secs, err := s.src.Settings()
	if err != nil {
		s.Status = "✗ " + err.Error()
		return
	}
	s.Secs = secs
	s.Sec = min(s.Sec, max(len(secs)-1, 0))
}

// Searching is whether the page lists a search's matches.
func (s *Settings) Searching() bool { return strings.TrimSpace(s.Query) != "" }

// Searched is the search's words.
func (s *Settings) Searched() string { return strings.TrimSpace(s.Query) }

// Groups are what the box lists: the section's groups, each a card with
// its title over it and its note under it; during a search, one group of
// its matches, with neither.
func (s *Settings) Groups() []demodata.Group {
	if s.Searching() {
		return []demodata.Group{{Settings: demodata.Search(s.Secs, s.Query)}}
	}
	if s.Sec >= len(s.Secs) {
		return nil
	}
	return slices.Clone(s.Secs[s.Sec].Groups)
}

// Shown are the settings the box lists, in its order: the cells' At
// counts them.
func (s *Settings) Shown() []demodata.Setting {
	var out []demodata.Setting
	for _, g := range s.Groups() {
		out = append(out, g.Settings...)
	}
	return out
}

// Title is the box's heading: the section's title, or Search.
func (s *Settings) Title() string {
	if !s.Searching() && s.Sec < len(s.Secs) {
		return s.Secs[s.Sec].Title
	}
	return "Search"
}

// Box is the box's key, by what it shows, so that another section starts
// at its top.
func (s *Settings) Box() string {
	if !s.Searching() && s.Sec < len(s.Secs) {
		return "box-" + s.Secs[s.Sec].ID
	}
	return "box-search"
}

// Control is st's control: switch, cycle (a few choices), select (more),
// number or text (a field that saves what is committed: Enter, or leaving
// it), or caption (the value in words).
func (s *Settings) Control(st demodata.Setting) string { return st.Control.String() }

// Draft is what st's field shows: what is typed, else the value.
func (s *Settings) Draft(st demodata.Setting) string {
	if v, ok := s.Drafts[st.Name]; ok {
		return v
	}
	return st.Value
}

// Search is the search field's input: the page lists its matches.
func (s *Settings) Search(q string) { s.Query, s.At = q, -1 }

// Pick shows section i, which empties the search.
func (s *Settings) Pick(i int) {
	if i < 0 || i >= len(s.Secs) {
		return
	}
	s.Sec, s.Query, s.At = i, "", -1
}

// Move moves the sidebar's selection d sections, at the ends no further;
// during a search, the first move selects General.
func (s *Settings) Move(d int) {
	if s.Searching() {
		s.Pick(0)
		return
	}
	s.Pick(max(min(s.Sec+d, len(s.Secs)-1), 0))
}

// Flip is a switch's click: it saves the other value.
func (s *Settings) Flip(st demodata.Setting) { s.set(st.Name, strconv.FormatBool(!st.On())) }

// Choose is a cycle's step or a select's pick: it saves the value.
func (s *Settings) Choose(st demodata.Setting, v string) { s.set(st.Name, v) }

// Edit is typing in st's field, which saves nothing until it is
// committed.
func (s *Settings) Edit(st demodata.Setting, v string) { s.Drafts[st.Name] = v }

// Commit is st's field committed, on Enter or on leaving it: it saves
// the value.
func (s *Settings) Commit(st demodata.Setting, v string) { s.set(st.Name, v) }

// Save saves what st's field holds, in the cells: on Enter, and on
// leaving it.
func (s *Settings) Save(st demodata.Setting) {
	if v, ok := s.Drafts[st.Name]; ok {
		s.set(st.Name, v)
	}
}

// set saves key's value to the overlay: the status says it, or why it
// was refused, and the field goes back to the value in force.
func (s *Settings) set(key, value string) {
	delete(s.Drafts, key)
	if err := s.src.Set(key, value); err != nil {
		s.Status = "✗ " + err.Error()
	} else {
		s.Status = key + " saved to the overlay"
	}
	s.reload()
}

// Reset takes st out of the overlay: the config file's value or the
// default is in force again.
func (s *Settings) Reset(st demodata.Setting) {
	delete(s.Drafts, st.Name)
	if err := s.src.Reset(st.Name); err != nil {
		s.Status = "✗ " + err.Error()
	} else {
		s.Status = st.Name + " taken out of the overlay"
	}
	s.reload()
}
