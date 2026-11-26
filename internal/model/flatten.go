package model

// Row is one visible line of a list's task tree.
type Row struct {
	Task  *Task
	Depth int
}

// Flatten returns the visible rows of a list in display order. Children of
// collapsed tasks are omitted. With hideDone, done tasks are omitted together
// with their subtrees.
func Flatten(l *List, hideDone bool) []Row {
	var rows []Row
	var rec func(ts []*Task, depth int)
	rec = func(ts []*Task, depth int) {
		for _, t := range ts {
			if hideDone && t.Done {
				continue
			}
			rows = append(rows, Row{Task: t, Depth: depth})
			if !t.Collapsed {
				rec(t.Children, depth+1)
			}
		}
	}
	rec(l.Tasks, 0)
	return rows
}
