package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "write the cells' goldens")

// sgr matches an SGR sequence, which the goldens leave out: they hold the
// layout, and TestCellsColours the colours.
var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// The greeting in cells, at 80 and 60 columns, as a terminal that is no
// host shows it: aligned, nothing cut at either width.
func TestCells(t *testing.T) {
	for _, cols := range []string{"80", "60"} {
		t.Run(cols, func(t *testing.T) {
			n := map[string]int{"80": 80, "60": 60}[cols]
			lines := cellsLines(fixture(), "neuroplastio", n)
			got := sgr.ReplaceAllString(strings.Join(lines, "\n"), "") + "\n"
			path := filepath.Join("testdata", "cells-"+cols+".txt")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("cells at %s:\n%s\nwant:\n%s", cols, got, want)
			}
			if strings.Contains(got, "…") {
				t.Errorf("cells at %s cut something:\n%s", cols, got)
			}
		})
	}
}

// The cells colour by the theme's roles in the terminal's sixteen: the
// mark and the name in the accent (magenta), the facts muted (dim), the
// shares as the accent's fill (reverse video).
func TestCellsColours(t *testing.T) {
	all := strings.Join(cellsLines(fixture(), "neuroplastio", 80), "\n")
	for _, want := range []string{
		"\x1b[0;35m󰬕",                // the mark's glyph, in the accent
		"\x1b[0;35;1mada@devbox",     // who, semibold in the accent
		"\x1b[0;2m · 6.12.4-arch1-1", // the facts, muted
		"\x1b[0;35;7m 29% ",          // a share, on the accent
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in\n%q", want, all)
		}
	}
}

func TestWords(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{duration(7 * 60), "7m"},
		{duration(3*3600 + 12*60), "3h 12m"},
		{duration(2*86400 + 4*3600 + 59), "2d 4h"},
		{amount(9_663_676_416, 33_285_996_544), "9.0/31.0 GiB"},
		{amount(412_316_860_416, 1_000_204_886_016), "384/932 GiB"},
		{amount(412_316_860_416, 2_000_000_000_000), "384 GiB/1.8 TiB"},
	} {
		if c.got != c.want {
			t.Errorf("%q, want %q", c.got, c.want)
		}
	}
	for _, c := range []struct {
		used, total uint64
		tone        string
	}{{1, 4, "accent"}, {80, 100, "warn"}, {95, 100, "error"}} {
		if g := gauge("x", c.used, c.total); g.Tone != c.tone {
			t.Errorf("%d of %d: %s, want %s", c.used, c.total, g.Tone, c.tone)
		}
	}
	if g := gauge("x", 0, 0); g.Used != "" {
		t.Errorf("an unknown total: %q, want nothing", g.Used)
	}
}

func TestProbeFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	osr := write("os-release", "NAME=\"Arch Linux\"\nPRETTY_NAME=\"Arch Linux (rolling)\"\nID=arch\n")
	if got := osName(osr); got != "Arch Linux (rolling)" {
		t.Errorf("os-release: %q", got)
	}
	osr = write("os-release-2", "NAME='Some OS'\n")
	if got := osName(osr); got != "Some OS" {
		t.Errorf("os-release without PRETTY_NAME: %q", got)
	}
	mi := write("meminfo", "MemTotal:       32505856 kB\nMemFree:         1000000 kB\nMemAvailable:   22068224 kB\n")
	used, total := memory(mi)
	if total != 32505856*1024 || used != (32505856-22068224)*1024 {
		t.Errorf("meminfo: %d of %d", used, total)
	}
	db := filepath.Join(dir, "local")
	for _, p := range []string{"a-1.0-1", "b-2.0-1", "c-3.0-1"} {
		if err := os.MkdirAll(filepath.Join(db, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("local/ALPM_DB_VERSION", "9\n")
	if got := pacman(db); got != 3 {
		t.Errorf("packages: %d, want 3", got)
	}
	if got := pacman(filepath.Join(dir, "none")); got != 0 {
		t.Errorf("no pacman: %d, want 0", got)
	}
}

// With its input or output not a terminal (a pipe, a file), it prints
// nothing and asks the terminal nothing.
func TestQuietWithoutTerminal(t *testing.T) {
	if os.Getenv("HOTTYFETCH_MAIN") == "1" {
		os.Args = []string{"hottyfetch", "-fixture"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestQuietWithoutTerminal$")
	cmd.Env = append(os.Environ(), "HOTTYFETCH_MAIN=1")
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if s := strings.TrimSuffix(strings.TrimSpace(string(out)), "PASS"); strings.TrimSpace(s) != "" {
		t.Errorf("printed %q", s)
	}
}
