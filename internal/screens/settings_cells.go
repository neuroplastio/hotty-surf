package screens

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/internal/screens/demodata"
	"neuroplast.io/hotty-surf/surfui"
)

// The settings page in cells (F7), from the same state: the title and the
// search, the sections on a line, then the settings under their groups'
// captions, a line each with its control's value at the right, the one
// with the keyboard in reverse video. ↑/↓ move between the search and the settings, Tab goes to the
// next section; Space or Enter changes a value, a field takes typing and
// saves on Enter, and Delete resets.

// cellsHint is the cells page's last line.
const cellsHint = "↑↓ move · tab section · space change · enter save · del reset"

// DrawCells is the page in cols by rows cells.
func (s *Settings) DrawCells(cols, rows int) []string {
	query := s.Query
	if s.At < 0 {
		query += "█"
	}
	out := []string{cellsLine(cols, false, []seg{{"Settings", "", true}}, seg{"Search: ", surfui.Muted, false}, seg{query, "", false})}
	var tabs []seg
	for i, sec := range s.Secs {
		t, pad := seg{" " + sec.Title, "", false}, " "
		if !s.Searching() && i == s.Sec {
			t, pad = seg{"[" + sec.Title + "]", surfui.Accent, true}, ""
		}
		tabs = append(tabs, t)
		if sec.Status != "" {
			tabs = append(tabs, seg{" " + sec.Status, surfui.Muted, false})
		}
		tabs = append(tabs, seg{pad + " ", "", false})
	}
	title := "Search"
	if !s.Searching() && s.Sec < len(s.Secs) {
		title = s.Secs[s.Sec].Title
	}
	out = append(out, cellsLine(cols, false, tabs),
		cellsLine(cols, false, []seg{{title, "", true}}, seg{s.Status, surfui.Muted, false}),
		"\x1b[90m"+strings.Repeat("─", cols)+"\x1b[39m")

	var body []string
	from, i := 0, 0
	for _, g := range s.Groups() {
		if g.Title != "" {
			if len(body) > 0 {
				body = append(body, "")
			}
			body = append(body, cellsLine(cols, false, []seg{{g.Title, "", true}}))
		}
		for _, st := range g.Settings {
			if i == s.At {
				from = len(body)
			}
			body = append(body, cellsLine(cols, i == s.At, []seg{{st.Label, "", true}}, s.cellsControl(st, i == s.At)))
			for _, l := range wrap(st.Description, cols-2) {
				body = append(body, cellsLine(cols, false, []seg{{"  " + l, surfui.Muted, false}}))
			}
			if st.Source != "" {
				src := []seg{{"  " + st.Source, surfui.Muted, false}}
				if st.Reset {
					src = append(src, seg{"  Reset: del", "", false})
				}
				body = append(body, cellsLine(cols, false, src))
			}
			i++
		}
		if g.Note != "" && len(g.Settings) > 0 {
			for _, l := range wrap(g.Note, cols) {
				body = append(body, cellsLine(cols, false, []seg{{l, surfui.Muted, false}}))
			}
		}
	}
	if s.Searching() && i == 0 {
		body = append(body, cellsLine(cols, false, []seg{{`No setting has "` + strings.TrimSpace(s.Query) + `" in it.`, "", false}}))
	}
	room := max(rows-len(out)-1, 1)
	from = max(min(from, len(body)-room), 0)
	out = append(out, body[from:min(from+room, len(body))]...)
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, cellsLine(cols, false, []seg{{cellsHint, surfui.Muted, false}}))
	return out[:min(len(out), rows)]
}

// cellsControl is st's control as a cells line shows it.
func (s *Settings) cellsControl(st demodata.Setting, focused bool) seg {
	switch st.Control {
	case demodata.Switch:
		if st.On() {
			return seg{"[on]", surfui.Accent, true}
		}
		return seg{"[off]", surfui.Muted, false}
	case demodata.Cycle:
		return seg{"‹ " + st.Value + " ›", "", false}
	case demodata.Select:
		return seg{st.Value + " ⌄", "", false}
	case demodata.Number, demodata.Text:
		v := s.Draft(st)
		if focused {
			v += "█"
		}
		if v == "" {
			return seg{"[" + st.Placeholder + "]", surfui.Muted, false}
		}
		return seg{"[" + v + "]", "", false}
	}
	return seg{st.Value, surfui.Muted, false}
}

// CellsKey is a key the page's bindings didn't take, in the cells.
func (s *Settings) CellsKey(key string) bool {
	shown := s.Shown()
	switch key {
	case "arrow_down":
		s.leave(shown)
		s.At = min(s.At+1, len(shown)-1)
		return true
	case "arrow_up":
		s.leave(shown)
		s.At = max(s.At-1, -1)
		return true
	case "tab", "shift+tab":
		s.leave(shown)
		d := 1
		if key == "shift+tab" {
			d = len(s.Secs) - 1
		}
		if s.Searching() {
			s.Pick(0)
		} else if len(s.Secs) > 0 {
			s.Pick((s.Sec + d) % len(s.Secs))
		}
		return true
	}
	if s.At < 0 || s.At >= len(shown) {
		return s.typeInto(key, s.Query, s.Search)
	}
	st := shown[s.At]
	switch {
	case key == "delete":
		if st.Reset {
			s.Reset(st)
		}
		return true
	case st.Control == demodata.Switch && (key == "space" || key == "enter"):
		s.set(st.Name, strconv.FormatBool(!st.On()))
	case (st.Control == demodata.Cycle || st.Control == demodata.Select) && len(st.Choices) > 0:
		d := 0
		switch key {
		case "space", "enter", "arrow_right":
			d = 1
		case "arrow_left":
			d = len(st.Choices) - 1
		default:
			return false
		}
		at := max(slices.Index(st.Choices, st.Value), 0)
		s.set(st.Name, st.Choices[(at+d)%len(st.Choices)])
	case st.Control == demodata.Number || st.Control == demodata.Text:
		if key == "enter" {
			s.Save(st)
			return true
		}
		return s.typeInto(key, s.Draft(st), func(v string) { s.Drafts[st.Name] = v })
	default:
		return false
	}
	return true
}

// leave saves what the field with the keyboard holds, as the keyboard
// leaves it, as a host's field commits when focus leaves it.
func (s *Settings) leave(shown []demodata.Setting) {
	if s.At >= 0 && s.At < len(shown) {
		if st := shown[s.At]; st.Control == demodata.Number || st.Control == demodata.Text {
			s.Save(st)
		}
	}
}

// typeInto types key into text, a field's, and gives set what it now
// holds: a character, or Backspace's deletion.
func (s *Settings) typeInto(key, text string, set func(string)) bool {
	if key == "backspace" {
		set(backspaced(text))
		return true
	}
	if t, ok := typed(key); ok {
		set(text + t)
		return true
	}
	return false
}
