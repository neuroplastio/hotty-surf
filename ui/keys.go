package ui

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// Action is a semantic keyboard action for interactive kit components.
// Specialized components may define additional Action constants as needed.
type Action string

const (
	ActDown     Action = "down"
	ActUp       Action = "up"
	ActLeft     Action = "left"
	ActRight    Action = "right"
	ActFirst    Action = "first"
	ActLast     Action = "last"
	ActPageUp   Action = "page_up"
	ActPageDown Action = "page_down"
	ActConfirm  Action = "confirm"
	ActToggle   Action = "toggle"
	ActCancel   Action = "cancel"
	ActOpen     Action = "open"
)

// KeyMap maps semantic Actions to the hotty key names that trigger them
// (e.g. "arrow_down", "j", "ctrl+n", "enter", "escape").
//
// When used as an override layer (in WithKeys or a component's Props.Keys):
//   - an omitted action inherits from the layer below,
//   - a non-empty slice replaces that action's keys,
//   - an empty slice []string{} explicitly unbinds that action.
type KeyMap map[Action][]string

// DefaultKeys returns a fresh copy of the kit's default component keybindings.
func DefaultKeys() KeyMap {
	return KeyMap{
		ActDown:     {"arrow_down"},
		ActUp:       {"arrow_up"},
		ActLeft:     {"arrow_left"},
		ActRight:    {"arrow_right"},
		ActFirst:    {"home"},
		ActLast:     {"end"},
		ActPageUp:   {"page_up"},
		ActPageDown: {"page_down"},
		ActConfirm:  {"enter"},
		ActToggle:   {"space"},
		ActCancel:   {"escape"},
	}
}

// VimKeys returns a KeyMap adding hjkl navigation alongside arrow keys.
func VimKeys() KeyMap {
	return KeyMap{
		ActDown:  {"arrow_down", "j"},
		ActUp:    {"arrow_up", "k"},
		ActLeft:  {"arrow_left", "h"},
		ActRight: {"arrow_right", "l"},
	}
}

// EmacsKeys returns a KeyMap adding ctrl+n/p/b/f navigation alongside arrow keys.
func EmacsKeys() KeyMap {
	return KeyMap{
		ActDown:  {"arrow_down", "ctrl+n"},
		ActUp:    {"arrow_up", "ctrl+p"},
		ActLeft:  {"arrow_left", "ctrl+b"},
		ActRight: {"arrow_right", "ctrl+f"},
	}
}

type keysCtxKey struct{}

// WithKeys returns a child context that layers overrides on top of any
// inherited KeyMap (and DefaultKeys). Pass it at a surface root or any subtree.
func WithKeys(ctx context.Context, overrides KeyMap) context.Context {
	if len(overrides) == 0 {
		return ctx
	}
	base := KeysOf(ctx)
	for a, ks := range overrides {
		base[a] = slices.Clone(ks)
	}
	return context.WithValue(ctx, keysCtxKey{}, base)
}

// KeysOf returns a fresh copy of the effective KeyMap for ctx
// (DefaultKeys merged with any WithKeys layers).
func KeysOf(ctx context.Context) KeyMap {
	out := DefaultKeys()
	if ctx == nil {
		return out
	}
	if km, ok := ctx.Value(keysCtxKey{}).(KeyMap); ok {
		for a, ks := range km {
			out[a] = slices.Clone(ks)
		}
	}
	return out
}

// ResolvedKeys is a component's effective keybinding table for one render.
type ResolvedKeys struct {
	keys  KeyMap
	byKey map[string]Action
}

// ResolveKeys merges DefaultKeys → ctx WithKeys → local overrides for the
// specified actions.
func ResolveKeys(ctx context.Context, local KeyMap, actions ...Action) ResolvedKeys {
	return resolveKeys(ctx, local, false, actions)
}

// ResolveTextKeys is like ResolveKeys for components whose focus lives in a
// text input (such as Combobox or Command): bare printable keys inherited from
// ctx (e.g. hjkl from VimKeys) are filtered out so they still type into the
// input, while modified/special keys and explicit local overrides apply.
func ResolveTextKeys(ctx context.Context, local KeyMap, actions ...Action) ResolvedKeys {
	return resolveKeys(ctx, local, true, actions)
}

func resolveKeys(ctx context.Context, local KeyMap, textInput bool, actions []Action) ResolvedKeys {
	inherited := KeysOf(ctx)
	km := make(KeyMap, len(actions))
	byKey := make(map[string]Action, len(actions))
	for _, a := range actions {
		ks, hasLocal := local[a]
		if !hasLocal {
			ks = inherited[a]
		}
		var kept []string
		for _, k := range ks {
			if k == "" {
				continue
			}
			if textInput && !hasLocal && isBarePrintable(k) {
				continue
			}
			kept = append(kept, k)
			byKey[k] = a
		}
		km[a] = kept
	}
	return ResolvedKeys{keys: km, byKey: byKey}
}

// Match returns the semantic Action bound to key, if any.
func (r ResolvedKeys) Match(key string) (Action, bool) {
	a, ok := r.byKey[key]
	return a, ok
}

// Declare returns the keys to pass to surf.Key(fn, keys...) for the given
// active actions. Standard navigation keys that hosts already forward
// (arrows, home, end, page_up, page_down, escape) are omitted so only keys
// needing host interception (enter, space, custom keys) are declared.
func (r ResolvedKeys) Declare(actions ...Action) []string {
	var out []string
	for _, a := range actions {
		for _, k := range r.keys[a] {
			if k == "" || slices.Contains(out, k) {
				continue
			}
			switch k {
			case "arrow_down", "arrow_up", "arrow_left", "arrow_right", "home", "end", "page_up", "page_down", "escape":
				continue
			}
			out = append(out, k)
		}
	}
	return out
}

func isBarePrintable(k string) bool {
	if strings.Contains(k, "+") {
		return false
	}
	r, size := utf8.DecodeRuneInString(k)
	return size == len(k) && r >= 0x20 && r != 0x7f
}

// BindKeys returns a surf.Bind handler for any host text-editing actions
// (converted from hotty.Action, e.g. ui.Action(hotty.WordBackward)) present
// in ctx's KeyMap or local overrides. Widget-only actions (ui.ActDown, etc.)
// are ignored. Returns a no-op Handler when no host actions are bound.
func BindKeys(ctx context.Context, local KeyMap) (surf.Handler, bool) {
	km := KeysOf(ctx)
	for a, ks := range local {
		km[a] = ks
	}
	var opts []surf.BindOption
	for a, ks := range km {
		ha := hotty.Action(a)
		if !isHostEditAction(ha) || len(ks) == 0 {
			continue
		}
		opts = append(opts, surf.Action(ha, ks...))
	}
	if len(opts) == 0 {
		return surf.Handler{}, false
	}
	return surf.Bind(surf.Group(opts...)), true
}

func isHostEditAction(a hotty.Action) bool {
	switch a {
	case hotty.CharBackward, hotty.CharForward,
		hotty.WordBackward, hotty.WordForward,
		hotty.LineStart, hotty.LineEnd,
		hotty.DeleteCharBackward, hotty.DeleteCharForward,
		hotty.DeleteWordBackward, hotty.DeleteWordForward,
		hotty.DeleteToLineStart, hotty.DeleteToLineEnd,
		hotty.LinePrevious, hotty.LineNext,
		hotty.PageUp, hotty.PageDown,
		hotty.InputStart, hotty.InputEnd,
		hotty.SelectAll, hotty.Newline, hotty.Submit,
		hotty.Copy, hotty.Cut, hotty.Paste, hotty.Undo, hotty.Redo:
		return true
	}
	return false
}
