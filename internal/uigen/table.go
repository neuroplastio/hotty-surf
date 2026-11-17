package main

// A Comp is a component Basecoat makes from CSS alone, one element: a row
// of the table.
type Comp struct {
	Name     string // its Go name: Checkbox
	File     string // Basecoat's file, components/<File>.css
	Doc      string // the function's doc, after its name
	PropsDoc string // more of the Props' doc, if any
	AttrsDoc string // the Attrs field's doc, if not the usual one
	Tag      string
	Void     bool   // no children (an input)
	Text     string // the field that is its text, not children (a textarea's Value)
	Slot     string // its data-slot
	Class    string // its own class, before Props.Class
	Bare     bool   // in an InputGroup, it leaves its own class out (the group draws it)
	Group    bool   // its children are an InputGroup's: a ui control among them is bare
	Fields   []Field
	Attrs    []Attr // in the markup's order, after data-slot
	Checks   []Check
}

// A Field is one of the Props' fields, in the struct's order (Attrs and
// Class come last, in every Props).
type Field struct {
	Name, Type, Doc string
}

// An Attr is one attribute the element writes: a constant (no Field), or
// a field's.
type Attr struct {
	Name    string
	Field   string // the field it comes from
	Value   string // a constant's value; for a bool field, what true writes ("" is a boolean attribute)
	Expr    string // a Go expression for its value, written when not ""
	IfUnset bool   // a constant written only when Attrs has no attribute of this name
	Default string // a string field's value written for ""
	Always  bool   // a string field's, written even when ""
}

// A Check faults a value the component doesn't take.
type Check struct {
	Field string
	Kind  string // variant, size, enum or call
	List  string // the Go list of the values it takes (check.go); for call, the Go call that checks

}

var table = []Comp{
	{
		Name: "Button", File: "button", Tag: "button", Slot: "button", Class: "btn",
		Doc: `is Basecoat's button, holding its children:

	@ui.Button(ui.ButtonProps{Attrs: surf.On(ctx, "save", surf.Click(a.Save))}) {
		Save
	}`,
		AttrsDoc: `Attrs are more attributes for the button element: its handlers, surf.On(ctx, "save", surf.Click(a.Save)), and any other. A type here replaces type="button"; any other attribute it writes is a fault here.`,
		Fields: []Field{
			{"Variant", "Variant", "Variant is Primary, Secondary, Outline, Ghost, Destructive or Link."},
			{"Size", "Size", "Size is Medium, XSmall, Small or Large."},
			{"IconOnly", "bool", "IconOnly makes the button a square that holds only an icon."},
			{"Disabled", "bool", ""},
		},
		Attrs: []Attr{
			{Name: "type", Value: "button", IfUnset: true},
			{Name: "disabled", Field: "Disabled"},
			{Name: "data-variant", Field: "Variant"},
			{Name: "data-size", Expr: "buttonSize(p)"},
		},
		Checks: []Check{{"Variant", "variant", "buttonVariants"}, {"Size", "size", "buttonSizes"}},
	},
	{
		Name: "Badge", File: "badge", Tag: "span", Slot: "badge", Class: "badge",
		Doc: "is Basecoat's badge, holding its children.",
		Fields: []Field{
			{"Variant", "Variant", "Variant is Primary, Secondary, Outline, Ghost, Destructive or Link."},
		},
		Attrs:  []Attr{{Name: "data-variant", Field: "Variant"}},
		Checks: []Check{{"Variant", "variant", "badgeVariants"}},
	},
	{
		Name: "Card", File: "card", Tag: "div", Slot: "card", Class: "card",
		Doc: `is Basecoat's card. Its parts are plain elements in its children, as Basecoat's markup has them: <header> (an <h2> and a <p>), <section> and <footer>.`,
		Fields: []Field{
			{"Size", "Size", "Size is Medium or Small (data-size): Small draws its padding and gaps tighter."},
		},
		Attrs:  []Attr{{Name: "data-size", Field: "Size"}},
		Checks: []Check{{"Size", "size", "cardSizes"}},
	},

	// Forms (B1).
	{
		Name: "Checkbox", File: "checkbox", Tag: "input", Void: true, Slot: "checkbox", Class: "input",
		Doc: `is Basecoat's check box, a native one: the host toggles it, and a surf.Check handler in Attrs hears it. Its label is beside it, in a Label or a Field:

	@ui.Label(ui.LabelProps{}) {
		@ui.Checkbox(ui.CheckboxProps{Checked: a.done,
			Attrs: surf.On(ctx, "done", surf.Check(a.setDone))})
		Done
	}`,
		Fields: []Field{
			{"Checked", "bool", "Checked is whether it is checked, which the surface sets on the host when it differs from what the host last reported."},
			{"Disabled", "bool", ""},
			{"Invalid", "bool", "Invalid marks it as failing its form's check (aria-invalid), in the destructive colour."},
		},
		Attrs: []Attr{
			{Name: "type", Value: "checkbox"},
			{Name: "checked", Field: "Checked"},
			{Name: "disabled", Field: "Disabled"},
			{Name: "aria-invalid", Field: "Invalid", Value: "true"},
		},
	},
	{
		Name: "Radio", File: "radio", Tag: "input", Void: true, Slot: "radio", Class: "input",
		Doc: `is Basecoat's radio button, a native one: the host checks it, unchecking the others of its Name, and a surf.Check handler in Attrs hears it. A group of them goes in an element with role="radiogroup", which Basecoat lays out as a column. Each has its own surf.On id, and its handler knows the value it stands for.`,
		Fields: []Field{
			{"Name", "string", "Name makes it one of the radios of that name, of which one at a time is checked."},
			{"Checked", "bool", "Checked is whether it is the one checked, which the surface sets on the host when it differs from what the host last reported."},
			{"Disabled", "bool", ""},
			{"Invalid", "bool", "Invalid marks it as failing its form's check (aria-invalid), in the destructive colour."},
		},
		Attrs: []Attr{
			{Name: "type", Value: "radio"},
			{Name: "name", Field: "Name"},
			{Name: "checked", Field: "Checked"},
			{Name: "disabled", Field: "Disabled"},
			{Name: "aria-invalid", Field: "Invalid", Value: "true"},
		},
	},
	{
		Name: "Switch", File: "switch", Tag: "input", Void: true, Slot: "switch", Class: "input",
		Doc: `is Basecoat's switch: a native check box drawn as a switch (role="switch"), which the host toggles; a surf.Check handler in Attrs hears it.`,
		Fields: []Field{
			{"Checked", "bool", "Checked is whether it is on, which the surface sets on the host when it differs from what the host last reported."},
			{"Size", "Size", "Size is Medium or Small."},
			{"Disabled", "bool", ""},
			{"Invalid", "bool", "Invalid marks it as failing its form's check (aria-invalid), in the destructive colour."},
		},
		Attrs: []Attr{
			{Name: "type", Value: "checkbox"},
			{Name: "role", Value: "switch"},
			{Name: "checked", Field: "Checked"},
			{Name: "data-size", Field: "Size"},
			{Name: "disabled", Field: "Disabled"},
			{Name: "aria-invalid", Field: "Invalid", Value: "true"},
		},
		Checks: []Check{{"Size", "size", "switchSizes"}},
	},
	{
		Name: "Input", File: "input", Tag: "input", Void: true, Slot: "input", Class: "input", Bare: true,
		Doc: `is Basecoat's text input, a native one: the host edits it, and a surf.Input handler in Attrs hears each edit:

	@ui.Input(ui.InputProps{Value: a.name, Placeholder: "Name",
		Attrs: surf.On(ctx, "name", surf.Input(a.setName))})`,
		Fields: []Field{
			{"Type", "string", `Type is "" or "text", "email", "password", "number", "tel", "url", "search", "date", "datetime-local", "month", "week" or "time" (its type attribute, none for ""). A check box and a radio are components of their own.`},
			{"Value", "string", `Value is its text. The input always writes it, so the surface sets the host's field whenever Value differs from what the host last reported: an app that keeps the text in its state and passes it back stays in step with the user's typing, and one that sets it to "" clears the field.`},
			{"Placeholder", "string", ""},
			{"Disabled", "bool", ""},
			{"Invalid", "bool", "Invalid marks it as failing its form's check (aria-invalid), in the destructive colour."},
		},
		Attrs: []Attr{
			{Name: "type", Field: "Type"},
			{Name: "value", Field: "Value", Always: true},
			{Name: "placeholder", Field: "Placeholder"},
			{Name: "disabled", Field: "Disabled"},
			{Name: "aria-invalid", Field: "Invalid", Value: "true"},
		},
		Checks: []Check{{"Type", "enum", "inputTypes"}},
	},
	{
		Name: "Textarea", File: "textarea", Tag: "textarea", Text: "Value", Slot: "textarea", Class: "textarea", Bare: true,
		Doc: `is Basecoat's textarea, a native one holding Value as its text: the host edits it, and a surf.Input handler in Attrs hears each edit. It is at least four lines high, and grows with its text where the host sizes a field to its content.`,
		Fields: []Field{
			{"Value", "string", "Value is its text, which the surface sets on the host whenever it differs from what the host last reported, as an Input's."},
			{"Placeholder", "string", ""},
			{"Disabled", "bool", ""},
			{"Invalid", "bool", "Invalid marks it as failing its form's check (aria-invalid), in the destructive colour."},
		},
		Attrs: []Attr{
			{Name: "placeholder", Field: "Placeholder"},
			{Name: "disabled", Field: "Disabled"},
			{Name: "aria-invalid", Field: "Invalid", Value: "true"},
		},
	},
	{
		Name: "Label", File: "label", Tag: "label", Slot: "label", Class: "label",
		Doc: `is Basecoat's label, holding its children: its text, and the control it labels when that is inside it. It dims in a disabled Field (FieldProps.Disabled).`,
		Fields: []Field{
			{"For", "string", "For is the id of the control it labels when that isn't inside it: the id the control's surf.On gave it."},
		},
		Attrs: []Attr{{Name: "for", Field: "For"}},
	},
	{
		Name: "Field", File: "field", Tag: "div", Slot: "field", Class: "field",
		Doc: `is Basecoat's field: a control with its label, description and error, which are plain elements in its children, as Basecoat's markup has them. A <label> (or a Label), the control, a <p> describing it and a <p role="alert"> for its error; in a horizontal field, a <section> holds the label and the description beside the control:

	@ui.Field(ui.FieldProps{Invalid: a.emailErr != ""}) {
		<label for="email">Email</label>
		@ui.Input(ui.InputProps{Type: "email", Value: a.email,
			Invalid: a.emailErr != "",
			Attrs:   surf.On(ctx, "email", surf.Input(a.setEmail))})
		<p role="alert">{ a.emailErr }</p>
	}`,
		Fields: []Field{
			{"Orientation", "string", `Orientation is "" or "vertical" (the label above the control), "horizontal" (beside it) or "responsive" (beside it where the field's group is wide enough): its data-orientation, none for "".`},
			{"Invalid", "bool", "Invalid writes the field's text in the destructive colour (data-invalid); the control takes its own Invalid."},
			{"Disabled", "bool", "Disabled dims its label and title (data-disabled); the control takes its own Disabled."},
		},
		Attrs: []Attr{
			{Name: "data-orientation", Field: "Orientation"},
			{Name: "data-invalid", Field: "Invalid", Value: "true"},
			{Name: "data-disabled", Field: "Disabled", Value: "true"},
		},
		Checks: []Check{{"Orientation", "enum", "fieldOrientations"}},
	},
	{
		Name: "FieldSet", File: "field", Tag: "fieldset", Slot: "fieldset", Class: "fieldset",
		Doc: `is Basecoat's fieldset: fields under a heading, all plain elements or Fields in its children. A <legend> (data-variant="label" sets it as a label is, rather than as a heading), a <p> describing the set, and its Fields, or a <div role="group"> of them.`,
	},
	{
		Name: "InputGroup", File: "input-group", Tag: "div", Slot: "input-group", Class: "input-group", Group: true,
		Doc: `is Basecoat's input group: one control with addons beside it, InputGroupAddons, in its children. The group draws the control's border, so an Input, a Textarea or a NativeSelect directly in it writes no class of its own, as Basecoat's markup has it:

	@ui.InputGroup(ui.InputGroupProps{}) {
		@ui.Input(ui.InputProps{Type: "search", Value: a.q, Placeholder: "Search",
			Attrs: surf.On(ctx, "q", surf.Input(a.search))})
		@ui.InputGroupAddon(ui.InputGroupAddonProps{Align: "inline-start"}) {
			@ui.Icon("search", "")
		}
	}`,
	},
	{
		Name: "InputGroupAddon", File: "input-group", Tag: "span", Slot: "input-group-addon",
		Doc: `is one of an InputGroup's addons, holding its children: an icon, text, a Kbd or a small Button.`,
		Fields: []Field{
			{"Align", "string", `Align is where it goes: "" or "inline-start" (before the control, on its line), "inline-end" (after it), "block-start" or "block-end" (above or below it, the group's width). surftest fails any other.`},
		},
		Attrs:  []Attr{{Name: "data-align", Field: "Align", Default: "inline-start"}},
		Checks: []Check{{"Align", "enum", "addonAligns"}},
	},

	// Display (B2).
	{
		Name: "Alert", File: "alert", Tag: "div", Slot: "alert", Class: "alert",
		Doc: `is Basecoat's alert (role="alert"). Its parts are plain elements in its children, as Basecoat's markup has them: an icon, its title (an <h2> or a <strong>), a <section> of <p>s or a list, and a <footer> for a button:

	@ui.Alert(ui.AlertProps{Variant: ui.Destructive}) {
		@ui.Icon("circle-alert", "")
		<h2>Not saved</h2>
		<section><p>The disk is full.</p></section>
	}`,
		Fields: []Field{
			{"Variant", "Variant", "Variant is Primary (Basecoat's default) or Destructive."},
		},
		Attrs: []Attr{
			{Name: "role", Value: "alert"},
			{Name: "data-variant", Field: "Variant"},
		},
		Checks: []Check{{"Variant", "variant", "alertVariants"}},
	},
	{
		Name: "Avatar", File: "avatar", Tag: "span", Slot: "avatar", Class: "avatar",
		Doc: `is Basecoat's avatar: a round picture of someone. Its parts are plain elements in its children, as Basecoat's markup has them: an <img> (which the host loads as it loads any image, SPEC §7), a <span> with what shows until it has (initials, an icon), and a <span class="avatar-badge"> for a status dot:

	@ui.Avatar(ui.AvatarProps{}) {
		<img src={ u.Photo } alt={ u.Name }/>
		<span>{ u.Initials }</span>
	}`,
		Fields: []Field{
			{"Size", "Size", "Size is Medium, Small or Large (data-size)."},
		},
		Attrs:  []Attr{{Name: "data-size", Field: "Size"}},
		Checks: []Check{{"Size", "size", "avatarSizes"}},
	},
	{
		Name: "AvatarGroup", File: "avatar", Tag: "div", Slot: "avatar-group", Class: "avatar-group",
		Doc: `is Basecoat's avatar group: Avatars side by side, overlapping, in its children, and after them, if the group shows only some, a plain <span data-count> with the rest's count ("+3"), which Basecoat draws as one more avatar.`,
	},
	{
		Name: "Empty", File: "empty", Tag: "div", Slot: "empty", Class: "empty",
		Doc: `is Basecoat's empty state: what a view shows with nothing in it. Its parts are plain elements in its children, as Basecoat's markup has them: a <header> (a <figure> with an icon or an image, an <h3> and a <p>), then a <section> or a <footer> for its buttons.`,
	},
	{
		Name: "Item", File: "item", Tag: "div", Slot: "item", Class: "item",
		Doc: `is Basecoat's item: one row of a list, with a <figure> (an icon or an image), a <section> (an <h3> and a <p>) and an <aside> (its buttons) as plain elements in its children. ItemGroup holds several.`,
		Fields: []Field{
			{"Variant", "Variant", "Variant is Primary (no border, Basecoat's default) or Outline (a border)."},
			{"Muted", "bool", "Muted draws it on the muted background (data-variant=\"muted\"); with Outline it is a fault."},
			{"Size", "Size", "Size is Medium, Small or XSmall (data-size): its padding and gaps."},
		},
		Attrs: []Attr{
			{Name: "data-variant", Expr: "itemVariant(p)"},
			{Name: "data-size", Field: "Size"},
		},
		Checks: []Check{{"Variant", "variant", "itemVariants"}, {"Size", "size", "itemSizes"}, {"", "call", `checkItem(ctx, p)`}},
	},
	{
		Name: "ItemGroup", File: "item", Tag: "div", Slot: "item-group", Class: "item-group", Doc: `is Basecoat's item group: Items in a column, in its children, with a plain <hr> between two where a line parts them.`,
	},
	{
		Name: "Skeleton", File: "skeleton", Tag: "div", Slot: "skeleton", Class: "skeleton",
		Doc: `is Basecoat's skeleton: a muted, pulsing block standing for something still loading, its size in Class ("h-4 w-[250px]", "size-12 rounded-full").`,
	},
}
