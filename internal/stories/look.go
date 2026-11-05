package stories

import (
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The look's two screens (gov R-24, the mock the maintainer took): a
// settings page, and the agents' sidebar with the palette open over the
// content, each drawn as the screens draw theirs, with the look's marks:
// an icon beside each label, a dot for a state, the sidebars' chrome and
// the sections' captions.
func init() {
	All = append(All,
		Story{"look/settings", 180, lookSettings},
		Story{"look/agents", 180, lookAgents},
	)
}

// The MDI glyphs of AdwaitaMono Nerd Font Propo the mock uses, by their
// names in the font.
const (
	mdCogOutline     = "\U000F08BB"
	mdSplitVertical  = "\U000F0BCC"
	mdDockLeft       = "\U000F10AA"
	mdKeyboard       = "\U000F097B"
	mdWindowRestore  = "\U000F05B2"
	mdRobotOutline   = "\U000F167A"
	mdMagnify        = "\U000F0349"
	mdPaletteOutline = "\U000F0E0C"
	mdConsole        = "\U000F018D"
	mdTab            = "\U000F04E9"
	mdPencilOutline  = "\U000F0CB6"
	mdContentCopy    = "\U000F018F"
	mdBellOutline    = "\U000F009C"
	mdSourceBranch   = "\U000F062C"
	mdPlus           = "\U000F0415"
	mdClose          = "\U000F0156"
	mdMenuDown       = "\U000F035D"
	mdConsoleLine    = "\U000F07B7"
	mdViewColumn     = "\U000F056D"
	mdThermometer    = "\U000F050F"
	mdHammerWrench   = "\U000F1323"
	mdServerOff      = "\U000F048E"
	mdMemory         = "\U000F035B"
	mdCloseCircle    = "\U000F015A"
	mdCheckCircle    = "\U000F05E1"
	mdSplitHoriz     = "\U000F0BCB"
	mdArrowExpand    = "\U000F0616"
	mdRestart        = "\U000F0709"
	mdArrowRight     = "\U000F0054"
	mdHexagonOutline = "\U000F02D9"
)

// hexagon is an SVG icon: a hexagon's outline, its glyph in the cells
// MDI's hexagon-outline.
func hexagon() *surfui.Node {
	return surfui.SVGIcon(mdHexagonOutline, "M12 2L21 7V17L12 22L3 17V7ZM12 4.3L5 8.2V15.8L12 19.7L19 15.8V8.2Z")
}

// mockKeys is the mock's key style: each key on a cap of its own, as a
// word in lower case. The mock draws a key, not a sequence, so it draws
// each in that style as it is (surfkeys.Node), whatever the gallery's.
var mockKeys = surfkeys.Style{Caps: "text", Case: "lower"}

// keyCap is one key's cap as the mock draws it.
func keyCap(name string) *surfui.Node { return surfkeys.Node(mockKeys, true, name) }

// caps are the mock's caps for names, side by side, kept whole.
func caps(names ...string) *surfui.Node {
	var ks []*surfui.Node
	for _, k := range names {
		ks = append(ks, keyCap(k))
	}
	return surfui.Row(ks...).Keep()
}

// lookSettings is plx's settings page: its sections in a sidebar, one
// with a status under its label; then a section's groups, each a card
// with its title over it, its settings in it, each its label and its
// control and a line about it, a hairline between them, and a note under
// one; and the page's buttons, Reset the destructive one.
func lookSettings() *surfui.Node {
	nav := func(glyph, title, status string, sel bool) *surfui.Node {
		label := surfui.Text(title)
		if status != "" {
			label = surfui.Stack(surfui.Text(title), surfui.Text(status).Small().Muted())
		}
		return surfui.Item(surfui.Icon(glyph), label).Gap(2).Selected(sel)
	}
	setting := func(label, about string, control *surfui.Node) *surfui.Node {
		return surfui.Row(surfui.Stack(surfui.Text(label).Bold(), surfui.Text(about).Small().Muted().Wrap()).Grow(), control).Gap(4).Pad(0, 1)
	}
	group := func(title, note string, settings ...*surfui.Node) *surfui.Node {
		var rows []*surfui.Node
		for i, st := range settings {
			if i > 0 {
				rows = append(rows, surfui.Rule())
			}
			rows = append(rows, st)
		}
		g := surfui.Stack(surfui.Text(title).Semibold(), surfui.Box(rows...).Card())
		if note != "" {
			g.Add(surfui.Text(note).Small().Muted().Wrap())
		}
		return g.Down(1)
	}
	sidebar := surfui.Stack(
		surfui.Row(surfui.Icon(mdCogOutline), surfui.Text("Settings").Bold()).Gap(2).Pad(2, 0),
		Force(surfui.List(
			nav(mdCogOutline, "General", "", true),
			nav(mdSplitVertical, "Panes", "", false),
			nav(mdDockLeft, "Docks and tools", "", false),
			nav(mdKeyboard, "Keys", "", false),
			nav(mdWindowRestore, "Windows", "", false),
			nav(mdRobotOutline, "Agents", "2 working", false),
		).Nav().Label("Sections"), "within").Down(1),
	).Width(30).Pad(1, 1).Chrome()
	page := surfui.Stack(
		surfui.Row(surfui.Heading("General").Grow(),
			surfui.Row(surfui.Icon(mdMagnify), surfui.Field().Bare().Placeholder("Search settings").Label("Search settings").Width(24)).Gap(2).Search(),
		).Gap(2),
		group("Terminal", "",
			setting("hotty enabled", "Draw panes with the terminal's surfaces where it has them; off behaves as a terminal without them.",
				surfui.Switch(true).Label("hotty enabled")),
			setting("Theme", "The colours plx draws in: neuroplastio is the terminal's own with neuroplastio's violet.",
				surfui.Select("neuroplastio", "neuroplastio", "terminal", "dracula", "nord", "gruvbox").Label("Theme"))),
		group("Panes", "Changes apply to panes opened after them.",
			setting("Pane border", "The border a pane has: a line rings each pane, a gap is a blank gutter.",
				surfui.Cycle("line", "line", "gap").Label("Pane border")),
			setting("Notify on exit", "Show a notice when a pane's program exits on its own.",
				surfui.Switch(false).Label("Notify on exit"))),
		surfui.Spacer(),
		surfui.Row(surfui.Text("set in config.overlay.kdl").Small().Muted().Grow(),
			surfui.Button("Reset section").Fill(surfui.Error), surfui.Button("Save").Fill(surfui.Accent)).Gap(2),
	).Grow().Pad(4, 1)
	return surfui.Row(sidebar, page).Top().Height(43)
}

// lookAgents is the agents' sidebar beside the content, and over it the
// palette, open as a popover: its query, the commands under their
// groups' captions with their keys, the selected one with the keyboard,
// and its footer.
func lookAgents() *surfui.Node {
	agent := func(t surfui.Tone, name, task, ago string, sel bool) *surfui.Node {
		return surfui.Item(surfui.Row(surfui.Dot().Tone(t),
			surfui.Stack(
				surfui.Row(surfui.Text(name).Bold().Grow(), surfui.Text(ago).Small().Muted()).Gap(2),
				surfui.Text(task).Small().Muted(),
			).Grow(),
		).Gap(1).Top().Grow()).Selected(sel)
	}
	sidebar := surfui.Stack(
		surfui.Row(surfui.Icon(mdRobotOutline), surfui.Text("Agents").Bold().Grow(), surfui.Badge("5")).Gap(2).Pad(2, 0),
		surfui.List(
			agent(surfui.Warn, "claude-1", "Fix the flaky parser test", "now", true),
			agent(surfui.Error, "claude-2", "Agents toolbar redesign", "2m", false),
			agent(surfui.Info, "pi-1", "Write the release notes", "5m", false),
			agent(surfui.Muted, "opencode-1", "idle", "1h", false),
			agent(surfui.Warn, "claude-4", "Settings page: groups by name", "3m", false),
		).Entries().Label("Agents").Down(1),
	).Width(34).Pad(1, 1).Chrome()
	command := func(glyph, title, detail string, sel bool, keys ...string) *surfui.Node {
		return surfui.Item(surfui.Icon(glyph), surfui.Text(title), surfui.Text(detail).Muted().Indent(1).Grow(), caps(keys...)).Gap(2).Selected(sel)
	}
	palette := surfui.Dialog(
		Force(surfui.Row(surfui.Icon(mdConsole), surfui.Field().Bare().Placeholder("Type a command…").Label("Command").Grow(), keyCap("escape")).Gap(2).Class("u-query"), "within"),
		surfui.Caption("Pane"),
		Force(surfui.List(
			command(mdSplitVertical, "Split right", "p-1", true, "ctrl", "b", "s"),
			command(mdWindowRestore, "Open in window", "p-1", false, "ctrl", "b", "o"),
			command(mdPencilOutline, "Rename", "p-1", false, "ctrl", "b", "r"),
			command(mdClose, "Close", "p-1", false, "ctrl", "b", "x"),
		).Label("Commands"), "within"),
		surfui.Caption("Tab"),
		surfui.List(
			command(mdTab, "New tab", "", false, "ctrl", "b", "c"),
			command(mdContentCopy, "Copy tab's layout", "", false),
			command(mdBellOutline, "Notices", "3 unread", false, "ctrl", "b", "!"),
		).Label("More commands"),
		surfui.Rule().Down(1),
		surfui.Row(surfui.Text("pane.split").Small().Muted().Grow(), surfui.Text("Actions").Small(), keyCap("ctrl"), keyCap("k"),
			surfui.Text("Run").Small().Indent(2), keyCap("enter")).Gap(1),
	).Width(76)
	content := surfui.Stack(
		surfui.Row(surfui.Spacer(), palette, surfui.Spacer()).Down(2),
		surfui.Spacer(),
		surfui.Row(surfui.Spacer(), surfui.Text("No pane focused. Pick an agent, or run a command.").Muted(), surfui.Spacer()),
		surfui.Spacer(),
	).Grow()
	return surfui.Row(sidebar, content).Top().Height(43)
}
