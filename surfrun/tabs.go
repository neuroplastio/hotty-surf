package surfrun

import "neuroplast.io/hotty-surf/surfui"

var tabsKeys = []string{"arrow_left", "arrow_right", "home", "end"}

func (s *Surface) focusedTabs() *surfui.Node {
	if !s.shown {
		return nil
	}
	n := s.index[s.focused]
	tabs, _, options, _, ok := surfui.TabsOf(n)
	if !ok || tabs != n || len(options) == 0 {
		return nil
	}
	if _, off := surfui.AttrOf(tabs, "disabled"); off {
		return nil
	}
	return tabs
}

func (s *Surface) tabsKey(key string) bool {
	n := s.focusedTabs()
	if n == nil {
		return false
	}
	_, value, options, _, _ := surfui.TabsOf(n)
	switch key {
	case "arrow_left":
		value = cycleBy(value, options, -1)
	case "arrow_right":
		value = cycleBy(value, options, 1)
	case "home":
		value = options[0]
	case "end":
		value = options[len(options)-1]
	default:
		return false
	}
	s.tabsTo(n, value)
	return true
}

func (s *Surface) tabsTo(n *surfui.Node, value string) {
	_, now, _, _, ok := surfui.TabsOf(n)
	if !ok || now == value {
		return
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return
	}
	if act, ok := surfui.Handler(n, "change"); ok {
		s.do(act, Event{Kind: "change", Target: surfui.IDOf(n), Value: value})
	}
}
