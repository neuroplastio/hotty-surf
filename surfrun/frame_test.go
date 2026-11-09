package surfrun

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

func TestWriteFrame(t *testing.T) {
	const name = "s"
	ops := []hotty.Op{hotty.OpReveal("a", "")}
	reveals := []hotty.Op{hotty.OpReveal("b", "")}
	fields := []Field{{ID: "f1"}, {ID: "f2"}}
	var fieldsOut string
	for _, fl := range fields {
		fieldsOut += hotty.SetField(name, fl.ID, fl.Change)
	}
	both := append(append([]hotty.Op{}, ops...), reveals...)
	l := &List{}
	const listOut = "<list>"
	writeList := func(b *strings.Builder, got *List) {
		if got != l {
			t.Errorf("list called with %p, want %p", got, l)
		}
		b.WriteString(listOut)
	}
	mf := &MenuFrame{}
	const menuOut = "<menu>"
	writeMenu := func(b *strings.Builder, got *MenuFrame) {
		if got != mf {
			t.Errorf("menu called with %p, want %p", got, mf)
		}
		b.WriteString(menuOut)
	}
	tests := []struct {
		name string
		f    Frame
		list func(*strings.Builder, *List)
		menu func(*strings.Builder, *MenuFrame)
		want string
	}{
		{"zero", Frame{}, writeList, writeMenu, ""},
		{"joined", Frame{Ops: ops, Reveals: reveals}, writeList, writeMenu, hotty.Delta(name, both)},
		{"reveals only", Frame{Reveals: reveals}, writeList, writeMenu, hotty.Delta(name, reveals)},
		{"split around focus", Frame{Ops: ops, Focus: "k", Reveals: reveals}, writeList, writeMenu,
			hotty.Delta(name, ops) + hotty.Focus(name, "k") + hotty.Delta(name, reveals)},
		{"fields", Frame{Fields: fields}, writeList, writeMenu, fieldsOut},
		{"blur", Frame{Blur: true}, writeList, writeMenu, hotty.Blur(name)},
		{"blur and focus", Frame{Blur: true, Focus: "k"}, writeList, writeMenu,
			hotty.Blur(name) + hotty.Focus(name, "k")},
		{"list in place", Frame{Ops: ops, Fields: fields, List: l, Blur: true, Focus: "k", Reveals: reveals}, writeList, writeMenu,
			hotty.Delta(name, ops) + fieldsOut + listOut + hotty.Blur(name) +
				hotty.Focus(name, "k") + hotty.Delta(name, reveals)},
		{"menu after the list", Frame{Ops: ops, List: l, Menu: mf, Blur: true}, writeList, writeMenu,
			hotty.Delta(name, ops) + listOut + menuOut + hotty.Blur(name)},
		{"menu alone", Frame{Menu: mf}, writeList, writeMenu, menuOut},
		{"nil menu callback", Frame{Menu: mf, Blur: true}, writeList, nil, hotty.Blur(name)},
		{"no list", Frame{Fields: fields, Blur: true}, writeList, writeMenu, fieldsOut + hotty.Blur(name)},
		{"nil callback", Frame{Fields: fields, List: l, Blur: true}, nil, writeMenu, fieldsOut + hotty.Blur(name)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			WriteFrame(&b, name, tt.f, tt.list, tt.menu)
			if got := b.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
