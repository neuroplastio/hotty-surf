package main

import (
	"strings"
)

// Details is the surface's state: the commits it shares with the program's
// list, the one selected, and the tag form. The view (details.templ) reads
// it.
type Details struct {
	Commits []*Commit
	Sel     int

	TagName string
	TagNote string
	Missing bool // the form was submitted without a name
}

func (d *Details) Commit() *Commit { return d.Commits[d.Sel] }

func (d *Details) SetTagName(v string) { d.TagName, d.Missing = v, false }
func (d *Details) SetTagNote(v string) { d.TagNote = v }

// AddTag puts the form's tag on the selected commit and clears the form.
func (d *Details) AddTag() {
	name := strings.TrimSpace(d.TagName)
	if name == "" {
		d.Missing = true
		return
	}
	c := d.Commit()
	c.Tags = append(c.Tags, Tag{Name: name, Note: strings.TrimSpace(d.TagNote)})
	d.TagName, d.TagNote = "", ""
}

// Move moves the selection by delta, within the list.
func (d *Details) Move(delta int) {
	d.Sel = max(0, min(len(d.Commits)-1, d.Sel+delta))
	d.TagName, d.TagNote, d.Missing = "", "", false
}
