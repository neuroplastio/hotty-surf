//go:build surfdev

package surf

import "neuroplast.io/hotty-surf/internal/rt"

func init() {
	rt.Peek.Pause = func(s any) func() {
		x := s.(*Surface)
		out, host, platform := x.out, x.host, x.platform
		x.out, x.host = nil, nil
		return func() { x.out, x.host, x.platform = out, host, platform }
	}
}
