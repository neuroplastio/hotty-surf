package surf

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Bind gives keys on the element to the host's own actions (SPEC §10.2):
// a text field's edit actions, the scroll actions, and prefix (§10.5).
// surf.Key gives keys to the program. Its options are bindings (Action,
// Unbind, Group, Readline) and at most one of Yield or Override, which
// sets how all of them meet the keys the program is given around the
// element. Without either, an Action wins and an Unbind and a group's
// bindings yield. A Bind with no binding is no handler; a nil option is
// none.
//
// The palette's query edits as a terminal's line does, in one line:
//
//	surf.On(ctx, "query", surf.Input(a.SetQuery), surf.Bind(surf.Readline()))
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
// group's bindings in their order). Then that binding's Yield or
// Override decides whether it gives way to the program's keys around the
// element (a surf.Key's declared key, a prefix, a ui.KeyScope's forwarded
// key, on an element around it): a binding that yields is dropped, and
// nothing under it on the element comes back. So a page's mod+k on its
// main still reaches its handler from a field under Readline, where
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

// A BindOption is one of Bind's options. Action, Unbind, Group, Readline,
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
// Readline without ctrl+k is surf.Bind(surf.Readline(),
// surf.Unbind("ctrl+k")).
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

// Readline is the group of the SDK's readline keymap (hotty.TerminalKeys:
// Bubble Tea's ctrl+e, ctrl+k, ctrl+u, ctrl+w, alt+b, alt+f, …), over the
// platform's default; mod+a still selects all. On an element that holds
// fields, it binds them all. A kit Command's or Combobox's field, which
// takes no handler of the app's, gets it from a Bind on an element around
// the component.
func Readline() BindOption {
	return &groupOption{read: readlineBinds}
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
// read makes for the frame's platform (Readline's): only the option's
// maker refers to read, so an app without it doesn't link the set.
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
// Readline, or a group that holds one.
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
// among the groups' the last read wins. A direct binding of a key a Key
// handler declares (keys) is a fault, but to prefix, which gives fn the
// key too; a group's is kept, unwritten (the program has the key), for
// the field's own use of the key when the handler declines it.
func bindsOf(f *rt.Frame, id string, keys []string, binds []rt.Bind) []rt.Bind {
	if len(binds) == 0 {
		return nil
	}
	var out []rt.Bind
	for _, b := range binds {
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
		default:
			// A direct binding over a group's; a later group's over an
			// earlier one's.
			out[i] = b
		}
	}
	return out
}

// readlineBinds is hotty.TerminalKeys for the frame's platform, read
// once a frame.
func readlineBinds(f *rt.Frame) []rt.Bind {
	if f.Readline == nil {
		m := hotty.ParseKeymap(f.Platform, hotty.TerminalKeys)
		for _, b := range strings.Fields(m.Format()) {
			k, a, _ := strings.Cut(b, "=")
			if c, ok := f.Platform.ReadKey(k); ok {
				f.Readline = append(f.Readline, rt.Bind{Key: c, Action: a})
			}
		}
	}
	return f.Readline
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
