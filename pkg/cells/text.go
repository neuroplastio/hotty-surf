package cells

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
)

// Text is laid out by grapheme cluster (Unicode UAX #29), each the cells
// a terminal gives it (displaywidth, as ultraviolet measures it): a wide
// one takes two. A cluster that takes none, a lone joiner or a control,
// is left out.

// clusters are line's clusters, a tab spaces to the next of every eight
// cells from its start.
func clusters(line string) []Cluster {
	var out []Cluster
	w := 0
	for g := displaywidth.StringGraphemes(line); g.Next(); {
		s := g.Value()
		if s == "\t" {
			n := 8 - w%8
			for range n {
				out = append(out, Cluster{" ", 1})
			}
			w += n
			continue
		}
		cw := g.Width()
		if cw <= 0 {
			continue
		}
		out = append(out, Cluster{s, cw})
		w += cw
	}
	return out
}

// width is the cells cs take.
func width(cs []Cluster) int {
	n := 0
	for _, c := range cs {
		n += c.Width
	}
	return n
}

// textWidth is the cells s takes on one line.
func textWidth(s string) int { return width(clusters(s)) }

// joined is cs as a string.
func joined(cs []Cluster) string {
	var b strings.Builder
	for _, c := range cs {
		b.WriteString(c.Text)
	}
	return b.String()
}

// ellipsis ends a text cut where it doesn't fit.
var ellipsis = Cluster{"…", 1}

// fit is cs in at most w cells, ending in … where it was cut.
func fit(cs []Cluster, w int) []Cluster {
	if w <= 0 {
		return nil
	}
	if width(cs) <= w {
		return cs
	}
	out := make([]Cluster, 0, w)
	n := 0
	for _, c := range cs {
		if n+c.Width > w-1 {
			break
		}
		out = append(out, c)
		n += c.Width
	}
	return append(out, ellipsis)
}

// lines are s's lines as white-space: pre has them, at its newlines.
func lines(s string) [][]Cluster {
	parts := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([][]Cluster, len(parts))
	for i, p := range parts {
		out[i] = clusters(p)
	}
	return out
}

// wrap is s in lines of at most w cells, as white-space: normal and
// overflow-wrap: anywhere have it: its runs of white space one space,
// broken at spaces and inside a word where parts says, a part longer
// than a line broken where the line ends.
func wrap(s string, w int) [][]Cluster {
	w = max(w, 1)
	var out [][]Cluster
	var line []Cluster
	n := 0
	for _, word := range strings.Fields(s) {
		for i, p := range parts(clusters(word)) {
			gap, pw := 0, width(p)
			if i == 0 && n > 0 {
				gap = 1
			}
			if n > 0 && n+gap+pw > w {
				out = append(out, line)
				line, n, gap = nil, 0, 0
			}
			for pw > w {
				k := fits(p, w)
				out = append(out, p[:k])
				p = p[k:]
				pw = width(p)
			}
			if gap > 0 {
				line = append(line, Cluster{" ", 1})
			}
			line = append(line, p...)
			n += gap + pw
		}
	}
	if n > 0 || len(out) == 0 {
		out = append(out, line)
	}
	return out
}

// parts are a word's pieces between the breaks a host's line breaker
// has inside it (UAX #14): after a "/", so that a path breaks between
// its names. A "/" it starts or ends with is none, nor one before
// another "/".
func parts(cs []Cluster) [][]Cluster {
	var out [][]Cluster
	from := 0
	for i := 1; i < len(cs)-1; i++ {
		if cs[i].Text == "/" && cs[i+1].Text != "/" {
			out = append(out, cs[from:i+1])
			from = i + 1
		}
	}
	return append(out, cs[from:])
}

// fits is how many of cs's clusters fit in w cells, at least one: where
// a part longer than a line breaks, as overflow-wrap: anywhere has it.
func fits(cs []Cluster, w int) int {
	n, nw := 0, 0
	for n < len(cs) && nw+cs[n].Width <= w {
		nw += cs[n].Width
		n++
	}
	return max(n, 1)
}
