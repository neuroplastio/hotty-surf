package util

import (
	"context"

	"github.com/a-h/templ"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/core/keydefault"
	"neuroplast.io/hotty-surf/internal/core/rt"
)

// KitOn registers handlers like surf.On and marks them as kit-owned keys for KeyScope.
func KitOn(ctx context.Context, id string, handlers ...surf.Handler) templ.Attributes {
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

func applyScope(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	if scope := rt.KeyScopeFrom(ctx); scope != nil && scope.Attach != nil {
		return scope.Attach(ctx, attrs, kind)
	}
	return attrs
}

// ScopeAttrs attaches any enclosing KeyScope metadata to attrs without mutating it.
func ScopeAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	return applyScope(ctx, attrs, "")
}

// ScopeTextAttrs attaches KeyScope metadata for a text input control.
func ScopeTextAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	rt.LinkText(keydefault.Text)
	return applyScope(ctx, attrs, "text")
}

// ScopeNativeAttrs attaches KeyScope metadata for a native control kind (e.g. "button").
func ScopeNativeAttrs(ctx context.Context, attrs templ.Attributes, kind string) templ.Attributes {
	return applyScope(ctx, attrs, kind)
}

// ScopeCompoundAttrs attaches KeyScope metadata for a compound text input (Combobox, Command).
func ScopeCompoundAttrs(ctx context.Context, attrs templ.Attributes) templ.Attributes {
	rt.LinkText(keydefault.Text)
	return applyScope(ctx, attrs, "compound-text")
}
