package surf

import (
	"golang.org/x/net/html"
	"neuroplast.io/hotty-surf/internal/rt"
)

const modalGuard = "data-hotty-modal-guard"

func activeModal(f *rt.Frame) string {
	if f == nil || len(f.Modal) == 0 {
		return ""
	}
	return f.Modal[len(f.Modal)-1]
}

func (s *Surface) modalScope(f *rt.Frame, id string) *html.Node {
	if f == nil || f.ModalOps == nil {
		return nil
	}
	return f.ModalOps.Scope(s.prev, s.byID, id)
}
func (s *Surface) modalApplies(f *rt.Frame, modal, id string) bool {
	return f != nil && f.ModalOps != nil && f.ModalOps.Applies(s.prev, s.byID, modal, id)
}
func (s *Surface) modalTarget(f *rt.Frame, id string, last bool) string {
	if f == nil || f.ModalOps == nil {
		return ""
	}
	return f.ModalOps.Target(s.byID, id, last)
}
