package demodata

import (
	"slices"
	"strings"
	"testing"
)

// values are rows' values, in order.
func values(rows []Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Value
	}
	return out
}

func palette(t *testing.T, s *Source) *Palette {
	t.Helper()
	p, err := s.Palette()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// With no query, the rows are the commands in order; a command that
// takes a value stages it (Run…) and has no Run and stay.
func TestRowsAtOpen(t *testing.T) {
	rows := palette(t, Mock()).Rows("")
	if len(rows) != len(commands) {
		t.Fatalf("%d rows, want the %d commands", len(rows), len(commands))
	}
	r := rows[0]
	if r.Value != "pane.new" || r.Group != "Pane" || r.Title != "New" || r.Keys != "ctrl+b p n" ||
		r.Kind != KindCommand || r.TakesValue || r.Used || r.Complete != "pane.new" {
		t.Errorf("row 0 %+v", r)
	}
	if got := actionIDs(r); !slices.Equal(got, []string{"run", "stay", "copy"}) {
		t.Errorf("pane.new's actions %q", got)
	}
	split := rows[slices.IndexFunc(rows, func(r Row) bool { return r.Value == "pane.split" })]
	if !split.TakesValue || split.Complete != "pane.split " || split.Detail != Origin ||
		split.Actions[0].Title != "Run…" || slices.Contains(actionIDs(split), "stay") {
		t.Errorf("pane.split %+v", split)
	}
}

func actionIDs(r Row) []string {
	var out []string
	for _, a := range r.Actions {
		out = append(out, a.ID)
	}
	return out
}

// A query lists the commands that hold each of its words, best first,
// their matches marked, then the agents, panes and settings that do.
func TestRowsQuery(t *testing.T) {
	p := palette(t, Mock())
	for _, c := range []struct {
		query, first     string
		match, group     [][2]int
		hasIcon, noMatch bool
	}{
		{query: "split", first: "pane.split", match: [][2]int{{0, 5}}},
		{query: "zoom", first: "pane.zoom", match: [][2]int{{7, 11}}},
		{query: "pane new", first: "pane.new", match: [][2]int{{0, 3}}, group: [][2]int{{0, 4}}},
		{query: "palette scale", first: "palette.style.scale", match: [][2]int{{0, 7}, {8, 13}}},
		{query: "agent", first: "agents.open", group: [][2]int{{0, 5}}, hasIcon: true},
		{query: "qqqq", noMatch: true},
	} {
		rows := p.Rows(c.query)
		if c.noMatch {
			if len(rows) != 0 {
				t.Errorf("%q: %q", c.query, values(rows))
			}
			continue
		}
		if len(rows) == 0 || rows[0].Value != c.first {
			t.Errorf("%q: rows %q, want %s first", c.query, values(rows), c.first)
			continue
		}
		if !slices.Equal(rows[0].Match, c.match) || !slices.Equal(rows[0].GroupMatch, c.group) {
			t.Errorf("%q: %s's marks %v %v, want %v %v", c.query, c.first, rows[0].Match, rows[0].GroupMatch, c.match, c.group)
		}
		icons := slices.ContainsFunc(rows, func(r Row) bool { return r.Icon != "" })
		if icons != c.hasIcon {
			t.Errorf("%q: icons %v", c.query, icons)
		}
	}
	setting := p.Rows("palette scale")[0]
	if setting.Group != "Setting" || setting.Title != "Palette scale" || setting.Detail != "palette.style.scale · 80" ||
		setting.Stage != "config.set palette.style.scale " || actionIDs(setting)[0] != "set" {
		t.Errorf("the setting's row %+v", setting)
	}
}

// A query that starts with @ looks only at the agents, the panes and the
// settings: all of them with nothing after the @.
func TestRowsAt(t *testing.T) {
	p := palette(t, Mock())
	all := p.Rows("@")
	if len(all) == 0 || all[0].Kind != KindAgent || all[0].Value != "claude-1" || all[0].Pane != "p-1" {
		t.Fatalf("@: %q", values(all))
	}
	if slices.ContainsFunc(all, func(r Row) bool { return r.Kind == KindCommand }) {
		t.Error("@ lists a command")
	}
	if got := values(p.Rows("@logs")); !slices.Equal(got, []string{"p-12"}) {
		t.Errorf("@logs: %q", got)
	}
	if got := values(p.Rows("@p-6")); !slices.Equal(got, []string{"claude-3"}) {
		t.Errorf("@p-6, claude-3's pane: %q", got)
	}
	if got := p.Rows("@qqqq"); len(got) != 0 {
		t.Errorf("@qqqq: %q", values(got))
	}
}

// A command's values: those of its argument, then the next argument's
// after them, and a typed value where the argument takes any.
func TestValues(t *testing.T) {
	p := palette(t, Mock())
	rows, empty := p.Values("pane.split", nil, "")
	if got := values(rows); !slices.Equal(got, []string{"right", "down", "left", "up"}) || empty != "" {
		t.Fatalf("pane.split's directions %q %q", got, empty)
	}
	if r := rows[1]; !r.TakesValue || r.Complete != "pane.split down " || r.Actions[0].Title != "Run…" {
		t.Errorf("down %+v", r)
	}
	if rows, empty := p.Values("pane.split", []string{"down"}, ""); len(rows) != 0 || empty != "Type command" {
		t.Errorf("pane.split's command: %q %q", values(rows), empty)
	}
	rows, _ = p.Values("pane.split", []string{"down"}, "htop")
	if got := values(rows); !slices.Equal(got, []string{"htop"}) || rows[0].TakesValue || rows[0].Actions[0].Title != "Run" {
		t.Errorf("a typed command %+v", rows)
	}
	rows, _ = p.Values("theme.use", nil, "")
	if rows[0].Value != "ink" || rows[0].Note != "in effect" || rows[1].Note != "" {
		t.Errorf("themes %+v", rows[:2])
	}
	rows, _ = p.Values("theme.use", nil, "us")
	if got := values(rows); !slices.Equal(got, []string{"dusk"}) || !slices.Equal(rows[0].Match, [][2]int{{1, 3}}) {
		t.Errorf("themes with us %+v", rows)
	}
	if rows, empty := p.Values("theme.use", nil, "zz"); len(rows) != 0 || empty != "No theme matches." {
		t.Errorf("no theme: %q %q", values(rows), empty)
	}
	if got, _ := p.Values("config.set", []string{"palette.position"}, ""); !slices.Equal(values(got), []string{"top", "center"}) {
		t.Errorf("palette.position's values %q", values(got))
	}
	if got, _ := p.Values("config.set", []string{"palette.style.scale"}, "30"); !slices.Equal(values(got), []string{"30"}) {
		t.Errorf("a typed scale %q", values(got))
	}
}

// A run is remembered as used, the latest first, with Remove from used;
// Forget takes it out. config.set and theme.use change the settings, or
// are refused in the settings' words.
func TestRun(t *testing.T) {
	s := Mock()
	before := palette(t, s) // a Palette reads the Source as it is called
	if err := s.Run("tab.new", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Run("pane.zoom", ""); err != nil {
		t.Fatal(err)
	}
	rows := before.Rows("")
	if got := values(rows[:3]); !slices.Equal(got, []string{"pane.zoom", "tab.new", "pane.new"}) {
		t.Errorf("used first: %q", got)
	}
	if !rows[0].Used || actionIDs(rows[0])[3] != "remove" || rows[2].Used {
		t.Errorf("used rows %+v", rows[:3])
	}
	if err := s.Forget("pane.zoom"); err != nil {
		t.Fatal(err)
	}
	if got := palette(t, s).Rows("")[0].Value; got != "tab.new" {
		t.Errorf("after forget, row 0 is %s", got)
	}

	err := s.Run("config.set", "palette.style.scale 30")
	if err == nil || err.Error() != "palette.style.scale wants a number in 60..200, got 30" {
		t.Errorf("a scale of 30: %v", err)
	}
	if err := s.Run("config.set", "palette.style.scale 150"); err != nil {
		t.Fatal(err)
	}
	if err := s.Run("theme.use", "moss"); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Style(); st.PaletteScale != 150 {
		t.Errorf("scale %d after config.set", st.PaletteScale)
	}
	if rows, _ := palette(t, s).Values("theme.use", nil, "moss"); rows[0].Note != "in effect" {
		t.Errorf("moss after theme.use %+v", rows[0])
	}
	for _, c := range []struct{ id, value, refusal string }{
		{"pane.split", "", "pane.split wants a direction"},
		{"pane.split", "sideways", "no direction sideways"},
		{"pane.swap", "p-99", "no pane p-99"},
		{"pane.new", "extra", "pane.new has no place for extra"},
		{"config.set", "no.such 1", "no setting no.such"},
		{"no.such", "", "no command no.such"},
	} {
		if err := s.Run(c.id, c.value); err == nil || err.Error() != c.refusal {
			t.Errorf("run %s %q: %v, want %q", c.id, c.value, err, c.refusal)
		}
	}
	// The last value takes the rest of the line where it takes any; a
	// value is trimmed.
	if err := s.Run("pane.split", "down make test"); err != nil {
		t.Error(err)
	}
	if err := s.Run("config.set", " palette.style.scale 90 "); err != nil {
		t.Error(err)
	}
	if st, _ := s.Style(); st.PaletteScale != 90 {
		t.Errorf("scale %d after a padded config.set", st.PaletteScale)
	}
	want := []string{
		"run tab.new", "run pane.zoom", "forget pane.zoom",
		"run config.set palette.style.scale 30", "run config.set palette.style.scale 150",
		"run theme.use moss", "run pane.split", "run pane.split sideways", "run pane.swap p-99",
		"run pane.new extra", "run config.set no.such 1", "run no.such",
		"run pane.split down make test", "run config.set palette.style.scale 90",
	}
	if got := s.Acts(); !slices.Equal(got, want) {
		t.Errorf("acts:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// GoTo and Close act on the panes there are; closing an agent's pane
// ends it, and every Watch hears.
func TestPanes(t *testing.T) {
	s := Mock()
	if err := s.GoTo("p-3"); err != nil {
		t.Error(err)
	}
	if err := s.GoTo("p-7"); err == nil || err.Error() != "devbox is not answering" {
		t.Errorf("p-7, on a host that has stopped answering: %v", err)
	}
	if err := s.GoTo("p-99"); err == nil || err.Error() != "no pane p-99" {
		t.Errorf("p-99: %v", err)
	}
	if err := s.Close("p-11"); err != nil {
		t.Fatal(err)
	}
	if got := values(palette(t, s).Rows("@shell")); !slices.Equal(got, []string{"p-13"}) {
		t.Errorf("shells after closing p-11: %q", got)
	}
	heard := make(chan []Agent, 1)
	done := watch(t, s, func(as []Agent) { heard <- as })
	defer done()
	<-heard // the agents as the watch starts
	if err := s.Close("p-2"); err != nil {
		t.Fatal(err)
	}
	if as := <-heard; slices.ContainsFunc(as, func(a Agent) bool { return a.Name == "claude-2" }) {
		t.Error("claude-2 is still there")
	}
	want := []string{"go p-3", "go p-7", "go p-99", "close p-11", "close p-2"}
	if got := s.Acts(); !slices.Equal(got, want) {
		t.Errorf("acts %q", got)
	}
}

func TestLine(t *testing.T) {
	for _, c := range []struct {
		id     string
		values []string
		want   string
	}{
		{"pane.new", nil, "pane.new"},
		{"pane.split", []string{"down", "make test"}, "pane.split down 'make test'"},
		{"pane.rename", []string{"it's"}, `pane.rename 'it'\''s'`},
	} {
		if got := Line(c.id, c.values...); got != c.want {
			t.Errorf("Line(%s, %q) = %s, want %s", c.id, c.values, got, c.want)
		}
	}
}

func TestStyle(t *testing.T) {
	s := Mock()
	st, err := s.Style()
	if err != nil {
		t.Fatal(err)
	}
	if st.Keys.Caps != "glyphs" || st.Keys.Case != "lower" || st.PaletteScale != 80 {
		t.Errorf("style %+v", st)
	}
	if err := s.Set("keys.caps", "text"); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Style(); st.Keys.Caps != "text" {
		t.Errorf("caps %q after keys.caps text", st.Keys.Caps)
	}
}
