package screens

import (
	"cmp"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// DrawCells is the roster in cells (F7), from the same state: each entry's
// lines, the one with the keyboard in reverse video.
func (l *Agents) DrawCells(cols, rows int) []string {
	var out []string
	if !l.Loaded {
		return out
	}
	if len(l.Agents) == 0 {
		for _, w := range wrap("No agents yet. An agent appears when its harness reports from its pane.", cols-1) {
			out = append(out, cellsLine(cols, false, []seg{{" " + w, surfui.Muted, false}}))
		}
	}
	from := 0
	for i, a := range l.Agents {
		if i == l.At {
			from = len(out)
		}
		dot, tone := l.Dot(a), surfui.Tone(l.Tone(a))
		icon := cmp.Or(a.Icon, "　")
		name := seg{a.Name, "", true}
		if a.Offline {
			name.tone = surfui.Muted
		}
		head := []seg{{" " + dot + " ", tone, false}, {icon + " ", "", false}, name}
		for _, b := range l.item.Badges(a) {
			head = append(head, seg{" " + b, surfui.Info, false})
		}
		if a.Host != "" {
			head = append(head, seg{" " + a.Host, surfui.Info, false})
		}
		var hand []seg
		if a.Attention {
			hand = append(hand, seg{demodata.Hand + " ", "", false})
		}
		sel := i == l.At
		out = append(out, cellsLine(cols, sel, head, hand...))
		for _, line := range l.item.Lines(a) {
			for _, w := range wrap(line, cols-6) {
				out = append(out, cellsLine(cols, sel, []seg{{"      " + w, surfui.Muted, false}}))
			}
		}
	}
	var end []string
	if l.Refused != "" {
		for _, w := range wrap(l.Refused, cols-1) {
			end = append(end, cellsLine(cols, false, []seg{{" " + w, surfui.Error, false}}))
		}
	}
	room := max(rows-len(end), 1)
	from = max(min(from, len(out)-room), 0)
	out = append(out[from:min(from+room, len(out))], end...)
	return out[:min(len(out), rows)]
}

// CellsKey moves the keyboard between the entries in the cells, as Tab
// and the arrows do; Enter and Space are the list's.
func (l *Agents) CellsKey(key string) bool {
	switch key {
	case "arrow_down", "tab", "j":
		l.At = min(l.At+1, max(len(l.Agents)-1, 0))
	case "arrow_up", "shift+tab", "k":
		l.At = max(l.At-1, 0)
	default:
		return false
	}
	return true
}
