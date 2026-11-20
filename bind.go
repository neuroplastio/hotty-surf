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
// CUAKeys and CocoaKeys, the keys a field doesn't use on that platform,
// and those of CUAEditShortcuts and CocoaEditShortcuts, the other
// platform's edit shortcuts. Its options are bindings (Action, Unbind,
// Group, the editing keymaps ReadlineKeys, CUAKeys and CocoaKeys, and the
// edit-shortcut groups CUAEditShortcuts and CocoaEditShortcuts, which are
// written on the fields inside the element) and at most one of Yield
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
// key no group binds. Then that binding's Yield or Override decides
// whether it gives way to the program's keys around the element (a
// surf.Key's declared key, a prefix, a ui.KeyScope's forwarded key, on an
// element around it): a binding that yields is dropped, and nothing under
// it on the element comes back. So a page's mod+k on its main still
// reaches its handler from a field under ReadlineKeys, where readline
// would delete to the line's end with ctrl+k.
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
// ReadlineKeys, CUAKeys, CocoaKeys, CUAEditShortcuts, CocoaEditShortcuts,
// Yield and Override make them, and nothing else does.
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
// shortcuts hold, and the host's scrolling, unless CUAEditShortcuts or
// CocoaEditShortcuts is in effect on the element, which choose them:
// CUAKeys then leaves out the keys those bind instead of the host's, and
// its binding of one of their keys to an edit action gives way to
// theirs. Outside a field the give-backs reach the program too: on a
// host whose mod is super, a focused button no longer scrolls with
// super+arrow_up and super+arrow_down, where the host did, unless
// another group on the element binds them (home, end and the page keys
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
// no longer scrolls with ctrl+home and ctrl+end, unless another group on
// the element binds them (ReadlineKeys does).
// surf.Bind(surf.ReadlineKeys(), surf.CocoaKeys()) is macOS's keys where
// they bind and readline's for the rest, ctrl+backspace among them.
func CocoaKeys() BindOption {
	return &groupOption{read: cocoaBinds}
}

// CUAEditShortcuts is the edit shortcuts of IBM's Common User Access,
// the Windows and Linux desktop's, a group: SPEC §10.2's edit shortcuts
// where mod is ctrl, whatever the host's mod and whatever it reports as
// its own: ctrl+a selects all, ctrl+c copies, ctrl+x cuts, ctrl+v pastes,
// ctrl+z undoes, and ctrl+shift+z and ctrl+y redo. Its source is SPEC
// §10.2's table, as hotty.DefaultEdit has it.
//
// The keys only macOS gives them (super+a, super+c, super+x, super+v,
// super+z, super+shift+z) are given back: from a field they go on as
// keys it doesn't use, to the terminal's shortcuts and then the program
// (SPEC §10.2), as on Linux. A give-back takes a key only from an edit
// shortcut: where the field would do something else with it, it stays.
// Keys the user added to the host's edit shortcuts stay the host's.
//
// It is for fields only: it is written on each text field inside the
// element that binds it, readonly ones included, and never on that
// element, so a selection in the surface's other text, and the
// terminal's, keep their copy keys (SPEC §11). A binding that changes
// nothing in the field is left out, so on a Linux host with its default
// shortcuts it writes nothing. Copy or cut with nothing selected, or in
// a password field, reaches the program (SPEC §10.2), so ctrl+c in an
// empty field still interrupts.
//
// It is a category of its own beside the editing keymaps, and it decides
// the six edit actions wherever it is in effect. On the element that
// binds it, where it binds a key it wins over ReadlineKeys', CUAKeys' and
// CocoaKeys' binding of that key in either order, and its give-backs
// fill only the keys they leave. On that element and every element
// inside it, those keymaps' binding of a key to an edit action
// (ReadlineKeys' mod+a) gives way where it binds the key, and they leave
// out the keys it binds instead of the host's edit shortcuts: so a
// field's own ReadlineKeys under it gives super+a back on a Mac host.
// Otherwise the nearer binding wins, as for every binding: an element
// between keeps its own Action, and its caret and delete keys. Bind it
// once, on the outermost element of each surface's view. Under it,
// Unbind gives a key back to its binding.
//
// A cells Screen has no clipboard: there the program copies on its own
// key (cells.Screen.Selection), which follows the app's setting as this
// does. surf.Platform still reports the host's edit shortcuts: an app
// that shows Copy's keycap shows its own setting's.
//
//	surf.On(ctx, "app", surf.Bind(surf.ReadlineKeys(), surf.CUAKeys(),
//		surf.CUAEditShortcuts()))
func CUAEditShortcuts() BindOption {
	return &groupOption{read: cuaEditBinds}
}

// CocoaEditShortcuts is the edit shortcuts of macOS's text system, a
// group: SPEC §10.2's edit shortcuts where mod is super, whatever the
// host's: super+a selects all, super+c copies, super+x cuts, super+v
// pastes, super+z undoes and super+shift+z redoes. The keys only Linux
// and Windows give them (ctrl+a, ctrl+c, ctrl+x, ctrl+v, ctrl+z,
// ctrl+shift+z, ctrl+y) are given back, as on macOS: ctrl+c interrupts
// even with text selected, and under CocoaKeys on the same element ctrl+a
// goes to the line's start, as it does on a Mac host, whose ctrl+a is
// no edit shortcut to give back.
//
// A host may keep super+c for itself, and a host that reads keys as
// bytes (a cells Screen, a host inside another terminal) may never see
// it. There the field has no copy key of its own, and only a terminal
// that is the host itself copies the field's selection with its own copy
// (SPEC §11). An app offers it as a choice, not as its default.
//
// Where it goes, what it leaves out, how it layers and Unbind under it
// are as CUAEditShortcuts says, with the two swapped.
func CocoaEditShortcuts() BindOption {
	return &groupOption{read: cocoaEditBinds}
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

// layout is an element's bindings, from flat (rt.Handlers.Flat) in
// order, as writeKeys writes them, under the edit shortcuts in effect
// around the element (in, nil where no edit-shortcut group is), into
// out, base and edits, which it reuses:
//   - out, one per key: a direct one holds its key
//     over the groups', two direct ones of a key to different actions
//     being a fault (the first stays); among the groups' the last read
//     wins, but a give-back (giveBack), which only fills a key no group
//     binds for real, in any order. A direct binding of a key a Key
//     handler declares (keys) is a fault, but to prefix, which gives fn
//     the key too; a group's is kept, unwritten (the program has the
//     key), for the field's own use of the key when the handler declines
//     it. A CUAKeys' or CocoaKeys' binding whose key the edit shortcuts
//     in effect hold (heldAt) is left out, and so is another group's
//     binding of a key to an edit action (ReadlineKeys' mod+a) where an
//     edit-shortcut group in effect, on the element or around it, binds
//     the key: they decide the edit actions.
//   - edits, the edit-shortcut groups' on the element, one per key: one's
//     real binding of a key takes it from the other groups' bindings of
//     it, in either order; a give-back otherwise fills only. A direct
//     binding, and a key a Key handler declares, hold their keys over
//     them.
//   - base, one per key: CUAKeys', CocoaKeys' and the
//     edit-shortcut groups' bindings, layered as the groups' are (a later
//     one winning but a give-back filling only, an edit-shortcut group's
//     as edits are), whatever holds the key: what an Unbind goes back to.
//
// Its faults are s's (nil: none).
func layout(s *Surface, id string, keys []string, flat []rt.Bind, in *tree, out, base, edits []rt.Bind) ([]rt.Bind, []rt.Bind, []rt.Bind) {
	for _, b := range flat {
		if b.Edit {
			edits = fill(edits, b)
		}
	}
	for _, b := range flat {
		if b.Edit {
			continue
		}
		if b.Base && !b.Direct && in.heldAt(flat, b) {
			continue
		}
		if !b.Direct && editAction(b.Action) && (slices.ContainsFunc(edits, func(e rt.Bind) bool { return e.Key == b.Key }) || in.editedAround(b.Key)) {
			// A group's binding to an edit action gives way to the
			// edit-shortcut groups' binding of its key in effect.
			continue
		}
		if b.Base {
			if i := slices.IndexFunc(base, func(x rt.Bind) bool { return x.Key == b.Key }); i >= 0 {
				if !giveBack(b) || giveBack(base[i]) {
					base[i] = b
				}
			} else {
				base = append(base, b)
			}
		}
		if b.Direct && b.Action != string(hotty.Prefix) && slices.Contains(keys, b.Key) {
			if s != nil {
				s.fault("surf.Bind: the id " + id + " binds " + strconv.Quote(b.Key) + ", which its surf.Key declares: a key has one use on an element")
			}
			continue
		}
		i := slices.IndexFunc(out, func(x rt.Bind) bool { return x.Key == b.Key })
		switch {
		case i < 0:
			out = append(out, b)
		case out[i].Direct && b.Direct:
			if out[i].Action != b.Action && s != nil {
				s.fault("surf.Bind: the id " + id + " binds " + strconv.Quote(b.Key) + " to " + out[i].Action + " and to " + b.Action + ": a key has one use on an element")
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
	if len(edits) == 0 {
		return out, base, nil
	}
	for _, e := range edits {
		if i := slices.IndexFunc(base, func(x rt.Bind) bool { return x.Key == e.Key }); i < 0 {
			base = append(base, e)
		} else if takes(e, base[i]) {
			base[i] = e
		}
	}
	edits = slices.DeleteFunc(edits, func(e rt.Bind) bool {
		if slices.Contains(keys, e.Key) {
			return true
		}
		i := slices.IndexFunc(out, func(x rt.Bind) bool { return x.Key == e.Key })
		switch {
		case i < 0:
			return false
		case out[i].Direct || !takes(e, out[i]):
			return true
		}
		out = slices.Delete(out, i, i+1)
		return false
	})
	return out, base, edits
}

// fill lays b over bs, the edit-shortcut groups' bindings so far, one per
// key: a later binding of a key replaces an earlier one, but a give-back,
// which fills only.
func fill(bs []rt.Bind, b rt.Bind) []rt.Bind {
	i := slices.IndexFunc(bs, func(x rt.Bind) bool { return x.Key == b.Key })
	switch {
	case i < 0:
		return append(bs, b)
	case !giveBack(b) || giveBack(bs[i]):
		bs[i] = b
	}
	return bs
}

// takes says whether an edit-shortcut group's binding e takes its key
// from another group's binding c of it on one element: a real one
// always does; a give-back only from a binding to an edit action.
func takes(e, c rt.Bind) bool {
	return !giveBack(e) || editAction(c.Action)
}

// editAction says whether a is one of the six edit shortcuts' actions
// (SPEC §10.2), which an edit-shortcut group decides on its element.
func editAction(a string) bool {
	switch hotty.Action(a) {
	case hotty.SelectAll, hotty.Copy, hotty.Cut, hotty.Paste, hotty.Undo, hotty.Redo:
		return true
	}
	return false
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

// cuaBinds and cocoaBinds are CUAKeys' and CocoaKeys' bindings for the
// frame's platform, made once a platform (rt.Frame.Editing).
func cuaBinds(f *rt.Frame) []rt.Bind   { return editingFor(f, 0) }
func cocoaBinds(f *rt.Frame) []rt.Bind { return editingFor(f, 1) }

// editingFor is editingTables()[i] for the frame's platform: each key
// as the platform reads it, every key the host's edit shortcuts hold
// (Platform.Edit, or DefaultEdit(Mod) where it reports none) marked Held,
// which stays the host's unless an edit-shortcut group is in effect
// (layout).
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
		for _, b := range editingTables()[i] {
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

// cuaEditBinds and cocoaEditBinds are CUAEditShortcuts' and
// CocoaEditShortcuts' bindings: their keys are the same on every
// platform, so they are made once a process. Reading one tells the frame
// which keys its fields resolve (rt.Frame.EditKeys).
func cuaEditBinds(f *rt.Frame) []rt.Bind   { return editShortcutsFor(f, 0) }
func cocoaEditBinds(f *rt.Frame) []rt.Bind { return editShortcutsFor(f, 1) }

func editShortcutsFor(f *rt.Frame, i int) []rt.Bind {
	t := editShortcutTables()
	f.EditKeys = t.keys
	return t.binds[i]
}

var (
	editShortcutsOnce  sync.Once
	editShortcutsTable struct {
		binds [2][]rt.Bind
		keys  []string
	}
)

// editShortcutTables is SPEC §10.2's edit shortcuts where mod is ctrl
// ([0], CUAEditShortcuts') and where it is super ([1],
// CocoaEditShortcuts'), as hotty.DefaultEdit has them, in SPEC's order of
// the actions (DefaultEdit is a map): each table's own keys, then the
// other's, given back (program). keys are all thirteen.
func editShortcutTables() *struct {
	binds [2][]rt.Bind
	keys  []string
} {
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
			t.binds[i] = append(t.binds[i], own[i]...)
			for _, b := range own[1-i] {
				b.Action = string(hotty.Program)
				t.binds[i] = append(t.binds[i], b)
			}
			for _, b := range own[i] {
				t.keys = append(t.keys, b.Key)
			}
		}
	})
	return &editShortcutsTable
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
