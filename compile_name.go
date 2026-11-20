//go:build !tinygo

package surf

import (
	"context"
	"reflect"
	"runtime"

	"github.com/a-h/templ"
)

// funcName is fn's name, as the runtime knows it: the same in every build
// of the same code, so that a reloaded build's instances keep their ids.
func funcName(fn any) string {
	if f := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()); f != nil {
		return f.Name()
	}
	return ""
}

// hasChildren says whether ctx carries a templ children block: templ gives
// its NopComponent for none, a func, which only its code tells apart.
func hasChildren(ctx context.Context) bool {
	c, ok := templ.GetChildren(ctx).(templ.ComponentFunc)
	return !ok || reflect.ValueOf(c).Pointer() != nop
}

var nop = reflect.ValueOf(templ.NopComponent).Pointer()
