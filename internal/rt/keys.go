package rt

import "context"

// KeyScope is one lexical keyboard override. Keys are canonical physical
// names for this frame's platform; Parent is the enclosing override.
type KeyScope struct {
	Keys   map[string]bool
	Handle func(string) bool
	Parent *KeyScope
}

type keyScopeContext struct{}

func WithKeyScope(ctx context.Context, scope *KeyScope) context.Context {
	return context.WithValue(ctx, keyScopeContext{}, scope)
}

func KeyScopeFrom(ctx context.Context) *KeyScope {
	s, _ := ctx.Value(keyScopeContext{}).(*KeyScope)
	return s
}
