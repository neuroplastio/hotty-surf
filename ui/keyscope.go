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

// KeyScope applies an application keyboard override to the kit controls
// in its children. keys names the keys the override wants; fn is offered
// only those keys while one of those controls has the keyboard. A true
// result takes the key. A false result tries the next enclosing KeyScope
// that names it, then the focused control's kit default. Keys no scope
// names keep their native or kit defaults.
//
// Scopes nest: the scope nearest the focused control is asked first.
// A scope may contain several controls, including a component's internal
// controls. A scope around one component overrides that instance. Scopes
// never affect another surface. Plain HTML elements are not kit controls;
// use surf.On and surf.Key for their program-side handlers.
//
// Names use surf.Key's notation, including mod ("arrow_down", "mod+s").
// keys are exact key names, not patterns or action names. An empty list or
// nil fn adds no override. Invalid names, duplicate names, and tab or
// shift+tab are faults; those entries are ignored. Tab and Shift+Tab are
// reserved for the host's focus traversal, including within a modal.
// An active modal confines focus and background controls do not receive
// keys; a scope enclosing it applies to its focused kit control.
//
// fn runs on the surface driver's goroutine, as other handlers do. Its
// children are rendered without adding an HTML wrapper. A KeyScope only
// handles keys while one of its kit controls has focus. A host's open
// native select list can keep keys until it closes (SPEC §10.2).
func KeyScope(keys []string, fn func(string) bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.GetChildren(ctx)
		if children == nil {
			return nil
		}
		if len(keys) == 0 || fn == nil {
			return children.Render(templ.ClearChildren(ctx), w)
		}
		p := hotty.Platform{}
		if f := rt.From(ctx); f != nil {
			p = f.Platform
		}
		scope := &rt.KeyScope{Keys: map[string]bool{}, Handle: fn, Parent: rt.KeyScopeFrom(ctx), Default: keyScopeDefault,
			Canonical: scopeCanonical, Attach: attachScope, Native: rt.NativeKeyDefault, Dispatch: rt.DispatchKeyScope}
		for _, key := range keys {
			k, ok := p.ReadKey(key)
			if !ok || k == "tab" || k == "shift+tab" || scope.Keys[k] {
				rt.Fault(ctx, fmt.Sprintf("ui.KeyScope: invalid or duplicate key %q", key))
				continue
			}
			scope.Keys[k] = true
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
