package ui

import (
	"strings"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyedit"
	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// keyScopeDefault is referenced by a KeyScope instance, not registered at
// init, so a program that uses no scopes does not link the editing fallback.
func keyScopeDefault(in rt.KeyDefault) rt.KeyResult {
	name := in.Key
	if i := strings.LastIndexByte(name, '+'); i >= 0 {
		name = name[i+1:]
	}
	plain := in.Key == name || in.Key == "shift+"+name
	switch in.Kind {
	case "checkbox", "radio", "switch":
		if !plain || name != "space" {
			return rt.KeyResult{}
		}
		next, changed := keydefault.Check(in.Checked, in.Kind == "radio")
		return rt.KeyResult{Taken: true, Changed: changed, Checked: next}
	case "select":
		if !plain {
			return rt.KeyResult{}
		}
		opts := make([]keydefault.Option, len(in.Options))
		for i, o := range in.Options {
			opts[i] = keydefault.Option{Value: o.Value, Label: o.Label, Disabled: o.Disabled}
		}
		value, changed, taken := keydefault.Select(opts, in.Value, in.Key, name)
		return rt.KeyResult{Taken: taken, Changed: changed, Value: value}
	case "text", "compound-text":
		return keyScopeTextDefault(in)
	}
	return rt.KeyResult{}
}

func keyScopeTextDefault(in rt.KeyDefault) rt.KeyResult {
	km := hotty.Resolve(in.Platform, in.Type == "textarea", in.Keymaps...)
	f := hottyedit.Field{Value: in.Value, Multiline: in.Type == "textarea", Password: in.Type == "password"}
	positions := []int{0}
	for g := graphemes.FromString(in.Value); g.Next(); {
		positions = append(positions, positions[len(positions)-1]+len(g.Value()))
	}
	selection := in.Selection
	if !in.HasSelection {
		selection = [2]int{len(in.Value), len(in.Value)}
	}
	char := func(b int) int {
		for i, pos := range positions {
			if pos >= b {
				return i
			}
		}
		return len(positions) - 1
	}
	f.Select(char(selection[0]), char(selection[1]))
	action, changed := f.Key(km, in.Key)
	if action == "" || action == hotty.Program || action == hotty.Copy || action == hotty.Cut || action == hotty.Paste {
		return rt.KeyResult{}
	}
	if in.Readonly && changed {
		return rt.KeyResult{Taken: true}
	}
	if action == hotty.Submit {
		return rt.KeyResult{Taken: true, Submit: true}
	}
	anchor, _ := f.Anchor()
	bytesAt := func(i int) int {
		g := graphemes.FromString(f.Value)
		off := 0
		for j := 0; j < i && g.Next(); j++ {
			off += len(g.Value())
		}
		return off
	}
	return rt.KeyResult{Taken: true, Changed: changed, Value: f.Value, HasSelection: true,
		Selection: [2]int{bytesAt(anchor), bytesAt(f.Caret)}}
}
