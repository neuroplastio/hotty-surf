// Package stories is ui's stories: each component in one state, with the
// little state a story keeps of its own. ui's tests render them (their
// looks in hotty-blitz and Chromium, and by keyboard in cells), the
// coverage test holds them to ui/coverage.tsv, and the book (L3) shows
// them.
//
// A story's view writes what its controls hold as text (a <p
// class="text-muted-foreground"> line), so a test in cells can read that
// the keys did what they should.
package stories

import (
	"neuroplast.io/hotty-surf/internal/book/story"
	"neuroplast.io/hotty-surf/surfui"
	"neuroplast.io/hotty-surf/surfui/alert"
	alertstory "neuroplast.io/hotty-surf/surfui/alert/story"
	"neuroplast.io/hotty-surf/surfui/alertdialog"
	alertdialogstory "neuroplast.io/hotty-surf/surfui/alertdialog/story"
	"neuroplast.io/hotty-surf/surfui/avatar"
	avatarstory "neuroplast.io/hotty-surf/surfui/avatar/story"
	"neuroplast.io/hotty-surf/surfui/badge"
	badgestory "neuroplast.io/hotty-surf/surfui/badge/story"
	"neuroplast.io/hotty-surf/surfui/breadcrumb"
	breadcrumbstory "neuroplast.io/hotty-surf/surfui/breadcrumb/story"
	"neuroplast.io/hotty-surf/surfui/button"
	buttonstory "neuroplast.io/hotty-surf/surfui/button/story"
	"neuroplast.io/hotty-surf/surfui/buttongroup"
	buttongroupstory "neuroplast.io/hotty-surf/surfui/buttongroup/story"
	"neuroplast.io/hotty-surf/surfui/card"
	cardstory "neuroplast.io/hotty-surf/surfui/card/story"
	"neuroplast.io/hotty-surf/surfui/chart"
	chartstory "neuroplast.io/hotty-surf/surfui/chart/story"
	"neuroplast.io/hotty-surf/surfui/checkbox"
	checkboxstory "neuroplast.io/hotty-surf/surfui/checkbox/story"
	codestory "neuroplast.io/hotty-surf/surfui/code/story"
	"neuroplast.io/hotty-surf/surfui/combobox"
	comboboxstory "neuroplast.io/hotty-surf/surfui/combobox/story"
	"neuroplast.io/hotty-surf/surfui/command"
	commandstory "neuroplast.io/hotty-surf/surfui/command/story"
	"neuroplast.io/hotty-surf/surfui/dialog"
	dialogstory "neuroplast.io/hotty-surf/surfui/dialog/story"
	"neuroplast.io/hotty-surf/surfui/drawer"
	drawerstory "neuroplast.io/hotty-surf/surfui/drawer/story"
	"neuroplast.io/hotty-surf/surfui/empty"
	emptystory "neuroplast.io/hotty-surf/surfui/empty/story"
	"neuroplast.io/hotty-surf/surfui/eventlog"
	eventlogstory "neuroplast.io/hotty-surf/surfui/eventlog/story"
	"neuroplast.io/hotty-surf/surfui/field"
	fieldstory "neuroplast.io/hotty-surf/surfui/field/story"
	"neuroplast.io/hotty-surf/surfui/icon"
	iconstory "neuroplast.io/hotty-surf/surfui/icon/story"
	"neuroplast.io/hotty-surf/surfui/iconbutton"
	iconbuttonstory "neuroplast.io/hotty-surf/surfui/iconbutton/story"
	"neuroplast.io/hotty-surf/surfui/input"
	inputstory "neuroplast.io/hotty-surf/surfui/input/story"
	"neuroplast.io/hotty-surf/surfui/inputgroup"
	inputgroupstory "neuroplast.io/hotty-surf/surfui/inputgroup/story"
	"neuroplast.io/hotty-surf/surfui/item"
	itemstory "neuroplast.io/hotty-surf/surfui/item/story"
	"neuroplast.io/hotty-surf/surfui/kbd"
	kbdstory "neuroplast.io/hotty-surf/surfui/kbd/story"
	"neuroplast.io/hotty-surf/surfui/label"
	labelstory "neuroplast.io/hotty-surf/surfui/label/story"
	"neuroplast.io/hotty-surf/surfui/listitem"
	listitemstory "neuroplast.io/hotty-surf/surfui/listitem/story"
	"neuroplast.io/hotty-surf/surfui/menu"
	menustory "neuroplast.io/hotty-surf/surfui/menu/story"
	"neuroplast.io/hotty-surf/surfui/navlist"
	navliststory "neuroplast.io/hotty-surf/surfui/navlist/story"
	"neuroplast.io/hotty-surf/surfui/popover"
	popoverstory "neuroplast.io/hotty-surf/surfui/popover/story"
	"neuroplast.io/hotty-surf/surfui/progress"
	progressstory "neuroplast.io/hotty-surf/surfui/progress/story"
	"neuroplast.io/hotty-surf/surfui/radio"
	radiostory "neuroplast.io/hotty-surf/surfui/radio/story"
	uiselect "neuroplast.io/hotty-surf/surfui/select"
	selectstory "neuroplast.io/hotty-surf/surfui/select/story"
	"neuroplast.io/hotty-surf/surfui/separator"
	separatorstory "neuroplast.io/hotty-surf/surfui/separator/story"
	"neuroplast.io/hotty-surf/surfui/skeleton"
	skeletonstory "neuroplast.io/hotty-surf/surfui/skeleton/story"
	"neuroplast.io/hotty-surf/surfui/slider"
	sliderstory "neuroplast.io/hotty-surf/surfui/slider/story"
	uiswitch "neuroplast.io/hotty-surf/surfui/switch"
	switchstory "neuroplast.io/hotty-surf/surfui/switch/story"
	"neuroplast.io/hotty-surf/surfui/symbol"
	symbolstory "neuroplast.io/hotty-surf/surfui/symbol/story"
	"neuroplast.io/hotty-surf/surfui/table"
	tablestory "neuroplast.io/hotty-surf/surfui/table/story"
	"neuroplast.io/hotty-surf/surfui/tabs"
	tabsstory "neuroplast.io/hotty-surf/surfui/tabs/story"
	"neuroplast.io/hotty-surf/surfui/textarea"
	textareastory "neuroplast.io/hotty-surf/surfui/textarea/story"
	"neuroplast.io/hotty-surf/surfui/toast"
	toaststory "neuroplast.io/hotty-surf/surfui/toast/story"
	"neuroplast.io/hotty-surf/surfui/tooltip"
	tooltipstory "neuroplast.io/hotty-surf/surfui/tooltip/story"
)

// Registry returns a surfui.Registry with all story components registered.
func Registry() *surfui.Registry {
	return surfui.NewRegistry().Register(
		alert.Register,
		alertdialog.Register,
		avatar.Register,
		badge.Register,
		breadcrumb.Register,
		button.Register,
		buttongroup.Register,
		card.Register,
		chart.Register,
		checkbox.Register,
		combobox.Register,
		command.Register,
		dialog.Register,
		drawer.Register,
		empty.Register,
		eventlog.Register,
		field.Register,
		icon.Register,
		iconbutton.Register,
		input.Register,
		inputgroup.Register,
		item.Register,
		kbd.Register,
		label.Register,
		listitem.Register,
		menu.Register,
		navlist.Register,
		popover.Register,
		progress.Register,
		radio.Register,
		uiselect.Register,
		separator.Register,
		skeleton.Register,
		slider.Register,
		uiswitch.Register,
		symbol.Register,
		table.Register,
		tabs.Register,
		textarea.Register,
		toast.Register,
		tooltip.Register,
	)
}

// Story is an alias for story.Story so callers can migrate incrementally.
type Story = story.Story

// Stories returns a story.Registry containing every registered story.
func Stories() *story.Registry {
	return reg
}

var reg = story.NewRegistry().Register(
	alertstory.Register,
	alertdialogstory.Register,
	avatarstory.Register,
	badgestory.Register,
	breadcrumbstory.Register,
	buttonstory.Register,
	buttongroupstory.Register,
	cardstory.Register,
	chartstory.Register,
	checkboxstory.Register,
	codestory.Register,
	comboboxstory.Register,
	commandstory.Register,
	dialogstory.Register,
	drawerstory.Register,
	emptystory.Register,
	eventlogstory.Register,
	fieldstory.Register,
	iconstory.Register,
	iconbuttonstory.Register,
	inputstory.Register,
	inputgroupstory.Register,
	itemstory.Register,
	kbdstory.Register,
	labelstory.Register,
	listitemstory.Register,
	menustory.Register,
	navliststory.Register,
	popoverstory.Register,
	progressstory.Register,
	radiostory.Register,
	selectstory.Register,
	separatorstory.Register,
	skeletonstory.Register,
	sliderstory.Register,
	switchstory.Register,
	symbolstory.Register,
	tablestory.Register,
	tabsstory.Register,
	textareastory.Register,
	toaststory.Register,
	tooltipstory.Register,
)

// All is every coverage story, by ID.
func All() []Story {
	return reg.All()
}

// Of is the coverage stories of a component, by ID.
func Of(component string) []Story {
	return reg.Of(component)
}

// BookAll returns every story shown in the storybook, in registration order.
func BookAll() []Story {
	return reg.Ordered()
}
