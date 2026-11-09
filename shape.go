package surf

import (
	"io/fs"
	"strings"
	"unicode"

	cellite "neuroplast.io/cel-lite"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

// Func is a helper an expression may call: a Go function, with its
// overloads (cel-lite's).
type Func = cellite.Func

// Shape is what a template can name of a type: a component's or a data
// struct's fields, which expressions read, and a component's methods,
// which handlers call and expressions may call where they return a value.
// surfgen generates a Go type's (docs/guide.md); a dynamic component,
// such as a Lua table, makes its own.
type Shape struct {
	// Name is the type's Go name, the struct's name in expressions'
	// types. A component's template is its kebab name's .surf
	// (AgentRow's agent-row.surf, notices' notices.surf).
	Name string
	// Component says it has a template and methods: a component's, not a
	// data struct's.
	Component bool
	// OneNumber says the host's language has one number type (Lua's,
	// JavaScript's): its numbers come in as an int where whole and
	// within int64, a double otherwise, and one given to a prop
	// declared int or double takes the prop's type (docs/surf.md,
	// "From Lua"). Go's shapes leave it false: their numbers are typed.
	OneNumber bool
	Fields    []Field
	Methods   []Method
}

// Field is one field of a shape, by its template name: the Go name in
// lower camel case (ID id, AgentName agentName).
type Field struct {
	Name string
	Type cellite.Type
	// Go is its Go type's name where that is a named basic type
	// (time.Duration, screens.Level), by its package's name: what a tool
	// shows the value as (the gallery shows a time.Duration as one). ""
	// for any other.
	Go string
}

// Method is one method of a component.
type Method struct {
	Name string
	// Args are its parameters' types, in order. A parameter of EventType
	// takes the event that ran the handler, which only a handler's
	// $event gives it (docs/spec.md, "Handlers").
	Args []cellite.Type
	// Result is what it returns, which makes it callable in an
	// expression; nil for a handler only. A method that takes the event
	// is a handler only, whatever it returns.
	Result *cellite.Type
}

// takesEvent says one of m's parameters is the event's.
func (m *Method) takesEvent() bool {
	for _, t := range m.Args {
		if isEventType(t) {
			return true
		}
	}
	return false
}

// function says m is a function of a template's: it returns a value and
// doesn't take the event.
func (m *Method) function() bool { return m.Result != nil && !m.takesEvent() }

// EventType is the type of a method's parameter that takes the event,
// surfrun.Event: a struct type no template names (no shape's name has a
// dot), which only a handler's $event passes.
var EventType = cellite.StructOf("surfrun.Event")

func isEventType(t cellite.Type) bool {
	return t.Kind == cellite.StructKind && t.Name == EventType.Name
}

// EventValue is ev as a handler's argument: what $event passes, a value
// of EventType, which Instance.Call gets at that parameter's place.
func EventValue(ev surfrun.Event) cellite.Value {
	return cellite.ObjectValue(EventType.Name, eventObject{ev})
}

// EventOf is the event v holds, which $event passed (EventValue); the
// zero Event for any other value.
func EventOf(v cellite.Value) surfrun.Event {
	if o, ok := v.Object().(eventObject); ok {
		return o.ev
	}
	return surfrun.Event{}
}

// eventObject is an event as a value: a handler's argument, never read
// by an expression.
type eventObject struct{ ev surfrun.Event }

func (eventObject) Field(int) cellite.Value { return cellite.NullValue() }

// ComponentType is the type of a surflive.Component as a template holds it:
// a field or a method's result of that Go type, which <embed :of> draws by
// its own Render. Like EventType, a struct type no template names: a
// prop's type that starts with a small letter has no _.
var ComponentType = cellite.StructOf("live_component")

func isComponentType(t cellite.Type) bool {
	return t.Kind == cellite.StructKind && t.Name == ComponentType.Name
}

// ComponentValue is c as a value of ComponentType: null for nil.
func ComponentValue(c surflive.Component) cellite.Value {
	if c == nil {
		return cellite.NullValue()
	}
	return cellite.ObjectValue(ComponentType.Name, componentObject{c})
}

// ComponentOf is the surflive.Component v holds (ComponentValue); nil for any
// other value.
func ComponentOf(v cellite.Value) surflive.Component {
	if o, ok := v.Object().(componentObject); ok {
		return o.c
	}
	return nil
}

// componentObject is a component as a value: drawn by <embed>, never read
// by an expression.
type componentObject struct{ c surflive.Component }

func (componentObject) Field(int) cellite.Value { return cellite.NullValue() }

// Instance is a component's value as its template reads and calls it.
// surfgen generates one for a Go type; a dynamic component makes its own.
type Instance interface {
	// Field is field i of its Shape's Fields.
	cellite.Object
	Shape() *Shape
	// Call runs method m of its Shape's Methods with args, one for each
	// of its parameters (at a parameter of EventType, the EventValue
	// that $event passed), and returns what it returns (null for
	// nothing).
	Call(m int, args []cellite.Value) cellite.Value
}

// Unwrapper is an Instance over a Go value: Unwrap is that value, so
// that the runtime can see what else it is (surfrun.Cells, surfrun.Attacher,
// surfrun.Sizer, surfrun.DensitySetter).
type Unwrapper interface{ Unwrap() any }

// method is m's index in s's methods, or -1.
func (s *Shape) method(name string) int {
	for i, m := range s.Methods {
		if m.Name == name {
			return i
		}
	}
	return -1
}

// TemplateName is the template a component's Go name draws with: its
// kebab name, AgentRow's agent-row.
func TemplateName(goName string) string {
	var b strings.Builder
	rs := []rune(goName)
	for i, r := range rs {
		if unicode.IsUpper(r) {
			// A capital starts a word, unless it continues a run of
			// capitals (ID, HTTPServer's HTTP) that isn't ending.
			if i > 0 && (!unicode.IsUpper(rs[i-1]) || i+1 < len(rs) && unicode.IsLower(rs[i+1])) {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// LowerCamel is a Go name as a template names it: ID id, IDs ids,
// AgentName agentName, HTTPServer httpServer.
func LowerCamel(goName string) string {
	rs := []rune(goName)
	n := 0
	for n < len(rs) && unicode.IsUpper(rs[n]) {
		n++
	}
	if n > 1 && n < len(rs) && string(rs[n:]) != "s" {
		n-- // HTTPServer: the S starts Server
	}
	for i := range n {
		rs[i] = unicode.ToLower(rs[i])
	}
	return string(rs)
}

// TypeOf is t's address, for a Method's Result in generated code.
func TypeOf(t cellite.Type) *cellite.Type { return &t }

// Deref is *p, or T's zero where p is nil: a struct passed by value to a
// handler, in generated code.
func Deref[T any](p *T) T {
	var z T
	if p == nil {
		return z
	}
	return *p
}

// Package is a package's templates and what they name of its types, for a
// tool that draws its components with templates of its own beside them
// (a gallery): surfgen -export writes it as the package's SurfPackage.
type Package struct {
	// Name is the Go package's name.
	Name string
	// Files are its templates.
	Files fs.FS
	// Shapes are its types' shapes, as Load takes them.
	Shapes []*Shape
	// Instance is v as the templates read it, where v is a pointer to one
	// of the package's components; nil for any other value.
	Instance func(v any) Instance
}
