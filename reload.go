package surf

import (
	"io/fs"
	"time"
)

// stamp is what a file was when the Set read it: its time, where the fs
// gives one, and its content. The content is kept as it is, not hashed:
// it is exact, and a program links no hash for hot reload.
type stamp struct {
	mod time.Time
	src string
}

func stampOf(fsys fs.FS, name string, src string) stamp {
	st := stamp{src: src}
	if fi, err := fs.Stat(fsys, name); err == nil {
		st.mod = fi.ModTime()
	}
	return st
}
