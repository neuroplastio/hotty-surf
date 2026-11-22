package surftui

import (
	"testing"

	"neuroplast.io/hotty-go"
)

func TestSelectionAndCut(t *testing.T) {
	var events []hotty.Event
	c := New(30, 4, func(e hotty.Event) { events = append(events, e) })
	c.Write([]byte(hotty.Doc("s", `<input id="f" data-on="input" value="héllo wörld"><input id="p" type="password" value="secret"><input id="r" readonly value="fixed">`)))
	c.Focus(true)
	if c.Selection() != "" {
		t.Error("a selection with no field focused")
	}
	c.Key("tab")
	if c.Selection() != "" {
		t.Errorf("a selection with nothing selected: %q", c.Selection())
	}
	c.Key("home")
	for range 5 {
		c.Key("shift+arrow_right")
	}
	if got := c.Selection(); got != "héllo" {
		t.Errorf("Selection = %q, want héllo", got)
	}
	events = nil
	if got := c.Cut(); got != "héllo" {
		t.Errorf("Cut = %q, want héllo", got)
	}
	if v := c.field(c.byID("f")).ed.Value; v != " wörld" {
		t.Errorf("after the cut the field holds %q", v)
	}
	if len(events) != 1 || events[0].Kind != hotty.EventInput || events[0].Value != " wörld" {
		t.Errorf("the cut sent %+v, want one input", events)
	}
	if c.Selection() != "" || c.Cut() != "" {
		t.Error("a selection left after the cut")
	}

	for _, id := range []string{"p", "r"} {
		c.Key("tab")
		c.Key("ctrl+a")
		c.Key("shift+end")
		if id == "p" && c.Selection() != "" {
			t.Errorf("a password's selection: %q", c.Selection())
		}
		if id == "r" {
			if c.Selection() != "fixed" {
				t.Errorf("readonly selection %q, want fixed", c.Selection())
			}
			if c.Cut() != "" || c.field(c.byID("r")).ed.Value != "fixed" {
				t.Error("a readonly field was cut")
			}
		}
	}
}
