package surf

import (
	"slices"
	"strconv"
	"strings"
	"sync"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Bind gives keys on the element to the host's own actions (SPEC §10.2):
// a text field's edit actions, the scroll actions, and prefix (§10.5).
// surf.Key gives keys to the program, and so do the give-backs of
// CUAKeys and CocoaKeys, the keys a field doesn't use on that platform.
// Its options are bindings (Action, Unbind, Group, and the editing
// keymaps ReadlineKeys, CUAKeys and CocoaKeys) and at most one of Yield
// or Override, which sets how all of them meet the keys the program is
// given around the element. Without either, an Action wins and an Unbind
// and a group's bindings yield. A Bind with no binding is no handler; a
// nil option is none.
//
// The groups layer: one Bind can hold several, read in order, a later
// binding replacing an earlier one of the same key. A give-back of
// CUAKeys or CocoaKeys (a key it hands the program) is the exception: it
// fills only a key no group on the element binds, whatever the order.
// So surf.Bind(surf.ReadlineKeys(), surf.CUAKeys()) is the Windows and
// Linux desktop's keys where they bind and readline's for the rest, and
// readline keeps ctrl+k, which CUAKeys gives back, in either order.
//
// The palette's query edits as a terminal's line does, in one line:
//
//	surf.On(ctx, "query", surf.Input(a.SetQuery),
//		surf.Bind(surf.ReadlineKeys()))
//
// The actions:
//   - a text field's edit actions (hotty.WordBackward, hotty.LineStart,
//     …), on a field or on an element that holds fields, for every field
//     inside; a multi-line one (hotty.Newline, hotty.LineNext, …) on a
//     textarea, an editing host or an element that holds one.
//   - hotty.Prefix (SPEC §10.5): a leader. The key reaches the program,
//     as a declared one does, and so does every key after it, Tab and
//     Escape too, until the sequence ends. A prefix a Key handler takes is
//     the surface's: it ends once the key after it has been through the
//     handlers. One no handler takes is the program's own keymap's (an
//     app's leader, surf.Bind(surf.Action(hotty.Prefix, "mod+k")) on its
//     main), and the program ends it with Surface.Focus. A printable
//     prefix still types in a field inside, as a declared key does.
//   - a scroll action (hotty.ScrollDown, hotty.ScrollPageUp, …) on an
//     element that isn't a field: a key the focused element doesn't use
//     scrolls the nearest element that scrolls (SPEC §10.2).
//
// Which binding holds a key on one element is, in order: a surf.Key's or
// a ui.KeyScope's key, which is the program's; then a direct Action or
// Unbind (one given to Bind, not in a group), where two of one key to
// different actions are a fault and the first stays; then the groups'
// bindings, the last one read winning (Bind by Bind, option by option, a
// group's bindings in their order), but a give-back, which holds only a
// key no group binds. Then that binding's Yield or
// Override decides whether it gives way to the program's keys around the
// element (a surf.Key's declared key, a prefix, a ui.KeyScope's forwarded
// key, on an element around it): a binding that yields is dropped, and
// nothing under it on the element comes back. So a page's mod+k on its
// main still reaches its handler from a field under ReadlineKeys, where
// readline would delete to the line's end with ctrl+k.
//
// These are faults, each binding dropped: an Action that isn't SPEC's,
// or is program, default (Unbind's) or insert; a key that doesn't parse,
// shift+tab and escape, and tab but for Unbind (tab=default gives Tab
// back to focus); a direct binding of a key a surf.Key on the element
// declares, but an Action to prefix, which still gives fn the key (a
// group's is silent); Yield and Override in one Bind, or in a Group; an
// edit action on a control that isn't a text field (a button, a select, a
// check box, a date field), one fault per binding, or per element for a
// group; a direct multi-line Action on an input (a group's is harmless);
// a scroll action on a text field. An edit action on an element that
// holds no field (yet) is harmless, and no fault.
//
// Bindings are a render's, as every handler is: a render that passes
// other options rebinds the element, and the host has it with the
// render's delta, an attribute change that keeps a field's caret; a key
// sequence in progress goes on. The surface writes the bindings into the
// element's data-keys (SPEC §10.2), as it writes Key's, one per key. A
// Bind is the host's action, not one of ui.Bindings'.
func Bind(opts ...BindOption) Handler {
	rt.LinkText(keydefault.Text)
	if !binds(opts) {
		return Handler{}
	}
	return Handler{kind: kindBind, fn: slices.Clone(opts)}
}

// A BindOption is one of Bind's options. Action, Unbind, Group,
// ReadlineKeys, CUAKeys, CocoaKeys, Yield and Override make them, and
// nothing else does.
type BindOption interface{ bindOption() }

// Action binds keys to one of SPEC §10.2's actions other than program
// (surf.Key's) and default (Unbind's): hotty.WordBackward,
// hotty.ScrollDown, hotty.Prefix, …. Given to Bind directly it wins, as
// SPEC's nearer binding does; in a Group it is one of the group's.
func Action(action hotty.Action, keys ...string) BindOption {
	return &keysOption{action: action, keys: slices.Clone(keys)}
}

// Unbind gives keys back to the host's own meaning on the element and
// inside it (SPEC §10.2, key=default): a field's platform default, a
// button's Space, the surface's scrolling. It replaces the bindings of
// those keys before it, on the element and around it, but yields to a key
// the program is given around the element unless its Bind says Override.
// Readline's keys without ctrl+k are surf.Bind(surf.ReadlineKeys(),
// surf.Unbind("ctrl+k")).
//
// Under CUAKeys or CocoaKeys, on the element or around it, a key the
// keymap binds goes back to the keymap's binding instead, not to the
// host's platform: its edit action, or, for a key it gives back, the
// program. So under CUAKeys on a host whose mod is super, readline's
// keys without ctrl+k give ctrl+k to the program, as on Linux, where the
// host's own default would delete to the line's end.
func Unbind(keys ...string) BindOption {
	return &keysOption{action: hotty.Default, keys: slices.Clone(keys), unbind: true}
}

// Group is a set of bindings given whole: an app's or the kit's, made
// once and bound wherever it's needed. Its options are bindings, read in
// order, a later one replacing an earlier one of the same key; Yield and
// Override are Bind's, and in a Group they are a fault. Group copies its
// options: a group stays as it was made.
//
//	var pager = surf.Group(
//		surf.Action(hotty.ScrollDown, "j", "arrow_down"),
//		surf.Action(hotty.ScrollUp, "k", "arrow_up"),
//	)
//
//	surf.On(ctx, "log", surf.Bind(pager))
func Group(opts ...BindOption) BindOption {
	return &groupOption{opts: slices.Clone(opts)}
}

// ReadlineKeys is the editing keymap of a terminal's line, a group: the
// SDK's hotty.TerminalKeys, the readline keys of Bubble Tea's text input
// (ctrl+e, ctrl+k, ctrl+u, ctrl+w, alt+b, alt+f, …), over the platform's
// default; mod+a still selects all. Its source is GNU Readline's manual,
// "Bindable Readline Commands" in the GNU Readline Library User
// Interface: https://tiswww.case.edu/php/chet/readline/rluserman.html
//
// On an element that holds fields, it binds them all. A kit Command's or
// Combobox's field, which takes no handler of the app's, gets it from a
// Bind on an element around the component. It layers with the other
// editing keymaps, as Bind says, in either order:
// surf.Bind(surf.ReadlineKeys(), surf.CUAKeys()) is the desktop's keys
// where they bind and readline's for the rest, since CUAKeys' give-backs
// don't take a key readline binds.
func ReadlineKeys() BindOption {
	return &groupOption{read: readlineBinds}
}

// CUAKeys is the editing keymap of IBM's Common User Access, the Windows
// and Linux desktop's, a group: a text field's caret and delete keys as
// SPEC §10.2 has them where mod is ctrl, whatever the host's mod:
// ctrl+arrow_left and ctrl+arrow_right move by words, ctrl+home and
// ctrl+end go to the value's ends, ctrl+backspace and ctrl+delete delete
// words. Its source is IBM's "Object-Oriented Interface Design: IBM
// Common User Access Guidelines", SC34-4399-00 (1992), which replaces
// SC34-4290, for the word moves and the moves to the value's ends; the
// word deletes, ctrl+backspace and ctrl+delete, aren't in it, and are
// later Windows and Linux practice, as SPEC §10.2 has them. IBM no
// longer hosts it; a scan is at https://archive.org/download/sc34-4399-00/
//
// The keys only macOS gives a field (super+arrow_left,
// super+backspace, ctrl+a, ctrl+e, ctrl+k, …) are given back: from a
// field they reach the program, as on Linux. A give-back only fills: a
// binding nearer the field still takes the key, and so does any group on
// the same element that binds it, before CUAKeys or after it.
//
// Only those keys change. What mod stands for stays the host's, and so
// do a shortcut's keycap, the edit shortcuts (select all, copy, paste,
// undo: the host's and the user's, SPEC §10.2), every key the edit
// shortcuts hold, and the host's scrolling. Outside a field the
// give-backs reach the program too: on a host whose mod is super, a
// focused button no longer scrolls with super+arrow_up and
// super+arrow_down, where the host did (home, end and the page keys
// still scroll). Under it, Unbind gives a key back to CUAKeys' binding,
// not to the host's platform.
//
// Bind it once, on the outermost element of each surface's view, so
// that every field inside has it, the kit's included: a field outside
// that element doesn't. It layers with the other editing keymaps, as
// Bind says: where two groups bind a key, the later one's binding wins,
// and its give-backs fill only the keys the others leave, so
// surf.Bind(surf.ReadlineKeys(), surf.CUAKeys()) is the desktop's keys
// where they bind and readline's for the rest. With CocoaKeys in the
// same Bind it is the two platforms' keys together. A field may bind
// CocoaKeys, or ReadlineKeys, nearer.
//
//	surf.On(ctx, "app", surf.Bind(surf.CUAKeys()))
func CUAKeys() BindOption {
	return &groupOption{read: cuaBinds}
}

// CocoaKeys is the editing keymap of macOS's text system, a group: a
// text field's caret and delete keys as SPEC §10.2 has them where mod is
// super, whatever the host's mod: super+arrow_left and super+arrow_right
// go to the line's ends, super+arrow_up and super+arrow_down to the
// value's, super+backspace deletes to the line's start, and ctrl+a,
// ctrl+e, ctrl+b, ctrl+f, ctrl+p, ctrl+n, ctrl+h, ctrl+d and ctrl+k move
// and delete as emacs's do. Its source is AppKit's StandardKeyBinding.dict,
// in Apple's "Text System Defaults and Key Bindings":
// https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/EventOverview/TextDefaultsBindings/TextDefaultsBindings.html
//
// The keys only Linux and Windows give a field (ctrl+arrow_left,
// ctrl+backspace, ctrl+home, …) are given back: from a field they reach
// the program, as on macOS, but only where no group on the element binds
// them, and a binding nearer the field still takes them.
//
// What it leaves the host's, where it goes, how it layers, and Unbind
// under it are as CUAKeys says, with CUAKeys and CocoaKeys swapped: on a
// host whose mod is ctrl, ctrl+a stays select all, and a focused button
// no longer scrolls with ctrl+home and ctrl+end.
// surf.Bind(surf.ReadlineKeys(), surf.CocoaKeys()) is macOS's keys where
// they bind and readline's for the rest, ctrl+backspace among them.
func CocoaKeys() BindOption {
	return &groupOption{read: cocoaBinds}
}

// Yield makes every binding of its Bind leave alone a key the program is
// given on the element or around it (a surf.Key's declared key, a prefix,
// a ui.KeyScope's forwarded key): that key stays the program's.
func Yield() BindOption { return yieldOption }

// Override makes every binding of its Bind take its keys even where the
// program is given them around the element: the nearer binding wins
// (SPEC §10.2). On the element itself, a surf.Key's or a KeyScope's key
// stays the program's.
func Override() BindOption { return overrideOption }

type keysOption struct {
	action hotty.Action
	keys   []string
	unbind bool
}

// A groupOption is Group's opts, or a predefined group's bindings, which
// read makes for the frame's platform (the editing keymaps'):
// only the option's maker refers to read, so an app without it doesn't
// link the set.
type groupOption struct {
	opts []BindOption
	read func(*rt.Frame) []rt.Bind
}

type modeOption struct{ yield bool }

var (
	yieldOption    = &modeOption{yield: true}
	overrideOption = &modeOption{yield: false}
)

func (*keysOption) bindOption()  {}
func (*groupOption) bindOption() {}
func (*modeOption) bindOption()  {}

// binds says whether opts hold a binding: an Action, an Unbind, a
// ReadlineKeys, or a group that holds one.
func binds(opts []BindOption) bool {
	for _, o := range opts {
		switch o := o.(type) {
		case *keysOption:
			if o != nil && len(o.keys) > 0 {
				return true
			}
		case *groupOption:
			if o != nil && (o.read != nil || binds(o.opts)) {
				return true
			}
		}
	}
	return false
}

// flatten is a Bind's bindings, in order, canonical for the frame's
// platform, each with its Bind's Yield or Override (a direct Action
// overrides, the rest yield, by default).
func flatten(f *rt.Frame, id string, opts []BindOption, out []rt.Bind) []rt.Bind {
	var mode *modeOption
	for _, o := range opts {
		if m, ok := o.(*modeOption); ok && m != nil {
			if mode != nil && mode != m {
				f.Faults = append(f.Faults, "surf.Bind: the id "+id+" has both Yield and Override in one Bind: the first holds")
				continue
			}
			mode = m
		}
	}
	return flattenIn(f, id, opts, mode, false, out)
}

func flattenIn(f *rt.Frame, id string, opts []BindOption, mode *modeOption, group bool, out []rt.Bind) []rt.Bind {
	for _, o := range opts {
		switch o := o.(type) {
		case *modeOption:
			if group && o != nil {
				f.Faults = append(f.Faults, "surf.Group: the id "+id+" binds a group with Yield or Override in it: set it where the group is bound, in its Bind")
			}
		case *keysOption:
			if o == nil {
				continue
			}
			yield := group || o.unbind
			if mode != nil {
				yield = mode.yield
			}
			for _, k := range o.keys {
				c, ok := f.Platform.ReadKey(k)
				switch {
				case !ok || c == "shift+tab" || c == "escape" || c == "tab" && !o.unbind:
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds the key "+strconv.Quote(k)+", which doesn't parse or keeps its own meaning (SPEC §10.2)")
				case !o.unbind && !bindable(o.action):
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(k)+" to "+strconv.Quote(string(o.action))+", which isn't an action Action takes: program is surf.Key's, and default is surf.Unbind")
				default:
					out = append(out, rt.Bind{Key: c, Action: string(o.action), Yield: yield, Direct: !group})
				}
			}
		case *groupOption:
			if o == nil {
				continue
			}
			if o.read != nil {
				yield := true
				if mode != nil {
					yield = mode.yield
				}
				for _, b := range o.read(f) {
					b.Yield, b.Direct = yield, false
					out = append(out, b)
				}
				continue
			}
			out = flattenIn(f, id, o.opts, mode, true, out)
		}
	}
	return out
}

// bindsOf is an element's bindings (rt.Handlers.Binds), one per key, from
// binds in order: a direct one holds its key over the groups', two direct
// ones of a key to different actions being a fault (the first stays);
// among the groups' the last read wins, but a give-back (giveBack), which
// only fills a key no group binds for real, in any order. A direct binding of a key a Key
// handler declares (keys) is a fault, but to prefix, which gives fn the
// key too; a group's is kept, unwritten (the program has the key), for
// the field's own use of the key when the handler declines it. base is
// CUAKeys' and CocoaKeys' bindings (rt.Handlers.Base), one per key, the last
// read winning, whatever holds the key.
func bindsOf(f *rt.Frame, id string, keys []string, binds []rt.Bind) (out, base []rt.Bind) {
	if len(binds) == 0 {
		return nil, nil
	}
	for _, b := range binds {
		if b.Base {
			if i := slices.IndexFunc(base, func(x rt.Bind) bool { return x.Key == b.Key }); i >= 0 {
				base[i] = b
			} else {
				base = append(base, b)
			}
		}
		if b.Direct && b.Action != string(hotty.Prefix) && slices.Contains(keys, b.Key) {
			f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(b.Key)+", which its surf.Key declares: a key has one use on an element")
			continue
		}
		i := slices.IndexFunc(out, func(x rt.Bind) bool { return x.Key == b.Key })
		switch {
		case i < 0:
			out = append(out, b)
		case out[i].Direct && b.Direct:
			if out[i].Action != b.Action {
				f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(b.Key)+" to "+out[i].Action+" and to "+b.Action+": a key has one use on an element")
			}
		case out[i].Direct:
			// A group's binding after a direct one: the direct one holds.
		case giveBack(b) && !giveBack(out[i]):
			// A give-back fills only a key no group binds for real.
		default:
			// A direct binding over a group's; a later group's over an
			// earlier one's.
			out[i] = b
		}
	}
	return out, base
}

// giveBack says whether b is a give-back: CUAKeys' or CocoaKeys' key of
// the other platform, a group's, bound to program.
func giveBack(b rt.Bind) bool {
	return b.Base && !b.Direct && b.Action == string(hotty.Program)
}

// readlineBinds is hotty.TerminalKeys for the frame's platform, made
// once a platform (rt.Frame.Editing).
func readlineBinds(f *rt.Frame) []rt.Bind {
	if f.Editing == nil {
		f.Editing = new([3][]rt.Bind)
	}
	if f.Editing[2] == nil {
		out := []rt.Bind{}
		m := hotty.ParseKeymap(f.Platform, hotty.TerminalKeys)
		for _, b := range strings.Fields(m.Format()) {
			k, a, _ := strings.Cut(b, "=")
			if c, ok := f.Platform.ReadKey(k); ok {
				out = append(out, rt.Bind{Key: c, Action: a})
			}
		}
		f.Editing[2] = out
	}
	return f.Editing[2]
}

// cuaBinds and cocoaBinds are CUAKeys' and CocoaKeys' bindings for the
// frame's platform, made once a platform (rt.Frame.Editing).
func cuaBinds(f *rt.Frame) []rt.Bind   { return editingFor(f, 0) }
func cocoaBinds(f *rt.Frame) []rt.Bind { return editingFor(f, 1) }

// editingFor is editingTables()[i] for the frame's platform: each key
// as the platform reads it, less every key the host's edit shortcuts hold
// (Platform.Edit, or DefaultEdit(Mod) where it reports none), which stay
// the host's.
func editingFor(f *rt.Frame, i int) []rt.Bind {
	if f.Editing == nil {
		f.Editing = new([3][]rt.Bind)
	}
	if f.Editing[i] == nil {
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
		for _, b := range editingTables()[i] {
			if c, ok := f.Platform.ReadKey(b.Key); ok && !held[c] {
				b.Key = c
				out = append(out, b)
			}
		}
		f.Editing[i] = out
	}
	return f.Editing[i]
}

var (
	editingOnce  sync.Once
	editingTable [2][]rt.Bind
)

// editingTables is SPEC §10.2's two tables of a field's keys by mod, as
// hotty.Resolve has them (no copy of them here): for every key whose
// action differs between a field's default keymap where mod is ctrl and
// where it is super, both without edit shortcuts, CUAKeys' binding ([0]) and
// CocoaKeys' ([1]), each the key's action on its platform or, where that
// platform's field doesn't use the key, program (a give-back). They
// don't depend on the host, so they are made once a process.
func editingTables() *[2][]rt.Bind {
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

// bindable says whether Bind takes the action: one of SPEC §10.2's but
// program, default and insert.
func bindable(a hotty.Action) bool {
	switch a {
	case hotty.Program, hotty.Default, hotty.Insert:
		return false
	}
	return new(hotty.Keymap).Bind("a", a)
}
