// Package stories holds a story for every primitive of the vocabulary
// and each of its states: what F5's gate renders in hotty-blitz and in
// Chromium, at 1.6× and 2×, and diffs.
//
// The gate renders still documents: nothing has the keyboard or the
// pointer. A story shows a focused or hovered look by forcing it (Force):
// its document's stylesheet reads `:focus`, `:focus-visible`,
// `:focus-within` and `:hover` as attribute selectors, which the story's
// nodes carry, in both engines alike. That shows the look draws the same;
// whether the pseudo-class matches is Blitz's element state, which
// hotty-blitz's own tests pin.
package stories

import (
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// Story is one state of one component, at a width in cells.
type Story struct {
	Name string
	Cols int
	Tree func() *surfui.Node
}

// Force draws n as if in states, space-separated: "focus" (`:focus` and
// `:focus-visible`), "within" (`:focus-within`), "hover" (`:hover`).
func Force(n *surfui.Node, states string) *surfui.Node { return n.Attr("data-force", states) }

var forced = strings.NewReplacer(
	":focus-visible", "[data-force~=focus]",
	":focus-within", "[data-force~=within]",
	":focus", "[data-force~=focus]",
	":hover", "[data-force~=hover]",
)

// Palette is the hues the stories draw in (hue/palette): five pairs, each
// 4.5:1 or over on both grounds. A story is a tree with no template, so
// the palette is declared here, and the stylesheet of the gate's documents
// and the gallery's runtime carry it.
var Palette = surfui.Palette{
	"coral": {Dark: "#ff8e7a", Light: "#b4321f", Cells: surfui.Cell(9)},
	"amber": {Dark: "#f2c14e", Light: "#8a5a00"},
	"lime":  {Dark: "#a3d977", Light: "#3f6b12"},
	"teal":  {Dark: "#4fd1c5", Light: "#0f766e", Cells: surfui.Cell(6)},
	"plum":  {Dark: "#c9a0ff", Light: "#6b3fa0"},
}

// CSS is surfui.CSS with its states forced: each of the pseudo-classes Force
// names written as the attribute it sets, and the stories' palette's rules
// after it. It is still, as the gate's documents are: a pulse dot is drawn
// at rest, at full opacity, which is what Chromium's shot draws too (it
// disables animations), so that no shot catches it mid-way.
func CSS() string { return forced.Replace(surfui.CSS) + Palette.CSS() + still }

// still stops what moves in surfui.CSS.
const still = "\n.u-dot.u-pulse { animation: none; }\n"

// Beside is css with each rule that reads a pseudo-class Force names
// followed by its forced copy: a node Force marks looks focused or
// hovered, and every other node keeps its real :focus and :hover. The
// gallery's states matrix draws so, beside live stories. A copy comes
// right after its rule, so the cascade's order is the rule's.
func Beside(css string) string {
	var b strings.Builder
	for css != "" {
		open := strings.IndexByte(css, '{')
		if open < 0 {
			b.WriteString(css)
			break
		}
		sel := css[:open]
		if strings.HasPrefix(strings.TrimSpace(sel), "@") {
			// A group rule (@media): its rules, each with its copy.
			depth, end := 1, open+1
			for ; end < len(css) && depth > 0; end++ {
				switch css[end] {
				case '{':
					depth++
				case '}':
					depth--
				}
			}
			b.WriteString(sel + "{" + Beside(css[open+1:end-1]) + "}")
			css = css[end:]
			continue
		}
		end := strings.IndexByte(css[open:], '}')
		if end < 0 {
			b.WriteString(css)
			break
		}
		rule := css[:open+end+1]
		b.WriteString(rule)
		if f := forced.Replace(sel); f != sel {
			b.WriteString("\n" + strings.TrimLeft(f, "\n") + css[open:open+end+1])
		}
		css = css[open+end+1:]
	}
	return b.String()
}

// Document is a resolved root as a document that `hotty render` and a
// browser open, with the stylesheet inline, its states forced (CSS).
func Document(root *surfui.Node) string {
	return `<!doctype html><html><head><meta charset="utf-8"><style>` + CSS() + `</style></head><body>` +
		surfui.HTML(root) + `</body></html>`
}

// KeyStyle is the style the stories draw their keys in (package surfkeys),
// in the gallery and in the gate's documents: plx's seed keys.style,
// glyph caps in lower case, glued, a gap between a sequence's chords, as
// the look's screens draw them.
var KeyStyle = surfkeys.Style{Caps: "glyphs", Case: "lower", Then: "space", Glue: "on"}

// All are the stories, in the order the gate and the book show them.
var All = []Story{
	{"text/tones", 60, func() *surfui.Node {
		return surfui.Stack(
			surfui.Text("Plain text, the terminal's colour"),
			surfui.Text("Muted, for what is secondary").Muted(),
			surfui.Row(surfui.Text("accent").Tone(surfui.Accent), surfui.Text("ok").Tone(surfui.OK), surfui.Text("warn").Tone(surfui.Warn),
				surfui.Text("error").Tone(surfui.Error), surfui.Text("info").Tone(surfui.Info)).Gap(1),
			surfui.Text("Bold").Bold(),
			surfui.Text("A long line that does not fit its box is cut with an ellipsis at its end, never wrapped"),
			surfui.Text("A long line that wraps goes on to the next row, at its box's edge, as prose does").Wrap(),
		)
	}},
	{"text/type", 50, func() *surfui.Node {
		return surfui.Stack(
			surfui.Heading("Settings"),
			surfui.Text("Pane border kind").Bold(),
			surfui.Text("The kind of border a pane has: line rings each pane, gap is a blank gutter").Small().Muted().Wrap(),
			surfui.Row(surfui.Text("set here (config.overlay.kdl)").Small().Muted(), surfui.Text("0 items").Small()).Gap(2),
		)
	}},
	{"text/path", 30, func() *surfui.Node {
		// A program's files, as its About page lists them: a path breaks
		// after a "/", on a host and in cells, not inside a name.
		return surfui.Stack(
			surfui.Text("Files").Bold(),
			surfui.Text("log /home/me/.cache/app/agents-shots/logs/program.log").Small().Muted().Wrap(),
			surfui.Text("state /srv/apps/1000/app.x1/cells-17/state/agents").Small().Muted().Wrap(),
		)
	}},
	{"line/runs", 30, func() *surfui.Node {
		// A palette's rows: each label one line of runs, its group muted
		// and its matches in the accent at weight 600, cut once, at its
		// end, beside keys that keep whole (text: a badge is a pixel wider
		// in Blitz, roomy, a gap of its own). The first fits, its detail
		// two cells on; the others are cut in their titles.
		row := func(sel bool, keys string, runs ...*surfui.Node) *surfui.Node {
			return surfui.Item(surfui.Line(runs...).Grow(), surfui.Text(keys).Muted().Keep()).Selected(sel)
		}
		return surfui.List(
			row(true, "ctrl+b s", surfui.Text("Pane").Muted(), surfui.Text(": ").Muted(), surfui.Text("Sp").Tone(surfui.Accent).Semibold(),
				surfui.Text("lit"), surfui.Text("p-1").Muted().Indent(2)),
			row(false, "ctrl+b p x", surfui.Text("S").Tone(surfui.Accent).Semibold(), surfui.Text("etting").Muted(), surfui.Text(": ").Muted(),
				surfui.Text("Pa").Tone(surfui.Accent).Semibold(), surfui.Text("lette position on the screen")),
			row(false, "w", surfui.Text("Workspace").Muted(), surfui.Text(": ").Muted(), surfui.Text("Close every pane but this one")),
		)
	}},
	{"line/paragraph", 40, func() *surfui.Node {
		// Prose with keys in it, as plx's onboarding writes it: its runs
		// and keys one paragraph, broken at its spaces, each key's caps
		// whole; a run in its own tone and weight across a break.
		return surfui.Stack(
			surfui.Line(surfui.Text("Press "), surfui.Key("ctrl+b"), surfui.Text(" and a hint shows what each next key does: "),
				surfui.Key("ctrl+b v"), surfui.Text(" splits right, "), surfui.Key("ctrl+b s"), surfui.Text(" below.")).Wrap(),
			surfui.Line(surfui.Text("What this does: ").Muted(), surfui.Text("it runs the command in the pane you picked").Bold(),
				surfui.Text(", and "), surfui.Key("escape"), surfui.Text(" closes the palette.")).Wrap(),
		).Gap(1)
	}},
	{"row/fit", 40, func() *surfui.Node {
		// A footer: its note, then flush right the actions that fit, each
		// whole, the most significant rightmost; those there is no room
		// for are left out. A long note leaves less room.
		act := func(title, key string) *surfui.Node { return surfui.Row(surfui.Text(title), surfui.Badge(key)).Gap(2) }
		foot := func(note string) *surfui.Node {
			return surfui.Row(surfui.Text(note).Muted(),
				surfui.Row(act("Run", "enter"), act("Stay", "mod+enter"), act("Copy", "alt+enter")).Fit().Gap(5)).Gap(5).Pad(3, 0)
		}
		return surfui.Stack(foot("pane.new"), foot("workspace.close-others"))
	}},
	{"row/fit-start", 40, func() *surfui.Node {
		// A footer: its note, then the hints that fit, in order from the
		// start, each whole; those there is no room for, the least
		// important last, are left out.
		hint := func(title, key string) *surfui.Node { return surfui.Row(surfui.Text(title), surfui.Key(key)).Gap(1) }
		return surfui.Stack(
			surfui.Row(surfui.Text("3 comments").Muted(),
				surfui.Row(hint("next", "j"), hint("prev", "k"), hint("comment", "c"), hint("approve", "a"), hint("quit", "q")).FitStart().Gap(2)).Gap(3).Pad(3, 0),
			surfui.Row(surfui.Text("Review").Muted(),
				surfui.Row(hint("next", "j"), hint("prev", "k"), hint("comment", "c"), hint("approve", "a"), hint("quit", "q")).FitStart().Gap(2)).Gap(3).Pad(3, 0),
		)
	}},
	{"badge/fills", 60, func() *surfui.Node {
		return surfui.Row(surfui.Badge("devbox"), surfui.Badge("ok").Fill(surfui.OK), surfui.Badge("warn").Fill(surfui.Warn),
			surfui.Badge("error").Fill(surfui.Error), surfui.Badge("info").Fill(surfui.Info), surfui.Badge("accent").Fill(surfui.Accent)).Gap(1)
	}},
	{"glyph/column", 40, func() *surfui.Node {
		return surfui.Stack(
			surfui.Row(surfui.Glyph("●").Tone(surfui.Warn), surfui.Glyph("🦀"), surfui.Text("alpha").Bold().Indent(1)),
			surfui.Row(surfui.Glyph("○").Tone(surfui.Muted), surfui.Glyph("🐙"), surfui.Text("beta").Bold().Indent(1)),
			surfui.Row(surfui.Glyph("◉").Tone(surfui.Error), surfui.Glyph(""), surfui.Text("codex-1").Bold().Indent(1)),
		)
	}},
	{"dot/shapes", 40, func() *surfui.Node {
		// A state told by its shape as well as its tone, as plx's agents
		// roster: working a dot that pulses (still in the gate, live in
		// the gallery), needs-you a target, idle a ring. A host
		// has no shape for offline (◌) until a dashed ring is proven
		// (blitz-gaps), so offline is a ring in the error's tone, its
		// name muted and its row saying so. On a sidebar's ground and a
		// selected row, the gap in a target is their fill.
		agent := func(dot, name *surfui.Node, state string, sel bool) *surfui.Node {
			return surfui.Item(dot, name.Grow(), surfui.Text(state).Small().Muted()).Gap(1).Selected(sel)
		}
		return surfui.List(
			agent(surfui.Dot().Tone(surfui.Info).Pulse(), surfui.Text("claude-1").Bold(), "working", false),
			agent(surfui.Dot().Tone(surfui.Error).Target(), surfui.Text("claude-2").Bold(), "needs you", true),
			agent(surfui.Dot().Tone(surfui.Muted).Ring(), surfui.Text("pi-1").Bold(), "idle", false),
			agent(surfui.Dot().Tone(surfui.Error).Ring(), surfui.Text("codex-1").Muted(), "offline", false),
		).Chrome()
	}},
	{"hue/palette", 50, func() *surfui.Node {
		// Named colours from data: each agent in the hue its user gave
		// it, a dot and its name, as the palette declares it (Palette). The
		// dark ground draws a hue's dark value and paper its light one,
		// each 4.5:1 or over, so text reads on both. An icon and a glyph
		// take a hue too.
		agent := func(hue, name, task string) *surfui.Node {
			return surfui.Item(surfui.Dot().Hue(hue), surfui.Text(name).Hue(hue).Bold(), surfui.Text(task).Small().Muted().Grow()).Gap(1)
		}
		return surfui.Stack(
			surfui.List(
				agent("coral", "claude-1", "refactoring the parser"),
				agent("amber", "codex-1", "waiting on review"),
				agent("lime", "pi-1", "tests are green"),
				agent("teal", "claude-2", "reading the logs"),
				agent("plum", "gemini-1", "drafting the plan"),
			).Chrome(),
			surfui.Row(surfui.Glyph("●").Hue("coral"), surfui.Icon("\U000F0493").Hue("teal"), surfui.Text("a glyph, an icon").Hue("plum")).Gap(1),
		).Gap(1)
	}},
	{"key/caps", 50, func() *surfui.Node {
		// A sheet of keys, where keys are the point: each a boxed cap.
		row := func(group, title string, sel bool, seq string) *surfui.Node {
			return surfui.Item(surfui.Text(group).Muted(), surfui.Text(title).Grow(), surfui.Key(seq).Cap()).Selected(sel)
		}
		hint := func(title, seq string) *surfui.Node {
			return surfui.Row(surfui.Text(title), surfui.Key(seq).Cap()).Gap(2)
		}
		return surfui.Stack(
			surfui.List(
				row("Pane: ", "New", true, "ctrl+b p n"),
				row("Pane: ", "Close now", false, "ctrl+b p shift+x"),
				row("Tab: ", "Toggle zoom", false, "ctrl+b z"),
			),
			surfui.Row(surfui.Text("pane.new").Muted(),
				surfui.Row(hint("Run", "enter"), hint("Run and stay", "mod+enter")).Fit().Gap(5)).Gap(5).Pad(3, 0),
		)
	}},
	{"key/styles", 80, func() *surfui.Node {
		// The same keys in each of plx's keys.style looks, as its
		// onboarding shows them while the person picks one: each drawn
		// in its own style (surfkeys.Node), whatever the gallery's.
		styles := []struct {
			name string
			st   surfkeys.Style
		}{
			{"defaults", surfkeys.Style{}},
			{"seed", KeyStyle},
			{"text", surfkeys.Style{Caps: "text", Case: "title"}},
			{"lower", surfkeys.Style{Caps: "text", Case: "lower", Sep: "-"}},
			{"shifted", surfkeys.Style{Caps: "text", Case: "shifted", Then: "then"}},
			{"mac", surfkeys.Style{OS: "mac"}},
		}
		st := surfui.Stack().Gap(1)
		for _, s := range styles {
			row := surfui.Row(surfui.Text(s.name).Muted().Width(10)).Gap(3)
			for _, seq := range []string{"ctrl+shift+t", "ctrl+b v", "mod+enter", "super+l"} {
				row.Add(surfkeys.Node(s.st, true, seq))
			}
			st.Add(row)
		}
		return st
	}},
	{"button/states", 60, func() *surfui.Node {
		return surfui.Row(surfui.Button("Restart"), surfui.Button("Focus").Fill(surfui.Accent), surfui.Button("Disabled").Disabled(true)).Gap(1)
	}},
	{"button/bare", 60, func() *surfui.Node {
		return surfui.Stack(
			surfui.Row(surfui.Text("set here (config.overlay.kdl)").Small().Muted(), surfui.Button("Reset").Bare()).Gap(2),
			surfui.Row(Force(surfui.Button("Focused"), "focus"), Force(surfui.Button("Hovered"), "hover"),
				Force(surfui.Button("Reset").Bare(), "focus"), Force(surfui.Button("Reset").Bare(), "hover")).Gap(2),
		).Gap(1)
	}},
	{"button/small", 50, func() *surfui.Node {
		// A small button fits its row: each row holding one is as tall
		// as the first, which holds none, at every density. The gate
		// measures it (data-same: the rows are one height in Chromium,
		// and the diff holds Blitz to Chromium). Plain on the selected
		// row, the primary, hovered, an icon alone, and bare.
		row := func(name, state string, tail ...*surfui.Node) *surfui.Node {
			kids := append([]*surfui.Node{surfui.Text(name).Grow(), surfui.Text(state).Small().Muted()}, tail...)
			return surfui.Item(kids...).Gap(1).Attr("data-same", "row")
		}
		return surfui.List(
			row("claude-1", "working"),
			row("claude-2", "waiting", surfui.Button("Stop").Small()).Selected(true),
			row("pi-1", "idle", surfui.Button("Start").Small().Fill(surfui.Accent)),
			row("codex-1", "done", Force(surfui.Button("Retry").Small(), "hover"),
				surfui.IconButton(surfui.Icon(mdClose), "", false).Label("Close").Small()),
			row("opencode-1", "idle", surfui.Button("Reset").Small().Bare()),
		)
	}},
	{"button/icon", 60, func() *surfui.Node {
		// An icon beside a button's label, before it or after it, centred
		// on its line a step from it, in the button's colour; an icon
		// alone, its label for a reader. An SVG icon is drawn as a glyph
		// is, and the whole button is its target: hovered and focused.
		icon := func(g string) *surfui.Node { return surfui.Icon(g) }
		return surfui.Stack(
			surfui.Row(surfui.IconButton(icon(mdRestart), "Restart", false), surfui.IconButton(icon(mdArrowRight), "Next", true),
				surfui.IconButton(icon(mdClose), "", false).Label("Close"), surfui.IconButton(hexagon(), "Mark", false),
				surfui.IconButton(icon(mdPlus), "Add", false).Fill(surfui.Accent)).Gap(1),
			surfui.Row(Force(surfui.IconButton(icon(mdRestart), "Focused", false), "focus"),
				Force(surfui.IconButton(hexagon(), "Hovered", false), "hover"),
				Force(surfui.IconButton(icon(mdContentCopy), "Copy", false).Bare(), "hover"),
				surfui.IconButton(icon(mdClose), "Off", false).Disabled(true)).Gap(1),
		).Gap(1)
	}},
	{"pill/states", 80, func() *surfui.Node {
		// A chrome control on a bar, in its states: at rest, the pointer
		// on it, what it opens shown, the keyboard on it, and what it
		// opens holding the keyboard; a bar's picker, raised, at rest and
		// under the pointer; a stack's tabs, small, the one whose stack
		// has the keyboard current, and shown without it, with an icon
		// button to add one.
		app := func() *surfui.Node { return surfui.IconButton(surfui.Icon(mdConsoleLine), "app", false).Pill() }
		picker := func() *surfui.Node {
			return surfui.IconButton(surfui.Icon(mdMenuDown), "neuroplastio", true).Pill().Raised()
		}
		tab := func(name string) *surfui.Node { return surfui.Button(name).Pill().Small() }
		return surfui.Stack(
			surfui.Row(app(), Force(app(), "hover"), app().Selected(true), Force(app(), "focus"), app().Current(true)).Gap(4),
			surfui.Row(picker(), Force(picker(), "hover"),
				surfui.Row(tab("editor").Current(true), tab("shell"), Force(tab("logs"), "hover"),
					surfui.IconButton(surfui.Icon(mdPlus), "", false).Label("New tab").Pill().Small()).Gap(1),
				surfui.Row(tab("editor").Selected(true), tab("shell")).Gap(1)).Gap(4),
		).Gap(1)
	}},
	{"pill/icon", 80, func() *surfui.Node {
		// An icon button, a circle: a stripe's tool at rest, under the
		// pointer, shown, holding the keyboard, tabbed to and disabled; a
		// stripe's toggle, off and on; and a header's actions, small, the
		// zoomed pane's zoom current and the pointer on close.
		tool := func() *surfui.Node {
			return surfui.IconButton(surfui.Icon(mdRobotOutline), "", false).Label("Agents").Pill()
		}
		act := func(g, label string) *surfui.Node {
			return surfui.IconButton(surfui.Icon(g), "", false).Label(label).Pill().Small()
		}
		toggle := func() *surfui.Node {
			return surfui.IconButton(surfui.Icon(mdViewColumn), "", false).Label("Stripes").Pill()
		}
		return surfui.Row(tool(), Force(tool(), "hover"), tool().Selected(true), tool().Current(true), Force(tool(), "focus"),
			tool().Disabled(true), toggle(), toggle().Selected(true),
			surfui.Row(act(mdSplitVertical, "Split right"), act(mdSplitHoriz, "Split down"),
				act(mdArrowExpand, "Zoom").Current(true), Force(act(mdClose, "Close"), "hover"))).Gap(4)
	}},
	{"segmented/states", 80, func() *surfui.Node {
		// A scope and a filter of three values each, as the board's: the
		// selected one raised, the pointer on one, the keyboard on one;
		// beside them a switch off, on and focused, the board's row.
		seg := func(sel int, hover, focus int, names ...string) *surfui.Node {
			var kids []*surfui.Node
			for i, s := range names {
				b := surfui.Button(s).Selected(i == sel)
				switch i {
				case hover:
					b = Force(b, "hover")
				case focus:
					b = Force(b, "focus")
				}
				kids = append(kids, b)
			}
			return surfui.Row(kids...).Segmented()
		}
		return surfui.Stack(
			surfui.Row(seg(0, 2, -1, "Workspace", "Machine", "All"), seg(0, -1, -1, "All", "Errors", "Agents"),
				surfui.Switch(false), surfui.Switch(true), Force(surfui.Switch(false), "focus")).Gap(5),
			surfui.Row(seg(1, -1, 2, "Workspace", "Machine", "All")),
		).Gap(1)
	}},
	{"strip/zones", 180, func() *surfui.Node {
		// A top strip: its start's picker, its centre's tabs, its end's
		// tools and toggle, each zone's items 4px apart.
		tool := func(g, label string) *surfui.Node {
			return surfui.IconButton(surfui.Icon(g), "", false).Label(label).Pill()
		}
		return surfui.Row(
			surfui.Row(surfui.IconButton(surfui.Icon(mdMenuDown), "neuroplastio", true).Pill().Raised()),
			surfui.Row(surfui.Button("editor").Pill().Small().Current(true), surfui.Button("shell").Pill().Small()),
			surfui.Row(tool(mdRobotOutline, "Agents"), tool(mdBellOutline, "Notices").Selected(true),
				tool(mdPaletteOutline, "Commands"), tool(mdViewColumn, "Stripes")),
		).Strip()
	}},
	{"grid/picker", 40, func() *surfui.Node {
		// An icon and emoji picker's grid: squares that wrap, the
		// pointer on one, the picked one.
		g := surfui.Row().Grid()
		for i, s := range []string{mdConsoleLine, mdRobotOutline, mdSourceBranch, mdBellOutline, mdCogOutline, mdPaletteOutline,
			mdKeyboard, mdMagnify, mdTab, mdPencilOutline, mdContentCopy, mdRestart} {
			b := surfui.IconButton(surfui.Icon(s), "", false).Label("icon " + strconv.Itoa(i))
			switch i {
			case 2:
				b.Selected(true)
			case 4:
				b = Force(b, "hover")
			}
			g.Add(b)
		}
		for _, e := range []string{"🦀", "🐙", "🚀", "🧪"} {
			g.Add(surfui.Button(e).Label("emoji " + e))
		}
		return surfui.Stack(g).Width(30)
	}},
	{"count/states", 60, func() *surfui.Node {
		// A tool's count on its button, on a strip's canvas: new (the
		// accent), waiting for you (warn), failed (error), informational
		// (muted), past 99, a dot with no number, and none once open.
		tool := func(c *surfui.Node) *surfui.Node {
			return surfui.IconButton(surfui.Icon(mdBellOutline), "", false).Label("Notices").Pill().WithCount(c)
		}
		return surfui.Row(tool(surfui.Count(3)), tool(surfui.Count(1).Tone(surfui.Warn)), tool(surfui.Count(2).Tone(surfui.Error)),
			tool(surfui.Count(12).Tone(surfui.Muted)), tool(surfui.Count(120)), tool(surfui.CountDot()),
			tool(nil).Selected(true)).Gap(5).Pad(2, 1).Chrome()
	}},
	{"widget/states", 140, func() *surfui.Node {
		// A widget's face on a strip's canvas: a clock at rest and
		// under the pointer, a branch whose popover shows, a build that
		// failed and one that passed, a warm machine, a host gone stale,
		// an icon alone and a value with a meter.
		return surfui.Stack(
			surfui.Row(surfui.Widget("", "14:32", ""), Force(surfui.Widget("", "14:32", ""), "hover"),
				surfui.Widget(mdSourceBranch, "main", "+2").Selected(true),
				surfui.Widget(mdCloseCircle, "make test", "").Tone(surfui.Error),
				surfui.Widget(mdThermometer, "84°", "").Tone(surfui.Warn),
				surfui.Widget(mdCheckCircle, "make test", "").Tone(surfui.OK)).Gap(5),
			surfui.Row(surfui.Widget(mdServerOff, "—", "devbox offline").Stale(), surfui.Widget(mdRobotOutline, "", ""),
				surfui.Widget(mdMemory, "23%", "").Meter(0.23)).Gap(5),
		).Gap(1).Pad(2, 1).Chrome()
	}},
	{"widget/own", 80, func() *surfui.Node {
		// A widget that draws its own face, the custom path: its widgets
		// in the kit's capsule, at rest, under the pointer and with its
		// popover shown; then one that mixes the data (an icon and a
		// value) with its own dots.
		agents := func() []*surfui.Node {
			return []*surfui.Node{surfui.Dot().Tone(surfui.OK), surfui.Dot().Tone(surfui.Warn),
				surfui.Dot().Ring().Tone(surfui.Muted), surfui.Text("3").Small()}
		}
		return surfui.Stack(
			surfui.Row(surfui.Widget("", "", "", agents()...), Force(surfui.Widget("", "", "", agents()...), "hover"),
				surfui.Widget("", "", "", agents()...).Selected(true),
				surfui.Widget(mdRobotOutline, "2", "", surfui.Dot().Tone(surfui.Warn))).Gap(5),
		).Gap(1).Pad(2, 1).Chrome()
	}},
	{"pop/shown", 50, func() *surfui.Node {
		// A popover a widget opens under it: its header (an icon, a
		// title, an action), a rule, and its body, a list.
		body := surfui.List(
			surfui.Item(surfui.Dot().Tone(surfui.OK), surfui.Text("reviewer").Grow(), surfui.Text("2m").Small().Muted()).Gap(2),
			surfui.Item(surfui.Dot().Tone(surfui.Warn), surfui.Text("builder").Grow(), surfui.Text("now").Small().Muted()).Gap(2),
		)
		head := surfui.Row(surfui.Icon(mdRobotOutline), surfui.Text("Agents").Semibold().Grow(),
			surfui.IconButton(surfui.Icon(mdClose), "", false).Label("Close").Pill().Small()).Gap(2).Pad(1, 0)
		return surfui.Row(surfui.Stack(surfui.Stack(head, surfui.Rule(), body.Pad(1, 1)).Popover()).Width(36))
	}},
	{"attention/states", 60, func() *surfui.Node {
		// What waits for you, in the warn: an agent's row, at rest and
		// as the keyboard's row, with a warn count; a tab and a tool
		// whose agent waits, and the tab current.
		agent := func(name, what, when string, att bool) *surfui.Node {
			end := surfui.Stack(surfui.Text(when).Small().Muted())
			if att {
				end.Add(surfui.Row(surfui.Spacer(), surfui.Count(1).Tone(surfui.Warn)))
			}
			return surfui.Item(surfui.Dot().Tone(map[bool]surfui.Tone{true: surfui.Warn, false: surfui.OK}[att]),
				surfui.Stack(surfui.Text(name).Semibold(), surfui.Text(what).Small().Muted()).Grow(), end).Gap(2).Top().Attention(att)
		}
		tab := func(name string) *surfui.Node { return surfui.Button(name).Pill().Small() }
		return surfui.Stack(
			surfui.List(agent("reviewer", "opus · reading the diff", "2m", false), agent("builder", "sonnet · waiting for you", "now", true)),
			Force(surfui.List(agent("builder", "sonnet · waiting for you", "now", true).Selected(true)), "within"),
			surfui.Row(tab("editor"), tab("agents").Attention(true), tab("agents").Attention(true).Current(true),
				surfui.IconButton(surfui.Icon(mdRobotOutline), "", false).Label("Agents").Pill().Attention(true)).Gap(1),
		).Gap(1)
	}},
	{"icon/svg", 50, func() *surfui.Node {
		// An SVG icon where a glyph would be: beside an item's label, the
		// search row's, at the icon's size and in its colour, the ink where
		// its item is selected.
		item := func(i *surfui.Node, title string, sel bool) *surfui.Node {
			return surfui.Item(i, surfui.Text(title)).Gap(2).Selected(sel)
		}
		return surfui.Stack(
			surfui.List(item(surfui.Icon(mdCogOutline), "Glyph", false), item(hexagon(), "Path", false),
				item(hexagon(), "Selected", true)).Nav(),
			surfui.Row(hexagon(), surfui.Field().Bare().Placeholder("Search").Width(16)).Gap(2).Search(),
		).Gap(1)
	}},
	{"icon/tone", 30, func() *surfui.Node {
		// A tone on an icon is its colour, a glyph's and an SVG's.
		return surfui.Row(surfui.Icon(mdCogOutline).Tone(surfui.OK), hexagon().Tone(surfui.Warn),
			surfui.Icon(mdCogOutline).Tone(surfui.Error), hexagon().Tone(surfui.Accent)).Gap(2)
	}},
	{"text/clickable", 50, func() *surfui.Node {
		// What acts on a click with no look of its own says so under the
		// pointer and with the keyboard: a text and an icon in the accent,
		// a badge with a control's hover-line, a row on the hover fill.
		// Each is at rest, then the same hovered or focused.
		click := func(n *surfui.Node) *surfui.Node { return n.On("click", nil) }
		return surfui.Stack(
			surfui.Row(click(surfui.Text("Fix the parser")), Force(click(surfui.Text("Fix the parser")), "hover")).Gap(2),
			surfui.Row(click(surfui.Text("#4182").Muted()), Force(click(surfui.Text("#4182").Muted().Focusable()), "focus"),
				click(surfui.Icon(mdContentCopy)), Force(click(surfui.Icon(mdContentCopy)), "hover"),
				click(hexagon()), Force(click(hexagon()), "hover"),
				click(surfui.Badge("devbox")), Force(click(surfui.Badge("devbox")), "hover")).Gap(2),
			Force(click(surfui.Row(surfui.Icon(mdSourceBranch), surfui.Text("main"), surfui.Text("3 ahead").Small().Muted()).Gap(1)), "hover"),
		).Gap(1)
	}},
	{"field/states", 60, func() *surfui.Node {
		return surfui.Stack(
			surfui.Row(surfui.Text("Name").Width(10), surfui.Field().Value("alpha").Grow()),
			surfui.Row(surfui.Text("Empty").Width(10), surfui.Field().Placeholder("Type a name").Grow()),
			surfui.Row(surfui.Text("Invalid").Width(10), surfui.Field().Value("1200").Invalid(true).Grow()),
			surfui.Text("0 to 1000").Tone(surfui.Error).Indent(10),
			surfui.Row(surfui.Text("Disabled").Width(10), surfui.Field().Value("top").Disabled(true).Grow()),
		)
	}},
	{"field/focus", 50, func() *surfui.Node {
		// The labels' column is as wide as "Bare, focused" and a cell, so
		// that no label is cut and none touches its field's ring.
		return surfui.Stack(
			surfui.Row(surfui.Text("Focused").Width(14), Force(surfui.Field().Placeholder("Search settings"), "focus").Grow()),
			surfui.Row(surfui.Text("Hovered").Width(14), Force(surfui.Field().Value("80"), "hover").Grow()),
			surfui.Row(surfui.Text("Bare").Width(14), surfui.Field().Bare().Value("split").Grow()),
			surfui.Row(surfui.Text("Bare, focused").Width(14), Force(surfui.Field().Bare(), "focus").Grow()),
		)
	}},
	{"switch/states", 40, func() *surfui.Node {
		return surfui.Stack(
			surfui.Row(surfui.Switch(true), surfui.Text("On")).Gap(1),
			surfui.Row(surfui.Switch(false), surfui.Text("Off")).Gap(1),
			surfui.Row(surfui.Switch(true).Disabled(true), surfui.Text("Disabled").Muted()).Gap(1),
		)
	}},
	{"switch/focus", 40, func() *surfui.Node {
		return surfui.Stack(
			surfui.Row(Force(surfui.Switch(true), "focus"), surfui.Text("On, focused")).Gap(2),
			surfui.Row(Force(surfui.Switch(false), "focus"), surfui.Text("Off, focused")).Gap(2),
			surfui.Row(Force(surfui.Switch(false), "hover"), surfui.Text("Off, hovered")).Gap(2),
		).Gap(1)
	}},
	{"select/closed", 40, func() *surfui.Node {
		row := func(label string, s *surfui.Node) *surfui.Node { return surfui.Row(surfui.Text(label).Width(10), s) }
		pos := func() *surfui.Node { return surfui.Select("top", "top", "center", "bottom") }
		return surfui.Stack(
			row("Position", pos()),
			row("Focused", Force(pos(), "focus")),
			row("Hovered", Force(pos(), "hover")),
			row("Disabled", pos().Disabled(true)),
		).Gap(1)
	}},
	// The list is a surface of its own over the Select's (run's
	// showList), here drawn under it: its first option hovered, the
	// picked one on its fill, the keys on the last.
	{"select/open", 40, func() *surfui.Node {
		options := []string{"top", "center", "bottom"}
		s := surfui.Select("center", options...)
		surfui.OpenSelect(s)
		list := surfui.Kids(surfui.SelectList(surfui.Surface(), "center", options, 2))[0]
		Force(surfui.Kids(list)[0], "hover")
		return surfui.Stack(
			surfui.Row(surfui.Text("Position").Width(10), s),
			surfui.Row(surfui.Box().Width(10), surfui.Box(list).Width(14)),
		)
	}},
	// A tooltip is a surface of its own over the view (surfrun's Tip),
	// here drawn under what it is of, as its host places it: a text one,
	// and a rich one of a row with a dot, bold text and a muted line.
	{"tip/shown", 50, func() *surfui.Node {
		// A tooltip's card as its host shows it, in a bar's tip's look,
		// each as wide as the host would make it (its cells' width): a
		// text that wraps, a label with a muted word, and two lines, a
		// host's name over its state.
		card := func(n *surfui.Node) *surfui.Node {
			return surfui.Kids(surfui.TipCard(n, surfui.TipOf(n)))[0]
		}
		cols := func(n *surfui.Node) int {
			t := surfui.TipCard(n, surfui.TipOf(n))
			surfui.Resolve(t, surfui.TipID)
			return cells.DrawTip(t, KeyStyle, Palette).Cols
		}
		copyBtn := surfui.Button("Copy").Title("Copies the note's text to the clipboard, as it is shown")
		agents := surfui.Button("Agents").Tip(surfui.Row(surfui.Text("Agents"), surfui.Text("shown").Muted()).Gap(2))
		host := Force(surfui.Item(surfui.Dot().Tone(surfui.OK), surfui.Text("devbox")).Gap(1), "hover")
		host.Tip(surfui.Stack(
			surfui.Text("devbox"),
			surfui.Text("connected · 1 workspace mounted").Muted(),
		))
		return surfui.Stack(
			surfui.Row(copyBtn, agents).Gap(1),
			surfui.Row(surfui.Box().Width(2), surfui.Stack(card(copyBtn)).Width(30)),
			surfui.Row(surfui.Box().Width(2), surfui.Stack(card(agents)).Width(cols(agents))),
			surfui.List(host).Width(20),
			surfui.Row(surfui.Box().Width(2), surfui.Stack(card(host)).Width(cols(host))),
		).Gap(1)
	}},
	{"menu/shown", 50, func() *surfui.Node {
		// A menu as its host shows it, the one look of every menu: a
		// pane's, its heading, actions with icons and keys, the keyboard
		// on one, one that can't be picked, a rule, and one that
		// destroys.
		items := []surfui.MenuItem{
			{Heading: true, Label: "Pane"},
			{Label: "Split right", Icon: "view-split-vertical", Key: "ctrl+b %"},
			{Label: "Split down", Icon: "view-split-horizontal", Key: `ctrl+b "`},
			{Label: "Zoom", Icon: "arrow-expand", Key: "ctrl+b z"},
			{Label: "Rename…", Icon: "pencil-outline"},
			{Label: "Copy selection", Icon: "content-copy", Disabled: true},
			{Sep: true},
			{Label: "Close pane", Icon: "close", Danger: true},
		}
		card := surfui.Kids(surfui.MenuCard(nil, items, 2))[0]
		return surfui.Row(surfui.Stack(card).Width(24))
	}},
	{"cycle/states", 50, func() *surfui.Node {
		row := func(label string, c *surfui.Node) *surfui.Node {
			return surfui.Row(surfui.Text(label).Bold().Grow(), c)
		}
		return surfui.Stack(
			row("Pane border kind", surfui.Cycle("line", "line", "gap")),
			row("Pane render", Force(surfui.Cycle("auto", "auto", "cells", "hotty"), "focus")),
			row("Pane gap", surfui.Cycle("1", "0", "1")),
			row("Keys case", Force(surfui.Cycle("lower", "auto", "title", "lower", "shifted"), "hover")),
		)
	}},
	{"list/selection", 50, func() *surfui.Node {
		return surfui.List(
			surfui.Item(surfui.Text("pane.split").Grow(), surfui.Badge("ctrl+b %")),
			surfui.Item(surfui.Text("pane.close").Grow(), surfui.Badge("ctrl+b x")).Selected(true),
			surfui.Item(surfui.Text("workspace.new").Grow()),
		)
	}},
	{"list/states", 50, func() *surfui.Node {
		// A list's rows in their states, as the board's hosts: at rest,
		// under the pointer, selected, and the keyboard's row, the
		// selection of a list that has the keyboard (the accent's soft
		// fill and its marker).
		row := func(on bool, name string, end *surfui.Node) *surfui.Node {
			dot := surfui.Dot().Tone(surfui.OK)
			if !on {
				dot = surfui.Dot().Ring().Tone(surfui.Muted)
			}
			return surfui.Item(dot, surfui.Text(name).Grow(), end).Gap(2)
		}
		return surfui.Stack(
			surfui.List(row(true, "devbox", surfui.Badge("1 ws")),
				Force(row(false, "ci-runner", surfui.Text("offline").Small().Muted()), "hover"),
				row(true, "spare", surfui.Badge("2 ws")).Selected(true)),
			Force(surfui.List(row(true, "this machine", surfui.Badge("3 ws")).Selected(true)), "within"),
		)
	}},
	{"list/error", 50, func() *surfui.Node {
		// A machine that failed, at rest and selected: on dark the
		// selected row's error is the lighter red (--u-error-on-sel), so
		// that it clears 4.5:1 on the selected fill.
		row := func(sel bool) *surfui.Node {
			return surfui.Item(surfui.Dot().Tone(surfui.Error), surfui.Text("build-02").Grow(),
				surfui.Text("connection refused").Small().Tone(surfui.Error)).Gap(2).Selected(sel)
		}
		return surfui.List(row(false), row(true))
	}},
	{"list/keyboard", 50, func() *surfui.Node {
		// A palette's rows: in Compact, the selection is the accent band
		// while the List's container has the keyboard (the Field that
		// filters it), and the tonal band without.
		rows := func(query string) *surfui.Node {
			return surfui.Stack(
				surfui.Field().Bare().Value(query),
				surfui.List(
					surfui.Item(surfui.Text("Pane: ").Muted(), surfui.Text("Sp").Bold().Tone(surfui.Accent), surfui.Text("lit"),
						surfui.Text("p-1").Muted().Indent(2)).Selected(true),
					surfui.Item(surfui.Text("Setting: ").Muted(), surfui.Text("S").Bold().Tone(surfui.Accent), surfui.Text("crollback "),
						surfui.Text("p").Bold().Tone(surfui.Accent), surfui.Text("ages")),
				),
			)
		}
		return surfui.Stack(Force(rows("sp"), "within"), rows("sp")).Gap(1)
	}},
	{"list/two-line", 50, func() *surfui.Node {
		row := func(dot string, tone surfui.Tone, icon, name, topic string, sel bool) *surfui.Node {
			return surfui.Item(surfui.Stack(
				surfui.Row(surfui.Glyph(dot).Tone(tone), surfui.Glyph(icon), surfui.Text(name).Bold().Indent(1).Grow()),
				surfui.Text(topic).Muted().Indent(5),
			).Grow()).Selected(sel)
		}
		return surfui.List(
			row("●", surfui.Warn, "🦀", "alpha", "Agents toolbar redesign", false),
			row("○", surfui.Muted, "🐙", "beta", "Fix the flaky parser test", true),
			row("◉", surfui.Error, "", "codex-1", "Refactor session index", false),
		)
	}},
	// Wide enough for "Docks and tools" in each of the three navs at every
	// density (dense takes the most): a nav is as wide as its widest item.
	{"list/nav", 64, func() *surfui.Node {
		nav := func(sel int) *surfui.Node {
			l := surfui.List().Nav()
			for i, s := range []string{"General", "Panes", "Docks and tools", "Keys"} {
				l.Add(surfui.Item(surfui.Text(s)).Selected(i == sel))
			}
			return l
		}
		return surfui.Row(nav(0), Force(nav(1), "within"), Force(nav(3), "hover")).Gap(4).Top()
	}},
	{"list/drop", 40, func() *surfui.Node {
		// Where a drop would land, as the runtime marks it while a row is
		// dragged (surfui.DropMark): a line in the accent at a row's top
		// (before) or its bottom (after), which takes no room, over a
		// selected row's fill too; an outline round a folder (into).
		row := func(name string, sel bool, where string) *surfui.Node {
			return surfui.DropMark(surfui.Item(surfui.Text(name).Grow()).Drag().Drop(surfui.DropHalves).Selected(sel), where)
		}
		folder := func(name, where string) *surfui.Node {
			return surfui.DropMark(surfui.Item(surfui.Text(name).Grow(), surfui.Text("3").Muted()).Drop(surfui.DropThirds), where)
		}
		return surfui.Stack(
			surfui.List(row("alpha", false, ""), row("beta", false, surfui.WhereBefore), row("gamma", true, "")),
			surfui.List(row("alpha", false, ""), row("beta", true, surfui.WhereAfter), row("gamma", false, "")),
			surfui.List(folder("Inbox", surfui.WhereInto), folder("Archive", "")),
		).Gap(1)
	}},
	{"list/entries", 30, func() *surfui.Node {
		entry := func(dot string, tone surfui.Tone, emoji, name string, hand bool, lines ...string) *surfui.Node {
			head := surfui.Row(surfui.Text(name).Bold().Grow())
			if hand {
				head.Add(surfui.Text("✋"))
			}
			text := surfui.Stack(head)
			for _, l := range lines {
				text.Add(surfui.Text(l).Small().Muted().Wrap())
			}
			return surfui.Item(surfui.Row(surfui.Glyph(dot).Tone(tone), surfui.Glyph(emoji), text.Grow()).Gap(2).Top().Grow())
		}
		return surfui.List(
			Force(entry("●", surfui.Warn, "🦀", "claude-1", false, "Fix the flaky parser test", "working"), "focus"),
			entry("●", surfui.Info, "🦊", "pi-1", true, "Write the release notes for 26.10.29", "done"),
			Force(entry("○", surfui.Muted, "　", "opencode-1", false, "idle"), "hover"),
		).Entries()
	}},
	{"section/nested", 50, func() *surfui.Node {
		return surfui.Stack(
			surfui.Section("Keys",
				surfui.Row(surfui.Text("Base").Width(12), surfui.Text("neuroplastio").Muted()),
				surfui.Section("Style",
					surfui.Row(surfui.Text("Caps").Width(10), surfui.Text("auto").Muted()),
					surfui.Row(surfui.Text("Case").Width(10), surfui.Text("title").Muted()),
				),
			),
		)
	}},
	// A box has no room of its own: its user pads it, here as a card, a
	// palette's query row's room (Pad(3, 1)) round its lines.
	{"box/rule", 50, func() *surfui.Node {
		return surfui.Stack(
			surfui.Box(surfui.Stack(surfui.Heading("A box"), surfui.Text("Raised, for a card or a dialog").Muted())).Pad(3, 1),
			surfui.Rule(),
			surfui.Text("After a rule"),
		).Gap(1)
	}},
	{"box/scroll", 50, func() *surfui.Node {
		nav := surfui.List().Nav()
		for i, s := range []string{"General", "Panes", "Docks and tools", "Keys", "Windows", "Agents"} {
			nav.Add(surfui.Item(surfui.Text(s)).Selected(i == 1))
		}
		body := surfui.Stack()
		for _, s := range []string{"Pane border kind", "Pane render", "Pane gap", "Pane top-left badges",
			"Pane top-right badges", "Lone pane header", "Lone pane footer", "Pane tint"} {
			body.Add(surfui.Text(s).Bold(), surfui.Text("What it does, in a line").Small().Muted(), surfui.Rule())
		}
		return surfui.Row(nav, surfui.Stack(surfui.Heading("Panes"), Force(surfui.Box(body).Pad(2, 0).Scroll(), "focus")).Grow()).Gap(2).Top()
	}},
	{"dialog/box", 50, func() *surfui.Node {
		// Without caps, whose borders are a gap of their own (key/caps).
		// Its height in rows, as a palette counts its lines, the least it
		// takes on a host: a row more than Dense's eleven lines, which it
		// keeps, where Compact's rows, with air, grow past it.
		row := func(group, title, detail string, sel bool) *surfui.Node {
			return surfui.Item(surfui.Text(group).Muted(), surfui.Text(title), surfui.Text(detail).Muted().Indent(2).Grow()).Selected(sel)
		}
		return surfui.Dialog(surfui.Stack(
			surfui.Row(surfui.Text("❯").Tone(surfui.Accent).Bold(), surfui.Field().Bare().Grow()).Gap(1).Pad(3, 1),
			surfui.Rule(),
			surfui.List(
				row("Pane: ", "New", "", true),
				row("Pane: ", "Split", "p-1", false),
				row("Tab: ", "Toggle zoom", "p-1", false),
			),
			surfui.Rule(),
			surfui.Row(surfui.Text("pane.new").Muted().Grow(), surfui.Text("Run")).Gap(2).Pad(3, 0),
		).Pad(1, 0)).Height(12)
	}},
	{"scale/zoom", 50, func() *surfui.Node {
		part := func() *surfui.Node {
			return surfui.Stack(
				surfui.Item(surfui.Text("Pane: ").Muted(), surfui.Text("Split").Bold().Tone(surfui.Accent), surfui.Spacer(), surfui.Badge("p-1")).Selected(true),
				surfui.Item(surfui.Text("Pane: ").Muted(), surfui.Text("Focus"), surfui.Spacer(), surfui.Badge("p-2").Fill(surfui.Accent)),
			).Width(40)
		}
		return surfui.Stack(part(), part().Scale(80))
	}},
}

// Surface is a story's tree in a surface of the theme and density, its
// keys drawn in KeyStyle, as the gallery draws them: a cell in from the
// edges. A dialog's story isn't: a dialog has its own padding, which a
// pad given it would take over (modifiers win, surf.css's last rules).
func Surface(s Story, theme string, d surfui.Density) *surfui.Node {
	tree := s.Tree()
	if !strings.HasPrefix(s.Name, "dialog/") {
		tree.Pad(1, 0)
	}
	root := surfui.Surface(tree).Theme(theme).Density(d)
	surfkeys.Expand(root, KeyStyle, true)
	return root
}
