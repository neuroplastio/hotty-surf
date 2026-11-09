package gallery

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"neuroplast.io/hotty-surf/surfui"
)

// The reference: a story of stories.All as F5's gate drew it (make gate,
// scripts/gate), under the story drawn live: hotty-blitz's render,
// Chromium's, and their diff, one under another, with the gate's verdict.
// The gate keeps each engine's render of each story, theme, density and
// scale in the checkout's .shots/gate/work/, and its verdicts in
// .shots/gate/report.md. The gallery sends each render as an image
// resource (Runtime.SetResource), which the canvas shows as
// <img src="cid:…"> (hotty SPEC §7.1). Only a gallery watching a
// checkout (Watch) has one, and only for stories.All's stories: they are
// the gate's.

// refScale is the scale of the renders shown: the gate draws each case
// at 1.6 and 2.
const refScale = "2"

// refs are the gate's three renders of a case, by its file's suffix,
// each with its heading.
var refs = []struct{ kind, head string }{
	{"blitz", "hotty-blitz"},
	{"chromium", "Chromium"},
	{"diff", "Their diff: red counts; orange, a speck that doesn't; blue, matched moved; yellow, within tolerance"},
}

// refID is the resource of the render kind.
func refID(kind string) string { return "gallery-ref-" + kind }

// HasReference says the story has the gate's renders: one of
// stories.All's, in a gallery watching a checkout.
func (g *Gallery) HasReference() bool { return g.cur != nil && g.cur.tree != nil && g.gate != "" }

// FlipReference shows the gate's renders under the story, or hides them.
func (g *Gallery) FlipReference() { g.Reference = !g.Reference && g.HasReference() }

// refCase is the gate's case of the story at the canvas's theme and
// density: <story>-<theme>-<density>-<scale>.
func (g *Gallery) refCase() string {
	s := g.cur.s
	return slug(s.group+"/"+s.name) + "-" + g.Theme + "-" + g.Density + "-" + refScale
}

// reference is the canvas's reference: the gate's verdict and its
// renders, each sent where it changed, or why there are none.
func (g *Gallery) reference() *surfui.Node {
	id := g.refCase()
	ref := surfui.Stack(surfui.Text("The gate's " + id + ": " + g.verdict(id)).Small().Muted().Wrap().Key("head")).Gap(1).Key("ref")
	shown := 0
	for _, r := range refs {
		data, err := os.ReadFile(filepath.Join(g.gate, "work", id+"."+r.kind+".png"))
		if err != nil {
			g.sendRef(r.kind, nil)
			continue
		}
		g.sendRef(r.kind, data)
		ref.Add(surfui.Stack(
			surfui.Text(r.head).Small().Bold().Wrap(),
			surfui.El("img").Attr("src", "cid:"+refID(r.kind)).Attr("alt", r.head+"'s render").Width(g.Width-2),
		).Key(r.kind))
		shown++
	}
	if shown == 0 {
		ref.Add(surfui.Text("No render of the gate's: make gate draws them into .shots/gate/.").Small().Tone(surfui.Warn).Wrap().Key("none"))
	}
	return ref
}

// verdict is the gate's report's line of the case id: pass or fail, and
// the area that differs.
func (g *Gallery) verdict(id string) string {
	report, err := os.ReadFile(filepath.Join(g.gate, "report.md"))
	if err != nil {
		return "no report"
	}
	for _, line := range strings.Split(string(report), "\n") {
		if !strings.HasPrefix(line, "| ["+id+"]") {
			continue
		}
		// | [id](id.png) | rows | moved | share | area | pass | note |
		f := strings.Split(line, "|")
		if len(f) < 8 {
			break
		}
		v := strings.TrimSpace(f[6]) + ", " + strings.TrimSpace(f[5]) + " CSS px² differ"
		if note := strings.TrimSpace(f[7]); note != "" {
			v += " (" + note + ")"
		}
		return v
	}
	return "not in the report"
}

// sendRef sends data as the render kind's resource, where it is not what
// was sent; nil deletes it.
func (g *Gallery) sendRef(kind string, data []byte) {
	old, sent := g.refSent[kind]
	if sent == (data != nil) && bytes.Equal(old, data) {
		return
	}
	if g.refSent == nil {
		g.refSent = map[string][]byte{}
	}
	if data == nil {
		delete(g.refSent, kind)
	} else {
		g.refSent[kind] = data
	}
	if s, ok := g.rt.(sender); ok {
		s.SetResource(refID(kind), "image/png", data)
	}
}

// unsendRefs deletes the renders sent: the reference is hidden.
func (g *Gallery) unsendRefs() {
	for _, r := range refs {
		g.sendRef(r.kind, nil)
	}
}
