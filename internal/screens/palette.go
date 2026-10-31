package screens

import (
	"cmp"
	"strings"
	"unicode/utf8"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Palette is a command palette (S5, S6, S12), a program in a pane of its
// own: a box near the pane's top, a query, the rows listed for it, and a
// footer with the selected row's actions. The query keeps the keyboard;
// the palette's keys, bound on its root, reach the program from it. Its
// data and its acts are demodata's. palette.surf draws it, a
// palette-row.surf a row, a foot-hint.surf one of the footer's actions.
type Palette struct {
	Query  string
	Rows   []demodata.Row
	Empty  string // what the list says with no rows
	Sel    int
	Top    int    // the first row shown
	Note   string // the status: "Copied pane.new"
	Failed bool   // the status is a refusal
	Panel  bool   // the actions panel is open
	Act    int    // the panel's selection
	Closed bool

	Cols, Lines int // the surface's size, in cells

	src     *demodata.Source
	data    *demodata.Palette // nil where the source refused
	style   demodata.Style
	rt      surfrun.Runner // to close
	stage   stage          // the command whose value the line is typing
	density surfui.Density // what it is drawn at, where its host picks it
}

// stage is the command a line stages a value for, and the values typed
// before the one being typed: `pane.split ` is pane.split's first.
type stage struct {
	on    bool
	cmd   demodata.Command
	prev  []string
	typed string
}

// NewPalette is the palette on src, its query empty.
func NewPalette(src *demodata.Source) *Palette {
	p := &Palette{src: src, Cols: 64, Lines: 44}
	p.style, _ = src.Style()
	data, err := src.Palette()
	if err != nil {
		p.Note, p.Failed = err.Error(), true
	}
	p.data = data
	p.list()
	return p
}

// Attach keeps the runner, which a run ends with when the palette closes.
func (p *Palette) Attach(r surfrun.Runner) { p.rt = r }

// Resize is the pane's size: the box's place, and as many rows as fit.
func (p *Palette) Resize(cols, rows int) {
	p.Cols, p.Lines = cols, rows
	p.keepInView()
}

// The box's place and size, for a surface cols by lines cells and a
// scale in percent: the palette inside is inner cells wide before its
// scale, at most 60, with 6 cells of the pane kept beside it; the box's
// top edge is at row top, a sixth of the way down; the list shows at
// most fit rows. The room for the palette is in drawn lines, two fifths
// of the pane and 5 more, rounded down at the scale; the chrome's five
// lines come off that room over the scale, the status's two kept
// whether or not it says anything, so that the box doesn't jump as a
// status comes and goes.
func place(cols, lines, scale int) (inner, top, fit int) {
	inner = max(min(60, (cols-6)*100/scale), 10)
	top = max(lines/6, 1)
	below := lines - 1 - top - 2
	zoom := float64(scale) / 100
	room := min(int(float64(min(below, lines*2/5+5))*zoom), below)
	fit = min(max(int(float64(room)/zoom)-5, 1), 30)
	return inner, top, fit
}

// Scale is the palette's scale, in percent.
func (p *Palette) Scale() int {
	if p.style.PaletteScale < 1 {
		return 100
	}
	return p.style.PaletteScale
}

// Inner is the palette's width before its scale, and Above the rows above
// the box.
func (p *Palette) Inner() int { inner, _, _ := place(p.Cols, p.Lines, p.Scale()); return inner }
func (p *Palette) Above() int { _, top, _ := place(p.Cols, p.Lines, p.Scale()); return top }

// fit is how many rows the box shows on a host.
func (p *Palette) fit() int {
	_, _, fit := place(p.Cols, p.Lines, p.Scale())
	return fit
}

// Status is the status row's text: the latest act's, else the stage's.
func (p *Palette) Status() string {
	switch {
	case p.Note != "":
		return p.Note
	case p.stage.on:
		if a := p.stage.arg(); a != nil && a.Name != "" {
			return p.stage.cmd.ID + " · " + a.Name
		}
		return p.stage.cmd.ID
	}
	return ""
}

// arg is the argument the stage's value is for, nil past the last.
func (s stage) arg() *demodata.Arg {
	if len(s.prev) < len(s.cmd.Args) {
		return &s.cmd.Args[len(s.prev)]
	}
	return nil
}

// staged is the stage line opens, where it is a command that takes a
// value, a space, and values: `config.set palette.position ce`.
func staged(data *demodata.Palette, line string) stage {
	id, rest, ok := strings.Cut(line, " ")
	if !ok || data == nil {
		return stage{}
	}
	c, ok := data.Command(id)
	if !ok || !c.TakesValue() {
		return stage{}
	}
	words := strings.Split(rest, " ")
	s := stage{on: true, cmd: c, typed: words[len(words)-1]}
	for _, w := range words[:len(words)-1] {
		if w != "" {
			s.prev = append(s.prev, w)
		}
	}
	return s
}

// list lists the rows for the line: the data's for the query, or the
// values of the command it stages.
func (p *Palette) list() {
	p.stage = staged(p.data, p.Query)
	switch {
	case p.data == nil:
		p.Rows, p.Empty = nil, ""
	case p.stage.on:
		p.Rows, p.Empty = p.data.Values(p.stage.cmd.ID, p.stage.prev, p.stage.typed)
	default:
		p.Rows, p.Empty = p.data.Rows(p.Query), "No command matches."
		if strings.HasPrefix(p.Query, "@") {
			p.Empty = "Nothing matches."
		}
	}
	p.keepInView()
}

// SetQuery is the user's typing, the query's input: row 0 is selected
// again, but where characters were deleted, which keeps the selection.
func (p *Palette) SetQuery(q string) {
	shorter := utf8.RuneCountInString(q) < utf8.RuneCountInString(p.Query)
	p.Query, p.Note, p.Failed, p.Panel = q, "", false, false
	if !shorter {
		p.Sel, p.Top = 0, 0
	}
	p.list()
}

// setLine is the program's rewrite of the line, a completion or a stage,
// which the runtime sends as a=field (S5).
func (p *Palette) setLine(line string) {
	p.Query, p.Note, p.Failed, p.Panel = line, "", false, false
	p.Sel, p.Top = 0, 0
	p.list()
}

// keepInView keeps the selection among the rows, and shown.
func (p *Palette) keepInView() {
	p.Sel = max(min(p.Sel, len(p.Rows)-1), 0)
	fit := p.fit()
	p.Top = max(min(p.Top, p.Sel, len(p.Rows)-fit), p.Sel-fit+1, 0)
}

// Current is the selected row, nil for none.
func (p *Palette) Current() *demodata.Row {
	if p.Sel < len(p.Rows) {
		return &p.Rows[p.Sel]
	}
	return nil
}

// ShowPanel is whether the actions panel is in the rows' place.
func (p *Palette) ShowPanel() bool { return p.Panel && p.Current() != nil }

// Move moves the selection d rows, round the ends: the panel's while it
// is open.
func (p *Palette) Move(d int) {
	if r := p.Current(); r != nil && p.Panel {
		n := len(r.Actions)
		p.Act = ((p.Act+d)%n + n) % n
		return
	}
	if n := len(p.Rows); n > 0 {
		p.Sel = ((p.Sel+d)%n + n) % n
		p.keepInView()
	}
}

// Page moves the selection by the rows shown, with no wrap.
func (p *Palette) Page(d int) {
	p.Panel = false
	p.Sel = max(min(p.Sel+d*p.fit(), len(p.Rows)-1), 0)
	p.keepInView()
}

// Enter is Enter: the panel's action, or the selected row's first; in a
// stage with no value to pick, the line as typed where its value may be
// left out.
func (p *Palette) Enter() {
	r := p.Current()
	switch {
	case r != nil && p.Panel:
		p.do(*r, r.Actions[p.Act])
	case r != nil && len(r.Actions) > 0:
		p.do(*r, r.Actions[0])
	case p.stage.on && p.stage.arg() != nil && !p.stage.arg().Required:
		p.run(p.stage.cmd.ID, p.stage.prev, true)
	}
}

// DoKey runs the selected row's action that key takes, if it has one: Run
// and stay, Copy command, Remove from used.
func (p *Palette) DoKey(key string) {
	p.Panel = false
	r := p.Current()
	if r == nil {
		return
	}
	for _, a := range r.Actions {
		if a.Key == key {
			p.do(*r, a)
			return
		}
	}
}

// TogglePanel opens or closes the actions panel (ctrl+k).
func (p *Palette) TogglePanel() {
	if p.Current() != nil {
		p.Panel, p.Act = !p.Panel, 0
	}
}

// Escape closes the panel, or the palette.
func (p *Palette) Escape() {
	if p.Panel {
		p.Panel = false
		return
	}
	p.close()
}

// Complete closes the panel and writes the selected row's completion on
// the line: a command's id, and a space where it takes a value.
func (p *Palette) Complete() {
	p.Panel = false
	if r := p.Current(); r != nil {
		p.setLine(cmp.Or(r.Complete, r.Title))
	}
}

// Click is a click on row i: it selects it, and a click on the selected
// row, or a double click's second, runs it.
func (p *Palette) Click(i int, e surfrun.Event) {
	if i == p.Sel || e.Count >= 2 {
		p.Sel = i
		p.Enter()
		return
	}
	p.Sel = i
	p.keepInView()
}

// ClickAct is a click on the panel's action i, as a row's.
func (p *Palette) ClickAct(i int, e surfrun.Event) {
	if i == p.Act || e.Count >= 2 {
		p.Act = i
		p.Enter()
		return
	}
	p.Act = i
}

// target is what r runs: the command and its values.
func (p *Palette) target(r demodata.Row) (id string, values []string) {
	if r.Kind == demodata.KindValue {
		return p.stage.cmd.ID, append(append([]string(nil), p.stage.prev...), r.Value)
	}
	return r.Value, nil
}

// do does r's action a through demodata: a run closes the palette, but
// for Run and stay; Run… and Set… stage the value on the line; a refusal
// is the status, and the palette stays.
func (p *Palette) do(r demodata.Row, a demodata.Action) {
	p.Panel, p.Note, p.Failed = false, "", false
	id, values := p.target(r)
	var err error
	switch a.ID {
	case "run":
		if r.TakesValue {
			p.setLine(r.Complete)
			return
		}
		err = p.run(id, values, true)
	case "stay":
		err = p.run(id, values, false)
	case "copy":
		line := demodata.Line(id, values...)
		if err = p.src.Copy(line); err == nil {
			p.Note = "Copied " + line
		}
	case "remove":
		if err = p.src.Forget(id); err == nil {
			p.list()
		}
	case "go":
		if err = p.src.GoTo(r.Pane); err == nil {
			p.close()
		}
	case "go-stay":
		err = p.src.GoTo(r.Pane)
	case "close":
		err = p.src.Close(r.Pane)
		p.list()
	case "set":
		p.setLine(r.Stage)
	}
	if err != nil {
		p.Note, p.Failed = err.Error(), true
		p.keepInView()
	}
}

// run runs id with values, then closes the palette, or stays with the
// line, the list now with id among the used.
func (p *Palette) run(id string, values []string, close bool) error {
	if err := p.src.Run(id, strings.Join(values, " ")); err != nil {
		return err
	}
	if close {
		p.close()
		return nil
	}
	p.list()
	return nil
}

// close ends the program: a palette closes once it has run a command.
func (p *Palette) close() {
	p.Closed = true
	if p.rt != nil {
		p.rt.Quit()
	}
}

// SetDensity is the density its host draws it at (surfrun.DensitySetter):
// the query's and the status's room is a row each side in Dense.
func (p *Palette) SetDensity(d surfui.Density) { p.density = d }

// BoxHeight is the box, in whole cells: the palette's lines at its
// scale, rounded up, and the dialog's half rows above and below.
func (p *Palette) BoxHeight() int { return (p.drawnLines()*p.Scale()+99)/100 + 1 }

// drawnLines are the palette's lines as drawn: the query's, the
// status's, the rows shown or the panel's, and the footer. The query and
// the status are a line with Pad's step above and below it: two lines in
// Compact and Roomy, whose step is half a row or less, three in Dense,
// whose step is a row.
func (p *Palette) drawnLines() int {
	padded := 2
	if p.density == surfui.Dense {
		padded = 3
	}
	n := padded + 1
	if p.Status() != "" {
		n += padded
	}
	r := p.Current()
	switch {
	case r != nil && p.Panel:
		return n + 1 + len(r.Actions)
	case r == nil:
		return n + 1
	}
	return n + min(p.fit(), len(p.Rows)-p.Top)
}

// Shown are the rows that fit, from Top.
func (p *Palette) Shown() []demodata.Row { return p.Rows[p.Top:min(p.Top+p.fit(), len(p.Rows))] }

// RowKey is shown row j's key: its value, or its kind and value where a
// row before it has that value.
func (p *Palette) RowKey(j int) string {
	shown := p.Shown()
	for _, r := range shown[:j] {
		if r.Value == shown[j].Value {
			return shown[j].Kind + ":" + shown[j].Value
		}
	}
	return shown[j].Value
}

// Icons is whether the rows shown have icons: then each row has its
// glyph, a blank where it has none.
func (p *Palette) Icons() bool {
	for _, r := range p.Shown() {
		if r.Icon != "" {
			return true
		}
	}
	return false
}

// Detail is what a row says muted after its title: a command's detail,
// a value's note.
func (p *Palette) Detail(r demodata.Row) string {
	if r.Kind == demodata.KindValue {
		return r.Note
	}
	return r.Detail
}

// Mark is a run of a row's group or title: a query's match, in the accent
// at weight 600, or the text between, in the row's tone.
type Mark struct {
	Text, Tone string
	Hit        bool
}

// GroupMarks and TitleMarks are r's group, muted, and its title, with the
// query's matches marked; no marks for no group.
func (p *Palette) GroupMarks(r demodata.Row) []Mark {
	if r.Group == "" {
		return nil
	}
	return marked(r.Group, r.GroupMatch, "muted")
}
func (p *Palette) TitleMarks(r demodata.Row) []Mark { return marked(r.Title, r.Match, "plain") }

// marked is s, in tone, with the runes in runs marked.
func marked(s string, runs [][2]int, tone string) []Mark {
	rs := []rune(s)
	var out []Mark
	at := 0
	for _, m := range runs {
		if m[0] > at {
			out = append(out, Mark{string(rs[at:m[0]]), tone, false})
		}
		out = append(out, Mark{string(rs[m[0]:m[1]]), "accent", true})
		at = m[1]
	}
	if at < len(rs) || len(out) == 0 {
		out = append(out, Mark{string(rs[at:]), tone, false})
	}
	return out
}

// KeyStyle is how the palette draws its keys: the settings' style
// (surfrun.KeyStyler), so that the surface draws key-caps.surf's in it.
func (p *Palette) KeyStyle() surfkeys.Style { return p.style.Keys }

// Hint is one of the footer's actions: its title and its key.
type Hint struct{ Title, Key string }

// hints are the footer's actions for the selected row, besides the panel's
// key: Enter's, then the rest that have a key, most significant first. In
// the panel: Run, then Back.
func (p *Palette) hints() (note string, acts []Hint) {
	r := p.Current()
	if r == nil {
		return "", nil
	}
	if r.Kind != demodata.KindValue {
		note = r.Value
	}
	if p.Panel {
		return note, []Hint{{"Run", "enter"}, {"Back", "escape"}}
	}
	for i, a := range r.Actions {
		if i == 0 {
			acts = append(acts, Hint{a.Title, "enter"})
		} else if a.Key != "" {
			acts = append(acts, Hint{a.Title, a.Key})
		}
	}
	return note, acts
}

// FootNote is the footer's note at its left: the selected row's id,
// cut where there is no room for all of it. Where even its first
// noteKeeps characters don't fit beside Enter's, it gives way: "".
func (p *Palette) FootNote() string {
	note, acts := p.hints()
	if len(acts) == 0 || !fits(p.Inner(), keeps(note), false, acts[0], nil) {
		return ""
	}
	return note
}

// Hints are the footer's actions flush right, the most significant
// first: Enter's, drawn rightmost, then the others leftwards (palette.surf
// lays them out right to left, in a fit row). Where they don't all fit in
// the palette's width with the note and Actions, the least significant
// give way; the fit row leaves out, whole, any the estimate kept that
// its density has no room for.
func (p *Palette) Hints() []Hint {
	note, acts := p.hints()
	if len(acts) == 0 {
		return nil
	}
	more := acts[1:]
	for len(more) > 0 && !fits(p.Inner(), note, !p.Panel, acts[0], more) {
		more = more[:len(more)-1]
	}
	return append([]Hint{acts[0]}, more...)
}

// ActionsHint is whether the footer shows Actions and its key, after the
// note: out of the panel, where it fits beside Enter's with the note's
// first noteKeeps characters. Narrower, it gives way, so that the note
// and Enter's stay legible.
func (p *Palette) ActionsHint() bool {
	note, acts := p.hints()
	if p.Panel || len(acts) == 0 {
		return false
	}
	return fits(p.Inner(), keeps(note), true, acts[0], nil)
}

// noteKeeps are the characters of the footer's note that it shows at
// least, as the legible gate holds a cut text to: Actions gives way to
// them.
const noteKeeps = 4

// keeps is the start of note that the footer keeps at least.
func keeps(note string) string {
	if rs := []rune(note); len(rs) > noteKeeps {
		return string(rs[:noteKeeps])
	}
	return note
}

// keyedFrom is the narrowest palette, in cells of its own, whose rows
// show their keys. A row's caps (three, some 14 cells, at the most) leave
// a label beside them some 7 characters there; narrower, the caps would
// leave it a letter or two, so the rows show their labels alone.
const keyedFrom = 24

// iconCells is the room the rows' icons take where they have them: the
// icon's two cells and the gap after it.
const iconCells = 3

// Keyed is whether the rows show their keys: where the palette is
// keyedFrom cells wide or more, and iconCells more where the rows have
// icons.
func (p *Palette) Keyed() bool {
	if p.Icons() {
		return p.Inner() >= keyedFrom+iconCells
	}
	return p.Inner() >= keyedFrom
}

// fits is whether the footer's note, Actions (with panel), first and more
// fit in inner cells of the palette's own (before its scale): an estimate
// of the type, the kit's Inter at about 8.4 px a character and the caps'
// mono at 9.6, a cell 9.4 px, as no host says how wide its text sets.
func fits(inner int, note string, panel bool, first Hint, more []Hint) bool {
	const cell, char, mono, capPad, gap, apart = 9.4, 8.4, 9.6, 14.8, 8.0, 20.0
	text := func(s string) float64 { return float64(utf8.RuneCountInString(s)) * char }
	caps := func(seq string) float64 {
		w := 0.0
		for _, c := range surfkeys.Caps(seedKeys, true, seq) {
			w += max(float64(utf8.RuneCountInString(c))*mono+capPad, 28) + 6.4
		}
		return w - 6.4
	}
	acts := append(append([]Hint(nil), more...), first)
	if panel {
		acts = append([]Hint{{"Actions", "ctrl+k"}}, acts...)
	}
	w := text(note)
	for _, a := range acts {
		w += apart + text(a.Title) + gap + caps(a.Key)
	}
	return w <= float64(inner)*cell-32
}

// seedKeys is the keys' style the footer's estimate takes its caps'
// widths from, glyph caps, whatever the style in force: an estimate need
// not follow it.
var seedKeys = surfkeys.Style{Caps: "glyphs", Case: "lower", Then: "space", Glue: "on"}
