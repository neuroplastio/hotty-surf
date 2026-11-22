//go:build !js

package surfapp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// TestCellsTyping: on a terminal that is no host, Tab reaches the field
// and typing fills it, through the surface's Input.
func TestCellsTyping(t *testing.T) {
	var got string
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := surf.On(ctx, "f", surf.Input(func(v string) { got = v }))
		_, err := io.WriteString(w, `<div><input id="f" data-on="`+a["data-on"].(string)+`" value="`+got+`"><button>b</button></div>`)
		return err
	})
	s := surf.New("t", page, "")
	c := newCellsHost(s, 40, 5)
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tab", "shift+t", "h"} {
		if _, err := c.key(k); err != nil {
			t.Fatal(err)
		}
		if c.err != nil {
			t.Fatal(c.err)
		}
		for {
			select {
			case <-s.Pending():
				if err := s.Update(); err != nil {
					t.Fatal(err)
				}
				continue
			default:
			}
			break
		}
	}
	if got != "Th" {
		t.Fatalf("the field heard %q; the screen:\n%s", got, strings.Join(c.scr.Lines(), "\n"))
	}
}

// A command the Screen refuses (an a=focus naming no element yet, which a
// host answers with ENOTARGET and run's host path ignores) is no error
// that ends the app.
func TestCellsRefusalIsNoError(t *testing.T) {
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		surf.Autofocus(ctx, "search") // the field isn't drawn yet
		_, err := io.WriteString(w, `<div><button>b</button></div>`)
		return err
	})
	s := surf.New("t", page, "")
	c := newCellsHost(s, 40, 5)
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Pending():
		if err := s.Update(); err != nil {
			t.Fatalf("the refusal reached the loop as an error, which ends inCells: %v", err)
		}
	default:
		t.Fatal("the autofocus asked for no update")
	}
}

// A refused delta has the document sent again, as a host's refusal has.
func TestCellsRefusedDeltaResends(t *testing.T) {
	n := 0
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<div><p>n=%d</p></div>`, n)
		return err
	})
	s := surf.New("t", page, "")
	c := newCellsHost(s, 20, 2)
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	// The Screen loses its document behind the surface's back: the next
	// delta has nothing to apply to.
	c.scr.Write([]byte(hotty.Doc("t", `<div id="root"></div>`)))
	n = 1
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if !c.stale {
		t.Fatal("the refused delta didn't mark the Screen stale")
	}
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(c.scr.Lines(), "\n"), "n=1") {
		t.Fatalf("not resent:\n%s", strings.Join(c.scr.Lines(), "\n"))
	}
}

// xterm is the rows a frame leaves on a terminal that keeps the cursor on
// the last column after it writes there, waiting to wrap (xterm, st): only
// CUP, EL, text and the modes and SGR it ignores. Every rune is a cell.
func xterm(frame string, cols, rows int) []string {
	grid := make([][]rune, rows)
	for i := range grid {
		grid[i] = []rune(strings.Repeat(" ", cols))
	}
	row, col, pending := 0, 0, false
	rs := []rune(frame)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\x1b' && i+1 < len(rs) && rs[i+1] == '[' {
			j := i + 2
			for j < len(rs) && (rs[j] < 0x40 || rs[j] > 0x7e) {
				j++
			}
			params := string(rs[i+2 : j])
			switch rs[j] {
			case 'H':
				r, c := 1, 1
				fmt.Sscanf(params, "%d;%d", &r, &c)
				row, col, pending = r-1, c-1, false
			case 'K':
				if params == "" || params == "0" {
					for k := col; k < cols; k++ {
						grid[row][k] = ' '
					}
				}
			}
			i = j
			continue
		}
		if pending {
			row, col, pending = row+1, 0, false
		}
		if row < rows {
			grid[row][col] = rs[i]
		}
		if col == cols-1 {
			pending = true
		} else {
			col++
		}
	}
	out := make([]string, rows)
	for i, l := range grid {
		out[i] = string(l)
	}
	return out
}

// A full row keeps its last cell on such a terminal.
func TestFrameKeepsTheLastColumn(t *testing.T) {
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<div class="flex justify-between"><span>left</span><span>right</span></div>`)
		return err
	})
	s := surf.New("t", page, "")
	c := newCellsHost(s, 20, 2)
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	if got := xterm(c.frame(), 20, 2)[0]; got != "left           right" {
		t.Fatalf("the row on the terminal: %q", got)
	}
}
