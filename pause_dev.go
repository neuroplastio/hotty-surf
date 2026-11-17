//go:build surfdev

package surf

import "neuroplast.io/hotty-surf/internal/rt"

func init() {
	rt.Peek.Pause = func(s any) func() {
		x := s.(*Surface)
		out, platform := x.out, x.platform
		x.out = nil
		return func() { x.out, x.platform = out, platform }
	}
}
