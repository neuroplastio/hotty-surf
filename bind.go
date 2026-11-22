package surf

import (
	"slices"
	"strconv"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/keybind"
	"neuroplast.io/hotty-surf/internal/core/keydefault"
	"neuroplast.io/hotty-surf/internal/core/rt"
	"neuroplast.io/hotty-surf/surfkey"
)

// Bind gives keys on the element to the host's own actions (SPEC §10.2):
// a text field's edit actions, the scroll actions, and prefix (§10.5).
// Options are constructed via package surfkey (surfkey.Action, surfkey.Unbind,
// surfkey.Group, surfkey.ReadlineKeys, surfkey.CUAKeys, surfkey.CocoaKeys,
// surfkey.CUAEditShortcuts, surfkey.CocoaEditShortcuts, surfkey.Yield,
// surfkey.Override). A Bind with no binding is no handler; a nil option is none.
func Bind(opts ...surfkey.Option) Handler {
	rt.LinkText(keydefault.Text)
	if !keybind.HasBinds(opts) {
		return Handler{}
	}
	return Handler{kind: kindBind, fn: slices.Clone(opts)}
}

func flatten(f *rt.Frame, id string, opts []surfkey.Option, out []rt.Bind) []rt.Bind {
	return keybind.Flatten(f, id, opts, out)
}

// layout is an element's bindings, from flat (rt.Handlers.Flat) in
// order, as writeKeys writes them, under the edit shortcuts in effect
// around the element (in, nil where no edit-shortcut group is), into
// out, base and edits, which it reuses.
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
		case giveBack(b) && !giveBack(out[i]):
		default:
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

func takes(e, c rt.Bind) bool {
	return !giveBack(e) || editAction(c.Action)
}

func editAction(a string) bool {
	switch hotty.Action(a) {
	case hotty.SelectAll, hotty.Copy, hotty.Cut, hotty.Paste, hotty.Undo, hotty.Redo:
		return true
	}
	return false
}

func giveBack(b rt.Bind) bool {
	return b.Base && !b.Direct && b.Action == string(hotty.Program)
}
