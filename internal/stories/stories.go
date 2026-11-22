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
	"neuroplast.io/hotty-surf/internal/story"
	"neuroplast.io/hotty-surf/ui"
	"neuroplast.io/hotty-surf/ui/alert"
	alertstory "neuroplast.io/hotty-surf/ui/alert/story"
	"neuroplast.io/hotty-surf/ui/avatar"
	avatarstory "neuroplast.io/hotty-surf/ui/avatar/story"
	"neuroplast.io/hotty-surf/ui/badge"
	badgestory "neuroplast.io/hotty-surf/ui/badge/story"
	"neuroplast.io/hotty-surf/ui/breadcrumb"
	breadcrumbstory "neuroplast.io/hotty-surf/ui/breadcrumb/story"
	"neuroplast.io/hotty-surf/ui/button"
	buttonstory "neuroplast.io/hotty-surf/ui/button/story"
	"neuroplast.io/hotty-surf/ui/buttongroup"
	buttongroupstory "neuroplast.io/hotty-surf/ui/buttongroup/story"
	"neuroplast.io/hotty-surf/ui/card"
	cardstory "neuroplast.io/hotty-surf/ui/card/story"
	"neuroplast.io/hotty-surf/ui/checkbox"
	checkboxstory "neuroplast.io/hotty-surf/ui/checkbox/story"
	"neuroplast.io/hotty-surf/ui/combobox"
	comboboxstory "neuroplast.io/hotty-surf/ui/combobox/story"
	"neuroplast.io/hotty-surf/ui/empty"
	emptystory "neuroplast.io/hotty-surf/ui/empty/story"
	"neuroplast.io/hotty-surf/ui/item"
	itemstory "neuroplast.io/hotty-surf/ui/item/story"
	"neuroplast.io/hotty-surf/ui/kbd"
	kbdstory "neuroplast.io/hotty-surf/ui/kbd/story"
	"neuroplast.io/hotty-surf/ui/label"
	labelstory "neuroplast.io/hotty-surf/ui/label/story"
	"neuroplast.io/hotty-surf/ui/progress"
	progressstory "neuroplast.io/hotty-surf/ui/progress/story"
	uiselect "neuroplast.io/hotty-surf/ui/select"
	selectstory "neuroplast.io/hotty-surf/ui/select/story"
	"neuroplast.io/hotty-surf/ui/separator"
	separatorstory "neuroplast.io/hotty-surf/ui/separator/story"
	"neuroplast.io/hotty-surf/ui/skeleton"
	skeletonstory "neuroplast.io/hotty-surf/ui/skeleton/story"
	"neuroplast.io/hotty-surf/ui/textarea"
	textareastory "neuroplast.io/hotty-surf/ui/textarea/story"
	"neuroplast.io/hotty-surf/ui/tooltip"
	tooltipstory "neuroplast.io/hotty-surf/ui/tooltip/story"
)

// Registry returns a ui.Registry with all story components registered.
func Registry() *ui.Registry {
	return ui.NewRegistry().Register(
		alert.Register,
		avatar.Register,
		badge.Register,
		breadcrumb.Register,
		button.Register,
		buttongroup.Register,
		card.Register,
		checkbox.Register,
		combobox.Register,
		empty.Register,
		item.Register,
		kbd.Register,
		label.Register,
		progress.Register,
		uiselect.Register,
		separator.Register,
		skeleton.Register,
		textarea.Register,
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
	avatarstory.Register,
	badgestory.Register,
	breadcrumbstory.Register,
	buttonstory.Register,
	buttongroupstory.Register,
	cardstory.Register,
	checkboxstory.Register,
	comboboxstory.Register,
	emptystory.Register,
	itemstory.Register,
	kbdstory.Register,
	labelstory.Register,
	progressstory.Register,
	selectstory.Register,
	separatorstory.Register,
	skeletonstory.Register,
	textareastory.Register,
	tooltipstory.Register,
)

func add(s Story) {
	reg.Add(s)
}

// All is every story, by ID.
func All() []Story {
	return reg.All()
}

// Of is the stories of a component, by ID.
func Of(component string) []Story {
	return reg.Of(component)
}

func onoff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// BookAll includes inspector component demonstrations alongside the kit's
// coverage stories. They are book examples rather than Basecoat coverage.
// They come in the order they were added, which keeps a component's
// variants in the order its file lists them, as the spike's book did
// (default, destructive, ghost with icon, …); the book sorts the rest.
func BookAll() []Story {
	return append(reg.Ordered(), bookStories...)
}

var bookStories []Story
