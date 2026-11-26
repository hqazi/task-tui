package model

import "time"

// FocusEntry is one row of the Focus view.
type FocusEntry struct {
	Task *Task
	List *List
	// Crumb is "List › Ancestor › Parent" (see Store.Breadcrumb).
	Crumb string
}

func (s *Store) addFocus(id string) {
	for _, x := range s.FocusOrder {
		if x == id {
			return
		}
	}
	s.FocusOrder = append(s.FocusOrder, id)
}

func (s *Store) removeFocus(ids ...string) {
	if len(ids) == 0 || len(s.FocusOrder) == 0 {
		return
	}
	drop := make(map[string]bool, len(ids))
	for _, id := range ids {
		drop[id] = true
	}
	out := make([]string, 0, len(s.FocusOrder))
	for _, id := range s.FocusOrder {
		if !drop[id] {
			out = append(out, id)
		}
	}
	s.FocusOrder = out
}

// NormalizeFocus repairs FocusOrder after loading: it drops unknown, unstarred
// and duplicate IDs and appends starred tasks that are missing from it.
func (s *Store) NormalizeFocus() {
	starred := map[string]bool{}
	var inTree []string
	for _, l := range s.Lists {
		walk(l.Tasks, func(t *Task) {
			if t.Today {
				starred[t.ID] = true
				inTree = append(inTree, t.ID)
			}
		})
	}
	out := make([]string, 0, len(inTree))
	seen := map[string]bool{}
	for _, id := range s.FocusOrder {
		if starred[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range inTree {
		if !seen[id] {
			out = append(out, id)
		}
	}
	s.FocusOrder = out
}

// startOfDay returns midnight of now's day in now's location.
func startOfDay(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// FocusTasks returns the starred tasks in FocusOrder, excluding tasks that
// were completed before today (relative to now). Starred tasks in hidden lists
// are included.
func (s *Store) FocusTasks(now time.Time) []FocusEntry {
	cutoff := startOfDay(now)
	var out []FocusEntry
	for _, id := range s.FocusOrder {
		loc, err := s.FindTask(id)
		if err != nil || !loc.Task.Today {
			continue
		}
		if t := loc.Task; t.Done && t.DoneAt != nil && t.DoneAt.Before(cutoff) {
			continue
		}
		out = append(out, FocusEntry{Task: loc.Task, List: loc.List, Crumb: breadcrumb(loc)})
	}
	return out
}

// MoveFocus moves a task up (dir < 0) or down (dir > 0) in the focus order,
// swapping with its neighbour among the currently visible entries (so tasks
// that dropped out of Focus are skipped). It reports whether anything moved.
func (s *Store) MoveFocus(id string, dir int, now time.Time) (bool, error) {
	if _, err := s.FindTask(id); err != nil {
		return false, err
	}
	vis := s.FocusTasks(now)
	vi := -1
	for i, e := range vis {
		if e.Task.ID == id {
			vi = i
		}
	}
	if vi < 0 {
		return false, nil
	}
	vj := vi + sign(dir)
	if dir == 0 || vj < 0 || vj >= len(vis) {
		return false, nil
	}
	a, b := -1, -1
	for i, x := range s.FocusOrder {
		switch x {
		case vis[vi].Task.ID:
			a = i
		case vis[vj].Task.ID:
			b = i
		}
	}
	s.FocusOrder[a], s.FocusOrder[b] = s.FocusOrder[b], s.FocusOrder[a]
	return true, nil
}
