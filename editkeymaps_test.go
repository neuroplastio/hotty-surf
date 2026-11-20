package surf

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/rt"
)

var (
	ctrlHost  = hotty.Caps{}
	superHost = hotty.Caps{Mod: "super"}
)

// bindPageOn is bindPage on a host of caps.
func bindPageOn(t *testing.T, caps hotty.Caps, markup string, ons func(ctx context.Context) []templ.Attributes) *Surface {
	t.Helper()
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var args []any
		for _, a := range ons(ctx) {
			args = append(args, attrsOf(a))
		}
		fmt.Fprintf(w, markup, args...)
		return nil
	}), "")
	s.Attach(io.Discard, caps)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	return s
}

// The tables are SPEC §10.2's, as hotty.Resolve has them: CUAKeys' six keys
// and CocoaKeys' fourteen, each giving back the other's, made once a process
// and read once a frame.
func TestEditingKeymapTables(t *testing.T) {
	tabs := editingTables()
	if tabs != editingTables() {
		t.Error("the tables were made twice")
	}
	for i, want := range [2]int{6, 14} {
		own, back := 0, 0
		for _, b := range tabs[i] {
			if b.Action == string(hotty.Program) {
				back++
			} else {
				own++
			}
			if !b.Base {
				t.Errorf("%v isn't marked", b)
			}
		}
		if own != want || back != 20-want {
			t.Errorf("table %d: %d keys of its own and %d given back, want %d and %d", i, own, back, want, 20-want)
		}
	}
	f := &rt.Frame{Platform: hotty.PlatformOf(superHost)}
	a, b := cuaBinds(f), cuaBinds(f)
	if len(a) != 20 || &a[0] != &b[0] {
		t.Errorf("CUAKeys on a super host: %d bindings, read once a frame: %v", len(a), &a[0] == &b[0])
	}
	if c := cuaBinds(&rt.Frame{}); len(c) != 19 {
		t.Errorf("CUAKeys on the zero platform: %d bindings, want 19 (ctrl+a is select all's)", len(c))
	}
}

// On every host, a field under CUAKeys or CocoaKeys resolves each of the tables'
// keys as that keymap's platform does with no data-keys, but a key the
// host's edit shortcuts hold, which stays the host's; and the surface's
// own use of a key (EditKeys, which cells' fallback reads) is the wire's.
func TestEditingKeymapsResolve(t *testing.T) {
	none := map[string][]string{}
	for _, caps := range []hotty.Caps{ctrlHost, superHost, {Edit: map[string][]string{"select-all": {"ctrl+shift+a"}}}} {
		host := hotty.PlatformOf(caps)
		edit := host.Edit
		if edit == nil {
			edit = hotty.DefaultEdit(host.Mod)
		}
		held := func(k string) bool {
			for _, keys := range edit {
				for _, e := range keys {
					if c, _ := host.ReadKey(e); c == k {
						return true
					}
				}
			}
			return false
		}
		for i, km := range []struct {
			name string
			opt  BindOption
			mod  string
		}{{"CUAKeys", CUAKeys(), "ctrl"}, {"CocoaKeys", CocoaKeys(), "super"}} {
			t.Run(fmt.Sprintf("%s on %q %v", km.name, host.Mod, caps.Edit != nil), func(t *testing.T) {
				s := bindPageOn(t, caps, `<div%s><textarea id="f"></textarea></div>`, func(ctx context.Context) []templ.Attributes {
					return []templ.Attributes{On(ctx, "app", Bind(km.opt))}
				})
				if len(s.faults) > 0 {
					t.Fatalf("faults %q", s.faults)
				}
				wire := attr(s.byID["app"], "data-keys")
				own := strings.Fields(rt.EditKeys(s.frame.On["app"], true))
				if got := strings.Fields(wire); !slices.Equal(got, slices.Sorted(slices.Values(own))) {
					t.Errorf("the wire %q, the surface's own %q", got, own)
				}
				on := hotty.Resolve(host, true, wire)
				theirs := hotty.Resolve(hotty.Platform{Mod: km.mod, Edit: none}, true)
				base := hotty.Resolve(host, true)
				for _, b := range editingTables()[i] {
					want := theirs.Lookup(b.Key)
					if held(b.Key) {
						want = base.Lookup(b.Key)
					}
					if got := on.Lookup(b.Key); got != want {
						t.Errorf("%s: %q, want %q", b.Key, got, want)
					}
				}
			})
		}
	}
}

// How CUAKeys and CocoaKeys meet the other bindings: a nearer ReadlineKeys keeps its
// keys (a give-back isn't a key given to the program); a page's key
// stays the page's; on one element the last group read wins; and an
// Unbind gives a key back to the editing keymap in effect, on the element
// or around it, not to the host's platform.
func TestEditingKeymapsCompose(t *testing.T) {
	take := func(string) bool { return true }
	cases := []struct {
		name      string
		caps      hotty.Caps
		app, main []Handler
		field     []BindOption
		has, not  []string // in the field's data-keys
	}{
		{"a nearer readline keeps ctrl+k", superHost, []Handler{Bind(CUAKeys())}, nil, []BindOption{ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+backspace=delete-word-backward"}, nil},
		{"a page's key stays the page's", superHost, nil, []Handler{Key(take, "ctrl+k", "ctrl+backspace")}, []BindOption{CUAKeys()},
			[]string{"ctrl+delete=delete-word-forward"}, []string{"ctrl+k=program", "ctrl+backspace=delete-word-backward"}},
		{"override takes a page's key", superHost, nil, []Handler{Key(take, "ctrl+backspace")}, []BindOption{CUAKeys(), Override()},
			[]string{"ctrl+backspace=delete-word-backward"}, nil},
		{"readline, then CUAKeys", superHost, nil, nil, []BindOption{ReadlineKeys(), CUAKeys()},
			[]string{"ctrl+k=program", "ctrl+w=delete-word-backward"}, []string{"ctrl+k=delete-to-line-end"}},
		{"CUAKeys, then readline", superHost, nil, nil, []BindOption{CUAKeys(), ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "super+arrow_left=program"}, nil},
		{"unbind on the element", superHost, nil, nil, []BindOption{CUAKeys(), Unbind("ctrl+k")},
			[]string{"ctrl+k=program"}, []string{"ctrl+k=default"}},
		{"unbind before the group", superHost, nil, nil, []BindOption{Unbind("ctrl+k"), CUAKeys()},
			[]string{"ctrl+k=program"}, []string{"ctrl+k=default"}},
		{"readline without ctrl+k, under CUAKeys", superHost, []Handler{Bind(CUAKeys())}, nil, []BindOption{ReadlineKeys(), Unbind("ctrl+k")},
			[]string{"ctrl+k=program", "ctrl+w=delete-word-backward"}, []string{"ctrl+k=default", "ctrl+k=delete-to-line-end"}},
		{"unbind of one of CocoaKeys' own", ctrlHost, []Handler{Bind(CocoaKeys())}, nil, []BindOption{ReadlineKeys(), Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=default"}},
		{"unbind of a key CocoaKeys gives back", ctrlHost, []Handler{Bind(CocoaKeys())}, nil, []BindOption{Unbind("ctrl+backspace")},
			[]string{"ctrl+backspace=program"}, []string{"ctrl+backspace=default"}},
		{"unbind of a key neither holds", superHost, []Handler{Bind(CUAKeys())}, nil, []BindOption{ReadlineKeys(), Unbind("ctrl+w")},
			[]string{"ctrl+w=default"}, nil},
		{"the nearer keymap is the one unbind goes back to", superHost, []Handler{Bind(CUAKeys())}, []Handler{Bind(CocoaKeys())}, []BindOption{ReadlineKeys(), Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=program", "ctrl+k=default"}},
		{"both in one Bind: the last wins", superHost, nil, nil, []BindOption{CUAKeys(), CocoaKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+backspace=program"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := bindPageOn(t, c.caps, `<div%s><div%s><input%s></div></div>`, func(ctx context.Context) []templ.Attributes {
				return []templ.Attributes{On(ctx, "app", c.app...), On(ctx, "main", c.main...), On(ctx, "f", Bind(c.field...))}
			})
			if len(s.faults) > 0 {
				t.Fatalf("faults %q", s.faults)
			}
			got := bindings(s, "f")
			for _, b := range c.has {
				if !slices.Contains(got, b) {
					t.Errorf("data-keys lack %s: %q", b, got)
				}
			}
			for _, b := range c.not {
				if slices.Contains(got, b) {
					t.Errorf("data-keys have %s: %q", b, got)
				}
			}
		})
	}
}

// The edit shortcuts stay the host's: CocoaKeys on a host whose mod is ctrl
// leaves ctrl+a to select all, unless the host's own edit shortcuts put
// select all elsewhere.
func TestEditingKeymapsLeaveEditShortcuts(t *testing.T) {
	for _, c := range []struct {
		caps hotty.Caps
		want bool
	}{{ctrlHost, false}, {hotty.Caps{Edit: map[string][]string{"select-all": {"ctrl+shift+a"}}}, true}} {
		s := bindPageOn(t, c.caps, `<div%s><input id="f"></div>`, func(ctx context.Context) []templ.Attributes {
			return []templ.Attributes{On(ctx, "app", Bind(CocoaKeys()))}
		})
		got := bindings(s, "app")
		if slices.Contains(got, "ctrl+a=line-start") != c.want || slices.ContainsFunc(got, func(b string) bool { return strings.HasPrefix(b, "ctrl+c=") }) {
			t.Errorf("edit %v: %q", c.caps.Edit, got)
		}
	}
}

// Outside a field: on a button directly CUAKeys is one fault, as ReadlineKeys
// is; on what holds a button its give-backs give the button's keys to
// the program (the host's super+arrow_up no longer scrolls from it).
func TestEditingKeymapsOutsideAField(t *testing.T) {
	s := bindPageOn(t, superHost, `<button%s>b</button>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "b", Bind(CUAKeys()))}
	})
	if len(s.faults) != 1 || hasAttr(s.byID["b"], "data-keys") {
		t.Errorf("on a button: faults %q, data-keys %q", s.faults, attr(s.byID["b"], "data-keys"))
	}
	s = bindPageOn(t, superHost, `<div%s><button id="b">b</button></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(CUAKeys()))}
	})
	if len(s.faults) > 0 {
		t.Fatalf("faults %q", s.faults)
	}
	km := hotty.ParseKeymap(hotty.PlatformOf(superHost), attr(s.byID["app"], "data-keys"))
	if !km.Program("super+arrow_up") || km.Scroll("super+arrow_up") != "" {
		t.Errorf("super+arrow_up on the button: program %v", km.Program("super+arrow_up"))
	}
}

// A render whose HTML is the last one's, the editing keymap around a
// field toggled, writes the field's Unbind anew.
func TestEditingKeymapAloneRepaints(t *testing.T) {
	on := false
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var app []Handler
		if on {
			app = append(app, Bind(CUAKeys()))
		}
		fmt.Fprintf(w, `<div%s><input%s></div>`, attrsOf(On(ctx, "app", app...)), attrsOf(On(ctx, "f", Bind(ReadlineKeys(), Unbind("ctrl+k")))))
		return nil
	}), "")
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "f"), "ctrl+k=default") {
		t.Fatalf("before: %q", bindings(s, "f"))
	}
	on = true
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "f"), "ctrl+k=program") {
		t.Errorf("after CUAKeys: %q", bindings(s, "f"))
	}
}

// The editing keymaps layer: in one Bind, read in order, a later one's
// binding of a key replaces an earlier one's, and an earlier one's other
// keys stay. One layering of each pair.
func TestEditingKeymapsLayer(t *testing.T) {
	cases := []struct {
		name     string
		caps     hotty.Caps
		opts     []BindOption
		has, not []string
	}{
		{"CUAKeys, then ReadlineKeys", superHost, []BindOption{CUAKeys(), ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+w=delete-word-backward", "super+arrow_left=program"}, []string{"ctrl+k=program"}},
		{"CocoaKeys, then ReadlineKeys", ctrlHost, []BindOption{CocoaKeys(), ReadlineKeys()},
			[]string{"ctrl+backspace=delete-word-backward", "ctrl+w=delete-word-backward", "super+arrow_left=line-start"}, []string{"ctrl+backspace=program"}},
		{"CocoaKeys, then CUAKeys", superHost, []BindOption{CocoaKeys(), CUAKeys()},
			[]string{"ctrl+k=program", "ctrl+backspace=delete-word-backward"}, []string{"ctrl+k=delete-to-line-end"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := bindPageOn(t, c.caps, `<div%s><input id="f"></div>`, func(ctx context.Context) []templ.Attributes {
				return []templ.Attributes{On(ctx, "app", Bind(c.opts...))}
			})
			if len(s.faults) > 0 {
				t.Fatalf("faults %q", s.faults)
			}
			got := bindings(s, "app")
			for _, b := range c.has {
				if !slices.Contains(got, b) {
					t.Errorf("data-keys lack %s: %q", b, got)
				}
			}
			for _, b := range c.not {
				if slices.Contains(got, b) {
					t.Errorf("data-keys have %s: %q", b, got)
				}
			}
		})
	}
}
