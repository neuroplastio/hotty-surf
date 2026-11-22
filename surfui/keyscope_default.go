package surfui

import (
	"strings"

	"neuroplast.io/hotty-surf/internal/core/keydefault"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// keyScopeDefault is a control's own use of a key a KeyScope's binding,
// or every Key handler that declared it, declined. ui links it for both
// (rt.KitDefault): a toggle's and a pick's, and a text field's through
// rt.TextDefault, which the kit's text fields and KeyScope link.
func init() { rt.KitDefault = keyScopeDefault }

// keyScopeDefault does the focused control's own use of in.Key.
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
		// Linked by what edits text (rt.LinkText), not here, so that an
		// app without a text field doesn't carry hottyedit.
		if text := rt.TextDefault(); text != nil {
			return text(in)
		}
	}
	return rt.KeyResult{}
}
