package ui

import (
	"context"
	_ "embed"
	"fmt"
	"io"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// contextMenuCSS is ContextMenu's, part of Stylesheet.
//
//go:embed context_menu.css
var contextMenuCSS string

// ContextMenuProps are a ContextMenu's attributes.
type ContextMenuProps struct {
	// ID names its elements; "" is "context-menu". Two on one surface need
	// IDs of their own.
	ID string
	// Items are the menu's, as a DropdownMenu's: groups, roles, shortcuts
	// shown, OnSelect.
	Items []MenuItem
	// Class is the area's, for the app's own.
	Class string
}

// ContextMenuState is what a ContextMenu remembers between renders:
// whether it is open, where, the item highlighted, and the element to give
// the keyboard back to. The zero value is closed.
type ContextMenuState struct {
	menu DropdownMenuState
	// at is where the menu opened, from the area's top left cell; placed
	// says a press placed it (a key opens it under the area).
	at     hotty.Cell
	placed bool
	// back is the element that had the keyboard when it opened.
	back string
	// into asks for the keyboard on the menu at render intoAt.
	into   bool
	intoAt uint64
}

// ContextMenu is a menu of actions on an area of the app's: its children.
// A secondary press in the area (surf.Secondary: the mouse's right button,
// a long press) opens it where the pointer was, with no item highlighted;
// shift+f10 or the Menu key (context_menu) on an element in it opens it
// under the area, the first item highlighted. Its items are a
// DropdownMenu's, and so are its keys while it is open (the arrows, Home,
// End, Enter, Space, Escape). It takes the keyboard while open; a pick or
// Escape gives it back to the element that had it, and a press outside the
// menu closes it.
//
// The area is a block (a card, a pane, a whole list or table),
// wrapped in an element of the kit's, and the menu's items act on it.
// A menu per row of a list or table is not this component's: the app
// puts surf.Secondary on its rows, keeps which row and where, and
// draws a ui.Menu at that cell itself.
//
// A host sends the key that opens a menu as it does every key the focused
// element doesn't use, and a host with no secondary press sends none, so
// the keys always open it. A key opens it under the area, not under the
// focused element: a focus event carries no area (a proposal for hotty's
// SPEC). It doesn't flip at the surface's edges: a host placing popovers
// within the surface is the same proposal's. In cells the menu is drawn
// after the area, as a DropdownMenu's is after its button.
//
// It is the surface's one open popup, as a DropdownMenu is: one opening
// closes the other.
func ContextMenu(st *ContextMenuState, p ContextMenuProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		v := &contextView{st: st, p: p, id: or(p.ID, "context-menu")}
		v.m = newMenuView(&st.menu, DropdownMenuProps{ID: v.id, Items: p.Items, Class: p.Class})
		return contextMenu(v, templ.GetChildren(ctx)).Render(templ.ClearChildren(ctx), w)
	})
}

// contextView is one render of a ContextMenu, over its menu's view.
type contextView struct {
	st *ContextMenuState
	p  ContextMenuProps
	id string
	m  *menuView
	// focused is where the keyboard was as this render began.
	focused string
}

func (v *contextView) join(ctx context.Context) string {
	v.m.join(ctx)
	v.focused = focusedOf(ctx)
	return ""
}

// press is the area's secondary press: the menu opens at the pointer.
func (v *contextView) press(at hotty.Cell, area hotty.Area) {
	v.st.at, v.st.placed = hotty.Cell{}, false
	if area != (hotty.Area{}) {
		v.st.at = hotty.Cell{Col: max(0, at.Col-area.Col), Row: max(0, at.Row-area.Row)}
		v.st.placed = true
	}
	v.open(pickNone)
}

// open opens the menu and gives it the keyboard at the next render.
func (v *contextView) open(initial uint8) {
	if !v.st.menu.open {
		v.st.back = v.focused
	}
	v.m.close(false)
	v.m.open(initial)
	v.st.into, v.st.intoAt = true, v.m.n+1
}

// areaKey is the area's: the keys that open a context menu.
func (v *contextView) areaKey(k string) bool {
	if k != "shift+f10" && k != "context_menu" {
		return false
	}
	v.st.at, v.st.placed = hotty.Cell{}, false
	v.open(pickFirst)
	return true
}

// menuKey is the open menu's: a DropdownMenu's keys, and the opening keys
// close it, as a native context menu's do.
func (v *contextView) menuKey(k string) bool {
	if !v.st.menu.open {
		return false
	}
	if k == "shift+f10" || k == "context_menu" {
		v.m.close(true)
		return true
	}
	return v.m.key(k)
}

func (v *contextView) outside() { v.m.close(false) }

// placement is the open menu's place in the area, in cells.
func (v *contextView) placement() templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf("top:calc(var(--hotty-cell-h) * %d);left:calc(var(--hotty-cell-w) * %d)", v.st.at.Row, v.st.at.Col))
}

// areaAttrs are the area's handlers: the press, and the keys from inside.
func (v *contextView) areaAttrs(ctx context.Context) templ.Attributes {
	return kitOn(ctx, v.id, surf.Secondary(v.press), surf.Key(v.areaKey))
}

// menuAttrs are the menu's: its keys, the pointer leaving it, and while
// open a press outside it.
func (v *contextView) menuAttrs(ctx context.Context) templ.Attributes {
	hs := []surf.Handler{surf.Key(v.menuKey), surf.Hover(v.m.leave)}
	if v.st.menu.open {
		hs = append(hs, surf.Outside(v.outside))
	}
	return kitOn(ctx, v.id+"-menu", hs...)
}

// into is whether this render gives the menu the keyboard.
func (v *contextView) into() bool { return oneShot(&v.st.into, v.st.intoAt, v.m.n) }

// back is whether this render gives the keyboard back, and where.
func (v *contextView) backTo() string {
	if v.m.refocus() && v.st.back != "" && v.st.back != v.id+"-menu" {
		return v.st.back
	}
	return ""
}

// focusedOf is the element the keyboard is on as the render begins.
func focusedOf(ctx context.Context) string {
	if f := rt.From(ctx); f != nil {
		return f.Focused()
	}
	return ""
}
