package ui_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/internal/stories"
)

// The kinds of test a component's row may need, besides spec and
// keys-cells (TestSpec and TestKeysCells, over the stories): each test
// file that covers a component adds it here.
var (
	renderTests       = map[string]bool{} // the markup equals the Jinja macro's (the oracle)
	differentialTests = map[string]bool{} // the Go behaviour equals Basecoat's script in Chromium
)

// A row of ui/coverage.tsv.
type coverageRow struct {
	component, markup, behaviour, browser string
	api, stories, tests                   []string
	status, notes                         string
}

func readCoverage(t *testing.T) []coverageRow {
	t.Helper()
	b, err := os.ReadFile("coverage.tsv")
	if err != nil {
		t.Fatal(err)
	}
	list := func(s string) []string {
		if s == "-" || s == "" {
			return nil
		}
		return strings.Split(s, ",")
	}
	var rows []coverageRow
	for i, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 9 {
			t.Fatalf("coverage.tsv:%d: %d columns, want 9", i+1, len(f))
		}
		if i == 0 {
			continue
		}
		rows = append(rows, coverageRow{f[0], f[1], f[2], f[3], list(f[4]), list(f[5]), list(f[6]), f[7], f[8]})
	}
	return rows
}

// approved is every ui name the approved API holds (api/v0.7.txt).
func approved(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile("../api/v0.7.txt")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		rest, ok := strings.CutPrefix(l, "pkg neuroplast.io/hotty-surf/ui, ")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 2 {
			continue
		}
		name := f[1]
		if f[0] == "func" || f[0] == "type" || f[0] == "const" || f[0] == "var" {
			name, _, _ = strings.Cut(name, "(")
			names["ui."+name] = true
		}
	}
	return names
}

// TestCoverage holds ui to ui/coverage.tsv, Basecoat's components one by
// one: a row that is done has its names in the approved API, exactly the
// stories it lists, and each test it lists. A row not done yet is logged
// with what it lacks.
func TestCoverage(t *testing.T) {
	api := approved(t)
	listed := map[string]bool{}
	for _, row := range readCoverage(t) {
		listed[row.component] = true
		var lacks []string
		for _, n := range row.api {
			if !api[n] {
				lacks = append(lacks, "API "+n)
			}
		}
		var have []string
		for _, st := range stories.Of(row.component) {
			have = append(have, st.ID)
		}
		for _, id := range row.stories {
			if !slices.Contains(have, id) {
				lacks = append(lacks, "story "+id)
			}
		}
		for _, id := range have {
			if !slices.Contains(row.stories, id) {
				lacks = append(lacks, "the row for story "+id)
			}
		}
		for _, kind := range row.tests {
			switch kind {
			case "spec":
				if len(have) == 0 {
					lacks = append(lacks, "spec (no story)")
				}
			case "keys-cells":
				for _, st := range stories.Of(row.component) {
					if len(st.Keys) == 0 {
						lacks = append(lacks, "keys-cells for "+st.ID)
					}
				}
			case "render":
				if !renderTests[row.component] {
					lacks = append(lacks, "render")
				}
			case "differential":
				if !differentialTests[row.component] {
					lacks = append(lacks, "differential")
				}
			default:
				t.Errorf("%s: no test kind %q", row.component, kind)
			}
		}
		switch {
		case len(lacks) == 0 && row.status != "done":
			t.Logf("%s: has all it lists; its status can be done", row.component)
		case len(lacks) > 0 && row.status == "done":
			t.Errorf("%s is done but lacks: %s", row.component, strings.Join(lacks, "; "))
		case len(lacks) > 0:
			t.Logf("%s (%s) lacks: %s", row.component, row.status, strings.Join(lacks, "; "))
		}
	}
	for _, st := range stories.All() {
		if !listed[st.Component()] {
			t.Errorf("story %s: no row for %s", st.ID, st.Component())
		}
	}
}
