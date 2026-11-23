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
	case button == uv.MouseBackward:
		return hotty.ButtonBack, true
	case button == uv.MouseForward:
		return hotty.ButtonForward, true
	case button == uv.MouseButton10 || button == uv.MouseButton11:
		// DOM button 5 is reserved for a pen eraser.
		return 6 + int(button-uv.MouseButton10), true
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
