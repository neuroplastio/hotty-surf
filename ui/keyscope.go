package ui

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/keydefault"
	"neuroplast.io/hotty-surf/internal/rt"
)

// A KeyBinding is one action a [KeyScope] offers: a key it takes, an
// action a host's menu or an app's palette lists, or both.
type KeyBinding struct {
	// Key is the key that runs it, in surf.Key's notation, mod included
	// ("mod+s", "arrow_down"); "" for an action only a pick runs.
	Key string
	// Label is what it does, as a menu and a palette show it: "Save".
	Label string
	// Name is the command a host's menu sends when the user picks it, and
	// what [Pick] takes: "save". "" for an action only its key runs. The
	// bindings of one name on a surface are one action: [Bindings] lists
	// it once, and a pick runs the one nearest the keyboard.
	Name string
	// Disabled is set while it does nothing now: its key goes on as if the
	// scope did not name it, a pick runs nothing, and a menu shows it
	// disabled.
	Disabled bool
	// Run does it, on the surface driver's goroutine. For a key, false
	// declines it: the next enclosing scope that names the key is tried,
	// then the focused control's default, as KeyScope says. A pick
	// reports what it returns.
	Run func() bool
}

// KeyScope gives its children's keys and actions to bindings, and names
// them, so a host's menu and an app's palette list them ([Bindings]).
// A binding's Run is offered its key while the keyboard is inside the
// scope, and runs when a host's menu or a palette picks its Name
// ([Pick]). A true result takes the key. A false result tries the next
// enclosing KeyScope that names it, then the focused control's kit
// default. Keys no scope names keep their native or kit defaults.
//
// The keyboard is inside a scope while it is on one of the scope's kit
// controls, on an element whose surf.On was called inside the scope, or
// on an element inside one of those; and a scope inside no other scope
// has it while nothing on the surface does. Scopes nest: the scope nearest
// the keyboard is asked first, before the kit control's default and any
// surf.Key handler. On a kit control, a scope takes its named keys from
// the control's native editing; on a plain element, it hears only keys
// the element doesn't use itself, as a surf.Key that declares no keys
// does. An element that types
// (a kit Input, Textarea or Combobox input, a plain text input, textarea
// or editing host) keeps its printable keys, with or without shift: no
// scope takes one from it. A scope around one component overrides that
// instance. Scopes never affect another surface.
//
// Keys use surf.Key's notation, including mod ("arrow_down", "mod+s").
// They are exact key names, not patterns or action names. A binding with
// a nil Run or with neither Key nor Name, an invalid key, a key the scope
// already binds, and tab or shift+tab are faults; those bindings are
// ignored. Tab and Shift+Tab are reserved for the host's focus traversal,
// including within a modal. An active modal confines focus and background
// controls do not receive keys; a scope enclosing it applies to its
// focused kit control.
//
// Its children are rendered without adding an HTML wrapper; a scope with
// no children still offers its bindings to a menu and a palette. Run runs
// on the surface driver's goroutine, as other handlers do. A host's open
// native select list can keep keys until it closes (SPEC §10.2).
func KeyScope(bindings []KeyBinding) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		linkText()
		children := templ.GetChildren(ctx)
		p := hotty.Platform{}
		f := rt.From(ctx)
		if f != nil {
			p = f.Platform
		}
		scope := &rt.KeyScope{Keys: map[string]bool{}, Parent: rt.KeyScopeFrom(ctx), Default: keyScopeDefault,
			Canonical: scopeCanonical, Attach: attachScope, Native: rt.NativeKeyDefault, Dispatch: rt.DispatchKeyScope}
		given := make([]KeyBinding, 0, len(bindings))
		byKey := map[string]int{}
		for _, b := range bindings {
			var k string
			switch {
			case b.Run == nil:
				rt.Fault(ctx, fmt.Sprintf("ui.KeyScope: binding %q has no Run", or(b.Name, or(b.Key, b.Label))))
				continue
			case b.Key != "":
				var ok bool
				k, ok = p.ReadKey(b.Key)
				_, dup := byKey[k]
				if !ok || k == "tab" || k == "shift+tab" || dup {
					rt.Fault(ctx, fmt.Sprintf("ui.KeyScope: invalid or duplicate key %q", b.Key))
					continue
				}
			case b.Name == "":
				rt.Fault(ctx, fmt.Sprintf("ui.KeyScope: binding %q has neither Key nor Name", b.Label))
				continue
			}
			if k != "" {
				byKey[k] = len(given)
				if !b.Disabled {
					scope.Keys[k] = true
				}
			}
			given = append(given, b)
			scope.Bindings = append(scope.Bindings, rt.Binding{Key: b.Key, Physical: k, Label: b.Label, Name: b.Name, Disabled: b.Disabled})
		}
		if len(given) == 0 {
			if children == nil {
				return nil
			}
			return children.Render(templ.ClearChildren(ctx), w)
		}
		scope.Handle = func(k string) bool {
			i, ok := byKey[k]
			return ok && given[i].Run()
		}
		scope.Pick = func(i int) bool { return given[i].Run() }
		if f != nil {
			f.Scopes = append(f.Scopes, scope)
		}
		if children == nil {
			return nil
		}
		return children.Render(rt.WithKeyScope(templ.ClearChildren(ctx), scope), w)
	})
}

func scopeCanonical(p hotty.Platform, key string) (string, bool) { return p.ReadKey(key) }

// KitOn registers handlers like surf.On and marks them as kit-owned keys for KeyScope.
func KitOn(ctx context.Context, id string, handlers ...surf.Handler) templ.Attributes {
	return kitOn(ctx, id, handlers...)
}

// ScopeAttrs attaches any enclosing KeyScope metadata to attrs without mutating it.
func ScopeAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	return scopeAttrs(ctx, attrs)
}

func kitOn(ctx context.Context, id string, handlers ...surf.Handler) templ.Attributes {
	f := rt.From(ctx)
	wasRegistered := f != nil && f.Get(id) != nil
	attrs := surf.On(ctx, id, handlers...)
	if f != nil && !wasRegistered {
		if h := f.Get(id); h != nil {
			h.KitKey = true
		}
	}
	return attrs
}

// scopeAttrs forwards the named keys of a kit control and records their
// lexical scopes in the frame. It copies attrs: those supplied by the app
// must never be modified. The marker is read by the surface runtime.
func applyScope(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	if scope := rt.KeyScopeFrom(ctx); scope != nil && scope.Attach != nil {
		return scope.Attach(ctx, attrs, kind)
	}
	return attrs
}

func scopeAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	return applyScope(ctx, attrs, "")
}

// attachScope is referenced by KeyScope itself, so unscoped controls do not
// link the map, sorting and attribute construction used for interception.
// It marks a kit control with the scope it is in, under the frame's
// secret, and keeps its kind in the frame; the surface takes the marker
// off and writes the keys the scopes forward to it (its data-keys).
func attachScope(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	scope, f := rt.KeyScopeFrom(ctx), rt.From(ctx)
	if scope == nil || f == nil || len(rt.Forwarded(scope, kind)) == 0 {
		return attrs
	}
	f.KeyScopes = append(f.KeyScopes, scope)
	f.KeyKinds = append(f.KeyKinds, kind)
	out := make(templ.Attributes, len(attrs)+1)
	for k, v := range attrs {
		out[k] = v
	}
	out["data-hotty-keyscope"] = f.KeyMark + strconv.Itoa(len(f.KeyScopes))
	if kind == "text" || kind == "compound-text" {
		on, _ := out["data-on"].(string)
		parts := strings.Fields(on)
		for _, name := range []string{"input", "caret"} {
			if !slices.Contains(parts, name) {
				parts = append(parts, name)
			}
		}
		out["data-on"] = strings.Join(parts, " ")
	}
	return out
}

func scopeTextAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	linkText()
	return applyScope(ctx, attrs, "text")
}

func scopeNativeAttrs(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	return applyScope(ctx, attrs, kind)
}

// scopeCompoundAttrs are a compound field's (a Command's or a
// Combobox's input): it types, and its kit default is its component's.
func scopeCompoundAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	linkText()
	return applyScope(ctx, attrs, "compound-text")
}

// linkText links the kit's text fallback (rt.LinkText): every component
// with a text field calls it as it renders, and KeyScope does.
func linkText() { rt.LinkText(keydefault.Text) }
