package demodata

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"neuroplast.io/hotty-surf/surfkeys"
)

// The kinds of palette row.
const (
	KindCommand = "command"
	KindAgent   = "agent"
	KindPane    = "pane"
	KindSetting = "setting"
	KindValue   = "value"
)

// Row is one row of the command palette.
//
// A command's row reads `Group: Title` ("Pane: Split"), its Detail muted
// after it. A value's row, while a command's values are staged, has no
// Group: it reads the value, its Note beside it.
type Row struct {
	// Value is what the row runs: a command's id ("pane.split"), an
	// agent's address, a pane's id, a setting's name, or on a value's row
	// the value itself ("down").
	Value string
	// Group is a command's noun, capitalised ("Pane"), or "Setting" on a
	// setting's row; "" on the other rows.
	Group string
	// Title is a command's title ("Split"), an agent's or a pane's name,
	// a setting's label, or the value on a value's row.
	Title string
	// Detail is what a command acts on ("p-1"), an agent's topic, a
	// pane's workspace, or a setting's name and value; "" for none.
	Detail string
	// Note is what a value is ("in effect"); "" for none, and always ""
	// on any other row.
	Note string
	// Keys are the keys that run a command, in myelin's names, a space
	// between a sequence's keys ("ctrl+b p n"); "" when no key does.
	Keys string
	// Match and GroupMatch are where the query matched the title and the
	// group, as [from, to) rune ranges, in order and apart. None with no
	// query, or where the query matched only the id.
	Match, GroupMatch [][2]int
	// Actions are what the row's keys do, Enter's first.
	Actions []Action
	// Kind is what the row is: KindCommand, KindAgent, KindPane,
	// KindSetting or KindValue.
	Kind string
	// TakesValue is whether Enter stages a value for it (Run…) rather
	// than running it.
	TakesValue bool
	// Used is whether a command's row was run from the palette before,
	// which lists it first and gives it Remove from used.
	Used bool
	// Complete is what Tab makes the line: a command's id, and a space
	// where it takes a value; an agent's or a pane's name; a value's
	// stage with the value.
	Complete string
	// Icon is an agent's emoji, drawn before its row; "" for none.
	Icon string
	// Pane is the pane an agent's or a pane's row acts on.
	Pane string
	// Stage is the line Enter puts in the field for a setting's row
	// (`config.set palette.position `), which lists the setting's values.
	Stage string
}

// Action is one thing a row's keys do.
type Action struct {
	// ID names it: run, stay, copy, remove, go, go-stay, close or set.
	ID string
	// Title is what the footer and the actions panel call it ("Run",
	// "Copy command").
	Title string
	// Key is the key that does it, in myelin's names ("mod+enter").
	Key string
}

// The actions a row may have.
var (
	actRun       = Action{ID: "run", Title: "Run", Key: "enter"}
	actStage     = Action{ID: "run", Title: "Run…", Key: "enter"}
	actStay      = Action{ID: "stay", Title: "Run and stay", Key: "mod+enter"}
	actCopy      = Action{ID: "copy", Title: "Copy command", Key: "alt+enter"}
	actForget    = Action{ID: "remove", Title: "Remove from used", Key: "shift+delete"}
	actGoPane    = Action{ID: "go", Title: "Go to pane", Key: "enter"}
	actStayPane  = Action{ID: "go-stay", Title: "Go to pane and stay", Key: "mod+enter"}
	actClosePane = Action{ID: "close", Title: "Close pane", Key: "shift+delete"}
	actGoAgent   = Action{ID: "go", Title: "Go to agent", Key: "enter"}
	actStayAgent = Action{ID: "go-stay", Title: "Go to agent and stay", Key: "mod+enter"}
	actEndAgent  = Action{ID: "close", Title: "End agent", Key: "shift+delete"}
	actSet       = Action{ID: "set", Title: "Set…", Key: "enter"}
)

// Command is one of the made-up app's commands.
type Command struct {
	// ID is its id ("pane.split"), and Title its title ("Split").
	ID, Title string
	// Detail is what it acts on: the pane the palette opened over
	// ("p-1"); "" for none.
	Detail string
	// Keys are the keys that run it, in myelin's names ("ctrl+b p s");
	// "" for none.
	Keys string
	// Args are the values it takes, in order; none for a command that
	// runs as it is.
	Args []Arg
}

// TakesValue is whether c takes a value, so that Enter stages it (Run…).
func (c Command) TakesValue() bool { return len(c.Args) > 0 }

// Arg is one value a command takes.
type Arg struct {
	// Name says what it is ("direction").
	Name string
	// Required is whether the command needs it; FreeForm whether it
	// takes any typed value, beside Values.
	Required, FreeForm bool
	// Values are the ones offered, in order; Notes says what some of
	// them are ("in effect").
	Values []string
	Notes  map[string]string
	// After are the values offered after each value of the argument
	// before, where it decides them (config.set's value, by its key).
	After map[string][]string
}

// Origin is the pane the palette opens over: the commands that act on a
// pane act on it.
const Origin = "p-1"

// commands are the made-up app's commands, in the palette's order. The
// keys are a leader keymap: ctrl+b, then p for a pane's, t for a tab's.
// The third has a shifted key, which the gallery's stories show.
var commands = []Command{
	{ID: "pane.new", Title: "New", Keys: "ctrl+b p n"},
	{ID: "pane.run", Title: "Run a command", Args: []Arg{{Name: "command", Required: true, FreeForm: true}}},
	{ID: "pane.reopen", Title: "Reopen closed", Keys: "ctrl+b p shift+x"},
	{ID: "pane.split", Title: "Split", Detail: Origin, Keys: "ctrl+b p s", Args: []Arg{
		{Name: "direction", Required: true, Values: []string{"right", "down", "left", "up"}},
		{Name: "command", FreeForm: true},
	}},
	{ID: "pane.close", Title: "Close", Detail: Origin, Keys: "ctrl+b p x"},
	{ID: "pane.zoom", Title: "Toggle zoom", Detail: Origin, Keys: "ctrl+b z"},
	{ID: "pane.rename", Title: "Rename", Detail: Origin, Keys: "ctrl+b p r",
		Args: []Arg{{Name: "name", Required: true, FreeForm: true}}},
	{ID: "pane.next", Title: "Next", Keys: "ctrl+b o"},
	{ID: "pane.swap", Title: "Swap", Detail: Origin, Args: []Arg{{Name: "pane", Required: true}}},
	{ID: "tab.new", Title: "New", Keys: "ctrl+b t n"},
	{ID: "tab.close", Title: "Close", Keys: "ctrl+b t x"},
	{ID: "tab.rename", Title: "Rename", Keys: "ctrl+b t r",
		Args: []Arg{{Name: "name", Required: true, FreeForm: true}}},
	{ID: "tab.next", Title: "Next", Keys: "ctrl+b ]"},
	{ID: "tab.prev", Title: "Previous", Keys: "ctrl+b ["},
	{ID: "theme.use", Title: "Use", Args: []Arg{{Name: "theme", Required: true}}},
	{ID: "config.set", Title: "Set", Args: []Arg{
		{Name: "setting", Required: true},
		{Name: "value", Required: true, FreeForm: true},
	}},
	{ID: "config.reload", Title: "Reload"},
	{ID: "dock.toggle", Title: "Toggle", Keys: "ctrl+b d"},
	{ID: "agents.open", Title: "Open", Keys: "ctrl+b a"},
	{ID: "notices.open", Title: "Open", Keys: "ctrl+b !"},
	{ID: "keys.show", Title: "Show all", Keys: "ctrl+b ?"},
	{ID: "window.open", Title: "Open pane in window", Detail: Origin},
	{ID: "window.title", Title: "Set title", Args: []Arg{{Name: "title", FreeForm: true}}},
}

// pane is one of the made-up workspace's panes.
type pane struct {
	id, name, workspace string
}

// shells are the panes no agent is in.
var shells = []pane{
	{"p-11", "shell", "api"},
	{"p-12", "logs", "api"},
	{"p-13", "shell", "docs"},
}

// Palette is the palette's data: the commands, the panes, the agents
// and the settings, read from its Source as each method is called, so
// that a run, a set or a close shows at once.
type Palette struct {
	src *Source
}

// resource is an agent, a pane or a setting as a query finds it: its
// row, and the words beside its title that a query may match.
type resource struct {
	row   Row
	words []string
}

// Palette is the palette's data.
func (s *Source) Palette() (*Palette, error) { return &Palette{src: s}, nil }

// snapshot is the palette's data at one moment.
type snapshot struct {
	cmds []Command
	res  []resource
	used []string
}

// now is the palette's data now.
func (p *Palette) now() snapshot {
	p.src.mu.Lock()
	defer p.src.mu.Unlock()
	return p.src.snapshot()
}

// snapshot is the palette's data now. The caller holds s.mu.
func (s *Source) snapshot() snapshot {
	n := snapshot{used: slices.Clone(s.used)}
	theme := s.valueOf("theme.name")
	var names []string
	after := map[string][]string{}
	var settings []resource
	for _, d := range page {
		for _, g := range d.groups {
			for _, k := range g.keys {
				if k.control == Caption {
					continue
				}
				names = append(names, k.name)
				if len(k.choices) > 0 {
					after[k.name] = slices.Clone(k.choices)
				}
				v := s.valueOf(k.name)
				stage := "config.set " + k.name + " "
				settings = append(settings, resource{
					row: Row{Kind: KindSetting, Value: k.name, Group: "Setting", Title: k.label,
						Detail: k.name + " · " + cmp.Or(v, "nothing"), Actions: []Action{actSet},
						TakesValue: true, Complete: stage, Stage: stage},
					words: []string{k.name},
				})
			}
		}
	}
	themes, _ := keyNamed("theme.name")
	for _, c := range commands {
		c.Args = slices.Clone(c.Args)
		for i := range c.Args {
			c.Args[i].Values = slices.Clone(c.Args[i].Values)
		}
		switch c.ID {
		case "theme.use":
			c.Args[0].Values = slices.Clone(themes.choices)
			c.Args[0].Notes = map[string]string{theme: "in effect"}
		case "config.set":
			c.Args[0].Values = names
			c.Args[1].After = after
		case "pane.swap":
			c.Args[0].Values = s.paneIDs()
		}
		n.cmds = append(n.cmds, c)
	}
	for _, a := range s.agents {
		n.res = append(n.res, resource{
			row: Row{Kind: KindAgent, Value: a.Address, Title: a.Name, Detail: cmp.Or(a.Topic, a.Status),
				Icon: a.Icon, Pane: a.Pane, Complete: a.Name,
				Actions: []Action{actGoAgent, actStayAgent, actEndAgent}},
			words: []string{"agent", a.Kind, a.Workspace, a.Host, a.Pane},
		})
	}
	for _, sh := range s.shells {
		n.res = append(n.res, resource{
			row: Row{Kind: KindPane, Value: sh.id, Title: sh.name, Detail: sh.workspace + " · " + sh.id,
				Pane: sh.id, Complete: sh.name, Actions: []Action{actGoPane, actStayPane, actClosePane}},
			words: []string{"pane", sh.id, sh.workspace},
		})
	}
	n.res = append(n.res, settings...)
	return n
}

// Command is the command whose id is id, with the values it offers now.
func (p *Palette) Command(id string) (Command, bool) { return p.now().command(id) }

// command is the command whose id is id.
func (n snapshot) command(id string) (Command, bool) {
	i := slices.IndexFunc(n.cmds, func(c Command) bool { return c.ID == id })
	if i < 0 {
		return Command{}, false
	}
	return n.cmds[i], true
}

// Rows are the rows for a query. With none, they are the commands, those
// used first, the latest first. With one, they are the commands whose
// label or id holds each of its words, best first, then the agents, the
// panes and the settings that do. A query that starts with @ looks only
// at the agents, the panes and the settings.
func (p *Palette) Rows(query string) []Row {
	n := p.now()
	q := strings.TrimSpace(query)
	if rest, ok := strings.CutPrefix(q, "@"); ok {
		return n.resources(strings.Fields(rest), true)
	}
	words := strings.Fields(q)
	if len(words) == 0 {
		var rows []Row
		for _, id := range n.used {
			if c, ok := n.command(id); ok {
				rows = append(rows, n.commandRow(c))
			}
		}
		for _, c := range n.cmds {
			if !slices.Contains(n.used, c.ID) {
				rows = append(rows, n.commandRow(c))
			}
		}
		return rows
	}
	var found []scored
	for _, c := range n.cmds {
		r := n.commandRow(c)
		if sc, ok := label(&r, words, c.ID); ok {
			found = append(found, scored{r, sc})
		}
	}
	return append(best(found), n.resources(words, false)...)
}

// resources are the agents, panes and settings whose title or words hold
// each of words, best first; with every, all of them where words is
// empty.
func (n snapshot) resources(words []string, every bool) []Row {
	var found []scored
	for _, res := range n.res {
		r := res.row
		if len(words) == 0 {
			if every {
				found = append(found, scored{r, 0})
			}
			continue
		}
		if sc, ok := label(&r, words, res.words...); ok {
			found = append(found, scored{r, sc})
		}
	}
	return best(found)
}

// scored is a row and how well the query matched it.
type scored struct {
	row   Row
	score int
}

// best are the rows, the best matched first, in their order where two
// match as well.
func best(found []scored) []Row {
	slices.SortStableFunc(found, func(a, b scored) int { return b.score - a.score })
	rows := make([]Row, len(found))
	for i, f := range found {
		rows[i] = f.row
	}
	return rows
}

// commandRow is c's row: its group from its id, its keys, and its
// actions.
func (n snapshot) commandRow(c Command) Row {
	noun, _, _ := strings.Cut(c.ID, ".")
	r := Row{Kind: KindCommand, Value: c.ID, Group: capital(noun), Title: c.Title, Detail: c.Detail,
		Keys: c.Keys, Used: slices.Contains(n.used, c.ID), Complete: c.ID, TakesValue: c.TakesValue()}
	if r.TakesValue {
		r.Complete += " "
		r.Actions = []Action{actStage, actCopy}
	} else {
		r.Actions = []Action{actRun, actStay, actCopy}
	}
	if r.Used {
		r.Actions = append(r.Actions, actForget)
	}
	return r
}

// capital is s with its first letter in upper case.
func capital(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

// Values are the rows for the value a command stages: those of its
// argument after the values prev, which hold typed, and typed itself
// where the argument takes any value. empty is what the list says where
// there are none.
func (p *Palette) Values(id string, prev []string, typed string) (rows []Row, empty string) {
	c, ok := p.now().command(id)
	if !ok || len(prev) >= len(c.Args) {
		return nil, ""
	}
	arg := c.Args[len(prev)]
	vals := arg.Values
	if len(prev) > 0 && arg.After != nil {
		vals = arg.After[prev[len(prev)-1]]
	}
	more := len(prev)+1 < len(c.Args)
	row := func(v string) Row {
		line := strings.Join(append(append([]string{id}, prev...), v), " ")
		r := Row{Kind: KindValue, Value: v, Title: v, Note: arg.Notes[v], Complete: line,
			TakesValue: more, Actions: []Action{actRun}}
		if more {
			r.Complete += " "
			r.Actions = []Action{actStage}
		}
		if typed != "" {
			if at, ok := whole(v, typed); ok {
				r.Match = runs(at)
			}
		}
		return r
	}
	for _, v := range vals {
		if typed == "" || strings.Contains(strings.ToLower(v), strings.ToLower(typed)) {
			rows = append(rows, row(v))
		}
	}
	if arg.FreeForm && typed != "" && !slices.Contains(vals, typed) {
		rows = append(rows, row(typed))
	}
	if len(rows) == 0 {
		if arg.FreeForm {
			return nil, "Type " + arg.Name
		}
		return nil, "No " + arg.Name + " matches."
	}
	return rows, ""
}

// label matches words against r's title, then its group, then each of
// also, and marks the title's and the group's matches on r. Each word
// must match one of them. score is how well they matched.
func label(r *Row, words []string, also ...string) (score int, ok bool) {
	var title, group []int
	for _, w := range words {
		if sc, at, ok := matchWord(r.Title, w); ok {
			score += sc
			title = append(title, at...)
			continue
		}
		if sc, at, ok := matchWord(r.Group, w); ok {
			score += sc
			group = append(group, at...)
			continue
		}
		hit := false
		for _, a := range also {
			if sc, _, ok := matchWord(a, w); ok {
				score += sc - 1
				hit = true
				break
			}
		}
		if !hit {
			return 0, false
		}
	}
	r.Match, r.GroupMatch = runs(title), runs(group)
	return score, true
}

// matchWord is where word is in s, case aside, as rune indexes: whole
// where s holds it whole, else its runes in order. score ranks the
// match: whole at the start of one of s's words is best, whole elsewhere
// next, in order last, the closer together the better.
func matchWord(s, word string) (score int, at []int, ok bool) {
	if word == "" || s == "" {
		return 0, nil, false
	}
	if at, ok := whole(s, word); ok {
		rs := []rune(s)
		if at[0] == 0 || !unicode.IsLetter(rs[at[0]-1]) {
			return 30, at, true
		}
		return 20, at, true
	}
	rs := []rune(strings.ToLower(s))
	qs := []rune(strings.ToLower(word))
	j := 0
	for i := 0; i < len(rs) && j < len(qs); i++ {
		if rs[i] == qs[j] {
			at = append(at, i)
			j++
		}
	}
	if j < len(qs) {
		return 0, nil, false
	}
	return max(10-(at[len(at)-1]-at[0]-len(at)+1), 1), at, true
}

// whole is where s holds word whole, case aside, as rune indexes.
func whole(s, word string) ([]int, bool) {
	rs := []rune(strings.ToLower(s))
	qs := []rune(strings.ToLower(word))
	for i := 0; i+len(qs) <= len(rs); i++ {
		if slices.Equal(rs[i:i+len(qs)], qs) {
			at := make([]int, len(qs))
			for k := range at {
				at[k] = i + k
			}
			return at, len(qs) > 0
		}
	}
	return nil, false
}

// runs are the indexes at as [from, to) ranges, in order and apart.
func runs(at []int) [][2]int {
	if len(at) == 0 {
		return nil
	}
	at = slices.Clone(at)
	slices.Sort(at)
	at = slices.Compact(at)
	var out [][2]int
	for _, i := range at {
		if n := len(out); n > 0 && out[n-1][1] == i {
			out[n-1][1] = i + 1
			continue
		}
		out = append(out, [2]int{i, i + 1})
	}
	return out
}

// Line is a command with its values as one line, a value with a space
// or a quote in single quotes: `pane.split down 'make test'`.
func Line(id string, values ...string) string {
	parts := []string{id}
	for _, v := range values {
		if v == "" || strings.ContainsAny(v, " '\"") {
			v = "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, " ")
}

// Run runs the command id with value, its values apart by spaces, or
// refuses it, saying why: a value missing, one not among those offered,
// or one the settings refuse. The last value takes the rest of the line
// where its argument takes any value. A run is remembered as used, the
// latest first. config.set and theme.use change the settings; the other
// commands change nothing here. The act is recorded either way: "run
// <id> <value>".
func (s *Source) Run(id, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value = strings.TrimSpace(value)
	s.record(strings.TrimSpace("run " + id + " " + value))
	c, ok := s.snapshot().command(id)
	if !ok {
		return fmt.Errorf("no command %s", id)
	}
	var vals []string
	rest := value
	for i, a := range c.Args {
		if rest == "" {
			if a.Required {
				return fmt.Errorf("%s wants a %s", id, a.Name)
			}
			break
		}
		v := rest
		if i < len(c.Args)-1 || !a.FreeForm {
			v, rest, _ = strings.Cut(rest, " ")
			rest = strings.TrimSpace(rest)
		} else {
			rest = ""
		}
		offered := a.Values
		if i > 0 && a.After != nil {
			offered = a.After[vals[i-1]]
		}
		if !a.FreeForm && !slices.Contains(offered, v) {
			return fmt.Errorf("no %s %s", a.Name, v)
		}
		vals = append(vals, v)
	}
	if rest != "" {
		return fmt.Errorf("%s has no place for %s", id, rest)
	}
	switch id {
	case "config.set":
		if err := s.set(vals[0], vals[1]); err != nil {
			return err
		}
	case "theme.use":
		if err := s.set("theme.name", vals[0]); err != nil {
			return err
		}
	}
	s.used = append([]string{id}, slices.DeleteFunc(s.used, func(u string) bool { return u == id })...)
	return nil
}

// Forget takes id out of the used commands. The act is recorded: "forget
// <id>".
func (s *Source) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("forget " + id)
	s.used = slices.DeleteFunc(s.used, func(u string) bool { return u == id })
	return nil
}

// Copy puts line on the clipboard. The act is recorded: "copy <line>".
func (s *Source) Copy(line string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("copy " + line)
	return nil
}

// GoTo focuses the pane whose id is id, or refuses where there is none,
// or where its host has stopped answering. The act is recorded either
// way: "go <id>".
func (s *Source) GoTo(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("go " + id)
	if !slices.Contains(s.paneIDs(), id) {
		return fmt.Errorf("no pane %s", id)
	}
	for _, a := range s.agents {
		if a.Pane == id && a.Offline {
			return fmt.Errorf("%s is not answering", a.Host)
		}
	}
	return nil
}

// Close closes the pane whose id is id, and ends the agent in it, or
// refuses where there is none. Every Watch hears an agent that ends. The
// act is recorded either way: "close <id>".
func (s *Source) Close(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.record("close " + id)
	if !slices.Contains(s.paneIDs(), id) {
		return fmt.Errorf("no pane %s", id)
	}
	s.shells = slices.DeleteFunc(s.shells, func(p pane) bool { return p.id == id })
	n := len(s.agents)
	s.agents = slices.DeleteFunc(s.agents, func(a Agent) bool { return a.Pane == id })
	if len(s.agents) < n {
		s.tell()
	}
	return nil
}

// paneIDs are the ids of the panes there are, the agents' first. The
// caller holds s.mu.
func (s *Source) paneIDs() []string {
	var ids []string
	for _, a := range s.agents {
		ids = append(ids, a.Pane)
	}
	for _, p := range s.shells {
		ids = append(ids, p.id)
	}
	return ids
}

// Style is what the demo screens draw by beside their data, from the
// settings in force.
type Style struct {
	// Keys is how keys are drawn on their caps: keys.caps's caps, in
	// lower case, a sequence's keys a gap apart.
	Keys surfkeys.Style
	// PaletteScale is palette.style.scale: the palette's size, in
	// percent.
	PaletteScale int
}

// Style is the style the settings say now.
func (s *Source) Style() (Style, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scale, err := strconv.Atoi(s.valueOf("palette.style.scale"))
	if err != nil || scale < 1 {
		scale = 100
	}
	return Style{
		Keys:         surfkeys.Style{Caps: s.valueOf("keys.caps"), Case: "lower", Then: "space", Glue: "on"},
		PaletteScale: scale,
	}, nil
}
