package cells_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/cells"
)

// screenOf is a Screen showing html as a surface's document.
func screenOf(t *testing.T, html string) *cells.Screen {
	t.Helper()
	c := cells.New(60, 12, nil)
	if _, err := c.Write([]byte(hotty.Doc("s", html))); err != nil {
		t.Fatal(err)
	}
	c.Focus(false)
	return c
}

// cellAt is the first cell of text on the screen.
func cellAt(t *testing.T, c *cells.Screen, text string) cells.Cell {
	t.Helper()
	for _, row := range c.Cells() {
		var s strings.Builder
		for _, k := range row {
			s.WriteString(k.Text)
		}
		line := s.String()
		if i := strings.Index(line, text); i >= 0 {
			// The index in cells: every cell is one byte here (ASCII).
			return row[i]
		}
	}
	t.Fatalf("no %q on the screen", text)
	return cells.Cell{}
}

// The roles of each kit component the Screen reads (K28): a Basecoat
// rename shows here, since api-check can't see a role's name.
func TestCellsRoles(t *testing.T) {
	for _, tc := range []struct {
		name, html, text string
		fg, bg           cells.Role
	}{
		{"primary button", `<button class="btn">Save</button>`, "Save", "primary-foreground", "primary"},
		{"secondary button", `<button class="btn" data-variant="secondary">Save</button>`, "Save", "secondary-foreground", "secondary"},
		{"destructive button", `<button class="btn" data-variant="destructive">Drop</button>`, "Drop", "destructive", "destructive/10"},
		{"ghost button", `<button class="btn" data-variant="ghost">Go</button>`, "Go", "", ""},
		{"link button", `<a class="btn" data-variant="link" href="#">Docs</a>`, "Docs", "primary", ""},
		{"outline, light", `<div class="light"><button class="btn" data-variant="outline">Out</button></div>`, "Out", "", ""},
		{"outline, dark", `<div class="dark"><button class="btn" data-variant="outline">Out</button></div>`, "Out", "", "input/30"},
		{"badge", `<span class="badge">New</span>`, "New", "primary-foreground", "primary"},
		{"alert", `<div class="alert"><p>Heads up</p></div>`, "Heads", "card-foreground", "card"},
		{"destructive alert", `<div class="alert" data-variant="destructive"><p>Failed</p></div>`, "Failed", "destructive", "card"},
		{"card", `<div class="card"><p>Inside</p></div>`, "Inside", "card-foreground", "card"},
		{"kbd", `<p>Press <kbd>K</kbd></p>`, "K", "", "muted"},
		{"muted text", `<p class="text-muted-foreground">quiet</p>`, "quiet", "muted-foreground", ""},
		{"own colour in muted", `<p class="text-muted-foreground">a <b class="text-destructive">loud</b></p>`, "loud", "destructive", ""},
		{"utility over the kit", `<button class="btn text-destructive">X1</button>`, "X1", "destructive", "primary"},
		{"tint", `<p class="bg-destructive/10">tinted</p>`, "tinted", "", "destructive/10"},
		{"opacity not a percent", `<p class="bg-destructive/[.5]">odd</p>`, "odd", "", "destructive"},
		{"arbitrary var", `<p class="text-[var(--plx-warn)]">warn</p>`, "warn", "plx-warn", ""},
		{"v4 shorthand", `<p class="text-(--plx-ok)/50">fine</p>`, "fine", "plx-ok/50", ""},
		{"inline var", `<p style="color: var(--hotty-ansi-3, red); background: var(--plx-band)">hue</p>`, "hue", "hotty-ansi-3", "plx-band"},
		{"literal colour", `<p class="text-red-500" style="color: red">lit</p>`, "lit", "", ""},
		{"no colour class", `<p class="text-sm bg-clip-padding">size</p>`, "size", "", ""},
		{"dark variant", `<div class="dark"><p class="dark:bg-input/30">field</p></div>`, "field", "", "input/30"},
		{"dark variant, light", `<div class="light"><p class="dark:bg-input/30">field</p></div>`, "field", "", ""},
		{"text field, dark", `<div class="dark"><input class="input" value="abc"></div>`, "abc", "", "input/30"},
		{"selected row", `<ul role="listbox"><li role="option" aria-selected="true">Picked</li><li role="option">Other</li></ul>`, "Picked", "accent-foreground", "selected"},
		{"unselected row", `<ul role="listbox"><li role="option" aria-selected="true">Picked</li><li role="option">Other</li></ul>`, "Other", "", ""},
		{"nav current", `<button class="nav-item" aria-current="true">Home</button>`, "Home", "accent-foreground", "selected"},
		{"selected tab", `<div class="tabs"><nav role="tablist"><button role="tab" aria-selected="true">One</button><button role="tab" aria-selected="false">Two</button></nav></div>`, "One", "accent-foreground", "selected"},
		{"tab list", `<div class="tabs"><nav role="tablist"><button role="tab" aria-selected="true">One</button><button role="tab" aria-selected="false">Two</button></nav></div>`, "Two", "", "muted"},
		{"progress", `<div role="progressbar" aria-valuenow="50"></div>`, "█", "primary", ""},
		{"rule", `<p>above</p><hr><p>below</p>`, "─", "border", ""},
		{"placeholder", `<input placeholder="Search">`, "Search", "muted-foreground", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := cellAt(t, screenOf(t, tc.html), tc.text)
			if k.Fg != tc.fg || k.Bg != tc.bg {
				t.Errorf("%s: Fg %q Bg %q, want %q %q", tc.html, k.Fg, k.Bg, tc.fg, tc.bg)
			}
		})
	}
}

// Muted text is Dim in Lines and a role, not Dim, in Cells; a fill
// covers its element's blank cells; focus adds Reverse and keeps the roles.
func TestCellsAttrs(t *testing.T) {
	c := screenOf(t, `<p class="text-muted-foreground">quiet</p><div class="card"><p>x</p></div><s>gone</s>`)
	if k := cellAt(t, c, "quiet"); k.Attr&cells.Dim != 0 {
		t.Errorf("muted text is Dim in Cells: %+v", k)
	}
	if !strings.Contains(c.Lines()[0], "\x1b[0;2m") {
		t.Errorf("muted text isn't dim in Lines: %q", c.Lines()[0])
	}
	if k := cellAt(t, c, "gone"); k.Attr&cells.Strike == 0 {
		t.Errorf("s isn't Strike: %+v", k)
	}
	row := c.Cells()[1]
	if row[0].Bg != "card" || row[30].Bg != "card" || row[30].Text != " " {
		t.Errorf("the card's row: %+v … %+v", row[0], row[30])
	}

	f := cells.New(40, 4, nil)
	f.Write([]byte(hotty.Doc("s", `<button class="btn" id="b">Save</button>`)))
	f.Focus(true)
	f.Key("tab")
	if k := cellAt(t, f, "Save"); k.Attr&cells.Reverse == 0 || k.Bg != "primary" {
		t.Errorf("focused button: %+v, want Reverse on primary", k)
	}
	if got, want := len(f.Cells()), 4; got != want || len(f.Cells()[0]) != 40 {
		t.Errorf("grid %d×%d, want 40×4", len(f.Cells()[0]), got)
	}
	if w := cells.New(4, 1, nil); true {
		w.Write([]byte(hotty.Doc("s", `<p>界a</p>`)))
		r := w.Cells()[0]
		if r[0].Text != "界" || r[1].Text != "" || r[2].Text != "a" || r[3].Text != " " {
			t.Errorf("wide row: %+v", r)
		}
	}
}
