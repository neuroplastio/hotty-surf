// Package differential is the browser half of ui's differential tests:
// Basecoat's own script driven in Chromium, step by step, on the macro's
// markup, recorded so that ui's tests compare its Go behaviour with it
// without a browser.
//
// A component's cases are ui/testdata/differential/<component>.json (a
// Spec). The recorder (cmd/record, `make differential`) writes each case's
// page (the oracle's markup for its arguments, the kit's stylesheet,
// basecoat.js and the component's script), drives it in a headless
// Chromium over the DevTools protocol (cdp.mjs: real mouse and key input),
// and writes what it saw after the load and after every step to
// <component>.browser.json. On a Basecoat bump, record again.
package differential

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf/internal/jinja"
)

// A Spec is a component's cases, as its JSON file has them.
type Spec struct {
	// Component is Basecoat's file name: select, dropdown-menu.
	Component string `json:"component"`
	// Macro is the Jinja macro the pages render (select,
	// dropdown_menu); "" for a component with no macro, whose cases give
	// their markup.
	Macro string `json:"macro"`
	// Scripts are the files of internal/basecoat/js the page loads after
	// basecoat.js: the component's own.
	Scripts []string `json:"scripts"`
	// Events are the events whose detail the page records: what the
	// component tells the page (select's change).
	Events []string `json:"events"`
	// EventsOn, when set, keeps only the events on what matches it, each
	// recorded as {"item": id} of that element unless it is a CustomEvent
	// with a detail of its own: a menu's chosen item, whose click is all
	// the page hears of it.
	EventsOn string `json:"eventsOn,omitempty"`
	// Ready is the selector of what the script marks once it started
	// (select.js's [data-select-initialized]): the page waits for it
	// before the first snapshot. "" for none.
	Ready string `json:"ready,omitempty"`
	// Drop are attributes left out of both sides at every step, as ui's
	// render tests drop them (tag@attr, tag#id@attr, *@attr,
	// tag@class~name), each with its reason in Why.
	Drop  []string `json:"drop,omitempty"`
	Why   string   `json:"why,omitempty"`
	Cases []Case   `json:"cases"`
}

// A Case is one sequence of steps on one render.
type Case struct {
	Name string `json:"name"`
	// Args are the macro's arguments; Markup, for a component with no
	// macro, is the page's markup instead.
	Args   map[string]any `json:"args"`
	Caller string         `json:"caller,omitempty"`
	Markup string         `json:"markup,omitempty"`
	// Root is the selector of the element compared after each step: by
	// default #<args.id>.
	Root  string `json:"root,omitempty"`
	Steps []Step `json:"steps"`
	// Known are differences explained (a hotty limit, a bug in the
	// script): at these steps, these aspects (document, focus, events).
	Known []Known `json:"known,omitempty"`
}

// A Step is one act of the user's: a click on an element (by id) or a
// key (hotty's name).
type Step struct {
	Click string `json:"click,omitempty"`
	Key   string `json:"key,omitempty"`
}

// Name is the step as the snapshots name it.
func Name(s Step) string {
	if s.Click != "" {
		return "click " + s.Click
	}
	return "key " + s.Key
}

// A Known difference.
type Known struct {
	Steps []int    `json:"steps"`
	What  []string `json:"what"`
	Why   string   `json:"why"`
}

// A Snap is what the browser had after a step (the first, after the load).
type Snap struct {
	Step   string            `json:"step"`
	Note   string            `json:"note,omitempty"`
	HTML   string            `json:"html"`
	Focus  string            `json:"focus"`
	Events []json.RawMessage `json:"events"`
}

// RootSel is the case's root selector.
func RootSel(c Case) string {
	if c.Root != "" {
		return c.Root
	}
	id, _ := c.Args["id"].(string)
	return "#" + id
}

// Load reads a spec.
func Load(path string) (Spec, error) {
	var s Spec
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// LoadBrowser reads a spec's recording: each case's snapshots by name.
func LoadBrowser(path string) (map[string][]Snap, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string][]Snap
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return out, nil
}

// Outside is what a case clicks outside of the component on, before it
// in the page: focusable by a click and skipped by Tab (tabindex -1), so
// the page's first Tab goes to the component's first stop from its body.
// OutsideHost is the same for ui's surface, on a hotty host, which focuses
// on a click only what a Tab reaches: there it is the first stop, which
// ui's test clicks to give the surface the keyboard, so its first Tab goes
// to the same element. (A Shift+Tab off the component's first stop is the
// one step where the two differ: no case has one.)
const (
	Outside     = `<p id="outside" tabindex="-1">Outside</p>`
	OutsideHost = `<p id="outside" tabindex="0">Outside</p>`
)

// Markup is the case's markup: the oracle's render of the macro, or the
// case's own.
func Markup(s Spec, c Case) (string, error) {
	if s.Macro == "" {
		return c.Markup, nil
	}
	var caller templ.Component
	ctx := context.Background()
	if c.Caller != "" {
		caller = templ.Raw(c.Caller)
		ctx = templ.WithChildren(ctx, caller)
	}
	m := jinja.Call(s.Macro, c.Args, caller)
	if m == nil {
		return "", fmt.Errorf("the oracle has no macro %s", s.Macro)
	}
	var b bytes.Buffer
	if err := m.Render(ctx, &b); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Page is the case's page: the stylesheet, Basecoat's scripts from jsDir,
// a recorder of the spec's events, and the markup, then Outside.
func Page(s Spec, c Case, css, jsDir string) (string, error) {
	markup, err := Markup(s, c)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">\n<style>")
	b.WriteString(css)
	b.WriteString("</style>\n")
	for _, f := range append([]string{"basecoat.js"}, s.Scripts...) {
		fmt.Fprintf(&b, "<script src=\"file://%s\"></script>\n", html.EscapeString(filepath.Join(jsDir, f)))
	}
	ev, _ := json.Marshal(s.Events)
	on, _ := json.Marshal(s.EventsOn)
	fmt.Fprintf(&b, `<script>
window.__events = [];
const on = %s;
for (const name of %s) document.addEventListener(name, (e) => {
  const t = on ? e.target.closest(on) : null;
  if (on && !t) return;
  const d = e instanceof CustomEvent ? e.detail : (t ? {item: t.id} : null);
  if (d !== undefined && d !== null) window.__events.push({type: e.type, detail: d});
}, true);
</script>
</head><body style="padding:16px">
`, on, ev)
	b.WriteString(Outside + "\n")
	b.WriteString(markup)
	b.WriteString("\n</body></html>\n")
	return b.String(), nil
}
