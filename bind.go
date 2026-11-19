package surf

import (
	"slices"
	"strconv"
	"strings"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// Bind declares keys, on the element, that the host does something with
// itself: SPEC §10.2's actions other than program, which is [Key]'s, and
// default, which is declining (a Key handler's false). No handler hears
// such a key, but for a prefix. What it binds are its options:
//   - [Keys]: keys bound to one action;
//   - a group: [Readline], the SDK's readline keymap, or an app's own
//     [Group];
//   - [Yield] or [Override]: how all of them meet the keys the program is
//     given there.
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
//     handlers (a field's ctrl+x and the key that follows). One no handler
//     takes is the program's own keymap's (an app's leader,
//     surf.Bind(surf.Keys(hotty.Prefix, "mod+k")) on its main), and the
//     program ends it with Surface.Focus. A printable prefix still types in
//     a field inside, as a declared key does.
//   - a scroll action (hotty.ScrollDown, hotty.ScrollPageUp, …) on an
//     element that isn't a field: while the keyboard is on it, or inside it
//     outside a field, a key it doesn't use scrolls the nearest element
//     that scrolls (SPEC §10.2, scrolling keys).
//
// Yield and override. The program is given a key on an element by a Key
// handler's declaration, a prefix or a ui.KeyScope's forwarded key. A
// binding that yields leaves such a key alone, on the element and on every
// element around it, so the farther binding holds: a page's mod+k on its
// main still reaches its handler from a field under Readline, where
// readline would delete to the line's end with ctrl+k. A binding that
// overrides is written anyway, the nearer binding winning (SPEC §10.2).
// Keys given to Bind directly override; every binding in a group yields.
// A Yield or Override option sets it for every binding of its Bind or
// Group, nested groups included, but for a nested group that sets its
// own. On the element itself a Key handler's declared key stays the
// program's either way: an overriding binding of it is a fault, but to
// hotty.Prefix, which still gives fn the key.
//
// Layering, on one element: bindings are read in order, Bind by Bind and
// option by option. A Keys given to Bind directly replaces a group's
// binding of its key, and a later group's binding replaces an earlier
// group's, as data-keys values layered. Two direct Keys of one key to
// different actions are a fault, and the first stays; one key bound twice
// to one action is one binding. Across elements SPEC's chain holds: the
// nearer binding of a key wins, where it is written.
//
// These are faults, each binding dropped: an action that isn't SPEC's, or
// is program, default or insert; a key that doesn't parse, and tab,
// shift+tab and escape, which keep their meaning; Yield and Override in
// one call; an edit action on a control that isn't a text field (a
// button, a select, a check box, a date field); a multi-line one bound
// directly on an input (a group's is harmless there); a scroll action on
// a text field. An edit action on an element that holds no field (yet) is
// harmless, and no fault.
//
// The surface writes the bindings into the element's data-keys (SPEC
// §10.2), as it writes Key's, one per key. A Bind is the host's action,
// not one of ui.Bindings'.
func Bind(opts ...BindOption) Handler {
	rt.LinkText(keydefault.Text)
	if len(opts) == 0 {
		return Handler{}
	}
	return Handler{kind: kindBind, fn: opts}
}

// A BindOption is one of [Bind]'s options: [Keys], [Group], [Readline],
// [Yield] and [Override] make them. The set is closed.
type BindOption interface{ bindOption() }

// Keys binds keys to action, in [Key]'s notation, read for the host's
// platform: surf.Keys(hotty.ScrollDown, "j", "arrow_down"). Given to Bind
// directly it overrides; in a [Group] it yields (see Bind).
func Keys(action hotty.Action, keys ...string) BindOption {
	return keysOption{action: action, keys: keys}
}

// Group is a set of bindings given whole, an app's or the kit's: Keys,
// other groups, and a Yield or Override for all of them. Its bindings
// yield unless it says Override. An app defines one once and binds it
// where it is needed:
//
//	var pager = surf.Group(
//		surf.Keys(hotty.ScrollDown, "j", "arrow_down"),
//		surf.Keys(hotty.ScrollUp, "k", "arrow_up"),
//	)
//
//	surf.On(ctx, "log", surf.Bind(pager))
//
// A group with no bindings binds nothing.
func Group(opts ...BindOption) BindOption {
	return groupOption{opts: opts}
}

// Readline is the group of the SDK's readline keymap, hotty.TerminalKeys:
// Bubble Tea's ctrl+e, ctrl+k, ctrl+u, ctrl+w, alt+b, alt+f, … over the
// platform's default; mod+a still selects all. Its bindings yield, as a
// group's do: a key the program is given on the field or around it stays
// the program's (vault/knowledge/field-keys.md, 2026-11-04);
// surf.Bind(surf.Readline(), surf.Override()) binds them anyway. A Keys
// beside it changes one of its keys. Bound on what holds fields, it is
// every field's inside; a kit Command's or Combobox's field, which takes
// no handler of the app's, gets it from a Bind on an element around the
// component. On a control that isn't a text field (a button, a select)
// it is a fault.
func Readline() BindOption {
	return groupOption{read: readlineBinds}
}

// Yield makes the bindings of the Bind or Group it is in leave alone the
// keys the program is given on the element or around it (see Bind).
func Yield() BindOption { return modeOption(yieldMode) }

// Override makes the bindings of the Bind or Group it is in bind the keys
// the program is given around the element too, the nearer binding winning
// (see Bind).
func Override() BindOption { return modeOption(overrideMode) }

type keysOption struct {
	action hotty.Action
	keys   []string
}

// A groupOption is Group's opts, or a predefined group's bindings, which
// read makes for the frame's platform (Readline's): only the option's
// maker refers to read, so an app without it doesn't link the set.
type groupOption struct {
	opts []BindOption
	read func(*rt.Frame) []rt.Bind
}

type modeOption int8

const (
	noMode modeOption = iota
	yieldMode
	overrideMode
)

func (keysOption) bindOption()  {}
func (groupOption) bindOption() {}
func (modeOption) bindOption()  {}

// flatten appends to out the bindings opts make, in order, canonical for
// the frame's platform: mode is the enclosing call's (noMode for none),
// and group says opts are a group's, whose bindings yield by default.
func flatten(f *rt.Frame, id string, opts []BindOption, mode modeOption, group bool, out []rt.Bind) []rt.Bind {
	own := noMode
	for _, o := range opts {
		if m, ok := o.(modeOption); ok && m != noMode {
			if own != noMode && own != m {
				f.Faults = append(f.Faults, "surf.Bind: the id "+id+" has both Yield and Override in one call: the first holds")
				continue
			}
			own = m
		}
	}
	if own != noMode {
		mode = own
	}
	yield := group
	if mode != noMode {
		yield = mode == yieldMode
	}
	for _, o := range opts {
		switch o := o.(type) {
		case keysOption:
			for _, k := range o.keys {
				c, ok := f.Platform.ReadKey(k)
				switch {
				case !ok || c == "shift+tab" || c == "escape" || c == "tab":
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds the key "+strconv.Quote(k)+", which doesn't parse or keeps its own meaning (SPEC §10.2)")
				case !bindable(o.action):
					f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(k)+" to "+strconv.Quote(string(o.action))+", which isn't an action Bind takes: program is surf.Key's, default is declining")
				default:
					out = append(out, rt.Bind{Key: c, Action: string(o.action), Yield: yield, Group: group})
				}
			}
		case groupOption:
			if o.read != nil {
				gy := true
				if mode != noMode {
					gy = mode == yieldMode
				}
				for _, b := range o.read(f) {
					b.Yield, b.Group = gy, true
					out = append(out, b)
				}
				continue
			}
			out = flatten(f, id, o.opts, mode, true, out)
		}
	}
	return out
}

// bindsOf is an element's bindings (rt.Handlers.Binds), from binds in
// order, one per key: a direct Keys's replaces a group's of its key, and
// a later group's an earlier group's; two direct ones of a key to
// different actions are a fault, and the first stays. A key a Key handler
// declares (keys) is the program's: a yielding binding of it is kept
// for the field's own use of a declined key but not written, and an
// overriding one is a fault, but to prefix, which gives fn the key too.
func bindsOf(f *rt.Frame, id string, keys []string, binds []rt.Bind) []rt.Bind {
	if len(binds) == 0 {
		return nil
	}
	var out []rt.Bind
	for _, b := range binds {
		if b.Action != string(hotty.Prefix) && slices.Contains(keys, b.Key) && !b.Yield {
			f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(b.Key)+", which its surf.Key declares: a key has one use on an element")
			continue
		}
		// A yielding one stays, unwritten (the program has the key): it
		// is the field's edit of the key when the handler declines it.
		i := slices.IndexFunc(out, func(x rt.Bind) bool { return x.Key == b.Key })
		switch {
		case i < 0:
			out = append(out, b)
		case out[i].Action == b.Action:
			if !b.Group {
				out[i].Group = false
			}
		case out[i].Group:
			// A direct binding replaces a group's; a later group's an
			// earlier group's.
			out[i] = b
		case !b.Group:
			f.Faults = append(f.Faults, "surf.Bind: the id "+id+" binds "+strconv.Quote(b.Key)+" to "+out[i].Action+" and to "+b.Action+": a key has one use on an element")
		}
		// A group's binding after a direct one of its key: the direct one
		// stays.
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
