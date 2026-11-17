package main

import (
	"neuroplast.io/hotty-surf/ui"
)

// File is one of the made-up files the program lists.
type File struct {
	Name, Kind, Size string
	Note             string
	Stars            int
}

// App is what the program and its surface share. The program's list moves
// Sel; the surface shows the file at Sel and edits it. Both run on the
// program's loop, which is the surface's driver.
type App struct {
	Files []File
	Sel   int
}

func NewApp() *App {
	return &App{Files: []File{
		{Name: "notes.md", Kind: "Markdown", Size: "4 KB"},
		{Name: "budget.csv", Kind: "Table", Size: "18 KB"},
		{Name: "logo.svg", Kind: "Image", Size: "2 KB"},
		{Name: "main.go", Kind: "Go", Size: "9 KB"},
	}}
}

func (a *App) file() *File { return &a.Files[a.Sel] }

func (a *App) SetNote(v string) { a.file().Note = v }
func (a *App) Star()            { a.file().Stars++ }

// CSS is the surface's stylesheet: the kit's.
var CSS = ui.Stylesheet()
