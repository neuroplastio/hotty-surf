package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"neuroplast.io/hotty-surf/internal/spikeui"
)

// Task is a row of upstream's data (data/schema.ts).
type Task struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Label    string `json:"label"`
	Priority string `json:"priority"`
}

// LoadTasks reads a data file in upstream's shape, checking each row as
// zod's taskSchema does.
func LoadTasks(path string) ([]Task, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ts []Task
	if err := json.Unmarshal(b, &ts); err != nil {
		return nil, err
	}
	for i, t := range ts {
		if t.ID == "" || t.Title == "" || t.Status == "" || t.Label == "" || t.Priority == "" {
			return nil, fmt.Errorf("%s: task %d: a field is missing", path, i)
		}
	}
	return ts, nil
}

// Option is a facet's value, as data/data.tsx lists them.
type Option struct{ Value, Label, Icon string }

var (
	labels   = []Option{{"bug", "Bug", ""}, {"feature", "Feature", ""}, {"documentation", "Documentation", ""}}
	statuses = []Option{{"backlog", "Backlog", "circle-help"}, {"todo", "Todo", "circle"}, {"in progress", "In Progress", "timer"},
		{"done", "Done", "circle-check-big"}, {"canceled", "Canceled", "circle-off"}}
	priorities = []Option{{"low", "Low", "arrow-down"}, {"medium", "Medium", "arrow-right"}, {"high", "High", "arrow-up"}}
)

func find(opts []Option, v string) *Option {
	for i := range opts {
		if opts[i].Value == v {
			return &opts[i]
		}
	}
	return nil
}

// App is the page's state: upstream's DataTable state (sorting, filters,
// visibility, selection, pagination) and its popups'. Everything TanStack
// Table computed in the TSX port (the filtered, faceted, sorted and
// paginated row models) is computed here, in Go, after each change.
type App struct {
	all []Task
	low []Task // all, lowercased once: what the filter and the sort compare

	filter    string // the title filter, as typed
	filterLow string
	facet     map[string][]string // column → the values picked, in the order picked
	sortCol   string
	sortDesc  bool
	hidden    map[string]bool
	selected  map[int]bool // by the row's index in all (TanStack's default row id)
	pageIndex int
	pageSize  int

	user     spikeui.Popup
	view     spikeui.Popup
	header   map[string]*spikeui.Popup
	facetCmd map[string]*spikeui.Command
	pageSel  spikeui.Popup
	rowMenu  spikeui.Popup
	rowFor   int // the row whose menu is open
	labelSub spikeui.Popup

	// The row models, made again by compute after every change.
	rows   []int // filtered and sorted
	counts map[string]map[string]int
}

func NewApp(all []Task) *App {
	a := &App{all: all, low: lower(all), facet: map[string][]string{}, hidden: map[string]bool{}, selected: map[int]bool{}, pageSize: 25,
		header: map[string]*spikeui.Popup{}, facetCmd: map[string]*spikeui.Command{}, rowFor: -1}
	for _, c := range []string{"title", "status", "priority"} {
		a.header[c] = &spikeui.Popup{Active: -1}
	}
	for _, c := range []string{"status", "priority"} {
		a.facetCmd[c] = spikeui.NewCommand()
	}
	a.user.Active, a.view.Active, a.pageSel.Active, a.rowMenu.Active, a.labelSub.Active = -1, -1, -1, -1, -1
	a.compute()
	return a
}

// lower is the tasks lowercased: TanStack lowercases in every comparison;
// once is enough.
func lower(all []Task) []Task {
	low := make([]Task, len(all))
	for i, t := range all {
		low[i] = Task{ID: strings.ToLower(t.ID), Title: strings.ToLower(t.Title), Status: strings.ToLower(t.Status),
			Label: strings.ToLower(t.Label), Priority: strings.ToLower(t.Priority)}
	}
	return low
}

func (a *App) value(i int, col string) string { return field(&a.all[i], col) }

func field(t *Task, col string) string {
	switch col {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return t.Status
	case "priority":
		return t.Priority
	}
	return ""
}

// passes says whether row i passes every filter but skip's.
func (a *App) passes(i int, skip string) bool {
	if a.filter != "" && skip != "title" && !strings.Contains(a.low[i].Title, a.filterLow) {
		return false
	}
	for col, vs := range a.facet {
		if col != skip && len(vs) > 0 && !slices.Contains(vs, a.value(i, col)) {
			return false
		}
	}
	return true
}

// compute makes the row models: TanStack's filtered row model (includesString
// on the title, the facets' value lists), its faceted unique values (each
// facet column counted over the rows every other filter passes), and its
// sorted row model (the column's auto sort function, ties by index).
func (a *App) compute() {
	a.rows = a.rows[:0]
	a.filterLow = strings.ToLower(a.filter)
	a.counts = map[string]map[string]int{"status": {}, "priority": {}}
	for i := range a.all {
		if a.passes(i, "") {
			a.rows = append(a.rows, i)
		}
		for _, col := range []string{"status", "priority"} {
			if a.passes(i, col) {
				a.counts[col][a.value(i, col)]++
			}
		}
	}
	if a.sortCol != "" {
		alnum := a.autoAlnum()
		col, desc := a.sortCol, a.sortDesc
		slices.SortFunc(a.rows, func(x, y int) int {
			vx, vy := field(&a.low[x], col), field(&a.low[y], col)
			var c int
			if alnum {
				c = compareAlphanumeric(vx, vy)
			} else {
				c = strings.Compare(vx, vy)
			}
			if c != 0 {
				if desc {
					return -c
				}
				return c
			}
			return x - y
		})
	}
	if n := a.pageCount(); a.pageIndex > 0 && a.pageIndex >= n {
		a.pageIndex = max(n-1, 0)
	}
}

// autoAlnum is TanStack's auto choice: alphanumeric when any of the first
// ten filtered rows' values has a digit in it, else text.
func (a *App) autoAlnum() bool {
	for k, i := range a.rows {
		if k == 10 {
			break
		}
		if strings.ContainsAny(a.value(i, a.sortCol), "0123456789") {
			return true
		}
	}
	return false
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func chunkEnd(s string, i int, num bool) int {
	j := i + 1
	for j < len(s) && isDigit(s[j]) == num {
		j++
	}
	return j
}

// compareAlphanumeric is TanStack's (sortFns.ts): digit runs compare as
// numbers, other runs as text.
func compareAlphanumeric(a, b string) int {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		an, bn := isDigit(a[ai]), isDigit(b[bi])
		ae, be := chunkEnd(a, ai, an), chunkEnd(b, bi, bn)
		if !an && !bn {
			if c := strings.Compare(a[ai:ae], b[bi:be]); c != 0 {
				return c
			}
		} else if an != bn {
			if an {
				return 1
			}
			return -1
		} else {
			x, y := strings.TrimLeft(a[ai:ae], "0"), strings.TrimLeft(b[bi:be], "0")
			if len(x) != len(y) {
				if len(x) > len(y) {
					return 1
				}
				return -1
			}
			if c := strings.Compare(x, y); c != 0 {
				return c
			}
		}
		ai, bi = ae, be
	}
	return chunks(a, ai) - chunks(b, bi)
}

func chunks(s string, i int) int {
	n := 0
	for i < len(s) {
		n++
		i = chunkEnd(s, i, isDigit(s[i]))
	}
	return n
}

func (a *App) pageCount() int { return (len(a.rows) + a.pageSize - 1) / a.pageSize }

// page is the rows shown: the paginated row model.
func (a *App) page() []int {
	lo := min(a.pageIndex*a.pageSize, len(a.rows))
	return a.rows[lo:min(lo+a.pageSize, len(a.rows))]
}

// The handlers: each changes the state, and compute makes the row models
// again. A change of the filters or the sort goes back to the first page,
// as TanStack's autoResetPageIndex does.

func (a *App) SetFilter(v string) {
	a.filter, a.pageIndex = v, 0
	a.compute()
}

func (a *App) ToggleFacet(col, v string) {
	vs := a.facet[col]
	if i := slices.Index(vs, v); i >= 0 {
		vs = slices.Delete(vs, i, i+1)
	} else {
		vs = append(vs, v)
	}
	a.facet[col], a.pageIndex = vs, 0
	a.compute()
}

func (a *App) ClearFacet(col string) {
	delete(a.facet, col)
	a.pageIndex = 0
	a.compute()
}

func (a *App) ResetFilters() {
	a.filter, a.facet, a.pageIndex = "", map[string][]string{}, 0
	a.compute()
}

func (a *App) filtered() bool {
	if a.filter != "" {
		return true
	}
	for _, vs := range a.facet {
		if len(vs) > 0 {
			return true
		}
	}
	return false
}

func (a *App) Sort(col string, desc bool) {
	a.sortCol, a.sortDesc, a.pageIndex = col, desc, 0
	a.compute()
}

func (a *App) Hide(col string, hidden bool) { a.hidden[col] = hidden }

func (a *App) SetPage(i int) { a.pageIndex = max(0, min(i, a.pageCount()-1)) }

// SetPageSize keeps the page's top row in view, as TanStack's setPageSize.
func (a *App) SetPageSize(n int) {
	a.pageIndex = a.pageSize * a.pageIndex / n
	a.pageSize = n
	a.compute()
}

func (a *App) Select(i int, on bool) {
	if on {
		a.selected[i] = true
	} else {
		delete(a.selected, i)
	}
}

// SelectPage is the header box: all the page's rows on, or all off.
func (a *App) SelectPage(on bool) {
	for _, i := range a.page() {
		a.Select(i, on)
	}
}

// pageSelected says whether all, or some, of the page's rows are selected.
func (a *App) pageSelected() (all, some bool) {
	p := a.page()
	n := 0
	for _, i := range p {
		if a.selected[i] {
			n++
		}
	}
	return len(p) > 0 && n == len(p), n > 0 && n < len(p)
}

// selectedCount is the selected rows among the filtered ones.
func (a *App) selectedCount() int {
	n := 0
	for _, i := range a.rows {
		if a.selected[i] {
			n++
		}
	}
	return n
}

// OpenRowMenu opens row i's … menu (one is open at a time).
func (a *App) OpenRowMenu(i int) {
	if a.rowMenu.Open && a.rowFor == i {
		a.rowMenu.Close()
		return
	}
	a.rowFor = i
	a.rowMenu.Close()
	a.rowMenu.Toggle()
	a.labelSub.Close()
}

// Small helpers the view uses.

func isPicked(vs []string, v string) bool { return slices.Contains(vs, v) }

func anyMatch(opts []Option, search string) bool {
	for _, o := range opts {
		if spikeui.Fuzzy(o.Label, search) {
			return true
		}
	}
	return false
}

func count(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

func toggleFacet(a *App, col, v string) func() { return func() { a.ToggleFacet(col, v) } }

func selState(on bool) string {
	if on {
		return "selected"
	}
	return "false"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func openState(b bool) string {
	if b {
		return "open"
	}
	return "closed"
}

// cls joins class names, leaving out the empty ones: cn, in Go.
func cls(names ...string) string {
	var b strings.Builder
	for _, n := range names {
		if n == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(n)
	}
	return b.String()
}

// when is class when ok, else "".
func when(ok bool, class string) string {
	if ok {
		return class
	}
	return ""
}

// slug is a value as an id's part: "in progress" → "in-progress".
func slug(v string) string { return strings.ReplaceAll(v, " ", "-") }
