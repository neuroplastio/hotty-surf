package rt

import (
	"context"

	"neuroplast.io/hotty-go"
)

// KeyScope is one lexical keyboard override. Keys are canonical physical
// names for this frame's platform; Parent is the enclosing override.
type KeyScope struct {
	Keys    map[string]bool
	Handle  func(string) bool
	Parent  *KeyScope
	Default func(KeyDefault) KeyResult
}

// KeyDefault describes a native control to the scope's optional fallback.
// The surface prepares it without linking an editing implementation.
type KeyDefault struct {
	Kind, Key, Value, Type          string
	Checked, Readonly, HasSelection bool
	Selection                       [2]int
	Platform                        hotty.Platform
	Keymaps                         []string
	Options                         []KeyOption
}

type KeyOption struct {
	Value, Label string
	Disabled     bool
}

type KeyResult struct {
	Taken, Changed, Submit, HasSelection bool
	Value                                string
	Checked                              bool
	Selection                            [2]int
}

type keyScopeContext struct{}

func WithKeyScope(ctx context.Context, scope *KeyScope) context.Context {
	return context.WithValue(ctx, keyScopeContext{}, scope)
}

func KeyScopeFrom(ctx context.Context) *KeyScope {
	s, _ := ctx.Value(keyScopeContext{}).(*KeyScope)
	return s
}
