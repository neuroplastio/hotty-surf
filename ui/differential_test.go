package ui_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/differential"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// The differential tests: each case of ui/testdata/differential/<c>.json,
// driven on ui's Go component in surftest, compared step by step with what
// Basecoat's own script did in Chromium on the macro's markup for the same
// case (<c>.browser.json, which `make differential` records). After the
// load and every step, the root element's markup (as the render tests
// compare it), where the keyboard is, and what the component reported.

// A subject is ui's component for a case, and what it has reported since
// the last call: each report as the keys of the browser's event detail
// that it carries.
type subject struct {
	view   templ.Component
	events func() []map[string]any
}

// subjects make each component's subject from a case's macro arguments.
var subjects = map[string]func(t *testing.T, args map[string]any) subject{
	"select":        selectSubject,
	"dropdown-menu": menuSubject,
	"tabs":          tabsSubject,
	"toast":         toastSubject,
	"dialog":        dialogSubject,
	"popover":       popoverSubject,
	"command":       commandSubject,
	"combobox":      comboboxSubject,
	"drawer":        drawerSubject,
}

func drawerSubject(_ *testing.T, args map[string]any) subject {
	var st ui.DrawerState
	p := ui.DrawerProps{}
	p.ID, _ = args["id"].(string)
	p.Side, _ = args["side"].(string)
	p.OnClose = func() { p.Open = false }
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			open := ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "drawer-open", surf.Click(func() { p.Open = true }))})
			if err := open.Render(templ.WithChildren(ctx, templ.Raw("Open")), w); err != nil {
				return err
			}
			inside := templ.Raw(`<button id="drawer-inside" type="button">Inside</button>`)
			return ui.Drawer(&st, p).Render(templ.WithChildren(ctx, inside), w)
		}),
		events: func() []map[string]any { return nil },
	}
}

func comboboxSubject(_ *testing.T, args map[string]any) subject {
	p := ui.ComboboxProps{Items: []ui.SelectItem{
		{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"}, {Value: "c", Label: "Cherry"},
	}}
	p.ID, _ = args["id"].(string)
	p.Placeholder, _ = args["placeholder"].(string)
	p.Multiple, _ = args["multiple"].(bool)
	var events []map[string]any
	p.OnChange = func(v string) { p.Value = v; events = append(events, map[string]any{"value": v}) }
	p.OnValuesChange = func(v []string) { p.Values = v; events = append(events, map[string]any{"value": v}) }
	var st ui.ComboboxState
	return subject{
		view:   templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return ui.Combobox(&st, p).Render(ctx, w) }),
		events: func() []map[string]any { result := events; events = nil; return result },
	}
}

func init() {
	for c := range subjects {
		differentialTests[c] = true
	}
}

func TestDifferential(t *testing.T) {
	for c, mk := range subjects {
		t.Run(c, func(t *testing.T) {
			sp, err := differential.Load(filepath.Join("testdata/differential", c+".json"))
			if err != nil {
				t.Fatal(err)
			}
			rec, err := differential.LoadBrowser(filepath.Join("testdata/differential", c+".browser.json"))
			if err != nil {
				t.Fatalf("no browser run (%v): make differential", err)
			}
			for _, cs := range sp.Cases {
				t.Run(cs.Name, func(t *testing.T) { runDifferential(t, sp, cs, rec[cs.Name], mk(t, cs.Args)) })
			}
		})
	}
}

func runDifferential(t *testing.T, sp differential.Spec, cs differential.Case, want []differential.Snap, sub subject) {
	if len(want) != len(cs.Steps)+1 {
		t.Fatalf("the recording has %d snapshots for %d steps: make differential", len(want), len(cs.Steps))
	}
	view := templ.Join(templ.Raw(differential.OutsideHost), sub.view)
	s := surf.New("diff", view, ui.Stylesheet())
	r := surftest.Start(t, s)
	// A page has the keyboard from its load, nothing focused; a surface
	// starts without it (SPEC §10.1). A click on the paragraph before the
	// component gives it the keyboard; the load's snapshot counts that as
	// nothing focused, and the first Tab goes to the component's first
	// stop, as the page's does from its body (differential.OutsideHost).
	r.Click("#outside")
	sub.events()
	root := differential.RootSel(cs)
	// pristine is that no step has moved the keyboard yet: a hover
	// doesn't, so the click on #outside still counts as nothing focused.
	pristine := true
	for i, w := range want {
		if i > 0 {
			st := cs.Steps[i-1]
			pristine = pristine && st.Hover != ""
			switch {
			case st.Click != "":
				r.Click("#" + st.Click)
			case st.Select != "":
				// The first match, as the browser's querySelector takes.
				ns := r.Find(st.Select)
				if len(ns) == 0 {
					t.Fatalf("step %d: nothing matches %s", i, st.Select)
				}
				r.Click("#" + attr(ns[0], "id"))
			case st.Hover != "":
				hover(t, s, r, st.Hover)
			case st.Type != "":
				// Into the field that has the keyboard, a key at a
				// time, as surftest's Type types once it has clicked.
				for _, c := range st.Type {
					k := string(c)
					if c == ' ' {
						k = "space"
					}
					r.Key(k)
				}
			default:
				r.Key(st.Key)
			}
		}
		var b bytes.Buffer
		if ns := r.Find(root); len(ns) > 1 {
			t.Fatalf("%q matches %d elements", root, len(ns))
		} else if len(ns) == 1 {
			if err := html.Render(&b, ns[0]); err != nil {
				t.Fatal(err)
			}
		}
		gotHTML := canonical(t, b.String(), sp.Drop)
		wantHTML := canonical(t, shown(t, sp, w.HTML), sp.Drop)
		gotFocus := focusOf(s, sp.Drop)
		if pristine && gotFocus == "outside" {
			gotFocus = ""
		}
		gotEvents := sub.events()

		var aspects, why []string
		if gotHTML != wantHTML {
			aspects = append(aspects, "document")
			why = append(why, "the document:\n"+lineDiff(wantHTML, gotHTML))
		}
		if gotFocus != w.Focus {
			aspects = append(aspects, "focus")
			why = append(why, fmt.Sprintf("focus: browser %q, ui %q", w.Focus, gotFocus))
		}
		if we, ge := eventsOf(t, w.Events, gotEvents), canonEvents(gotEvents); we != ge {
			aspects = append(aspects, "events")
			why = append(why, fmt.Sprintf("events: browser %s, ui %s", we, ge))
		}
		if len(aspects) == 0 {
			continue
		}
		known := ""
		for _, k := range cs.Known {
			if slices.Contains(k.Steps, i) && !slices.ContainsFunc(aspects, func(a string) bool { return !slices.Contains(k.What, a) }) {
				known = k.Why
			}
		}
		msg := fmt.Sprintf("step %d (%s): %s", i, w.Step, strings.Join(why, "\n"))
		if known != "" {
			t.Logf("known: %s: %s", msg, known)
		} else {
			t.Error(msg)
		}
	}
}

// shown is the browser's root, or "" for one the spec's Shown attribute
// says isn't shown.
func shown(t *testing.T, sp differential.Spec, root string) string {
	if sp.Shown == "" {
		return root
	}
	n, err := html.ParseFragment(strings.NewReader(root), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range n {
		if x.Type == html.ElementNode {
			for _, a := range x.Attr {
				if a.Key == sp.Shown {
					return root
				}
			}
			return ""
		}
	}
	return ""
}

// focusOf is where the surface holds the keyboard is, as the browser names
// it: the element's id, or its tag for one whose id is only ui's (the
// runtime's, or one drop leaves out as tag#id@id).
func focusOf(s *surf.Surface, drop []string) string {
	id := rt.Peek.Focused(s)
	if !runtimeID.MatchString(id) && !slices.ContainsFunc(drop, func(d string) bool { return strings.HasSuffix(d, "#"+id+"@id") }) {
		return id
	}
	var tag string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				tag = strings.ToUpper(n.Data)
			}
		}
		for c := n.FirstChild; c != nil && tag == ""; c = c.NextSibling {
			walk(c)
		}
	}
	walk(rt.Peek.Tree(s))
	return tag
}

// eventsOf is the browser's events in the form ui's are compared in: each
// detail cut to the keys ui's reports carry.
func eventsOf(t *testing.T, raw []json.RawMessage, like []map[string]any) string {
	t.Helper()
	var keys []string
	for _, m := range like {
		for k := range m {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	var out []map[string]any
	for _, r := range raw {
		var e struct {
			Detail map[string]any `json:"detail"`
		}
		if err := json.Unmarshal(r, &e); err != nil {
			t.Fatal(err)
		}
		m := map[string]any{}
		for k, v := range e.Detail {
			if len(keys) == 0 || slices.Contains(keys, k) {
				m[k] = v
			}
		}
		out = append(out, m)
	}
	return canonEvents(out)
}

func canonEvents(es []map[string]any) string {
	if len(es) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(es) // a map's keys are sorted
	return string(b)
}

// selectSubject is a Select for select's macro arguments: id, items
// (groups, separators, options with aria-disabled), placeholder and
// selected.
func selectSubject(t *testing.T, args map[string]any) subject {
	p := ui.SelectProps{}
	p.ID, _ = args["id"].(string)
	p.Placeholder, _ = args["placeholder"].(string)
	p.Value, _ = args["selected"].(string)
	var walk func(items []any, group string)
	walk = func(items []any, group string) {
		for _, x := range items {
			it := x.(map[string]any)
			switch it["type"] {
			case "separator":
			case "group":
				walk(it["items"].([]any), it["label"].(string))
			default:
				v, _ := it["value"].(string)
				l, _ := it["label"].(string)
				attrs, _ := it["attrs"].(map[string]any)
				p.Items = append(p.Items, ui.SelectItem{Value: v, Label: l, Group: group, Disabled: attrs["aria-disabled"] == "true"})
			}
		}
	}
	walk(args["items"].([]any), "")
	// With no selected and no placeholder the macro chooses its first
	// option (its default_option); ui's Value "" is none, so the app says
	// it.
	if _, ok := args["selected"]; !ok && p.Placeholder == "" && len(p.Items) > 0 {
		p.Value = p.Items[0].Value
	}
	var heard []map[string]any
	p.OnChange = func(v string) {
		p.Value = v
		heard = append(heard, map[string]any{"value": v})
	}
	var st ui.SelectState
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Select(&st, p).Render(ctx, w)
		}),
		events: func() []map[string]any { h := heard; heard = nil; return h },
	}
}

// menuSubject is a DropdownMenu for dropdown_menu's macro arguments: id,
// trigger (text), and items (groups, separators, items with aria-disabled
// or data-variant in attrs). An item's OnSelect reports it by the id the
// macro gives it: <id>-items-<n>, or -<n>-<m> in a group, every entry
// counted.
func menuSubject(t *testing.T, args map[string]any) subject {
	p := ui.DropdownMenuProps{}
	p.ID, _ = args["id"].(string)
	if tr, ok := args["trigger"].(string); ok {
		p.Trigger = templ.Raw(tr)
	}
	var heard []map[string]any
	add := func(it map[string]any, group, id string) {
		l, _ := it["label"].(string)
		attrs, _ := it["attrs"].(map[string]any)
		mi := ui.MenuItem{Label: l, Group: group, Disabled: attrs["aria-disabled"] == "true",
			OnSelect: func() { heard = append(heard, map[string]any{"item": id}) }}
		if attrs["data-variant"] == "destructive" {
			mi.Variant = ui.Destructive
		}
		p.Items = append(p.Items, mi)
	}
	for n, x := range args["items"].([]any) {
		it := x.(map[string]any)
		id := fmt.Sprintf("%s-items-%d", p.ID, n+1)
		switch it["type"] {
		case "separator":
		case "group":
			for m, y := range it["items"].([]any) {
				add(y.(map[string]any), it["label"].(string), fmt.Sprintf("%s-%d", id, m+1))
			}
		default:
			add(it, "", id)
		}
	}
	var st ui.DropdownMenuState
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.DropdownMenu(&st, p).Render(ctx, w)
		}),
		events: func() []map[string]any { h := heard; heard = nil; return h },
	}
}

// commandSubject is a Command for command's macro arguments: id, and
// items (label, keywords, disabled; separators; groups of them). Its items
// report their clicks, as the page's do.
func commandSubject(t *testing.T, args map[string]any) subject {
	p := ui.CommandProps{}
	p.ID, _ = args["id"].(string)
	var heard []map[string]any
	add := func(it map[string]any, group, id string) {
		l, _ := it["label"].(string)
		k, _ := it["keywords"].(string)
		p.Items = append(p.Items, ui.MenuItem{Label: l, Keywords: k, Group: group, Disabled: it["disabled"] == true,
			OnSelect: func() { heard = append(heard, map[string]any{"item": id}) }})
	}
	for n, x := range args["items"].([]any) {
		it := x.(map[string]any)
		id := fmt.Sprintf("%s-items-%d", p.ID, n+1)
		switch it["type"] {
		case "separator":
		case "group":
			for m, y := range it["items"].([]any) {
				add(y.(map[string]any), it["label"].(string), fmt.Sprintf("%s-%d", id, m+1))
			}
		default:
			add(it, "", id)
		}
	}
	var st ui.CommandState
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Command(&st, p).Render(ctx, w)
		}),
		events: func() []map[string]any { h := heard; heard = nil; return h },
	}
}

// tabsSubject is a Tabs for tabs' macro arguments: id, tabsets (tab text,
// tab_attrs' disabled) and default_tab_index (1-based, 1 by default). Its
// OnChange shows the tab chosen; the macro's tabs.js tells the page
// nothing, so it reports nothing.
func tabsSubject(t *testing.T, args map[string]any) subject {
	p := ui.TabsProps{}
	p.ID, _ = args["id"].(string)
	for i, x := range args["tabsets"].([]any) {
		ts := x.(map[string]any)
		attrs, _ := ts["tab_attrs"].(map[string]any)
		p.Tabs = append(p.Tabs, ui.Tab{Value: fmt.Sprint(i + 1), Label: ts["tab"].(string), Disabled: attrs["disabled"] == true})
	}
	p.Value = "1"
	if n, ok := args["default_tab_index"].(float64); ok {
		p.Value = fmt.Sprint(int(n))
	}
	p.OnChange = func(v string) { p.Value = v }
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Tabs(p).Render(templ.WithChildren(ctx, templ.Raw("<p>"+p.Value+"</p>")), w)
		}),
		events: func() []map[string]any { return nil },
	}
}

// popoverSubject is a Popover for popover's macro arguments: id, trigger,
// and popover_attrs' data-side and data-align. popover.js tells the page
// nothing, so it reports nothing.
func popoverSubject(t *testing.T, args map[string]any) subject {
	var st ui.PopoverState
	p := ui.PopoverProps{}
	p.ID, _ = args["id"].(string)
	trigger, _ := args["trigger"].(string)
	p.Trigger = templ.Raw(trigger)
	if pa, ok := args["popover_attrs"].(map[string]any); ok {
		p.Side, _ = pa["data-side"].(string)
		p.Align, _ = pa["data-align"].(string)
	}
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Popover(&st, p).Render(templ.WithChildren(ctx, templ.Raw("<p>Hi</p>")), w)
		}),
		events: func() []map[string]any { return nil },
	}
}

// toastSubject is a Toaster for toaster's macro arguments: toasts, each
// a category, title, description and duration (-1 to stay open, with a
// cancel button). toast.js tells the page nothing, so it reports nothing.
func toastSubject(t *testing.T, args map[string]any) subject {
	var st ui.ToasterState
	for _, x := range args["toasts"].([]any) {
		m := x.(map[string]any)
		ts := ui.Toast{}
		ts.Category, _ = m["category"].(string)
		ts.Title, _ = m["title"].(string)
		ts.Description, _ = m["description"].(string)
		if d, ok := m["duration"].(float64); ok && d < 0 {
			ts.Duration = -1
		}
		st.Show(ts)
	}
	return subject{
		view:   templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return ui.Toaster(&st).Render(ctx, w) }),
		events: func() []map[string]any { return nil },
	}
}

// dialogSubject is a Dialog for dialog's macro arguments (id, title,
// description, the caller as its body), and the macro's trigger as a
// ui.Button that opens it, before it as the macro writes it. The dialog
// tells the page nothing, so it reports nothing.
func dialogSubject(t *testing.T, args map[string]any) subject {
	p := ui.DialogProps{}
	p.ID, _ = args["id"].(string)
	p.Title, _ = args["title"].(string)
	p.Description, _ = args["description"].(string)
	p.OnClose = func() { p.Open = false }
	trigger, _ := args["trigger"].(string)
	return subject{
		view: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			open := ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, p.ID+"-trigger", surf.Click(func() { p.Open = true }))})
			if err := open.Render(templ.WithChildren(ctx, templ.Raw(trigger)), w); err != nil {
				return err
			}
			return ui.Dialog(p).Render(templ.WithChildren(ctx, templ.Raw(dialogBody)), w)
		}),
		events: func() []map[string]any { return nil },
	}
}

// dialogBody is the dialog cases' caller, in the <section> the macro
// wraps it in: ui's children go straight into the box.
const dialogBody = `<section><input id="d-name" class="input"><button id="d-save" type="button" class="btn">Save</button></section>`
