package main

import (
	"strconv"
	"unicode/utf8"
)

// DrawCells is the screen in cells (surfrun.Cells), for a terminal that is
// no hotty host, or a host that failed: the same state, as lines.
func (h *Hello) DrawCells(cols, rows int) []string {
	lines := []string{"Hello, " + h.Name + "!", "",
		strconv.Itoa(h.Count) + " clicks (enter adds one)"}
	return lines[:min(len(lines), rows)]
}

// CellsKey is a key that no binding took: a character goes into the
// name, backspace takes one off, and enter counts.
func (h *Hello) CellsKey(key string) bool {
	switch {
	case key == "enter":
		h.Add()
	case key == "backspace" && h.Name != "":
		_, n := utf8.DecodeLastRuneInString(h.Name)
		h.Name = h.Name[:len(h.Name)-n]
	case key == "space":
		h.Name += " "
	case utf8.RuneCountInString(key) == 1:
		h.Name += key
	default:
		return false
	}
	return true
}
