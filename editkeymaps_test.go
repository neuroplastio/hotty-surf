package surf

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/keybind"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/surfkey"
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
// and CocoaKeys' fourteen, each giving back the other's, made once a
// process, and filtered once for a frame without a surface's.
func TestEditingKeymapTables(t *testing.T) {
	tabs := keybind.EditingTables()
	if tabs != keybind.EditingTables() {
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
	a, b := keybind.CUABinds(f), keybind.CUABinds(f)
	if len(a) != 20 || &a[0] != &b[0] {
		t.Errorf("CUAKeys on a super host: %d bindings, made once: %v", len(a), &a[0] == &b[0])
	}
	held := func(b rt.Bind) bool { return b.Held }
	if c := keybind.CUABinds(&rt.Frame{}); len(c) != 20 || slices.IndexFunc(c, held) < 0 || c[slices.IndexFunc(c, held)].Key != "ctrl+a" || len(slices.DeleteFunc(slices.Clone(c), held)) != 19 {
		t.Errorf("CUAKeys on the zero platform: %v, want 20 with ctrl+a held (select all's)", c)
	}
}

// On every host, a field under CUAKeys or CocoaKeys resolves each of the
// tables' keys as that keymap's platform does with no data-keys, but a key the
// host's edit shortcuts hold, which stays the host's; and the surface's own use
// of a key (EditKeys, which cells' fallback reads) is the wire's.
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
			opt  surfkey.Option
			mod  string
		}{{"CUAKeys", surfkey.CUAKeys(), "ctrl"}, {"CocoaKeys", surfkey.CocoaKeys(), "super"}} {
			t.Run(fmt.Sprintf("%s on %q %v", km.name, host.Mod, caps.Edit != nil), func(t *testing.T) {
				s := bindPageOn(t, caps, `<div%s><textarea id="f"></textarea></div>`, func(ctx context.Context) []templ.Attributes {
					return []templ.Attributes{On(ctx, "app", Bind(km.opt))}
				})
				if len(s.faults) > 0 {
					t.Fatalf("faults %q", s.faults)
				}
				wire := attr(s.byID["app"], "data-keys")
				own := strings.Fields(rt.EditKeys(s.frame.Get("app"), true))
				if got := strings.Fields(wire); !slices.Equal(got, slices.Sorted(slices.Values(own))) {
					t.Errorf("the wire %q, the surface's own %q", got, own)
				}
				on := hotty.Resolve(host, true, wire)
				theirs := hotty.Resolve(hotty.Platform{Mod: km.mod, Edit: none}, true)
				base := hotty.Resolve(host, true)
				for _, b := range keybind.EditingTables()[i] {
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

// How CUAKeys and CocoaKeys meet the other bindings: a nearer ReadlineKeys
// keeps its keys (a give-back isn't a key given to the program); a page's key
// stays the page's; on one element the last group read wins where both
// bind a key, and a give-back fills only a key no group binds; and an Unbind
// gives a key back to the editing keymap in effect, on the element or around
// it, not to the host's platform.
func TestEditingKeymapsCompose(t *testing.T) {
	take := func(string) bool { return true }
	cases := []struct {
		name      string
		caps      hotty.Caps
		app, main []Handler
		field     []surfkey.Option
		has, not  []string // in the field's data-keys
	}{
		{"a nearer readline keeps ctrl+k", superHost, []Handler{Bind(surfkey.CUAKeys())}, nil, []surfkey.Option{surfkey.ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+backspace=delete-word-backward"}, nil},
		{"a page's key stays the page's", superHost, nil, []Handler{Key(take, "ctrl+k", "ctrl+backspace")}, []surfkey.Option{surfkey.CUAKeys()},
			[]string{"ctrl+delete=delete-word-forward"}, []string{"ctrl+k=program", "ctrl+backspace=delete-word-backward"}},
		{"override takes a page's key", superHost, nil, []Handler{Key(take, "ctrl+backspace")}, []surfkey.Option{surfkey.CUAKeys(), surfkey.Override()},
			[]string{"ctrl+backspace=delete-word-backward"}, nil},
		{"readline, then CUAKeys", superHost, nil, nil, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.CUAKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+w=delete-word-backward", "super+arrow_left=program"}, []string{"ctrl+k=program"}},
		{"readline, then CUAKeys, unbind", superHost, nil, nil, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.CUAKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=program"}, []string{"ctrl+k=default", "ctrl+k=delete-to-line-end"}},
		{"CUAKeys, then readline", superHost, nil, nil, []surfkey.Option{surfkey.CUAKeys(), surfkey.ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "super+arrow_left=program"}, nil},
		{"unbind on the element", superHost, nil, nil, []surfkey.Option{surfkey.CUAKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=program"}, []string{"ctrl+k=default"}},
		{"unbind before the group", superHost, nil, nil, []surfkey.Option{surfkey.Unbind("ctrl+k"), surfkey.CUAKeys()},
			[]string{"ctrl+k=program"}, []string{"ctrl+k=default"}},
		{"readline without ctrl+k, under CUAKeys", superHost, []Handler{Bind(surfkey.CUAKeys())}, nil, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=program", "ctrl+w=delete-word-backward"}, []string{"ctrl+k=default", "ctrl+k=delete-to-line-end"}},
		{"unbind of one of CocoaKeys' own", ctrlHost, []Handler{Bind(surfkey.CocoaKeys())}, nil, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=default"}},
		{"unbind of a key CocoaKeys gives back", ctrlHost, []Handler{Bind(surfkey.CocoaKeys())}, nil, []surfkey.Option{surfkey.Unbind("ctrl+backspace")},
			[]string{"ctrl+backspace=program"}, []string{"ctrl+backspace=default"}},
		{"unbind of a key neither holds", superHost, []Handler{Bind(surfkey.CUAKeys())}, nil, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+w")},
			[]string{"ctrl+w=default"}, nil},
		{"the nearer keymap is the one unbind goes back to", superHost, []Handler{Bind(surfkey.CUAKeys())}, []Handler{Bind(surfkey.CocoaKeys())}, []surfkey.Option{surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=program", "ctrl+k=default"}},
		{"unbind under CocoaKeys, then CUAKeys", superHost, []Handler{Bind(surfkey.CocoaKeys(), surfkey.CUAKeys())}, nil, []surfkey.Option{surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=program", "ctrl+k=default"}},
		{"unbind under CUAKeys, then CocoaKeys", superHost, []Handler{Bind(surfkey.CUAKeys(), surfkey.CocoaKeys())}, nil, []surfkey.Option{surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=program", "ctrl+k=default"}},
		{"unbind of ctrl+backspace under CUAKeys, then CocoaKeys", superHost, []Handler{Bind(surfkey.CUAKeys(), surfkey.CocoaKeys())}, nil, []surfkey.Option{surfkey.Unbind("ctrl+backspace")},
			[]string{"ctrl+backspace=delete-word-backward"}, []string{"ctrl+backspace=program", "ctrl+backspace=default"}},
		{"unbind on the element under CocoaKeys, then CUAKeys", superHost, nil, nil, []surfkey.Option{surfkey.CocoaKeys(), surfkey.CUAKeys(), surfkey.Unbind("ctrl+k")},
			[]string{"ctrl+k=delete-to-line-end"}, []string{"ctrl+k=program", "ctrl+k=default"}},
		{"both in one Bind: their keys together", superHost, nil, nil, []surfkey.Option{surfkey.CUAKeys(), surfkey.CocoaKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+backspace=delete-word-backward"}, []string{"ctrl+k=program", "ctrl+backspace=program"}},
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
			return []templ.Attributes{On(ctx, "app", Bind(surfkey.CocoaKeys()))}
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
		return []templ.Attributes{On(ctx, "b", Bind(surfkey.CUAKeys()))}
	})
	if len(s.faults) != 1 || hasAttr(s.byID["b"], "data-keys") {
		t.Errorf("on a button: faults %q, data-keys %q", s.faults, attr(s.byID["b"], "data-keys"))
	}
	s = bindPageOn(t, superHost, `<div%s><button id="b">b</button></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(surfkey.CUAKeys()))}
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
			app = append(app, Bind(surfkey.CUAKeys()))
		}
		fmt.Fprintf(w, `<div%s><input%s></div>`, attrsOf(On(ctx, "app", app...)), attrsOf(On(ctx, "f", Bind(surfkey.ReadlineKeys(), surfkey.Unbind("ctrl+k")))))
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
// keys stay; a give-back takes no key the other binds. One layering of
// each pair.
func TestEditingKeymapsLayer(t *testing.T) {
	cases := []struct {
		name     string
		caps     hotty.Caps
		opts     []surfkey.Option
		has, not []string
	}{
		{"CUAKeys, then ReadlineKeys", superHost, []surfkey.Option{surfkey.CUAKeys(), surfkey.ReadlineKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+w=delete-word-backward", "super+arrow_left=program"}, []string{"ctrl+k=program"}},
		{"CocoaKeys, then ReadlineKeys", ctrlHost, []surfkey.Option{surfkey.CocoaKeys(), surfkey.ReadlineKeys()},
			[]string{"ctrl+backspace=delete-word-backward", "ctrl+w=delete-word-backward", "super+arrow_left=line-start"}, []string{"ctrl+backspace=program"}},
		{"CocoaKeys, then CUAKeys", superHost, []surfkey.Option{surfkey.CocoaKeys(), surfkey.CUAKeys()},
			[]string{"ctrl+k=delete-to-line-end", "ctrl+backspace=delete-word-backward"}, []string{"ctrl+k=program", "ctrl+backspace=program"}},
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

// A surface makes CUAKeys' bindings once for its platform: every render
// after the first, and a compiled instance's check, share them, and
// Attach, which may bring another platform, makes them anew.
func TestEditingKeymapsOncePerPlatform(t *testing.T) {
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s><input id="f"></div>`, attrsOf(On(ctx, "app", Bind(surfkey.CUAKeys()))))
		return nil
	}), "")
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	first := s.editing.Keymaps[0]
	if len(first) != 20 {
		t.Fatalf("on a super host: %d bindings", len(first))
	}
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if &s.editing.Keymaps[0][0] != &first[0] {
		t.Error("a second render made the bindings again")
	}
	s.Attach(io.Discard, ctrlHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if len(s.editing.Keymaps[0]) != 20 || slices.Contains(bindings(s, "app"), "ctrl+a=program") {
		t.Errorf("attached where mod is ctrl: %d bindings, %q", len(s.editing.Keymaps[0]), bindings(s, "app"))
	}
}

// A give-back fills only a key no group in the Bind binds for real, in
// either order; between two real bindings of a key the later wins. Each
// order of each pair, on a host whose mod is ctrl (a cells Screen's
// platform) and on one whose mod is super, as a field resolves the wire.
func TestEditingKeymapsFill(t *testing.T) {
	groups := map[string]surfkey.Option{"ReadlineKeys": surfkey.ReadlineKeys(), "CUAKeys": surfkey.CUAKeys(), "CocoaKeys": surfkey.CocoaKeys()}
	pairs := [][2]string{{"ReadlineKeys", "CUAKeys"}, {"ReadlineKeys", "CocoaKeys"}, {"CUAKeys", "CocoaKeys"}}
	for _, caps := range []hotty.Caps{ctrlHost, superHost} {
		host := hotty.PlatformOf(caps)
		for _, p := range pairs {
			for _, order := range [][2]string{p, {p[1], p[0]}} {
				t.Run(fmt.Sprintf("%s, then %s, on %s", order[0], order[1], cmp.Or(string(host.Mod), "ctrl")), func(t *testing.T) {
					// What each holds, on its own, for this platform.
					f := &rt.Frame{Platform: host}
					want := map[string]string{}
					var keys []string
					for _, name := range order {
						var binds []rt.Bind
						binds = flatten(f, "app", []surfkey.Option{groups[name]}, binds)
						for _, b := range binds {
							if b.Held {
								continue // the host's edit shortcuts'
							}
							if !slices.Contains(keys, b.Key) {
								keys = append(keys, b.Key)
							}
							if _, ok := want[b.Key]; ok && giveBack(b) {
								continue // fills only
							}
							want[b.Key] = b.Action
						}
					}
					s := bindPageOn(t, caps, `<div%s><textarea id="f"></textarea></div>`, func(ctx context.Context) []templ.Attributes {
						return []templ.Attributes{On(ctx, "app", Bind(groups[order[0]], groups[order[1]]))}
					})
					if len(s.faults) > 0 {
						t.Fatalf("faults %q", s.faults)
					}
					on := hotty.Resolve(host, true, attr(s.byID["app"], "data-keys"))
					for _, k := range keys {
						w := hotty.Action(want[k])
						if w == hotty.Program {
							w = ""
						}
						if got := on.Lookup(k); got != w {
							t.Errorf("%s: %q, want %q", k, got, w)
						}
					}
				})
			}
		}
	}
}

// ReadlineKeys' bindings too are made once a platform: a second render
// shares them, and Attach where mod is super makes them anew, mod+a
// read as super+a.
func TestReadlineKeysOncePerPlatform(t *testing.T) {
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div><input%s></div>`, attrsOf(On(ctx, "f", Bind(surfkey.ReadlineKeys()))))
		return nil
	}), "")
	s.Attach(io.Discard, ctrlHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	first := s.editing.Keymaps[2]
	if len(first) == 0 || !slices.Contains(bindings(s, "f"), "ctrl+a=select-all") {
		t.Fatalf("where mod is ctrl: %d bindings, %q", len(first), bindings(s, "f"))
	}
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if &s.editing.Keymaps[2][0] != &first[0] {
		t.Error("a second render made the bindings again")
	}
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if got := bindings(s, "f"); &s.editing.Keymaps[2][0] == &first[0] || !slices.Contains(got, "super+a=select-all") || slices.Contains(got, "ctrl+a=select-all") {
		t.Errorf("attached where mod is super: %q", got)
	}
}
