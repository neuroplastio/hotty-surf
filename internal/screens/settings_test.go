package screens

import (
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
)

// The settings page's parts, by id.
var (
	setNav     = id("split", "side", "nav")
	setSearch  = id("split", "main", "top", "search", "query")
	setHeading = id("split", "main", "top", 0)
	setStatus  = id("split", "main", "top", 1)
)

// setNavItem is section sec's item in the sidebar.
func setNavItem(sec string) string { return id("split", "side", "nav", sec) }

// setGroup is group g of the box that shows box (a section's id, or
// "search"), with its parts.
func setGroup(box string, g int, parts ...any) string {
	return id(append([]any{"split", "main", "box-" + box, strconv.Itoa(g)}, parts...)...)
}

// setRow is a part of setting name's block in group g's card in the box
// that shows box: its label and its control are in "label", the label's
// text, description and source in "label", "text".
func setRow(box string, g int, name string, parts ...any) string {
	return setGroup(box, g, append([]any{"card", name}, parts...)...)
}

// openSettings is the page in a tab of its own, on demodata's mock.
func openSettings(t *testing.T, m mode) (*app, *Settings) {
	t.Helper()
	src := demodata.Mock()
	s := NewSettings(src)
	return start(t, m.settings(s), src, true, 160, 43), s
}

// At open: the search has the keyboard, General is selected and shown,
// each section has its icon, Agents its status under its title; each
// group is a card with its title over it, as written, a rule between its
// settings and a note under it where it has one; and each setting has
// its label, control and description; one at its default has no source.
func TestSettingsOpens(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openSettings(t, m)
		if f := a.Surface().Focused(); f != setSearch {
			t.Errorf("focus on %q, want the search", f)
		}
		if got, _ := a.Surface().Attr(setSearch, "placeholder"); got != "Search settings" {
			t.Errorf("search placeholder %q", got)
		}
		if got := a.text(setHeading); got != "General" {
			t.Errorf("heading %q", got)
		}
		if !a.Has(setNavItem("general"), "is-sel") {
			t.Error("General is not selected")
		}
		var titles []string
		for _, sec := range []string{"general", "panes", "tools", "keys", "windows", "agents"} {
			titles = append(titles, a.text(id("split", "side", "nav", sec, "label", "title")))
		}
		if got := strings.Join(titles, "|"); got != "General|Panes|Docks and tools|Keys|Windows|Agents" {
			t.Errorf("sections %q", got)
		}
		if got := a.text(id("split", "side", "nav", "general", 0)); got != "\U000F08BB" {
			t.Errorf("General's icon %q", got)
		}
		if !a.Has(id("split", "side", "nav", "general", 0), "u-icon") || !a.Has(id("split", "side"), "u-chrome") {
			t.Error("no icon, or no chrome")
		}
		if got := a.text(id("split", "side", "nav", "agents", "label", "status")); got != "2 working" {
			t.Errorf("Agents' status %q", got)
		}
		if strings.Contains(a.html(), id("split", "side", "nav", "general", "label", "status")) {
			t.Error("General has a status")
		}
		if got := a.text(setGroup("general", 0, "title")) + "|" + a.text(setGroup("general", 1, "title")); got != "Terminal|Panes" {
			t.Errorf("titles %q", got)
		}
		if !a.Has(setGroup("general", 0, "card"), "u-card") {
			t.Error("the group is no card")
		}
		if h := a.html(); strings.Contains(h, setRow("general", 0, "hotty.enabled", "rule")) || !strings.Contains(h, setRow("general", 0, "theme.name", "rule")) {
			t.Error("a rule over the first setting, or none over the second")
		}
		if got := a.text(setGroup("general", 1, "note")); got != "Changes apply to panes opened after them." {
			t.Errorf("Panes' note %q", got)
		}
		if strings.Contains(a.html(), setGroup("general", 0, "note")) {
			t.Error("Terminal has a note")
		}
		if got := a.text(setRow("general", 0, "hotty.enabled", "label", "text", 0)); got != "hotty enabled" {
			t.Errorf("label %q", got)
		}
		if !a.Has(setRow("general", 0, "hotty.enabled", "label", "control"), "is-on") {
			t.Error("hotty enabled is not on")
		}
		if got := a.text(setRow("general", 0, "hotty.enabled", "label", "text", "about")); !strings.HasPrefix(got, "Draw the screens as hotty surfaces") {
			t.Errorf("description %q", got)
		}
		if strings.Contains(a.html(), setRow("general", 0, "hotty.enabled", "label", "text", "from")) {
			t.Error("a source row for the default")
		}
		if got := a.text(setRow("general", 0, "theme.name", "label", "control", 0)); got != "ink" {
			t.Errorf("theme select %q", got)
		}
		if acts := a.src.Acts(); len(acts) != 0 {
			t.Errorf("acts at open: %q", acts)
		}
	})
}

// A switch, a cycle and a select save at once, through the source, and
// the status says so; Reset takes the setting out of the overlay.
func TestSettingsSaveAtOnce(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openSettings(t, m)
		a.click(setNavItem("tools"))
		a.Wait("Docks and tools", func() bool { return a.text(setHeading) == "Docks and tools" })

		a.click(setRow("tools", 0, "dock.controls", "label", "control"))
		a.acted("set dock.controls false")
		a.Wait("the switch off", func() bool { return !a.Has(setRow("tools", 0, "dock.controls", "label", "control"), "is-on") })
		if got := a.text(setStatus); got != "dock.controls saved to the overlay" {
			t.Errorf("status %q", got)
		}

		side := setRow("tools", 0, "dock.side", "label", "control", 1) // the cycle's value, which steps on
		a.click(side)
		a.acted("set dock.controls false", "set dock.side right")
		a.Wait("right", func() bool { return a.text(side) == "right" })
		if got := a.text(setRow("tools", 0, "dock.side", "label", "text", "from")); got != demodata.FromOverlay+"Reset" {
			t.Errorf("source row after the set %q", got)
		}

		a.click(setRow("tools", 0, "dock.side", "label", "text", "from", "reset"))
		a.acted("set dock.controls false", "set dock.side right", "reset dock.side")
		a.Wait("left again", func() bool { return a.text(side) == "left" })
		if got := a.text(setStatus); got != "dock.side taken out of the overlay" {
			t.Errorf("status %q", got)
		}
		if strings.Contains(a.html(), setRow("tools", 0, "dock.side", "label", "text", "from")) {
			t.Error("a source row for the default")
		}

		a.click(setNavItem("general"))
		a.Wait("General", func() bool { return a.text(setHeading) == "General" })
		a.Pick(setRow("general", 0, "theme.name", "label", "control"), "dusk")
		a.Wait("the theme set", func() bool { return len(a.src.Acts()) == 4 && a.src.Acts()[3] == "set theme.name dusk" })
	})
}

// A field saves what is committed, on Enter or on leaving it, once: a
// refusal is the status, and the field goes back to the value in force
// (S7); a value the source takes is saved.
func TestSettingsField(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, s := openSettings(t, m)
		a.click(setNavItem("windows"))
		scale := setRow("windows", 0, "palette.style.scale", "label", "control")
		a.Wait("the scale's field", func() bool { return a.value(scale) == "80" })
		if got, _ := a.Surface().Attr(scale, "placeholder"); got != "100" {
			t.Errorf("placeholder %q", got)
		}
		if got := a.text(setRow("windows", 0, "palette.style.scale", "label", "text", "from")); got != demodata.FromConfig {
			t.Errorf("source row %q", got)
		}
		a.fill(scale, "30")
		a.until("typed", func() bool { return s.Drafts["palette.style.scale"] == "30" })
		if acts := a.src.Acts(); len(acts) != 0 {
			t.Fatalf("typing saved: %q", acts)
		}
		a.keys("enter")
		a.acted("set palette.style.scale 30")
		a.Wait("the refusal", func() bool {
			return a.text(setStatus) == "✗ palette.style.scale wants a number in 60..200, got 30"
		})
		a.Wait("the value in force again", func() bool { return a.value(scale) == "80" })
		a.fill(scale, "150")
		a.keys("enter")
		a.acted("set palette.style.scale 30", "set palette.style.scale 150")
		a.Wait("saved", func() bool { return a.text(setStatus) == "palette.style.scale saved to the overlay" })
		if got := a.value(scale); got != "150" {
			t.Errorf("field %q", got)
		}
		// Leaving the field saves what it holds, and leaving it again after
		// Enter saves nothing more.
		a.fill(scale, "180")
		a.keys("tab")
		a.acted("set palette.style.scale 30", "set palette.style.scale 150", "set palette.style.scale 180")
		a.fill(scale, "90")
		a.keys("enter", "tab")
		a.acted("set palette.style.scale 30", "set palette.style.scale 150", "set palette.style.scale 180",
			"set palette.style.scale 90")
		a.Wait("saved", func() bool { return a.value(scale) == "90" })
		if acts := a.src.Acts(); len(acts) != 4 {
			t.Errorf("acts %q", acts)
		}
	})
}

// The search lists every section's matches under Search, in one group
// with no title and no section selected; with none, it says so;
// picking a section empties it.
func TestSettingsSearch(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openSettings(t, m)
		a.fill(setSearch, "palette")
		a.Wait("Search", func() bool { return a.text(setHeading) == "Search" })
		var found []string
		for _, k := range []string{"palette.position", "palette.style.scale"} {
			if strings.Contains(a.html(), `id="`+setRow("search", 0, k)+`"`) {
				found = append(found, k)
			}
		}
		if len(found) != 2 {
			t.Errorf("found %q, want the two", found)
		}
		if strings.Contains(a.html(), setGroup("search", 0, "title")) {
			t.Error("a search's group has a title")
		}
		if strings.Contains(a.html(), "is-sel") {
			t.Error("a section is selected during a search")
		}
		a.fill(setSearch, "qqqq")
		a.Wait("no match", func() bool {
			return a.text(id("split", "main", "box-search", "none")) == `No setting has "qqqq" in it.`
		})
		a.click(setNavItem("keys"))
		a.Wait("Keys", func() bool { return a.text(setHeading) == "Keys" && a.value(setSearch) == "" })
	})
}

// The sidebar's keys move its selection, the section showing at once;
// during a search the first arrow_down selects General. The sidebar comes
// before the search, as the look has it: Shift+Tab goes back to it.
func TestSettingsNavKeys(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, s := openSettings(t, m)
		a.fill(setSearch, "palette")
		a.Wait("Search", func() bool { return a.text(setHeading) == "Search" })
		a.keys("shift+tab")
		a.Wait("the sidebar has the keyboard", func() bool { return a.Surface().Focused() == setNav })
		a.keys("arrow_down")
		a.Wait("General", func() bool { return a.text(setHeading) == "General" && a.value(setSearch) == "" })
		for _, step := range []struct{ key, want string }{
			{"arrow_down", "Panes"}, {"j", "Docks and tools"}, {"k", "Panes"}, {"shift+g", "Agents"},
			{"g", "General"}, {"end", "Agents"}, {"home", "General"}, {"page_down", "Agents"}, {"arrow_up", "Windows"},
		} {
			a.keys(step.key)
			a.Wait(step.key+": "+step.want, func() bool { return a.text(setHeading) == step.want })
		}
		a.until("the model agrees", func() bool { return s.Sec == 4 })
	})
}

// One change sends a few ops (F10): a switch's flip is its class, its
// source row and the status, not the page.
func TestSettingsFlipIsAFewOps(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openSettings(t, m)
		a.click(setNavItem("tools"))
		a.Wait("Docks and tools", func() bool { return a.text(setHeading) == "Docks and tools" })
		n := len(a.Ops())
		a.click(setRow("tools", 0, "dock.controls", "label", "control"))
		a.Wait("the switch off", func() bool { return !a.Has(setRow("tools", 0, "dock.controls", "label", "control"), "is-on") })
		ops := a.opsSince(n)
		if len(ops) == 0 || len(ops) > 8 {
			t.Errorf("%d ops for a flip: %+v", len(ops), ops)
		}
		for _, op := range ops {
			if op.Op == "morph" || op.Op == "replace" {
				t.Errorf("a flip sent %s %s", op.Op, op.Target)
			}
		}
	})
}

// The cells UI (F7): the same page as lines, the groups' titles among
// them and Agents' status beside its tab, which takes typing into the search, moves between the settings
// and changes them.
func TestSettingsCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, s := openSettings(t, m)
		a.cells()
		a.screenHas("Settings")
		a.screenHas("[General]")
		a.screenHas("Terminal")
		a.screenHas("Agents 2 working")
		a.screenHas("hotty enabled")
		for _, ch := range "palette" {
			a.keys(string(ch))
		}
		a.until("the search typed", func() bool { return s.Query == "palette" })
		a.screenHas("Palette position")
		a.keys("arrow_down", "space")
		a.acted("set palette.position top")
		a.screenHas("‹ top ›")
		a.keys("arrow_down")
		a.until("on the scale", func() bool { return s.At == 1 })
		a.keys("backspace", "backspace", "backspace", "6", "5", "enter")
		a.acted("set palette.position top", "set palette.style.scale 65")
		a.keys("backspace", "backspace", "7", "0", "arrow_up") // leaving the field saves it
		a.acted("set palette.position top", "set palette.style.scale 65", "set palette.style.scale 70")
		a.keys("arrow_down")
		a.keys("tab")
		a.until("General", func() bool { return s.Sec == 0 && s.Query == "" })
		a.keys("tab")
		a.until("Panes", func() bool { return s.Sec == 1 })
		a.screenHas("[Panes]")
		a.screenHas("Header")
	})
}
