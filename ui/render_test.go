package ui_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	x "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/jinja"
	"neuroplast.io/hotty-surf/ui"
)

// The render tests of ui/coverage.tsv: a Jinja component's markup, as ui
// renders it for its Props, against Basecoat's macro for the arguments the
// Props stand for, translated to templ by machine (internal/jinja, the
// oracle). Both are parsed and printed canonically; what ui writes on
// purpose that the macro doesn't (the attributes Basecoat's script sets
// once it has started, the runtime's handlers) is taken out first, each
// with its reason in the case.

// A renderCase is one component's Props and the macro call they stand for.
type renderCase struct {
	name   string
	ui     templ.Component
	macro  string
	args   map[string]any
	caller string // the macro's caller (children), as markup
	// drop are attributes left out of both renders, written by one of
	// them on purpose, each with its reason in the case's comment:
	// tag@attr, tag#id@attr, or *@attr for any element; tag@class~name
	// leaves out one class.
	drop []string
}

var (
	runtimeID = regexp.MustCompile(`^_[0-9a-z]+$`)
	boolAttrs = map[string]bool{"disabled": true, "hidden": true, "open": true, "checked": true, "selected": true}
)

func runRender(t *testing.T, component string, cases []renderCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(component+"/"+c.name, func(t *testing.T) {
			got := canonical(t, uiMarkup(t, c.ui), c.drop)
			var caller templ.Component
			ctx := context.Background()
			if c.caller != "" {
				caller = templ.Raw(c.caller)
				ctx = templ.WithChildren(ctx, caller)
			}
			m := jinja.Call(c.macro, c.args, caller)
			if m == nil {
				t.Fatalf("the oracle has no macro %s", c.macro)
			}
			var b strings.Builder
			if err := m.Render(ctx, &b); err != nil {
				t.Fatal(err)
			}
			want := canonical(t, b.String(), c.drop)
			if got != want {
				t.Errorf("ui's markup differs from the macro's:\n%s", lineDiff(want, got))
			}
		})
	}
}

// uiMarkup is c rendered on a surface: the markup under #root.
func uiMarkup(t *testing.T, c templ.Component) string {
	t.Helper()
	doc, err := surf.New("render", c, "").Document()
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(doc, `<div id="root">`)
	if i < 0 {
		t.Fatalf("no #root in %s", doc)
	}
	inner := doc[i+len(`<div id="root">`):]
	return strings.TrimSuffix(inner, "</div>")
}

// canonical is s parsed as HTML and printed one tag or text per line,
// attributes sorted, whitespace collapsed, and the attributes of drop
// left out.
func canonical(t *testing.T, s string, drop []string) string {
	t.Helper()
	ns, err := x.ParseFragment(strings.NewReader(s), &x.Node{Type: x.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		t.Fatal(err)
	}
	dropped := func(n *x.Node, key string) bool {
		id := ""
		for _, a := range n.Attr {
			if a.Key == "id" {
				id = a.Val
			}
		}
		return slices.Contains(drop, "*@"+key) || id != "" && slices.Contains(drop, n.Data+"#"+id+"@"+key) ||
			slices.Contains(drop, n.Data+"@"+key)
	}
	var b strings.Builder
	var walk func(n *x.Node, depth int)
	walk = func(n *x.Node, depth int) {
		for _, a := range n.Attr {
			if a.Key == "data-hotty-modal-guard" {
				return // the runtime's focus guard, not the macro's markup
			}
		}
		pad := strings.Repeat("  ", depth)
		switch n.Type {
		case x.TextNode:
			if s := strings.Join(strings.Fields(n.Data), " "); s != "" {
				b.WriteString(pad + s + "\n")
			}
		case x.ElementNode:
			var attrs []string
			for _, a := range n.Attr {
				// The runtime's own: the ids it numbers elements with
				// and the events it asks the host for (ARCHITECTURE.md).
				if dropped(n, a.Key) || a.Key == "data-on" || a.Key == "data-keys" || a.Key == "id" && runtimeID.MatchString(a.Val) {
					continue
				}
				v := strings.Join(strings.Fields(a.Val), " ")
				if boolAttrs[a.Key] {
					v = "" // Jinja writes True for a boolean attribute
				}
				if a.Key == "class" {
					cs := slices.DeleteFunc(strings.Fields(a.Val), func(c string) bool {
						return dropped(n, "class~"+c)
					})
					slices.Sort(cs)
					v = strings.Join(cs, " ")
					if v == "" {
						continue
					}
				}
				attrs = append(attrs, a.Key+`="`+v+`"`)
			}
			slices.Sort(attrs)
			b.WriteString(pad + "<" + n.Data)
			for _, a := range attrs {
				b.WriteString(" " + a)
			}
			b.WriteString(">\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, depth+1)
			}
		}
	}
	for _, n := range ns {
		walk(n, 0)
	}
	return b.String()
}

// lineDiff shows two canonical renders line by line from the first line
// they differ at.
func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	i := 0
	for i < len(w) && i < len(g) && w[i] == g[i] {
		i++
	}
	var b strings.Builder
	for j := max(0, i-3); j < max(len(w), len(g)) && j < i+12; j++ {
		wl, gl := "", ""
		if j < len(w) {
			wl = w[j]
		}
		if j < len(g) {
			gl = g[j]
		}
		mark := " "
		if wl != gl {
			mark = "≠"
		}
		fmt.Fprintf(&b, "%s macro: %s\n%s ui:    %s\n", mark, wl, mark, gl)
	}
	return b.String()
}

func TestRenderSelect(t *testing.T) {
	var st ui.SelectState
	runRender(t, "select", []renderCase{{
		name: "default",
		ui: ui.Select(&st, ui.SelectProps{ID: "size", Value: "m", Items: []ui.SelectItem{
			{Value: "s", Label: "Small"}, {Value: "m", Label: "Medium"},
		}}),
		macro: "select",
		// select.js marks the select it started.
		drop: []string{"div#size@data-select-initialized"},
		args: map[string]any{"id": "size", "selected": "m", "items": []any{
			map[string]any{"value": "s", "label": "Small"}, map[string]any{"value": "m", "label": "Medium"},
		}},
	}})
}

func TestRenderSelectGrouped(t *testing.T) {
	var st ui.SelectState
	runRender(t, "select", []renderCase{{
		name: "grouped",
		ui: ui.Select(&st, ui.SelectProps{ID: "fruit", Placeholder: "Pick one", Items: []ui.SelectItem{
			{Value: "apple", Label: "Apple", Group: "Fruits"}, {Value: "banana", Label: "Banana", Group: "Fruits"},
			{Value: "leek", Label: "Leek", Group: "Vegetables", Disabled: true}, {Value: "none", Label: "None"},
		}}),
		macro: "select",
		// select.js marks the select it started, and dims the
		// placeholder it shows.
		drop: []string{"div#fruit@data-select-initialized", "span@class~text-muted-foreground"},
		args: map[string]any{"id": "fruit", "placeholder": "Pick one", "items": []any{
			map[string]any{"type": "group", "label": "Fruits", "items": []any{
				map[string]any{"value": "apple", "label": "Apple"}, map[string]any{"value": "banana", "label": "Banana"}}},
			map[string]any{"type": "separator"},
			map[string]any{"type": "group", "label": "Vegetables", "items": []any{
				map[string]any{"value": "leek", "label": "Leek", "attrs": map[string]any{"aria-disabled": "true"}}}},
			map[string]any{"type": "separator"},
			map[string]any{"value": "none", "label": "None"},
		}},
	}})
}

func TestRenderDropdownMenu(t *testing.T) {
	var st ui.DropdownMenuState
	runRender(t, "dropdown-menu", []renderCase{{
		name: "default",
		ui: ui.DropdownMenu(&st, ui.DropdownMenuProps{ID: "m", Trigger: templ.Raw("Account"), Items: []ui.MenuItem{
			{Label: "Profile"}, {Label: "Billing", Disabled: true},
			{Label: "Delete", Group: "Danger", Variant: ui.Destructive},
		}}),
		macro: "dropdown_menu",
		// dropdown-menu.js marks the menu it started. The trigger is
		// an outline button, as Basecoat's docs write it (trigger_attrs).
		drop: []string{"div#m@data-dropdown-menu-initialized"},
		args: map[string]any{"id": "m", "trigger": "Account",
			"trigger_attrs": map[string]any{"class": "btn", "data-variant": "outline"}, "items": []any{
				map[string]any{"label": "Profile"}, map[string]any{"label": "Billing", "attrs": map[string]any{"aria-disabled": "true"}},
				map[string]any{"type": "separator"},
				map[string]any{"type": "group", "label": "Danger", "items": []any{
					map[string]any{"label": "Delete", "attrs": map[string]any{"data-variant": "destructive"}}}},
			}},
	}})
}

func TestRenderDialog(t *testing.T) {
	runRender(t, "dialog", []renderCase{{
		name: "open",
		ui: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Dialog(ui.DialogProps{ID: "d", Open: true, Title: "Rename", Description: "A new name."}).Render(templ.WithChildren(ctx, templ.Raw("<section><p>Body</p></section>")), w)
		}),
		macro: "dialog",
		// The macro's inline handlers are its script (closing on the
		// backdrop and the close button), which ui does in Go, on the
		// box and the button it names. ui's children go straight into
		// the box, where the macro wraps its caller in a <section>.
		drop:   []string{"dialog@onclick", "dialog@onpointerdown", "button@onclick", "div#d-content@id", "button#d-close@id"},
		args:   map[string]any{"id": "d", "open": true, "title": "Rename", "description": "A new name."},
		caller: "<p>Body</p>",
	}})
}

func TestRenderTabs(t *testing.T) {
	runRender(t, "tabs", []renderCase{{
		name: "default",
		ui: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Tabs(ui.TabsProps{ID: "t", Value: "b", Tabs: []ui.Tab{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}, {Value: "c", Label: "C", Disabled: true}}}).
				Render(templ.WithChildren(ctx, templ.Raw("<p>b</p>")), w)
		}),
		macro: "tabs",
		// tabs.js marks the tabs it started.
		drop: []string{"div#t@data-tabs-initialized"},
		args: map[string]any{"id": "t", "default_tab_index": 2, "tabsets": []any{
			map[string]any{"tab": "A"}, map[string]any{"tab": "B", "panel": "<p>b</p>"},
			map[string]any{"tab": "C", "tab_attrs": map[string]any{"disabled": true}}}},
	}})
}

func TestRenderPopover(t *testing.T) {
	var st ui.PopoverState
	runRender(t, "popover", []renderCase{{
		name: "default",
		ui: templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return ui.Popover(&st, ui.PopoverProps{ID: "p", Trigger: templ.Raw("Open"), Side: "top", Align: "end"}).
				Render(templ.WithChildren(ctx, templ.Raw("<p>Hi</p>")), w)
		}),
		macro:  "popover",
		caller: "<p>Hi</p>",
		// popover.js marks the popovers it started.
		drop: []string{"div#p@data-popover-initialized"},
		args: map[string]any{"id": "p", "trigger": "Open",
			"trigger_attrs": map[string]any{"class": "btn", "data-variant": "outline"},
			"popover_attrs": map[string]any{"data-side": "top", "data-align": "end"}},
	}})
}

// A Command and a CommandDialog, as their macros write them; what
// command.js sets as it starts (each item's aria-hidden, the first
// highlighted, the field's aria-activedescendant) and the field's value,
// which the surface keeps, aside.
func TestRenderCommand(t *testing.T) {
	var st ui.CommandState
	items := []ui.MenuItem{
		{Label: "New note"}, {Label: "Search", Keywords: "find, lookup"}, {Label: "Archive", Disabled: true},
		{Label: "Profile", Group: "Settings"}, {Label: "Inbox", Group: "Settings"},
	}
	argItems := []any{
		map[string]any{"label": "New note"}, map[string]any{"label": "Search", "keywords": "find, lookup"},
		map[string]any{"label": "Archive", "disabled": true}, map[string]any{"type": "separator"},
		map[string]any{"type": "group", "label": "Settings", "items": []any{
			map[string]any{"label": "Profile"}, map[string]any{"label": "Inbox"}}},
	}
	js := []string{"*@aria-hidden", "div@class~active", "input@aria-activedescendant", "input@value", "*@data-command-initialized"}
	runRender(t, "command", []renderCase{{
		name:  "default",
		ui:    ui.Command(&st, ui.CommandProps{ID: "c", Items: items}),
		macro: "command",
		drop:  js,
		args:  map[string]any{"id": "c", "items": argItems},
	}, {
		name:  "words",
		ui:    ui.Command(&st, ui.CommandProps{ID: "c", Items: items[:1], Placeholder: "Find…", Empty: "Nothing.", Class: "max-w-md"}),
		macro: "command",
		drop:  js,
		args: map[string]any{"id": "c", "items": argItems[:1], "placeholder": "Find…", "empty_text": "Nothing.",
			"main_attrs": map[string]any{"class": "max-w-md"}},
	}, {
		name:  "dialog",
		ui:    ui.CommandDialog(&st, ui.CommandDialogProps{ID: "cd", Open: true, Items: items, OnClose: func() {}}),
		macro: "command_dialog",
		// The macro's inline handlers are Go here, and ui names the box
		// for its handlers, as Dialog's.
		drop: append([]string{"dialog@onclick", "dialog@onpointerdown", "div#cd-content@id"}, js...),
		args: map[string]any{"id": "cd", "items": argItems, "open": true},
	}})
}

func TestRenderToast(t *testing.T) {
	var st ui.ToasterState
	st.Show(ui.Toast{Title: "Saved", Description: "All of it."})
	runRender(t, "toast", []renderCase{{
		name:  "default",
		ui:    ui.Toaster(&st, ui.ToasterProps{}),
		macro: "toaster",
		// toast.js marks the toaster and each toast it started; ui
		// names each toast for its handlers.
		drop: []string{"div#toaster@data-toaster-initialized", "*@data-toast-initialized", "div#toast-1@id"},
		args: map[string]any{"toasts": []any{
			map[string]any{"category": "success", "title": "Saved", "description": "All of it."}}},
	}})
}

func init() {
	for _, c := range []string{"select", "dropdown-menu", "dialog", "tabs", "toast", "popover", "command"} {
		renderTests[c] = true
	}
}
