package util

import (
	"encoding/base64"
	"fmt"
	"strings"
)

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
