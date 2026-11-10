package cells

import (
	"strconv"

	"neuroplast.io/hotty-surf/surfui"
)

// A Slider in the cells (A8): a track of ─ with a ● thumb, and the value
// after it, a cell off, as wide as the widest of its ends so that the
// track keeps its length as the value changes. The track is the part a
// drag measures (SPEC §9.1's step, from the middle of a cell, as in
// stepIn), so the value is outside the element's steps (trackWidth).

// sliderTrack is the track's natural length in cells.
const sliderTrack = 20

// trackWidth is the cells of the track in a Slider w cells wide: what
// the value and the cell before it leave.
func trackWidth(n *surfui.Node, w int) int {
	return max(w-1-surfui.SliderWidth(n), 1)
}

// thumbAt is the column of the thumb for the value at step i of count in a
// track tw cells long: the middle of the cells whose pointer step is i, so
// that a press on the thumb reads the value it shows; where there are more
// steps than cells, the cell nearest.
func thumbAt(i, count, tw int) int {
	if count <= 0 || tw <= 1 {
		return 0
	}
	first, last := -1, -1
	for c := range tw {
		if step(count, c, tw) == i {
			if first < 0 {
				first = c
			}
			last = c
		}
	}
	if first >= 0 {
		return (first + last) / 2
	}
	return min(max((i*(tw-1)+count/2)/count, 0), tw-1)
}

// slider draws a Slider: the track, filled in the accent up to its thumb,
// the rest muted; the thumb in the ink, the accent while it has the
// keyboard; then the value.
func (p *painter) slider(b *box, x, y, w int, in ink, clip rect) {
	lo, hi, step, value, ok := surfui.SliderOf(b.n)
	if !ok {
		return
	}
	tw := trackWidth(b.n, w)
	count := int(surfui.SliderCount(lo, hi, step))
	at := thumbAt(int((value-lo)/max(step, 1)), count, tw)
	fill, rest, thumb := Accent, Muted, Fg
	if p.focus == b.n {
		thumb = Accent
	}
	if b.f&fDisabled != 0 {
		fill, rest, thumb = Faint, Faint, Faint
	}
	for c := range tw {
		switch {
		case c == at:
			p.g.put(x+c, y, "●", 1, style{fg: thumb}, clip)
		case c < at:
			p.g.put(x+c, y, "─", 1, style{fg: fill}, clip)
		default:
			p.g.put(x+c, y, "─", 1, style{fg: rest}, clip)
		}
	}
	if vw := w - tw - 1; vw > 0 {
		v := strconv.FormatInt(value, 10)
		p.write(x+tw+1, y, vw, clusters(v), in.style(), clip)
	}
}
