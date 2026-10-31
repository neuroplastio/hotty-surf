package screens

import (
	"strings"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The palette in cells (F7): its box in box-drawing lines where the host
// has it, from the same state as the host's. A cells line is a cell
// tall, as a row is at scale 1.

// DrawCells is the palette's box: the query, the status, the rows or the
// panel, and the footer, each a line.
func (p *Palette) DrawCells(cols, rows int) []string {
	inner, top, fit := place(cols, rows, 100)
	inner = min(inner, max(cols-4, 1))
	left := max((cols-inner-4)/2, 0)
	line := func(selected bool, l []seg, r ...seg) string {
		return boxed(left, cellsLine(inner, selected, l, r...))
	}
	rule := boxed(left, "\x1b[90m"+strings.Repeat("─", inner)+"\x1b[39m")
	out := make([]string, top-1, rows)
	out = append(out, boxEdge(left, inner, true))
	out = append(out, line(false, []seg{{"❯ ", surfui.Accent, true}, {p.Query + "█", "", false}}), rule)
	if s := p.Status(); s != "" {
		st := seg{s, surfui.Muted, false}
		if p.Failed {
			st = seg{"✗ " + s, surfui.Error, false}
		}
		out = append(out, line(false, []seg{st}), rule)
	}
	r := p.Current()
	switch {
	case r != nil && p.Panel:
		out = append(out, line(false, []seg{{"Actions: " + r.Title, surfui.Muted, true}}))
		for i, a := range r.Actions {
			key := a.Key
			if i == 0 {
				key = "enter"
			}
			out = append(out, line(i == p.Act, []seg{{a.Title, "", false}}, seg{capsText(p.style.Keys, key), surfui.Muted, false}))
		}
	case r == nil:
		out = append(out, line(false, []seg{{p.Empty, surfui.Muted, false}}))
	default:
		from := max(min(p.Top, p.Sel), p.Sel-fit+1, 0)
		for i := from; i < min(from+fit, len(p.Rows)); i++ {
			out = append(out, line(i == p.Sel, p.cellsRow(p.Rows[i]), seg{capsText(p.style.Keys, p.Rows[i].Keys), surfui.Muted, false}))
		}
	}
	note, acts := p.hints()
	foot := []seg{{note, surfui.Muted, false}}
	if !p.Panel && len(acts) > 0 {
		foot = append(foot, seg{"  Actions ", "", false}, seg{capsText(p.style.Keys, "ctrl+k"), surfui.Muted, false})
	}
	var right []seg
	for i := len(acts) - 1; i >= 0; i-- {
		right = append(right, seg{"  " + acts[i].Title + " ", "", false}, seg{capsText(p.style.Keys, acts[i].Key), surfui.Muted, false})
	}
	out = append(out, rule, line(false, foot, right...), boxEdge(left, inner, false))
	return out[:min(len(out), rows)]
}

// cellsRow is a row's text: its icon, its group muted, its title with the
// query's matches in bold, and its detail muted.
func (p *Palette) cellsRow(r demodata.Row) []seg {
	var out []seg
	if r.Icon != "" {
		out = append(out, seg{r.Icon + " ", "", false})
	}
	if r.Group != "" {
		out = append(out, marks(r.Group, r.GroupMatch, surfui.Muted)...)
		out = append(out, seg{": ", surfui.Muted, false})
	}
	out = append(out, marks(r.Title, r.Match, "")...)
	detail := r.Detail
	if r.Kind == demodata.KindValue {
		detail = r.Note
	}
	if detail != "" {
		out = append(out, seg{"  " + detail, surfui.Muted, false})
	}
	return out
}

// marks is s in tone, the runes in runs bold in the accent: a cells
// line's matches.
func marks(s string, runs [][2]int, tone surfui.Tone) []seg {
	rs := []rune(s)
	var out []seg
	at := 0
	for _, m := range runs {
		out = append(out, seg{string(rs[at:m[0]]), tone, false}, seg{string(rs[m[0]:m[1]]), surfui.Accent, true})
		at = m[1]
	}
	return append(out, seg{string(rs[at:]), tone, false})
}

// CellsKey types into the query: a letter, a space, Backspace. The
// palette's other keys are its root's bindings, which the cells take as
// the host's do.
func (p *Palette) CellsKey(key string) bool {
	if key == "backspace" {
		p.SetQuery(backspaced(p.Query))
		return true
	}
	if t, ok := typed(key); ok {
		p.SetQuery(p.Query + t)
		return true
	}
	return false
}
