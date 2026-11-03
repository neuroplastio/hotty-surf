package surf

import (
	"sort"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-surf/pkg/markup"
)

// The compiled tree, read-only, for a generator: surfgen writes the
// compiled mode from it, the same tree the interpreter walks, so that
// the two draw alike.

type (
	// Template is a template compiled for an owner.
	Template = tmpl
	// Elem is one node of a compiled template.
	Elem = elem
	// ElemKind is what an Elem is.
	ElemKind = elemKind
	// AttrValue is an attribute's value: a literal, or a program.
	AttrValue = value
	// Mod is a modifier and its value.
	Mod = mod
	// TextPart is a literal run of text, or a program.
	TextPart = part
	// EventBinding is a handler bound to an event.
	EventBinding = event
	// Handler is a method call a handler makes: Event says the event
	// fills in an argument, which From says.
	Handler = handler
	// Branch is one branch of a :if chain.
	Branch = branch
	// PropArg is a prop as a use gives it.
	PropArg = propArg
	// PropSlot is a template's prop and its slot.
	PropSlot = propSlot
)

// The kinds of Elem.
const (
	ElemNone      = kNone
	ElemWidget    = kWidget
	ElemText      = kText
	ElemComponent = kComponent
	ElemPartial   = kPartial
	ElemSlot      = kSlot
	ElemBind      = kBind
	ElemIf        = kIf
	ElemEach      = kEach
	ElemEmbed     = kEmbed // its State is its :of, its Mods its key
	ElemMenu      = kMenu  // a widget's Menu: its Kids are its items
	ElemAction    = kAction
	ElemSep       = kSep
)

// Menu is a widget's <menu>, nil for none.
func (e *elem) Menu() *Elem { return e.menu }

// Ord is an <action>'s or a <sep>'s place among its menu's items in the
// source, from 0: its id (MenuAction, MenuSep).
func (e *elem) Ord() int { return e.ord }

// An <action>'s icon, key label, :disabled and do; its label is its Text.
func (e *elem) Icon() AttrValue     { return e.icon }
func (e *elem) KeyLabel() AttrValue { return e.keyLabel }
func (e *elem) Disabled() AttrValue { return e.off }
func (e *elem) Do() Handler         { return e.do }

// After says a <button>'s <icon>, among its Kids, is after its text.
func (e *elem) After() bool { return e.after }

// Templates are the Set's compiled templates, by Key.
func (s *Set) Templates() []*Template {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.tmpls))
	for k := range s.tmpls {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*Template, len(keys))
	for i, k := range keys {
		out[i] = s.tmpls[k]
	}
	return out
}

// Programs are the programs of the template whose Key is key, in the
// order it compiled them.
func (s *Set) Programs(key string) []*cellite.Program {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tmpls[key]; ok {
		return t.progs
	}
	return nil
}

// Key names t in its Set: its name and its owner's.
func (t *tmpl) Key() string { return t.file.name + "\x00" + t.owner.Name }

// Name is t's template name; Path its file.
func (t *tmpl) Name() string { return t.file.name }
func (t *tmpl) Path() string { return t.file.path }

// Owner is the component whose fields and methods t names.
func (t *tmpl) Owner() *Shape { return t.owner }

// Props are t's props and their slots.
func (t *tmpl) Props() []PropSlot { return t.props }

// Root is t's root.
func (t *tmpl) Root() *Elem { return t.root }

// Programs are t's programs, in the order it compiled them.
func (t *tmpl) Programs() []*cellite.Program { return t.progs }

func (p propSlot) Name() string       { return p.name }
func (p propSlot) Type() cellite.Type { return p.typ }
func (p propSlot) Slot() int          { return p.slot }

func (e *elem) Kind() ElemKind             { return e.kind }
func (e *elem) Pos() markup.Pos            { return e.pos }
func (e *elem) Widget() *Widget            { return e.w }
func (e *elem) Args() []AttrValue          { return e.args }
func (e *elem) Mods() []Mod                { return e.mods }
func (e *elem) Text() []TextPart           { return e.text }
func (e *elem) Events() []EventBinding     { return e.evs }
func (e *elem) Kids() []*Elem              { return e.kids }
func (e *elem) List() *cellite.Program     { return e.list }
func (e *elem) Item() int                  { return e.item }
func (e *elem) Index() int                 { return e.idx }
func (e *elem) Body() *Elem                { return e.body }
func (e *elem) Branches() []Branch         { return e.branches }
func (e *elem) Sub() *Template             { return e.sub }
func (e *elem) State() *cellite.Program    { return e.state }
func (e *elem) Props() []PropArg           { return e.props }
func (e *elem) Fills() map[string][]*Elem  { return e.fills }
func (e *elem) SlotName() string           { return e.slotName }
func (e *elem) BindKey() AttrValue         { return e.bindKey }
func (e *elem) BindLabel() AttrValue       { return e.bindLabel }
func (e *elem) BindDo() Handler            { return e.bindDo }
func (e *elem) BindScroll() string         { return e.bindScroll }
func (v value) Given() bool                { return v.set }
func (v value) Lit() cellite.Value         { return v.lit }
func (v value) Pad() [2]int64              { return v.pad }
func (v value) Program() *cellite.Program  { return v.prog }
func (m mod) Attr() *Attr                  { return m.attr }
func (m mod) Value() AttrValue             { return m.v }
func (p part) Lit() string                 { return p.lit }
func (p part) Program() *cellite.Program   { return p.prog }
func (e event) Kind() string               { return e.kind }
func (e event) Handler() Handler           { return e.h }
func (h handler) Method() int              { return h.method }
func (h handler) Args() []*cellite.Program { return h.args }
func (h handler) Event() bool              { return h.ev }
func (b branch) Cond() *cellite.Program    { return b.cond }
func (b branch) Body() *Elem               { return b.body }
func (p propArg) Slot() int                { return p.slot }
func (p propArg) Type() cellite.Type       { return p.typ }
func (p propArg) Value() AttrValue         { return p.v }

// From is where argument i comes from: its program (Args, ArgExpr), or
// the event, which fills it in ($value, $event; Args has nil there).
func (h handler) From(i int) EventArg {
	if h.from == nil {
		return ArgExpr
	}
	return h.from[i]
}
