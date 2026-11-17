package ui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
)

// KeyBinding is one action a [KeyScope] offers: a key the scope takes, an
// action a menu or a palette lists, or both.
type KeyBinding struct {
	// Key is the key that reaches it, in surf.Key's notation, mod
	// included ("mod+s", "arrow_down"); empty for an action no key
	// reaches, which only a pick runs ([Pick]).
	Key string
	// Label is what it does, as a host's menu and an app's palette show
	// it: "Save". Empty leaves it out of the menu's labels.
	Label string
	// Name is the command a pick sends and [Pick] takes ("save"); empty
	// for an action only its key reaches. Names are unique on a surface;
	// the first in [Bindings]' order wins.
	Name string
	// Disabled is set while it does nothing: its key goes on as if the
	// scope did not name it, a pick does nothing, and the menu shows it
	// disabled.
	Disabled bool
}

// KeyScope applies an application keyboard override to the kit controls
// in its children, and names the actions it offers. bindings are its keys
// and actions; fn is offered a binding when its key is pressed while one
// of those controls has the keyboard, or when it is picked ([Pick]). A
// true result takes the key. A false result tries the next enclosing
// KeyScope that names it, then the focused control's kit default. Keys no
// scope names keep their native or kit defaults. fn gets the binding as
// given, so it can switch on Name or Key.
//
// Scopes nest: the scope nearest the focused control is asked first.
// A scope may contain several controls, including a component's internal
// controls. A scope around one component overrides that instance. Scopes
// never affect another surface. Plain HTML elements are not kit controls;
// use surf.On and surf.Key for their program-side handlers.
//
// Keys use surf.Key's notation, including mod ("arrow_down", "mod+s").
// They are exact key names, not patterns or action names. A binding with
// neither Key nor Name, an invalid key, a key the scope already binds, and
// tab or shift+tab are faults; those bindings are ignored. Tab and
// Shift+Tab are reserved for the host's focus traversal, including within
// a modal. An active modal confines focus and background controls do not
// receive keys; a scope enclosing it applies to its focused kit control.
//
// The bindings of every scope rendered are what [Bindings] lists and
// [Menu] gives a host, whether or not a kit control is in the scope.
// No bindings or a nil fn adds no override.
//
// fn runs on the surface driver's goroutine, as other handlers do. Its
// children are rendered without adding an HTML wrapper. A KeyScope only
// handles keys while one of its kit controls has focus. A host's open
// native select list can keep keys until it closes (SPEC §10.2).
func KeyScope(bindings []KeyBinding, fn func(KeyBinding) bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.GetChildren(ctx)
		if len(bindings) == 0 || fn == nil {
			if children == nil {
				return nil
			}
			return children.Render(templ.ClearChildren(ctx), w)
		}
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
			if b.Key != "" {
				var ok bool
				k, ok = p.ReadKey(b.Key)
				_, dup := byKey[k]
				if !ok || k == "tab" || k == "shift+tab" || dup {
					rt.Fault(ctx, fmt.Sprintf("ui.KeyScope: invalid or duplicate key %q", b.Key))
					continue
				}
			} else if b.Name == "" {
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
		scope.Handle = func(k string) bool {
			i, ok := byKey[k]
			return ok && fn(given[i])
		}
		scope.Pick = func(i int) bool { return fn(given[i]) }
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

func kitOn(ctx context.Context, id string, handlers ...surf.Handler) templ.Attributes {
	f := rt.From(ctx)
	wasRegistered := f != nil && f.On[id] != nil
	attrs := surf.On(ctx, id, handlers...)
	if f != nil && !wasRegistered {
		if h := f.On[id]; h != nil {
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
func attachScope(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	scope, f := rt.KeyScopeFrom(ctx), rt.From(ctx)
	if scope == nil || f == nil {
		return attrs
	}
	keys := map[string]bool{}
	for s := scope; s != nil; s = s.Parent {
		for k := range s.Keys {
			keys[k] = true
		}
	}
	if len(keys) == 0 {
		return attrs
	}
	f.KeyScopes = append(f.KeyScopes, scope)
	out := make(templ.Attributes, len(attrs)+2)
	for k, v := range attrs {
		out[k] = v
	}
	if old, ok := out["data-keys"].(string); ok {
		out["data-hotty-default-keys"] = old
	}
	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	sort.Strings(list)
	var bindings []string
	if old, ok := out["data-keys"].(string); ok && old != "" {
		bindings = append(bindings, old)
	}
	for _, key := range list {
		bindings = append(bindings, key+"=program")
	}
	out["data-keys"] = strings.Join(bindings, " ")
	out["data-hotty-keyscope"] = strconv.Itoa(len(f.KeyScopes))
	if kind != "" {
		out["data-hotty-keykind"] = kind
	}
	if kind == "text" || kind == "compound-text" {
		on, _ := out["data-on"].(string)
		parts := strings.Fields(on)
		for _, name := range []string{"input", "caret"} {
			found := false
			for _, part := range parts {
				if part == name {
					found = true
					break
				}
			}
			if !found {
				parts = append(parts, name)
			}
		}
		out["data-on"] = strings.Join(parts, " ")
	}
	return out
}

func scopeTextAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	return applyScope(ctx, attrs, "text")
}

func scopeNativeAttrs(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	return applyScope(ctx, attrs, kind)
}

func scopeWithKeys(ctx context.Context, attrs templ.Attributes, keys string) templ.Attributes {
	if keys != "" {
		out := make(templ.Attributes, len(attrs)+1)
		for k, v := range attrs {
			out[k] = v
		}
		out["data-keys"] = keys
		attrs = out
	}
	return scopeAttrs(ctx, attrs)
}

func scopeCommandAttrs(ctx context.Context, attrs templ.Attributes, keys string) templ.Attributes {
	if keys != "" {
		out := make(templ.Attributes, len(attrs)+1)
		for k, v := range attrs {
			out[k] = v
		}
		out["data-keys"] = keys
		attrs = out
	}
	return applyScope(ctx, attrs, "compound-text")
}
