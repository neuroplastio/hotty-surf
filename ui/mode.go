package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Mode is the target rendition a component renders for.
type Mode uint8

const (
	// ModeHotty is an interactive HTML/CSS surface on a hotty host
	// (hottyterm, xterm-addon-hotty, hotty run). It is the zero value.
	ModeHotty Mode = iota
	// ModeTUI is an interactive character-grid TUI in a terminal without a
	// hotty host, driven by cells.Screen (focus, hover, pointer, keys).
	ModeTUI
	// ModeANSI is non-interactive styled terminal output (static CLI cards,
	// summaries, scrollback snapshots) with ANSI styling and box-drawing
	// but no interactive focus or pointer chrome.
	ModeANSI
	// ModePlain is non-interactive unstyled plain text for pipes, CI logs,
	// and programmatic consumers (no escape sequences, no box chrome).
	ModePlain
)

// Modes lists every rendering Mode in display order.
var Modes = []Mode{ModeHotty, ModeTUI, ModeANSI, ModePlain}

// String names m as "hotty", "tui", "ansi", or "plain".
func (m Mode) String() string {
	switch m {
	case ModeHotty:
		return "hotty"
	case ModeTUI:
		return "tui"
	case ModeANSI:
		return "ansi"
	case ModePlain:
		return "plain"
	default:
		return "hotty"
	}
}

// Interactive reports whether m takes user input (ModeHotty or ModeTUI).
func (m Mode) Interactive() bool {
	return m == ModeHotty || m == ModeTUI
}

type modeCtxKey struct{}

// WithMode sets the rendering Mode on ctx.
func WithMode(ctx context.Context, m Mode) context.Context {
	return context.WithValue(ctx, modeCtxKey{}, m)
}

// ModeOf returns the rendering Mode on ctx, defaulting to ModeHotty.
func ModeOf(ctx context.Context) Mode {
	m, _ := ctx.Value(modeCtxKey{}).(Mode)
	return m
}

// InMode wraps c so it renders under WithMode(ctx, m).
func InMode(m Mode, c templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(WithMode(ctx, m), w)
	})
}
