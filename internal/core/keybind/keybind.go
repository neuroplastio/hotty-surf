// Package keybind implements the internal key-binding options, flattening, and
// platform keymap tables exposed through package surfkey and surf.Bind.
package keybind

import (
	"slices"
	"strconv"
	"strings"
	"sync"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// Option is a sealed key-binding option for surf.Bind.
type Option interface{ keybindOption() }

// KeysOption binds one or more keys to an action (or Unbind).
type KeysOption struct {
	Action hotty.Action
	Keys   []string
	Unbind bool
}

// GroupOption is a named or predefined group of key bindings.
type GroupOption struct {
	Opts []Option
	Read func(*rt.Frame) []rt.Bind
}

// ModeOption selects Yield or Override for a surf.Bind call.
type ModeOption struct{ Yield bool }

var (
	YieldOption    = &ModeOption{Yield: true}
	OverrideOption = &ModeOption{Yield: false}
)

func (*KeysOption) keybindOption()  {}
func (*GroupOption) keybindOption() {}
func (*ModeOption) keybindOption()  {}

// HasBinds reports whether opts hold at least one binding.
func HasBinds(opts []Option) bool {
	for _, o := range opts {
		switch o := o.(type) {
		case *KeysOption:
			if o != nil && len(o.Keys) > 0 {
				return true
			}
		case *GroupOption:
			if o != nil && (o.Read != nil || HasBinds(o.Opts)) {
				return true
			}
		}
	}
	return false
}

// Flatten expands opts into canonical rt.Binds for f.Platform.
func Flatten(f *rt.Frame, id string, opts []Option, out []rt.Bind) []rt.Bind {
	var mode *ModeOption
	for _, o := range opts {
		if m, ok := o.(*ModeOption); ok && m != nil {
			if mode != nil && mode != m {
				f.Faults = append(f.Faults, "surf.Bind: the id "+id+" has both Yield and Override in one Bind: the first holds")
				continue
			}
			mode = m
		}
	}
	return flattenIn(f, id, opts, mode, false, out)
}

func flattenIn(f *rt.Frame, id string, opts []Option, mode *ModeOption, group bool, out []rt.Bind) []rt.Bind {
	for _, o := range opts {
		switch o := o.(type) {
		case *ModeOption:
			if group && o != nil {
				f.Faults = append(f.Faults, "surfkey.Group: the id "+id+" binds a group with Yield or Override in it: set it where the group is bound, in its Bind")
			}
		case *KeysOption:
			if o == nil {
				continue
			}
			yield := group || o.Unbind
			if mode != nil {
				yield = mode.Yield
			}
			for _, k := range o.Keys {
				c, ok := f.Platform.ReadKey(k)
				switch {
				case !ok || c == "shift+tab" || c == "escape" || c == "tab" && !o.Unbind:
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds the key "+strconv.Quote(k)+", which doesn't parse or keeps its own meaning (SPEC §10.2)")
				case !o.Unbind && !bindable(o.Action):
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(k)+" to "+strconv.Quote(string(o.Action))+", which isn't an action Action takes: program is surf.Key's, and default is surfkey.Unbind")
				default:
					out = append(out, rt.Bind{Key: c, Action: string(o.Action), Yield: yield, Direct: !group})
				}
			}
		case *GroupOption:
			if o == nil {
				continue
			}
			if o.Read != nil {
				yield := true
				if mode != nil {
					yield = mode.Yield
				}
				for _, b := range o.Read(f) {
					b.Yield, b.Direct = yield, false
					out = append(out, b)
				}
				continue
			}
			out = flattenIn(f, id, o.Opts, mode, true, out)
		}
	}
	return out
}

// ReadlineBinds is hotty.TerminalKeys for the frame's platform.
func ReadlineBinds(f *rt.Frame) []rt.Bind {
	if f.Editing == nil {
		f.Editing = new(rt.Editing)
	}
	if f.Editing.Keymaps[2] == nil {
		out := []rt.Bind{}
		m := hotty.ParseKeymap(f.Platform, hotty.TerminalKeys)
		for _, b := range strings.Fields(m.Format()) {
			k, a, _ := strings.Cut(b, "=")
			if c, ok := f.Platform.ReadKey(k); ok {
				out = append(out, rt.Bind{Key: c, Action: a})
			}
		}
		f.Editing.Keymaps[2] = out
	}
	return f.Editing.Keymaps[2]
}

// CUABinds and CocoaBinds are CUAKeys' and CocoaKeys' bindings for the frame's platform.
func CUABinds(f *rt.Frame) []rt.Bind   { return editingFor(f, 0) }
func CocoaBinds(f *rt.Frame) []rt.Bind { return editingFor(f, 1) }

func editingFor(f *rt.Frame, i int) []rt.Bind {
	if f.Editing == nil {
		f.Editing = new(rt.Editing)
	}
	if f.Editing.Keymaps[i] == nil {
		edit := f.Platform.Edit
		if edit == nil {
			edit = hotty.DefaultEdit(f.Platform.Mod)
		}
		held := map[string]bool{}
		for _, keys := range edit {
			for _, k := range keys {
				if c, ok := f.Platform.ReadKey(k); ok {
					held[c] = true
				}
			}
		}
		out := []rt.Bind{}
		for _, b := range EditingTables()[i] {
			if c, ok := f.Platform.ReadKey(b.Key); ok {
				b.Key, b.Held = c, held[c]
				out = append(out, b)
			}
		}
		f.Editing.Keymaps[i] = out
	}
	return f.Editing.Keymaps[i]
}

var (
	editingOnce  sync.Once
	editingTable [2][]rt.Bind
)

// EditingTables returns the cached CUA ([0]) and Cocoa ([1]) editing keymap tables.
func EditingTables() *[2][]rt.Bind {
	editingOnce.Do(func() {
		none := map[string][]string{}
		var keys []string
		var acts [2]map[string]string
		for i, mod := range [2]string{"ctrl", "super"} {
			acts[i] = map[string]string{}
			for _, b := range strings.Fields(hotty.Resolve(hotty.Platform{Mod: mod, Edit: none}, true).Format()) {
				j := strings.LastIndexByte(b, '=')
				if !slices.Contains(keys, b[:j]) {
					keys = append(keys, b[:j])
				}
				acts[i][b[:j]] = b[j+1:]
			}
		}
		for _, k := range keys {
			if acts[0][k] == acts[1][k] {
				continue
			}
			for i := range acts {
				a := acts[i][k]
				if a == "" {
					a = string(hotty.Program)
				}
				editingTable[i] = append(editingTable[i], rt.Bind{Key: k, Action: a, Base: true})
			}
		}
	})
	return &editingTable
}

// CUAEditBinds and CocoaEditBinds are CUAEditShortcuts' and CocoaEditShortcuts' bindings.
func CUAEditBinds(f *rt.Frame) []rt.Bind   { return editShortcutsFor(f, 0) }
func CocoaEditBinds(f *rt.Frame) []rt.Bind { return editShortcutsFor(f, 1) }

func editShortcutsFor(f *rt.Frame, i int) []rt.Bind {
	t := EditShortcutTables()
	f.EditKeys = t.Keys
	return t.Binds[i]
}

// EditShortcutTable holds the cached CUA ([0]) and Cocoa ([1]) edit shortcut tables.
type EditShortcutTable struct {
	Binds [2][]rt.Bind
	Keys  []string
}

var (
	editShortcutsOnce  sync.Once
	editShortcutsTable EditShortcutTable
)

// EditShortcutTables returns the cached CUA and Cocoa edit shortcut tables.
func EditShortcutTables() *EditShortcutTable {
	editShortcutsOnce.Do(func() {
		order := []hotty.Action{hotty.SelectAll, hotty.Copy, hotty.Cut, hotty.Paste, hotty.Undo, hotty.Redo}
		var own [2][]rt.Bind
		for i, mod := range [2]string{"ctrl", "super"} {
			edit := hotty.DefaultEdit(mod)
			for _, a := range order {
				for _, k := range edit[string(a)] {
					own[i] = append(own[i], rt.Bind{Key: k, Action: string(a), Base: true, Edit: true})
				}
			}
		}
		t := &editShortcutsTable
		for i := range own {
			t.Binds[i] = append(t.Binds[i], own[i]...)
			for _, b := range own[1-i] {
				b.Action = string(hotty.Program)
				t.Binds[i] = append(t.Binds[i], b)
			}
			for _, b := range own[i] {
				t.Keys = append(t.Keys, b.Key)
			}
		}
	})
	return &editShortcutsTable
}

func bindable(a hotty.Action) bool {
	switch a {
	case hotty.Program, hotty.Default, hotty.Insert:
		return false
	}
	return new(hotty.Keymap).Bind("a", a)
}
