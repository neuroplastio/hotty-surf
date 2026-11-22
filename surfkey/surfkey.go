// Package surfkey provides keymap options, action bindings, and platform
// editing presets for surf.Bind.
package surfkey

import (
	"slices"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/core/keybind"
)

// Option is one of surf.Bind's options. Action, Unbind, Group, ReadlineKeys,
// CUAKeys, CocoaKeys, CUAEditShortcuts, CocoaEditShortcuts, Yield, and
// Override make them, and nothing else does.
type Option = keybind.Option

// Action binds keys to one of SPEC §10.2's actions other than program
// (surf.Key's) and default (Unbind's): hotty.WordBackward, hotty.ScrollDown,
// hotty.Prefix, ….
func Action(action hotty.Action, keys ...string) Option {
	return &keybind.KeysOption{Action: action, Keys: slices.Clone(keys)}
}

// Unbind gives keys back to the host's own meaning on the element and inside
// it (SPEC §10.2, key=default).
func Unbind(keys ...string) Option {
	return &keybind.KeysOption{Action: hotty.Default, Keys: slices.Clone(keys), Unbind: true}
}

// Group is a reusable set of key-binding options applied in order.
func Group(opts ...Option) Option {
	return &keybind.GroupOption{Opts: slices.Clone(opts)}
}

// ReadlineKeys is the GNU Readline editing keymap (hotty.TerminalKeys).
func ReadlineKeys() Option {
	return &keybind.GroupOption{Read: keybind.ReadlineBinds}
}

// CUAKeys is the Windows/Linux CUA editing keymap (mod=ctrl).
func CUAKeys() Option {
	return &keybind.GroupOption{Read: keybind.CUABinds}
}

// CocoaKeys is the macOS Cocoa editing keymap (mod=super).
func CocoaKeys() Option {
	return &keybind.GroupOption{Read: keybind.CocoaBinds}
}

// CUAEditShortcuts binds the six CUA edit shortcuts (ctrl+a/c/x/v/z/y).
func CUAEditShortcuts() Option {
	return &keybind.GroupOption{Read: keybind.CUAEditBinds}
}

// CocoaEditShortcuts binds the six macOS edit shortcuts (super+a/c/x/v/z).
func CocoaEditShortcuts() Option {
	return &keybind.GroupOption{Read: keybind.CocoaEditBinds}
}

// Yield makes every binding of its surf.Bind yield to surrounding program keys.
func Yield() Option { return keybind.YieldOption }

// Override makes every binding of its surf.Bind take precedence over
// surrounding program keys.
func Override() Option { return keybind.OverrideOption }
