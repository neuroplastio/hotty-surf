package main

import (
	"slices"
	"testing"

	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
)

const (
	notes  = "r.0.kf2" // notes.md
	status = "r.0.kstatus"
)

// A right click on a row opens its menu, the arrows and Enter pick, and
// the program's own code is the status line.
func TestFilesMenuOnAHost(t *testing.T) {
	a := surftest.Start(t, surflive.New(&files{}), surfrun.Options{Name: "files"})
	if err := a.Host.SecondaryPress(a.Name, notes, 10, 3); err != nil {
		t.Fatal(err)
	}
	a.Wait("the menu", func() bool { m := a.Menu(); return m != nil && m.Placed() })
	a.Host.Key("arrow_down")
	a.Host.Key("arrow_down")
	a.Host.Key("enter")
	a.Wait("the status", func() bool { return a.Surface().TextOf(status) == "Rename: notes.md" })
	a.Wait("the menu gone", func() bool { return a.Menu() == nil })
}

// Delete is disabled on LICENSE only.
func TestFilesDisabled(t *testing.T) {
	f := &files{}
	for i, n := range names {
		var off []string
		for _, it := range f.menu(n) {
			if it.Disabled {
				off = append(off, it.Label)
			}
		}
		if want := n == "LICENSE"; want != slices.Equal(off, []string{"Delete"}) || !want && len(off) != 0 {
			t.Errorf("row %d %s: disabled %q", i, n, off)
		}
	}
}
