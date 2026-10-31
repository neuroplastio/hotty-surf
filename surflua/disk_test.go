//go:build !tinygo

package surflua

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// Preload reads the disk: surf.load() is the directory of the file the
// script is. (TinyGo's wasm has no disk.)
func TestDisk(t *testing.T) {
	dir := t.TempDir()
	for name, src := range todoFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	L := lua.NewState()
	defer L.Close()
	Preload(L)
	if err := L.DoFile(filepath.Join(dir, "todo.lua")); err != nil {
		t.Fatal(err)
	}
	if got, want := render(t, L), todoTree("", nil); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}
