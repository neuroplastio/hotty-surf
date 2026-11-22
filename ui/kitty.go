package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type kittyKey struct{}

// WithKitty explicitly overrides Kitty graphics protocol support detection on ctx.
func WithKitty(ctx context.Context, supported bool) context.Context {
	return context.WithValue(ctx, kittyKey{}, supported)
}

// SupportsKitty reports whether the target terminal supports the Kitty graphics
// protocol (used by TUI components like Avatar to render crisp inline images
// when available, falling back to text/initials otherwise).
func SupportsKitty(ctx context.Context) bool {
	if ctx != nil {
		if v, ok := ctx.Value(kittyKey{}).(bool); ok {
			return v
		}
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") {
		return true
	}
	prog := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch prog {
	case "ghostty", "wezterm", "hottyterm", "iterm.app", "kitty":
		return true
	}
	return os.Getenv("HOTTY_SURF_HOTTY") != ""
}

// KittyImageSequence encodes PNG bytes as a Kitty Graphics Protocol APC escape
// sequence sized to cols×rows terminal cells (chunked at 4096 base64 bytes).
func KittyImageSequence(pngBytes []byte, cols, rows int) string {
	if len(pngBytes) == 0 || cols <= 0 || rows <= 0 {
		return ""
	}
	enc := base64.StdEncoding.EncodeToString(pngBytes)
	const chunk = 4096
	var b strings.Builder
	for i := 0; i < len(enc); i += chunk {
		end := min(i+chunk, len(enc))
		more := 0
		if end < len(enc) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&b, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2,m=%d;%s\x1b\\", cols, rows, more, enc[i:end])
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, enc[i:end])
		}
	}
	return b.String()
}
