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

// KeyScope applies a keyboard override to kit controls in its children.
// keys names the keys to offer fn, using surf.Key notation. When fn takes
// one, the default does not run; otherwise enclosing scopes are tried,
// nearest first, and then the kit default. Unnamed keys keep their defaults.
// The scope adds no wrapper and does not apply to plain HTML elements.
// Empty keys or a nil fn adds no override. Duplicate, invalid, tab and
// shift+tab keys are faults and are ignored. Tab stays focus traversal.
// fn runs on the surface driver's goroutine. A host's open native select
// list can keep keys until it closes.
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
		scope := &rt.KeyScope{Keys: map[string]bool{}, Handle: fn, Parent: rt.KeyScopeFrom(ctx), Default: keyScopeDefault}
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
func scopeAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
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
	return out
}

func scopeTextAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	if rt.KeyScopeFrom(ctx) == nil || rt.From(ctx) == nil {
		return attrs
	}
	out := scopeNativeAttrs(ctx, attrs, "text")
	if _, ok := out["data-hotty-keyscope"]; !ok {
		return out
	}
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
	return out
}

func scopeNativeAttrs(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	out := scopeAttrs(ctx, attrs)
	if rt.KeyScopeFrom(ctx) != nil && rt.From(ctx) != nil && out != nil {
		if _, ok := out["data-hotty-keyscope"]; ok {
			out["data-hotty-keykind"] = kind
		}
	}
	return out
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
	out := scopeTextAttrs(ctx, attrs)
	if _, ok := out["data-hotty-keyscope"]; ok {
		out["data-hotty-keykind"] = "compound-text"
	}
	return out
}
