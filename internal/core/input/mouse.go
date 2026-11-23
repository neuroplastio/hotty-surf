package input

import (
	uv "github.com/charmbracelet/ultraviolet"
	"neuroplast.io/hotty-go"
)

// MouseButton translates a decoded SGR button to its DOM number.
func MouseButton(button uv.MouseButton) (int, bool) {
	switch {
	case button >= uv.MouseLeft && button <= uv.MouseRight:
		return int(button - uv.MouseLeft), true
	case button >= uv.MouseBackward && button <= uv.MouseButton11:
		return int(button-uv.MouseBackward) + hotty.ButtonBack, true
	default:
		return 0, false
	}
}

// MouseMods translates terminal mouse modifiers to hotty's modifier mask.
func MouseMods(mod uv.KeyMod) int {
	var bits int
	if mod.Contains(uv.ModShift) {
		bits |= hotty.ModShift
	}
	if mod.Contains(uv.ModAlt) {
		bits |= hotty.ModAlt
	}
	if mod.Contains(uv.ModCtrl) {
		bits |= hotty.ModCtrl
	}
	if mod.Contains(uv.ModSuper) {
		bits |= hotty.ModSuper
	}
	return bits
}
