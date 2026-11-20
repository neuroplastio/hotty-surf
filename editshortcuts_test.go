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

// With no edit-shortcut group, the editing keymaps write what they wrote
// before the groups came: each combination's data-keys, on a host whose
// mod is ctrl and on one whose mod is super, as the branch had them
// (R ReadlineKeys, U CUAKeys, C CocoaKeys, in order).
func TestEditingKeymapsWithoutEditShortcuts(t *testing.T) {
	groups := map[rune]BindOption{'R': ReadlineKeys(), 'U': CUAKeys(), 'C': CocoaKeys()}
	for _, c := range []struct{ mod, combo, want string }{
		{"", "U", "ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=program ctrl+backspace=delete-word-backward ctrl+d=program ctrl+delete=delete-word-forward ctrl+e=program ctrl+end=input-end ctrl+f=program ctrl+h=program ctrl+home=input-start ctrl+k=program ctrl+n=program ctrl+p=program super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"", "C", "ctrl+arrow_left=program ctrl+arrow_right=program ctrl+b=char-backward ctrl+backspace=program ctrl+d=delete-char-forward ctrl+delete=program ctrl+e=line-end ctrl+end=program ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=program ctrl+k=delete-to-line-end ctrl+n=line-next ctrl+p=line-previous super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"", "RU", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=select-all ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"", "UR", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=select-all ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"", "RC", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=select-all ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"", "CR", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=select-all ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"", "UC", "ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+n=line-next ctrl+p=line-previous super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"super", "U", "ctrl+a=program ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=program ctrl+backspace=delete-word-backward ctrl+d=program ctrl+delete=delete-word-forward ctrl+e=program ctrl+end=input-end ctrl+f=program ctrl+h=program ctrl+home=input-start ctrl+k=program ctrl+n=program ctrl+p=program super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"super", "C", "ctrl+a=line-start ctrl+arrow_left=program ctrl+arrow_right=program ctrl+b=char-backward ctrl+backspace=program ctrl+d=delete-char-forward ctrl+delete=program ctrl+e=line-end ctrl+end=program ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=program ctrl+k=delete-to-line-end ctrl+n=line-next ctrl+p=line-previous super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"super", "RU", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=program ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+a=select-all super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"super", "UR", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=program ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+a=select-all super+arrow_down=program super+arrow_left=program super+arrow_right=program super+arrow_up=program super+backspace=program"},
		{"super", "RC", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=line-start ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+a=select-all super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"super", "CR", "alt+<=input-start alt+>=input-end alt+arrow_left=word-backward alt+arrow_right=word-forward alt+b=word-backward alt+backspace=delete-word-backward alt+d=delete-word-forward alt+delete=delete-word-forward alt+f=word-forward arrow_down=line-next arrow_left=char-backward arrow_right=char-forward arrow_up=line-previous backspace=delete-char-backward ctrl+a=line-start ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+m=newline ctrl+n=line-next ctrl+p=line-previous ctrl+u=delete-to-line-start ctrl+w=delete-word-backward delete=delete-char-forward end=line-end home=line-start page_down=page-down page_up=page-up super+a=select-all super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
		{"super", "UC", "ctrl+a=line-start ctrl+arrow_left=word-backward ctrl+arrow_right=word-forward ctrl+b=char-backward ctrl+backspace=delete-word-backward ctrl+d=delete-char-forward ctrl+delete=delete-word-forward ctrl+e=line-end ctrl+end=input-end ctrl+f=char-forward ctrl+h=delete-char-backward ctrl+home=input-start ctrl+k=delete-to-line-end ctrl+n=line-next ctrl+p=line-previous super+arrow_down=input-end super+arrow_left=line-start super+arrow_right=line-end super+arrow_up=input-start super+backspace=delete-to-line-start"},
	} {
		var opts []BindOption
		for _, g := range c.combo {
			opts = append(opts, groups[g])
		}
		s := bindPageOn(t, hotty.Caps{Mod: c.mod}, `<div%s><input id="f"></div>`, func(ctx context.Context) []templ.Attributes {
			return []templ.Attributes{On(ctx, "app", Bind(opts...))}
		})
		if got := attr(s.byID["app"], "data-keys"); got != c.want {
			t.Errorf("%s where mod is %q:\n got %s\nwant %s", c.combo, c.mod, got, c.want)
		}
		if got := attr(s.byID["f"], "data-keys"); got != "" {
			t.Errorf("%s where mod is %q: the field has %q", c.combo, c.mod, got)
		}
	}
}

// chain is the data-keys of the elements from the root down to id, id's
// last: its keymap as SPEC §10.2 reads it.
func chain(s *Surface, id string) []string {
	var vs []string
	for n := s.byID[id]; n != nil; n = n.Parent {
		if hasAttr(n, "data-keys") {
			vs = append([]string{attr(n, "data-keys")}, vs...)
		}
	}
	return vs
}

// lookup is what the field id does with key, on a host of caps.
func lookup(s *Surface, caps hotty.Caps, id, key string) hotty.Action {
	return hotty.Resolve(hotty.PlatformOf(caps), true, chain(s, id)...).Lookup(key)
}

// The tables are SPEC §10.2's edit shortcuts, in its order of the actions,
// each giving back the other's: made once a process, the same on every
// platform, every binding marked.
func TestEditShortcutsTables(t *testing.T) {
	tabs := editShortcutTables()
	if tabs != editShortcutTables() {
		t.Error("the tables were made twice")
	}
	want := [2]string{
		"ctrl+a=select-all ctrl+c=copy ctrl+x=cut ctrl+v=paste ctrl+z=undo ctrl+shift+z=redo ctrl+y=redo super+a=program super+c=program super+x=program super+v=program super+z=program super+shift+z=program",
		"super+a=select-all super+c=copy super+x=cut super+v=paste super+z=undo super+shift+z=redo ctrl+a=program ctrl+c=program ctrl+x=program ctrl+v=program ctrl+z=program ctrl+shift+z=program ctrl+y=program",
	}
	for i := range tabs.binds {
		var got []string
		for _, b := range tabs.binds[i] {
			got = append(got, b.Key+"="+b.Action)
			if !b.Base || !b.Edit || b.Held || b.Direct {
				t.Errorf("%v isn't marked", b)
			}
		}
		if strings.Join(got, " ") != want[i] {
			t.Errorf("table %d:\n got %s\nwant %s", i, strings.Join(got, " "), want[i])
		}
	}
	if len(tabs.keys) != 13 {
		t.Errorf("%d keys", len(tabs.keys))
	}
	f := &rt.Frame{}
	if b := cuaEditBinds(f); &b[0] != &tabs.binds[0][0] || len(f.EditKeys) != 13 {
		t.Errorf("a frame's: shared %v, keys %d", &b[0] == &tabs.binds[0][0], len(f.EditKeys))
	}
}

// editPageOn is a page on a host of caps: app's options on the outermost
// element, form's on one between it and the field f (an input of type
// typ, or a textarea), and a paragraph and a button beside.
func editPageOn(t *testing.T, caps hotty.Caps, app, form []BindOption, field string) *Surface {
	t.Helper()
	if field == "" {
		field = `<input id="f"%s>`
	}
	return bindPageOn(t, caps, `<div%s><p id="p">text</p><button id="b">b</button><form%s>`+field+`</form></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(app...)), On(ctx, "form", Bind(form...)), On(ctx, "f")}
	})
}

var (
	cuaEdit   = CUAEditShortcuts()
	cocoaEdit = CocoaEditShortcuts()
)

// The review's matrix: on one element the edit-shortcut group owns the six
// edit actions, in every order of the Bind's groups. ctrl+a and super+a are
// the only keys both categories touch.
func TestEditShortcutsOnOneElement(t *testing.T) {
	rows := []struct {
		name          string
		caps          hotty.Caps
		groups        []BindOption
		ctrlA, superA hotty.Action // "" reaches the program; "-" elided
	}{
		{"Cocoa + CocoaEdit", ctrlHost, []BindOption{CocoaKeys(), cocoaEdit}, hotty.LineStart, hotty.SelectAll},
		{"Readline + CocoaEdit", ctrlHost, []BindOption{ReadlineKeys(), cocoaEdit}, "", hotty.SelectAll},
		{"Readline + Cocoa + CocoaEdit", ctrlHost, []BindOption{ReadlineKeys(), CocoaKeys(), cocoaEdit}, hotty.LineStart, hotty.SelectAll},
		{"Readline + CUAEdit", superHost, []BindOption{ReadlineKeys(), cuaEdit}, hotty.SelectAll, ""},
		{"CUA + CUAEdit", superHost, []BindOption{CUAKeys(), cuaEdit}, hotty.SelectAll, ""},
		{"CUAEdit alone", ctrlHost, []BindOption{cuaEdit}, "-", "-"},
	}
	for _, r := range rows {
		var seen string
		for _, order := range permutations(r.groups) {
			s := editPageOn(t, r.caps, order, nil, "")
			if len(s.faults) > 0 {
				t.Fatalf("%s: faults %q", r.name, s.faults)
			}
			field := attr(s.byID["f"], "data-keys")
			if seen == "" {
				seen = field + "|" + attr(s.byID["app"], "data-keys")
			} else if seen != field+"|"+attr(s.byID["app"], "data-keys") {
				t.Errorf("%s: another order wrote another keymap", r.name)
			}
			for k, want := range map[string]hotty.Action{"ctrl+a": r.ctrlA, "super+a": r.superA} {
				if want == "-" {
					if strings.Contains(field, k+"=") {
						t.Errorf("%s: %s written on the field: %q", r.name, k, field)
					}
					continue
				}
				if got := lookup(s, r.caps, "f", k); got != want {
					t.Errorf("%s: %s is %q, want %q (field %q)", r.name, k, got, want, field)
				}
			}
			if strings.Contains(attr(s.byID["app"], "data-keys"), "c=") {
				t.Errorf("%s: a copy key on the app element: %q", r.name, attr(s.byID["app"], "data-keys"))
			}
		}
	}
}

// permutations is every order of gs.
func permutations(gs []BindOption) [][]BindOption {
	if len(gs) <= 1 {
		return [][]BindOption{gs}
	}
	var out [][]BindOption
	for i := range gs {
		rest := slices.Concat(gs[:i:i], gs[i+1:])
		for _, p := range permutations(rest) {
			out = append(out, append([]BindOption{gs[i]}, p...))
		}
	}
	return out
}

// Fields only: the group writes nothing on the elements that aren't text
// fields, so a paragraph's copy key and a button's keys are the host's,
// and the field's are the group's, but where they change nothing.
func TestEditShortcutsFieldsOnly(t *testing.T) {
	s := editPageOn(t, superHost, []BindOption{cuaEdit}, nil, "")
	if len(s.faults) > 0 {
		t.Fatalf("faults %q", s.faults)
	}
	for _, id := range []string{"app", "p", "b", "form"} {
		if hasAttr(s.byID[id], "data-keys") {
			t.Errorf("%s has %q", id, attr(s.byID[id], "data-keys"))
		}
	}
	want := "ctrl+a=select-all ctrl+c=copy ctrl+shift+z=redo ctrl+v=paste ctrl+x=cut ctrl+y=redo ctrl+z=undo super+a=program super+c=program super+shift+z=program super+v=program super+x=program super+z=program"
	if got := attr(s.byID["f"], "data-keys"); got != want {
		t.Errorf("the field:\n got %s\nwant %s", got, want)
	}
	if n := len(attr(s.byID["f"], "data-keys")); n < 180 || n > 220 {
		t.Errorf("%d bytes on the field", n)
	}
	// On a Linux host it changes nothing: nothing written.
	s = editPageOn(t, ctrlHost, []BindOption{cuaEdit}, nil, "")
	if hasAttr(s.byID["f"], "data-keys") {
		t.Errorf("where mod is ctrl: %q", attr(s.byID["f"], "data-keys"))
	}
	// CocoaEditShortcuts there: the paragraph's copy key stays ctrl+c.
	s = editPageOn(t, ctrlHost, []BindOption{cocoaEdit}, nil, "")
	p := hotty.Resolve(hotty.PlatformOf(ctrlHost), false, chain(s, "p")...)
	if p.Lookup("ctrl+c") != hotty.Copy || p.Lookup("super+c") == hotty.Copy {
		t.Errorf("the paragraph's copy key: ctrl+c %q, super+c %q", p.Lookup("ctrl+c"), p.Lookup("super+c"))
	}
	if lookup(s, ctrlHost, "f", "super+c") != hotty.Copy || lookup(s, ctrlHost, "f", "ctrl+c") != "" {
		t.Errorf("the field: %q", attr(s.byID["f"], "data-keys"))
	}
}

// Every element between the group's and the field wins its keys, not
// only the field's own: a form's direct Action, and a form's CocoaKeys
// under CocoaEditShortcuts, whose give-back of ctrl+a only fills. Under
// CUAEditShortcuts, which binds ctrl+a, CocoaKeys on the form leaves
// ctrl+a out, as the edit shortcuts in effect there hold it.
func TestEditShortcutsElementsBetween(t *testing.T) {
	for _, c := range []struct {
		name       string
		app, form  []BindOption
		ctrlA      hotty.Action
		other, act string
	}{
		{"a direct Action", []BindOption{cuaEdit}, []BindOption{Action(hotty.LineStart, "ctrl+a")}, hotty.LineStart, "ctrl+c", ""},
		{"CocoaKeys under CocoaEdit", []BindOption{cocoaEdit}, []BindOption{CocoaKeys()}, hotty.LineStart, "ctrl+e", "line-end"},
		{"CocoaKeys under CUAEdit", []BindOption{cuaEdit}, []BindOption{CocoaKeys()}, hotty.SelectAll, "ctrl+e", "line-end"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := editPageOn(t, ctrlHost, c.app, c.form, "")
			if len(s.faults) > 0 {
				t.Fatalf("faults %q", s.faults)
			}
			if got := lookup(s, ctrlHost, "f", "ctrl+a"); got != c.ctrlA {
				t.Errorf("ctrl+a: %q, want %q (%q)", got, c.ctrlA, chain(s, "f"))
			}
			if c.act != "" {
				if got := lookup(s, ctrlHost, "f", c.other); string(got) != c.act {
					t.Errorf("%s: %q", c.other, got)
				}
			}
		})
	}
	// On a host whose mod is super, CocoaKeys on the form under the root's
	// CUAEditShortcuts: ctrl+a selects all, the edit group's; the form
	// writes no ctrl+a.
	s := editPageOn(t, superHost, []BindOption{cuaEdit}, []BindOption{CocoaKeys()}, "")
	if got := lookup(s, superHost, "f", "ctrl+a"); got != hotty.SelectAll || strings.Contains(attr(s.byID["form"], "data-keys"), "ctrl+a=") {
		t.Errorf("ctrl+a: %q, the form %q", got, attr(s.byID["form"], "data-keys"))
	}
}

// An Unbind on a form under the group goes back to the group's binding,
// and is written only on the fields inside: never a copy key on the form.
func TestEditShortcutsUnbindNotOnAContainer(t *testing.T) {
	// The form binds ctrl+c to line-end, and the element inside it
	// unbinds it.
	s := bindPageOn(t, superHost, `<div%s><form%s><div%s><p id="p">text</p><input id="f"></div></form></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(cuaEdit)), On(ctx, "form", Bind(Action(hotty.LineEnd, "ctrl+c"))), On(ctx, "in", Bind(Unbind("ctrl+c")))}
	})
	if len(s.faults) > 0 {
		t.Fatalf("faults %q", s.faults)
	}
	if strings.Contains(attr(s.byID["in"], "data-keys"), "ctrl+c") {
		t.Errorf("the container: %q", attr(s.byID["in"], "data-keys"))
	}
	if got := lookup(s, superHost, "f", "ctrl+c"); got != hotty.Copy {
		t.Errorf("the field's ctrl+c: %q (%q)", got, chain(s, "f"))
	}
	p := hotty.Resolve(hotty.PlatformOf(superHost), false, chain(s, "p")...)
	if p.Lookup("ctrl+c") == hotty.Copy || p.Lookup("super+c") != hotty.Copy {
		t.Errorf("the paragraph's copy key: ctrl+c %q, super+c %q", p.Lookup("ctrl+c"), p.Lookup("super+c"))
	}
	// On the field itself, an Unbind goes back to the group's: undo.
	s = bindPageOn(t, superHost, `<div%s><input%s></div>`, func(ctx context.Context) []templ.Attributes {
		return []templ.Attributes{On(ctx, "app", Bind(cuaEdit)), On(ctx, "f", Bind(ReadlineKeys(), Unbind("ctrl+z")))}
	})
	if got := lookup(s, superHost, "f", "ctrl+z"); got != hotty.Undo {
		t.Errorf("ctrl+z under an Unbind on the field: %q", got)
	}
}

// Bound directly on a control that isn't a text field it is one fault,
// shared with the editing keymaps', and nothing is written; on a
// container it is harmless.
func TestEditShortcutsFaults(t *testing.T) {
	for _, opts := range [][]BindOption{{cuaEdit}, {CUAKeys(), cuaEdit}, {Group(cocoaEdit)}} {
		s := bindPageOn(t, superHost, `<div><button%s>b</button></div>`, func(ctx context.Context) []templ.Attributes {
			return []templ.Attributes{On(ctx, "b", Bind(opts...))}
		})
		if len(s.faults) != 1 || hasAttr(s.byID["b"], "data-keys") {
			t.Errorf("%d options: faults %q, data-keys %q", len(opts), s.faults, attr(s.byID["b"], "data-keys"))
		}
	}
}

// Every text field is written the group, readonly, number, password and
// editing hosts among them, and a group nested in a Group is the same;
// what isn't a text field isn't.
func TestEditShortcutsEveryField(t *testing.T) {
	for _, field := range []string{
		`<input id="f"%s readonly>`, `<input type="number" id="f"%s>`, `<input type="password" id="f"%s>`,
		`<textarea id="f"%s></textarea>`, `<div contenteditable id="f"%s></div>`,
	} {
		for _, opt := range []BindOption{cuaEdit, Group(Group(cuaEdit))} {
			s := editPageOn(t, superHost, []BindOption{opt}, nil, field)
			if got := lookup(s, superHost, "f", "ctrl+c"); got != hotty.Copy {
				t.Errorf("%s: ctrl+c %q", field, got)
			}
		}
	}
	for _, field := range []string{`<div contenteditable="False" id="f"%s></div>`, `<select id="f"%s></select>`, `<input type="checkbox" id="f"%s>`} {
		s := editPageOn(t, superHost, []BindOption{cuaEdit}, nil, field)
		if hasAttr(s.byID["f"], "data-keys") || len(s.faults) > 0 {
			t.Errorf("%s: %q, faults %q", field, attr(s.byID["f"], "data-keys"), s.faults)
		}
	}
}

// The group's bindings yield to a key given the program around the field
// (a page's surf.Key), unless the Bind says Override; the field's own
// surf.Key always wins.
func TestEditShortcutsYieldOverride(t *testing.T) {
	take := func(string) bool { return true }
	for _, c := range []struct {
		name      string
		app       []BindOption
		form, own []Handler
		want      string
	}{
		{"yields", []BindOption{cuaEdit}, []Handler{Key(take, "ctrl+c")}, nil, "ctrl+c=program"},
		{"overrides", []BindOption{cuaEdit, Override()}, []Handler{Key(take, "ctrl+c")}, nil, "ctrl+c=copy"},
		{"the field's own", []BindOption{cuaEdit, Override()}, nil, []Handler{Key(take, "ctrl+c")}, "ctrl+c=program"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := bindPageOn(t, superHost, `<div%s><form%s><input%s></form></div>`, func(ctx context.Context) []templ.Attributes {
				return []templ.Attributes{On(ctx, "app", Bind(c.app...)), On(ctx, "form", c.form...), On(ctx, "f", c.own...)}
			})
			if len(s.faults) > 0 {
				t.Fatalf("faults %q", s.faults)
			}
			km := hotty.ParseKeymap(hotty.PlatformOf(superHost), strings.Join(chain(s, "f"), " "))
			got := "ctrl+c=copy"
			if km.Program("ctrl+c") {
				got = "ctrl+c=program"
			}
			if got != c.want {
				t.Errorf("%s, want %s (%q)", got, c.want, chain(s, "f"))
			}
		})
	}
}

// A host that reports its own edit shortcuts: with copy moved to
// ctrl+shift+c, CUAEditShortcuts still writes ctrl+c=copy, and leaves
// ctrl+shift+c the host's.
func TestEditShortcutsOverAHostsOwn(t *testing.T) {
	edit := hotty.DefaultEdit("ctrl")
	edit[string(hotty.Copy)] = []string{"ctrl+shift+c"}
	caps := hotty.Caps{Edit: edit}
	s := editPageOn(t, caps, []BindOption{cuaEdit}, nil, "")
	got := attr(s.byID["f"], "data-keys")
	if got != "ctrl+c=copy" {
		t.Errorf("the field: %q", got)
	}
	if lookup(s, caps, "f", "ctrl+shift+c") != hotty.Copy {
		t.Errorf("ctrl+shift+c: %q", lookup(s, caps, "f", "ctrl+shift+c"))
	}
}

// OwnKeys follows the group: under CocoaEditShortcuts on a Linux host, a
// focused field takes super+c itself, and gives ctrl+c to the program.
func TestEditShortcutsOwnKeys(t *testing.T) {
	s := editPageOn(t, ctrlHost, []BindOption{cocoaEdit}, nil, "")
	s.focused = "f"
	own := s.OwnKeys()
	if !slices.Contains(own, "super+c") || slices.Contains(own, "ctrl+c") {
		t.Errorf("OwnKeys %q", own)
	}
}

type editFieldProps struct{ Label string }

var editField = Compile(func(p editFieldProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div><span>%s</span><input%s></div>`, p.Label, attrsOf(On(ctx, "q")))
		return nil
	})
})

// A compiled instance filled alone writes the group around it on its
// fields, as the first render did (record.edit), and a group switched at
// the root repaints a kept instance's fields.
func TestEditShortcutsInAnInstance(t *testing.T) {
	label, group := "a", cuaEdit
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s>`, attrsOf(On(ctx, "app", Bind(group))))
		if err := editField.With("k", editFieldProps{Label: label}).Render(ctx, w); err != nil {
			return err
		}
		fmt.Fprint(w, `</div>`)
		return nil
	}), "")
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(bindings(s, "q"), "ctrl+c=copy") {
		t.Fatalf("first render: %q", bindings(s, "q"))
	}
	label = "b"
	fills := s.c.fills
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if s.c.fills == fills || !slices.Contains(bindings(s, "q"), "ctrl+c=copy") {
		t.Errorf("filled alone (%d fills): %q", s.c.fills-fills, bindings(s, "q"))
	}
	if w := s.frame.Get("q").Written; !slices.ContainsFunc(w, func(b rt.Bind) bool { return b.Key == "ctrl+c" && b.Edit }) {
		t.Errorf("Written: %v", w)
	}
	group = cocoaEdit
	s.Do(func() {})
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if got := bindings(s, "q"); len(got) != 0 {
		t.Errorf("under CocoaEditShortcuts where mod is super: %q", got)
	}
}

type editFormProps struct{ Name string }

var editForm = Compile(func(p editFormProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div%s><span>%s</span><input%s></div>`, attrsOf(On(ctx, "form", Bind(cuaEdit))), p.Name, attrsOf(On(ctx, "q")))
		return nil
	})
})

// A kept compiled instance that binds the group itself: a render that
// changes only what is outside it still finds the group's keys, and the
// instance's field keeps its bindings.
func TestEditShortcutsInAKeptInstance(t *testing.T) {
	n := 0
	s := New("t", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintf(w, `<div><p>%d</p>`, n)
		if err := editForm.With("k", editFormProps{Name: "a"}).Render(ctx, w); err != nil {
			return err
		}
		fmt.Fprint(w, `</div>`)
		return nil
	}), "")
	s.Attach(io.Discard, superHost)
	if _, err := s.Document(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if !slices.Contains(bindings(s, "q"), "ctrl+c=copy") || len(s.faults) > 0 {
			t.Fatalf("render %d: %q, faults %q", n, bindings(s, "q"), s.faults)
		}
		n++
		s.Do(func() {})
		if err := s.Update(); err != nil {
			t.Fatal(err)
		}
	}
}
