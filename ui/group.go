package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// An InputGroup draws its control's border, and the kit's own class on an
// Input, Textarea or NativeSelect would draw a second (kit rule K9). The
// group marks the ctx its children render in; a ui control under the mark
// leaves its class out, and every ui component that renders children of
// its own clears the mark, so only the group's own control is bare.

type groupKey struct{}

func inGroup(ctx context.Context) bool {
	in, _ := ctx.Value(groupKey{}).(bool)
	return in
}

// bare is class, or "" for a control that is an InputGroup's own.
func bare(ctx context.Context, class string) string {
	if inGroup(ctx) {
		return ""
	}
	return class
}

// scoped renders c with its children (from the ctx it renders in) under
// the group's mark when group, or out of any group: every ui component
// that holds children goes through it, so that only an InputGroup's own
// control is bare. templ clears the children at the top of a component,
// so it is done here, around it.
func scoped(c templ.Component, group bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		kids := templ.GetChildren(ctx)
		if kids == nil || !group && !inGroup(ctx) {
			return c.Render(ctx, w)
		}
		marked := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return kids.Render(context.WithValue(ctx, groupKey{}, group), w)
		})
		return c.Render(templ.WithChildren(ctx, marked), w)
	})
}

// outOfGroup is ctx without an InputGroup's mark.
func outOfGroup(ctx context.Context) context.Context {
	if inGroup(ctx) {
		return context.WithValue(ctx, groupKey{}, false)
	}
	return ctx
}

// plain renders c out of any InputGroup: content a component holds that
// isn't its children (a menu's trigger).
func plain(c templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(outOfGroup(ctx), w)
	})
}
