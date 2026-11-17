// Command record writes the pages of ui's differential cases and the
// manifest cdp.mjs drives: for each spec in <specs dir> (a
// <component>.json), <out>/<component>/<case>.html and
// <out>/<component>.manifest.json. scripts/differential.sh runs it, then
// Chromium.
//
//	go run ./internal/differential/record <specs dir> <out dir>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"neuroplast.io/hotty-surf/internal/differential"
)

type manifestCase struct {
	Name  string              `json:"name"`
	Root  string              `json:"root"`
	Steps []differential.Step `json:"steps"`
}

type manifest struct {
	Ready string         `json:"ready"`
	Cases []manifestCase `json:"cases"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: record <specs dir> <out dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "record:", err)
		os.Exit(1)
	}
}

func run(specs, out string) error {
	css, err := os.ReadFile("ui/kit.css")
	if err != nil {
		return fmt.Errorf("run from the repository's root: %w", err)
	}
	js, err := filepath.Abs("internal/basecoat/js")
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(specs, "*.json"))
	if err != nil {
		return err
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".browser.json") {
			continue
		}
		sp, err := differential.Load(f)
		if err != nil {
			return err
		}
		dir := filepath.Join(out, sp.Component)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		m := manifest{Ready: sp.Ready}
		for _, c := range sp.Cases {
			page, err := differential.Page(sp, c, string(css), js)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", sp.Component, c.Name, err)
			}
			if err := os.WriteFile(filepath.Join(dir, c.Name+".html"), []byte(page), 0o644); err != nil {
				return err
			}
			m.Cases = append(m.Cases, manifestCase{c.Name, differential.RootSel(c), c.Steps})
		}
		b, err := json.MarshalIndent(m, "", " ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, sp.Component+".manifest.json"), b, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s: %d cases\n", sp.Component, len(sp.Cases))
	}
	return nil
}
