package rt

import (
	"context"
	"slices"
	"strings"

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
	Node *html.Node
	// Kind is the focused kit control's, as ui marked it ("button",
	// "text", …), or the native element's kind for a declared key.
	Kind string
	// Declined says every Key handler that declared the key declined it:
	// the field's keymap is its default and the chain's surf.Bind and
	// ReadlineKeys bindings, the declared keys left out.
	Declined                      bool
	Focused                       string
	On                            *Frame // the frame's handlers (Get)
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
	OwnKeys                         string
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

// KeyScopeBoundary marks a display-contents wrapper made for a shared scope
// after the surface runs out of authored ancestors.
const KeyScopeBoundary = "data-hotty-scope-boundary"

// HasKeyBindings reports whether the scope declares a physical key.
func HasKeyBindings(scope *KeyScope) bool {
	return slices.ContainsFunc(scope.Bindings, func(b Binding) bool { return b.Physical != "" })
}

// ScopeKeys returns the enabled keys of one scope in wire order.
func ScopeKeys(scope *KeyScope) []string {
	var out []string
	for key := range scope.Keys {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

// Forwarded returns the enabled keys of the scope and its ancestors.
func Forwarded(s *KeyScope) []string {
	var out []string
	for ; s != nil; s = s.Parent {
		for k := range s.Keys {
			if !slices.Contains(out, k) {
				out = append(out, k)
			}
		}
	}
	slices.Sort(out)
	return out
}

// NativeKind is the row of SPEC §10.2's table n is in: "text" (multiline
// for a textarea or an editing host), "select", "date", "activate", or ""
// for an element that uses no keys itself.
func NativeKind(n *html.Node) (kind string, multiline bool) {
	if hasAttr(n, "contenteditable") && !strings.EqualFold(attr(n, "contenteditable"), "false") {
		return "text", true
	}
	switch n.Data {
	case "textarea":
		return "text", true
	case "select":
		return "select", false
	case "button", "summary":
		return "activate", false
	case "a":
		if hasAttr(n, "href") {
			return "activate", false
		}
	case "input":
		switch t := strings.ToLower(attr(n, "type")); t {
		case "hidden", "file", "color", "range", "image":
			return "", false
		case "checkbox", "radio", "submit", "reset", "button":
			return "activate", false
		case "date", "time", "datetime-local", "month", "week":
			return "date", false
		}
		return "text", false
	}
	return "", false
}

// KeyReader identifies the protocol reader of an element.
func KeyReader(n *html.Node) hotty.Reader {
	if n == nil {
		return hotty.ReaderElement
	}
	kind, multiline := NativeKind(n)
	switch kind {
	case "text":
		if multiline {
			return hotty.ReaderMultiline
		}
		return hotty.ReaderInput
	case "select":
		return hotty.ReaderSelect
	case "date":
		return hotty.ReaderDate
	case "activate":
		return hotty.ReaderButton
	default:
		return hotty.ReaderElement
	}
}

// ResolveKeys resolves the focused element's own and inherited attributes.
// With nothing focused, or for document selection, body stands for the document.
func ResolveKeys(p hotty.Platform, n, body *html.Node, reader hotty.Reader, legacy bool) *hotty.Keymap {
	var inner []string
	name := "data-keys-inner"
	if legacy {
		name = "data-keys"
	}
	if n == nil || reader == hotty.ReaderSelection {
		n = body
		if n != nil {
			inner = append(inner, attr(n, name))
		}
	}
	if n == nil {
		return hotty.Resolve(p, reader, "")
	}
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if v := attr(parent, name); v != "" {
			inner = append(inner, v)
		}
	}
	slices.Reverse(inner)
	return hotty.Resolve(p, reader, attr(n, "data-keys"), inner...)
}
