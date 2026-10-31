package demodata

import (
	"slices"
	"strings"
	"testing"
)

// find is the setting called name on the page, and whether there is one.
func find(t *testing.T, s *Source, name string) Setting {
	t.Helper()
	secs, err := s.Settings()
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range secs {
		for _, g := range sec.Groups {
			for _, st := range g.Settings {
				if st.Name == name {
					return st
				}
			}
		}
	}
	t.Fatalf("no setting %s", name)
	return Setting{}
}

func TestPage(t *testing.T) {
	secs, err := Mock().Settings()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, sec := range secs {
		ids = append(ids, sec.ID)
		if sec.Icon == "" || sec.Title == "" || len(sec.Groups) == 0 {
			t.Errorf("section %s: icon %q, title %q, %d groups", sec.ID, sec.Icon, sec.Title, len(sec.Groups))
		}
	}
	if want := []string{"general", "panes", "tools", "keys", "windows", "agents"}; !slices.Equal(ids, want) {
		t.Errorf("sections %v, want %v", ids, want)
	}
	var general []string
	for _, g := range secs[0].Groups {
		for _, st := range g.Settings {
			general = append(general, g.Title+"/"+st.Label)
		}
	}
	want := []string{"Terminal/hotty enabled", "Terminal/Theme", "Panes/Pane border", "Panes/Notify on exit"}
	if !slices.Equal(general, want) {
		t.Errorf("General is %v, want the look's %v", general, want)
	}

	// What the settings screen's cases need, somewhere on the page.
	seen := map[Control]bool{}
	names := map[string]bool{}
	var overlay, config, longSelect, oneGroup, manyGroups bool
	for _, sec := range secs {
		if sec.ID != "general" {
			oneGroup = oneGroup || len(sec.Groups) == 1
			manyGroups = manyGroups || len(sec.Groups) > 1
		}
		for _, g := range sec.Groups {
			if g.Title == "" {
				t.Errorf("section %s has a group with no title", sec.ID)
			}
			for _, st := range g.Settings {
				if names[st.Name] {
					t.Errorf("%s is on the page twice", st.Name)
				}
				names[st.Name] = true
				seen[st.Control] = true
				overlay = overlay || (st.Reset && st.Source == FromOverlay)
				config = config || st.Source == FromConfig
				longSelect = longSelect || (st.Control == Select && len(st.Choices) >= 5)
			}
		}
	}
	for c := Caption; c <= Text; c++ {
		if !seen[c] {
			t.Errorf("no %s on the page", c)
		}
	}
	if !overlay || !config || !longSelect || !oneGroup || !manyGroups {
		t.Errorf("overlay %v, config %v, a select of 5+ %v, one group %v, many groups %v",
			overlay, config, longSelect, oneGroup, manyGroups)
	}
}

func TestSetAndReset(t *testing.T) {
	s := Mock()
	st := find(t, s, "palette.style.scale")
	if st.Value != "80" || st.Placeholder != "100" || st.Source != FromConfig || st.Reset {
		t.Fatalf("palette.style.scale starts as %+v", st)
	}
	err := s.Set("palette.style.scale", "30")
	if err == nil || err.Error() != "palette.style.scale wants a number in 60..200, got 30" {
		t.Fatalf("set 30: %v", err)
	}
	if st := find(t, s, "palette.style.scale"); st.Value != "80" {
		t.Errorf("after a refusal the value is %q, want the one in force, 80", st.Value)
	}
	if err := s.Set("palette.style.scale", "150"); err != nil {
		t.Fatal(err)
	}
	if st := find(t, s, "palette.style.scale"); st.Value != "150" || st.Source != FromOverlay || !st.Reset {
		t.Errorf("after set 150: %+v", st)
	}
	if err := s.Reset("palette.style.scale"); err != nil {
		t.Fatal(err)
	}
	if st := find(t, s, "palette.style.scale"); st.Value != "80" || st.Source != FromConfig || st.Reset {
		t.Errorf("after reset: %+v", st)
	}

	if st := find(t, s, "palette.position"); st.Value != "center" || !st.Reset {
		t.Fatalf("palette.position starts as %+v", st)
	}
	if err := s.Reset("palette.position"); err != nil {
		t.Fatal(err)
	}
	if st := find(t, s, "palette.position"); st.Value != "top" || st.Source != "" || st.Reset {
		t.Errorf("palette.position after reset: %+v", st)
	}

	if err := s.Set("theme.name", "nope"); err == nil {
		t.Error("theme.name took a theme it doesn't list")
	}
	if err := s.Set("pane.badges", "3 items"); err == nil {
		t.Error("a caption took a value")
	}
	if err := s.Set("hotty.enabled", "false"); err != nil || find(t, s, "hotty.enabled").On() {
		t.Errorf("switch off: %v", err)
	}
	if err := s.Set("no.such", "1"); err == nil {
		t.Error("a setting that isn't there took a value")
	}
	// A number is kept as its digits, as its row shows it.
	if err := s.Set("pane.gap", " 03"); err != nil {
		t.Fatal(err)
	}
	if st := find(t, s, "pane.gap"); st.Value != "3" {
		t.Errorf("pane.gap set to %q, want 3", st.Value)
	}

	want := []string{
		"set palette.style.scale 30",
		"set palette.style.scale 150",
		"reset palette.style.scale",
		"reset palette.position",
		"set theme.name nope",
		"set pane.badges 3 items",
		"set hotty.enabled false",
		"set no.such 1",
		"set pane.gap  03",
	}
	if got := s.Acts(); !slices.Equal(got, want) {
		t.Errorf("acts:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSettingsIsACopy(t *testing.T) {
	s := Mock()
	secs, _ := s.Settings()
	secs[0].Groups[0].Settings[1].Choices[0] = "changed"
	secs[0].Groups[0].Settings[0].Value = "changed"
	if st := find(t, s, "theme.name"); st.Choices[0] != "ink" {
		t.Errorf("a caller's change reached the source: %v", st.Choices)
	}
	if st := find(t, s, "hotty.enabled"); st.Value != "true" {
		t.Errorf("a caller's change reached the source: %q", st.Value)
	}
}

func TestSearch(t *testing.T) {
	secs, _ := Mock().Settings()
	var got []string
	for _, st := range Search(secs, "  PALETTE ") {
		got = append(got, st.Name)
	}
	if want := []string{"palette.position", "palette.style.scale"}; !slices.Equal(got, want) {
		t.Errorf("search palette: %v, want %v", got, want)
	}
	got = got[:0]
	for _, st := range Search(secs, "blocked") {
		got = append(got, st.Name)
	}
	if want := []string{"agents.notify"}; !slices.Equal(got, want) {
		t.Errorf("search by label: %v, want %v", got, want)
	}
	got = got[:0]
	for _, st := range Search(secs, "double click") {
		got = append(got, st.Name)
	}
	if want := []string{"pane.word"}; !slices.Equal(got, want) {
		t.Errorf("search by description: %v, want %v", got, want)
	}
	if Search(secs, "  ") != nil {
		t.Error("an empty query found settings")
	}
}

func TestControlNames(t *testing.T) {
	var got []string
	for c := Caption; c <= Text; c++ {
		got = append(got, c.String())
	}
	if want := []string{"caption", "switch", "cycle", "select", "number", "text"}; !slices.Equal(got, want) {
		t.Errorf("%v", got)
	}
	if Control(9).String() != "control(9)" {
		t.Error(Control(9).String())
	}
}
