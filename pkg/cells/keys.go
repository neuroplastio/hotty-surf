package cells

import (
	"strings"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/myelin/myelinkey"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// KeyUse is a key Key would take (Program.Keys, Renderer.Keys).
type KeyUse struct {
	// Name is the key as Key takes it (SPEC §10.4's name): "enter",
	// "ctrl+s", a binding's "mod+k" as the Platform reads it.
	Name string
	// Title is the binding's title (surfui.Binding); "" for a key of the
	// focused element's own.
	Title string
	// Own says the focused element takes the key itself: a field's Enter,
	// which commits it and goes on to a binding of Enter; a Select's keys,
	// its open list's, a Cycle's; the Escape that blurs. In a renderer's
	// Keys, the keys the renderer takes ahead of its program, too.
	Own bool
}

// Chars is what the focused element does with a key that types a
// character, with Shift at most: "a", "shift+a", "1", ">" (Program.Keys).
type Chars uint8

const (
	// CharsBound is no element that takes characters: such a key goes to
	// the bindings, as any key does.
	CharsBound Chars = iota
	// CharsPick is a Select, its open list or a Cycle, which takes each
	// such key ahead of the bindings, and goes to the next option that
	// starts with it.
	CharsPick
	// CharsText is a text field, which types them, ahead of the bindings
	// (SPEC §10.2): a host keeps them for it, and Key never hears one.
	CharsText
)

// Keys are the keys Key would take now, in the order it tries them, each
// once, and what the focused element does with characters: the
// program's Keys, with the keys the renderer takes ahead of the program
// first, marked Own. They are for a host that decides a key without
// calling Key (a multiplexer, whose renderer is in another process),
// from what it last heard; they are good after Show, Draw and each input
// call, until the program's model changes off them.
//
// The renderer's own are Tab and Shift+Tab; a focused text field's keys
// (its keymap, hotty.Resolve with the data-keys from the root down to
// it, but those it gives the program: program and prefix, copy and
// paste, cut with nothing selected), with Shift where a key without it
// selects so, and Space, which it types; a button's, a summary's or a link's Space and Enter, with
// Shift or without; and the keys that scroll, while a view from the
// focused element outward (or the page, with nothing focused) has more
// than it shows. A key the focused element's keymap gives the program is
// the program's, where the program lists it. A scrolling key that moves
// nothing goes on to the program (SPEC §5.3), so Own is where Key takes
// it first, not that the program never hears it. Characters are the
// text field's while one is focused (CharsText), else as the program
// says.
func (r *Renderer) Keys() ([]KeyUse, Chars) {
	if r.menu != nil {
		// The menu has the keyboard: only the program's, which are its.
		return r.p.Keys()
	}
	var keys []KeyUse
	seen := map[string]bool{}
	own := func(names ...string) {
		for _, k := range names {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, KeyUse{Name: k, Own: true})
			}
		}
	}
	own("tab", "shift+tab")
	progKeys, chars := r.p.Keys()
	n := r.index[r.focused]
	switch {
	case r.focused == "" || n == nil:
		r.ownScroll(nil, nil, own)
	case surfui.IsTextField(n):
		chars = CharsText
		if st := r.fields[r.focused]; st != nil {
			own(r.fieldKeys(n, st)...)
		}
	default:
		km := hotty.ParseKeymap(r.Platform, strings.Join(r.keymaps(n), " "))
		if presses(n) {
			for _, k := range []string{"space", "enter", "shift+space", "shift+enter"} {
				if !km.Program(k) {
					own(k)
				}
			}
		}
		r.ownScroll(n, km, own)
	}
	for _, u := range progKeys {
		if !seen[u.Name] {
			seen[u.Name] = true
			keys = append(keys, u)
		}
	}
	return keys, chars
}

// ownScroll lists the keys that scroll from n outward (the page with n
// nil), while a view there has more than it shows: those km binds to a
// scroll action, then those a browser scrolls with, but those km gives
// the program.
func (r *Renderer) ownScroll(n *surfui.Node, km *hotty.Keymap, own func(...string)) {
	if !r.scrolls() || !r.canScroll(n) {
		return
	}
	if km != nil {
		for _, b := range bindingsOf(r.Platform, km) {
			if b.a.Scrolls() {
				own(b.key)
			}
		}
	}
	for _, k := range scrollOrder {
		if km == nil || !km.Program(k) {
			own(k)
		}
	}
}

// scrollOrder is scrollKeys' keys, in a fixed order.
var scrollOrder = []string{"arrow_up", "arrow_down", "page_up", "page_down", "space", "shift+space", "home", "end"}

// canScroll reports whether a view from n outward, or the page, has more
// than it shows.
func (r *Renderer) canScroll(n *surfui.Node) bool {
	r.ensure()
	for _, pt := range r.viewsFrom(n) {
		if content, view := r.extent(pt); content > view {
			return true
		}
	}
	return false
}

// fieldKeys are the keys the focused text field n takes: its keymap's,
// but those it gives the program, as fieldKey routes them.
func (r *Renderer) fieldKeys(n *surfui.Node, st *fieldState) []string {
	km := hotty.Resolve(r.Platform, surfui.Tag(n) == "textarea", r.keymaps(n)...)
	_, cuts := selected(&st.ed)
	copies := cuts && r.Copy != nil
	var out []string
	take := func(k string) {
		switch a := km.Lookup(k); {
		case a == "", a == hotty.Paste, a == hotty.Copy && !copies, a == hotty.Cut && !cuts, km.Prefix(k):
		default:
			out = append(out, k)
		}
	}
	bs := bindingsOf(r.Platform, km)
	for _, b := range bs {
		take(b.key)
	}
	// A key with Shift that no binding names is looked up without it
	// (SPEC §10.2): Shift and a move selects.
	for _, b := range bs {
		if s, ok := withShift(b.key); ok {
			take(s)
		}
	}
	// Space types, but is no character to a host (Chars).
	take("space")
	take("shift+space")
	return out
}

// binding is a key a keymap binds, as the Platform reads it, and its
// action.
type binding struct {
	key string
	a   hotty.Action
}

// bindingsOf are km's bindings in the order first bound (Keymap.Format).
func bindingsOf(p hotty.Platform, km *hotty.Keymap) []binding {
	var out []binding
	for _, w := range strings.Fields(km.Format()) {
		i := strings.LastIndexByte(w, '=')
		if i < 0 {
			continue
		}
		if k, ok := p.ReadKey(w[:i]); ok {
			out = append(out, binding{k, hotty.Action(w[i+1:])})
		}
	}
	return out
}

// withShift is k with Shift, for a key that is not a character and has
// no Shift yet.
func withShift(k string) (string, bool) {
	key, err := myelinkey.Parse(k)
	if err != nil {
		return "", false
	}
	mods, _, kind := myelinkey.Split(key)
	if mods&myelinkey.Shift != 0 || kind == myelinkey.Character {
		return "", false
	}
	s, err := myelinkey.Parse("shift+" + k)
	if err != nil {
		return "", false
	}
	return string(s), true
}
