package rt

import (
	"context"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
)

// KeyScope is one lexical keyboard override. Keys are canonical physical
// names for this frame's platform, of its enabled keyed bindings; Parent is
// the enclosing override. Handle is offered a canonical key of Keys.
type KeyScope struct {
	Keys   map[string]bool
	Handle func(string) bool
	// Bindings are the scope's, as ui.KeyScope was given them (its valid
	// ones), and Pick runs the one at an index of them.
	Bindings  []Binding
	Pick      func(int) bool
	Parent    *KeyScope
	Default   func(KeyDefault) KeyResult
	Canonical func(hotty.Platform, string) (string, bool)
	Attach    func(context.Context, templ.Attributes, string) templ.Attributes
	Native    func(*KeyEnvironment, string, func(KeyDefault) KeyResult) bool
	Dispatch  func([]*KeyScope, string, string, string) (bool, bool, func(KeyDefault) KeyResult, func(*KeyEnvironment, string, func(KeyDefault) KeyResult) bool)
}

// Binding is one of a scope's bindings: Key as the app wrote it, and
// Physical its canonical name for the frame's platform ("" for none).
type Binding struct {
	Key, Physical, Label, Name string
	Disabled                   bool
}

// KeyEnvironment exposes only the frame and field state a scoped native
// fallback needs. The fallback itself is linked by a KeyScope instance.
type KeyEnvironment struct {
	Node                          *html.Node
	Focused                       string
	On                            map[string]*Handlers
	Platform                      hotty.Platform
	Values                        map[string]string
	Checked                       map[string]bool
	Revisions                     map[string]int
	Selections, PendingSelections map[string][2]int
	PendingNative                 map[string]hotty.FieldChange
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
