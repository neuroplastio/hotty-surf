package keydefault

import (
	"github.com/clipperhouse/uax29/v2/graphemes"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyedit"

	"neuroplast.io/hotty-surf/internal/core/rt"
)

// Text is a text field's own use of in.Key, as its keymap (the platform's
// default, then in.Keymaps) has it: hottyedit's edit of in.Value and its
// selection. It is the kit's text fallback, which only what edits text
// links (rt.LinkText): ui's text components, ui.KeyScope and
// surf.Bind; an app without them doesn't carry hottyedit.
func Text(in rt.KeyDefault) rt.KeyResult {
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
