package demodata

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// Control is what a setting's row changes it with.
type Control int

const (
	// Caption is no control: the value in words ("2 items"), set only in
	// the config file.
	Caption Control = iota
	// Switch is on or off. It saves at once.
	Switch
	// Cycle steps round a few choices. It saves at once.
	Cycle
	// Select picks one of more choices from a list. It saves at once.
	Select
	// Number is a field for a whole number in a range. It saves on Enter.
	Number
	// Text is a field for words. It saves on Enter.
	Text
)

// String is the control's name in lower case: "switch", "cycle" and so
// on, as a template compares it.
func (c Control) String() string {
	names := []string{"caption", "switch", "cycle", "select", "number", "text"}
	if c < 0 || int(c) >= len(names) {
		return "control(" + strconv.Itoa(int(c)) + ")"
	}
	return names[c]
}

// Section is one of the settings page's sections: its sidebar entry and
// its groups, in order.
type Section struct {
	// ID names it: "general", "tools".
	ID string
	// Title is its sidebar's label: "Docks and tools".
	Title string
	// Icon is its sidebar's glyph, a Material Design Icons code point in
	// the hosts' Nerd Font.
	Icon string
	// Status is a line under its sidebar label, small and muted, where it
	// says something: "2 working". Empty, the label is one line.
	Status string
	// Groups are what the page draws for it, in order.
	Groups []Group
}

// Group is a run of settings under a caption.
type Group struct {
	// Title is its title over its card, in sentence case, as the page
	// draws it: "Terminal", "Remote hosts".
	Title string
	// Note is a line under the group, small and muted, where it says
	// something: "Changes apply to new panes."
	Note string
	// Settings are the group's, in order.
	Settings []Setting
}

// Setting is one setting as its row on the settings page shows it.
type Setting struct {
	// Name is its key, dotted: "pane.border".
	Name string
	// Label is its row's label, as written: "hotty enabled".
	Label string
	// Description is what the row says under its label. A Caption's ends
	// with "(set it in the config file)".
	Description string
	// Control is what its row changes it with.
	Control Control
	// Choices are a Cycle's or a Select's, in order.
	Choices []string
	// Value is the value in force as the control shows it: "true" or
	// "false" for a Switch, a choice, a number's digits, the words of a
	// Text; for a Caption, the value in words ("2 items").
	Value string
	// Placeholder is what a Number's or a Text's empty field shows: the
	// default's text, or "nothing".
	Placeholder string
	// Source is where the value in force comes from, in the row's words:
	// "set here (overlay)", "from the config file", or "" for the
	// default, which shows none.
	Source string
	// Reset is whether the overlay holds the setting, so that its row has
	// a Reset after the source.
	Reset bool
}

// On is whether a Switch's value is on.
func (s Setting) On() bool { return s.Value == "true" }

// The words a row shows for where its value comes from.
const (
	FromOverlay = "set here (overlay)"
	FromConfig  = "from the config file"
)

// Settings is the settings page as it is now: a fresh copy each call, so
// that a caller may keep or change it.
func (s *Source) Settings() ([]Section, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Section, len(page))
	for i, d := range page {
		sec := Section{ID: d.id, Title: d.title, Icon: d.icon, Status: d.status}
		for _, g := range d.groups {
			grp := Group{Title: g.title, Note: g.note}
			for _, k := range g.keys {
				grp.Settings = append(grp.Settings, s.setting(k))
			}
			sec.Groups = append(sec.Groups, grp)
		}
		out[i] = sec
	}
	return out, nil
}

// setting is k as its row shows it now. The caller holds s.mu.
func (s *Source) setting(k key) Setting {
	st := Setting{
		Name:        k.name,
		Label:       k.label,
		Description: k.description,
		Control:     k.control,
		Value:       k.def,
		Placeholder: k.def,
	}
	if len(k.choices) > 0 {
		st.Choices = append([]string(nil), k.choices...)
	}
	if k.config != "" {
		st.Value, st.Source = k.config, FromConfig
	}
	if v, ok := s.overlay[k.name]; ok {
		st.Value, st.Source, st.Reset = v, FromOverlay, true
	}
	switch k.control {
	case Caption:
		st.Description += " (set it in the config file)"
		st.Placeholder = ""
	case Text:
		if st.Placeholder == "" {
			st.Placeholder = "nothing"
		}
	case Switch, Cycle, Select:
		st.Placeholder = ""
	}
	return st
}

// Set puts value in the overlay for the setting called name, so that it
// is the value in force, or refuses it, saying why in words a row can
// show after "✗ ". The act is recorded either way.
func (s *Source) Set(name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("set " + name + " " + value)
	return s.set(name, value)
}

// set is Set without the record. The caller holds s.mu.
func (s *Source) set(name, value string) error {
	k, ok := keyNamed(name)
	if !ok {
		return fmt.Errorf("no setting %s", name)
	}
	if err := k.check(value); err != nil {
		return err
	}
	if k.control == Number {
		// check took it: the digits alone, as the row shows them.
		n, _ := strconv.Atoi(strings.TrimSpace(value))
		value = strconv.Itoa(n)
	}
	s.overlay[name] = value
	return nil
}

// valueOf is the value in force of the setting called name: the
// overlay's, the config file's or the default. The caller holds s.mu.
func (s *Source) valueOf(name string) string {
	k, _ := keyNamed(name)
	if v, ok := s.overlay[name]; ok {
		return v
	}
	return cmp.Or(k.config, k.def)
}

// Reset takes the setting called name out of the overlay, so that the
// config file's value or the default is in force again. The act is
// recorded either way.
func (s *Source) Reset(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("reset " + name)
	if _, ok := keyNamed(name); !ok {
		return fmt.Errorf("no setting %s", name)
	}
	delete(s.overlay, name)
	return nil
}

// Search is the settings a query finds, as the page's search does: in the
// page's order, those whose name, label or description holds the query,
// trimmed, case aside; none for an empty query.
func Search(secs []Section, query string) []Setting {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []Setting
	for _, sec := range secs {
		for _, g := range sec.Groups {
			for _, st := range g.Settings {
				if strings.Contains(strings.ToLower(st.Name), q) ||
					strings.Contains(strings.ToLower(st.Label), q) ||
					strings.Contains(strings.ToLower(st.Description), q) {
					out = append(out, st)
				}
			}
		}
	}
	return out
}

// key is one setting of the made-up page: what it is, and the values it
// has in the default, the config file and the overlay to start with.
type key struct {
	name, label, description string
	control                  Control
	choices                  []string
	// def is the default, as the control shows it (a Caption's words).
	def string
	// config is the config file's value, "" for none.
	config string
	// overlay is the overlay's value when a Mock starts, "" for none.
	overlay string
	// lo and hi bound a Number.
	lo, hi int
}

// check is nil where value is one the setting takes, else why not.
func (k key) check(value string) error {
	switch k.control {
	case Caption:
		return fmt.Errorf("%s is set only in the config file", k.name)
	case Switch:
		if value != "true" && value != "false" {
			return fmt.Errorf("%s wants true or false, got %s", k.name, value)
		}
	case Cycle, Select:
		for _, c := range k.choices {
			if c == value {
				return nil
			}
		}
		return fmt.Errorf("%s wants one of %s, got %s",
			k.name, strings.Join(k.choices, ", "), value)
	case Number:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < k.lo || n > k.hi {
			return fmt.Errorf("%s wants a number in %d..%d, got %s",
				k.name, k.lo, k.hi, value)
		}
	}
	return nil
}

// keyNamed is the setting called name.
func keyNamed(name string) (key, bool) {
	for _, d := range page {
		for _, g := range d.groups {
			for _, k := range g.keys {
				if k.name == name {
					return k, true
				}
			}
		}
	}
	return key{}, false
}

type groupDef struct {
	title, note string
	keys        []key
}

type sectionDef struct {
	id, title, icon, status string
	groups                  []groupDef
}

// The sidebar's glyphs: Material Design Icons in the hosts' Nerd Font, as
// the look draws them.
const (
	iconCog      = "\U000F08BB" // cog-outline
	iconSplit    = "\U000F0BCC" // split-vertical
	iconDock     = "\U000F10AA" // dock-left
	iconKeyboard = "\U000F097B" // keyboard
	iconWindow   = "\U000F05B2" // window-restore
	iconRobot    = "\U000F167A" // robot-outline
)

// onOff is a Switch's choices.
var onOff = []string{"true", "false"}

// page is the made-up settings page. Between them its sections hold one
// of each control; a Number whose value in force is the config file's,
// not its default; a setting the overlay holds; sections of one group and
// of several; a group with a note under it and a section with a status
// under its label; and, in Panes, enough rows that the page scrolls in a
// terminal 43 rows tall.
var page = []sectionDef{
	{id: "general", title: "General", icon: iconCog, groups: []groupDef{
		{title: "Terminal", keys: []key{
			{name: "hotty.enabled", label: "hotty enabled", control: Switch, choices: onOff, def: "true",
				description: "Draw the screens as hotty surfaces where the terminal can"},
			{name: "theme.name", label: "Theme", control: Select, def: "ink",
				choices:     []string{"ink", "paper", "dusk", "harbor", "moss", "ember"},
				description: "The colours of the frame, the panes and the screens"},
		}},
		{title: "Panes", keys: []key{
			{name: "pane.border", label: "Pane border", control: Cycle, def: "line",
				choices:     []string{"line", "gap"},
				description: "What goes between two panes: a line, or a gap"},
			{name: "pane.notify", label: "Notify on exit", control: Switch, choices: onOff, def: "false",
				description: "Raise a notice when a pane's program ends"},
		}, note: "Changes apply to panes opened after them."},
	}},
	{id: "panes", title: "Panes", icon: iconSplit, groups: []groupDef{
		{title: "Header", keys: []key{
			{name: "pane.header.show", label: "Show headers", control: Switch, choices: onOff, def: "true",
				description: "A line above each pane with its name"},
			{name: "pane.header.title", label: "Header title", control: Text, def: "@name",
				description: "What the header says: words, and @name for the pane's name"},
			{name: "pane.header.align", label: "Header alignment", control: Cycle, def: "left",
				choices:     []string{"left", "center", "right"},
				description: "Where the header's title sits"},
		}},
		{title: "Badges", keys: []key{
			{name: "pane.badges", label: "Badges", control: Caption, def: "2 items",
				description: "The small labels in a pane's corners"},
			{name: "pane.tint", label: "Tint by program", control: Switch, choices: onOff, def: "true",
				description: "Colour a pane's border by the program it runs"},
		}},
		{title: "Content", keys: []key{
			{name: "pane.gap", label: "Gap", control: Number, def: "1", lo: 0, hi: 4,
				description: "Cells between two panes when the border is a gap"},
			{name: "pane.padding", label: "Padding", control: Number, def: "0", lo: 0, hi: 8,
				description: "Cells between a pane's border and its program"},
			{name: "pane.scrollback", label: "Scrollback", control: Number, def: "10000", lo: 100, hi: 100000,
				config:      "50000",
				description: "Lines a pane keeps above its screen"},
			{name: "pane.cursor", label: "Cursor", control: Cycle, def: "block",
				choices:     []string{"block", "bar", "underline"},
				description: "The cursor's shape where the program doesn't set one"},
			{name: "pane.bell", label: "Bell", control: Select, def: "badge",
				choices:     []string{"none", "flash", "sound", "badge", "notice"},
				description: "What a pane's bell does"},
			{name: "pane.dim", label: "Dim unfocused", control: Switch, choices: onOff, def: "false",
				description: "Draw the panes without the focus a little darker"},
			{name: "pane.word", label: "Word characters", control: Text, def: "",
				description: "Characters a double click takes as part of a word"},
		}},
	}},
	{id: "tools", title: "Docks and tools", icon: iconDock, groups: []groupDef{
		{title: "Docks", keys: []key{
			{name: "dock.side", label: "Dock side", control: Cycle, def: "left",
				choices:     []string{"left", "right"},
				description: "Where a tool's dock opens"},
			{name: "dock.size", label: "Dock size", control: Number, def: "30", lo: 20, hi: 60,
				description: "A dock's width, in cells"},
			{name: "dock.controls", label: "Dock controls", control: Switch, choices: onOff, def: "true",
				description: "Buttons to close and move a dock in its header"},
			{name: "tool.tabs", label: "Tool tabs", control: Cycle, def: "justified",
				choices:     []string{"justified", "packed"},
				description: "How a dock's tabs share its width"},
		}},
	}},
	{id: "keys", title: "Keys", icon: iconKeyboard, groups: []groupDef{
		{title: "Leader", keys: []key{
			{name: "keys.leader", label: "Leader key", control: Text, def: "ctrl+b",
				description: "The key that starts a sequence"},
			{name: "keys.timeout", label: "Sequence timeout", control: Number, def: "1000", lo: 200, hi: 5000,
				description: "Milliseconds to wait for a sequence's next key"},
		}},
		{title: "Hints", keys: []key{
			{name: "keys.hints", label: "Show hints", control: Switch, choices: onOff, def: "true",
				description: "After the leader, show what each next key does"},
			{name: "keys.caps", label: "Key caps", control: Cycle, def: "glyphs",
				choices:     []string{"glyphs", "text"},
				description: "Draw modifiers as glyphs or as words"},
		}},
	}},
	{id: "windows", title: "Windows", icon: iconWindow, groups: []groupDef{
		{title: "Palette", keys: []key{
			{name: "palette.position", label: "Palette position", control: Cycle, def: "top",
				choices: []string{"top", "center"}, overlay: "center",
				description: "Where the command palette opens"},
			{name: "palette.style.scale", label: "Palette scale", control: Number, def: "100", lo: 60, hi: 200,
				config:      "80",
				description: "The palette's size, in percent of the frame's type"},
		}},
		{title: "Frame", keys: []key{
			{name: "frame.title", label: "Window title", control: Text, def: "",
				description: "The terminal window's title; nothing for the workspace's name"},
			{name: "frame.status", label: "Status line", control: Switch, choices: onOff, def: "true",
				description: "A line at the bottom with the workspace and the time"},
		}},
	}},
	{id: "agents", title: "Agents", icon: iconRobot, status: "2 working", groups: []groupDef{
		{title: "List", keys: []key{
			{name: "agents.dock", label: "Agents dock", control: Cycle, def: "left",
				choices:     []string{"left", "right"},
				description: "Where the agents list opens"},
			{name: "agents.columns", label: "Columns", control: Text, def: "@name, @status",
				description: "What each agent's row shows, in order"},
			{name: "agents.notify", label: "Notify when blocked", control: Switch, choices: onOff, def: "true",
				description: "Raise a notice when an agent waits for you"},
			{name: "agents.order", label: "Order", control: Select, def: "recent",
				choices:     []string{"recent", "name", "status", "workspace", "manual"},
				description: "How the agents list is sorted"},
		}},
	}},
}
