package ui

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// stopTabindexByReplace is what List did before stopTabindex: one
// strings.Replace over the whole body per item.
func stopTabindexByReplace(out, tag string, n, stop int) string {
	for i := 0; i < n; i++ {
		val := "-1"
		if i == stop {
			val = "0"
		}
		out = strings.Replace(out, `tabindex="`+tag+strconv.Itoa(i)+`"`, `tabindex="`+val+`"`, 1)
	}
	return out
}

func listBody(r *rand.Rand, tag string, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(`<li tabindex="` + tag + strconv.Itoa(i) + `">item ` + strconv.Itoa(i) + `</li>`)
		switch r.Intn(6) {
		case 0: // an item's placeholder again, as an item's markup may repeat it
			b.WriteString(`<span tabindex="` + tag + strconv.Itoa(r.Intn(n+2)) + `"></span>`)
		case 1: // another list's tag, or a broken one
			b.WriteString(`<li tabindex="ui-list-0xother-3">x</li><b tabindex="` + tag + `x"></b>`)
		case 2:
			b.WriteString(`<i tabindex="` + tag)
		}
	}
	return b.String()
}

func TestStopTabindexAsReplace(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	tag := "ui-list-0xc000123-"
	for k := 0; k < 2000; k++ {
		n := r.Intn(30)
		body := listBody(r, tag, n)
		stop := r.Intn(n+2) - 1
		if got, want := stopTabindex(body, tag, n, stop), stopTabindexByReplace(body, tag, n, stop); got != want {
			t.Fatalf("n=%d stop=%d\nbody %q\ngot  %q\nwant %q", n, stop, body, got, want)
		}
	}
}

func BenchmarkStopTabindex(b *testing.B) {
	tag := "ui-list-0xc000123-"
	body := listBody(rand.New(rand.NewSource(2)), tag, 48)
	b.Run("pass", func(b *testing.B) {
		for range b.N {
			stopTabindex(body, tag, 48, 7)
		}
	})
	b.Run("replace", func(b *testing.B) {
		for range b.N {
			stopTabindexByReplace(body, tag, 48, 7)
		}
	})
}
