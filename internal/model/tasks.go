package model

import "time"

func (s *Store) newTask(text string) (*Task, error) {
	text, err := cleanText(text)
	if err != nil {
		return nil, err
	}
	return &Task{ID: s.newID(), Text: text}, nil
}

// AddTaskAfter inserts a new sibling directly below the task afterID.
func (s *Store) AddTaskAfter(afterID, text string) (*Task, error) {
	loc, err := s.FindTask(afterID)
	if err != nil {
		return nil, err
	}
	t, err := s.newTask(text)
	if err != nil {
		return nil, err
	}
	sibs := loc.Siblings()
	*sibs = insertAt(*sibs, loc.Index+1, t)
	return t, nil
}

// AddTaskEnd appends a new top-level task to a list.
func (s *Store) AddTaskEnd(listID, text string) (*Task, error) {
	l := s.List(listID)
	if l == nil {
		return nil, ErrListNotFound
	}
	t, err := s.newTask(text)
	if err != nil {
		return nil, err
	}
	l.Tasks = append(l.Tasks, t)
	return t, nil
}

// AddChild appends a new last child to parentID and expands the parent.
func (s *Store) AddChild(parentID, text string) (*Task, error) {
	loc, err := s.FindTask(parentID)
	if err != nil {
		return nil, err
	}
	t, err := s.newTask(text)
	if err != nil {
		return nil, err
	}
	loc.Task.Children = append(loc.Task.Children, t)
	loc.Task.Collapsed = false
	return t, nil
}

// EditTask replaces a task's text.
func (s *Store) EditTask(id, text string) error {
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	text, err = cleanText(text)
	if err != nil {
		return err
	}
	loc.Task.Text = text
	return nil
}

// DeleteTask removes a task and its whole subtree, dropping any of them from
// the focus order.
func (s *Store) DeleteTask(id string) error {
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	s.removeFocus(subtreeIDs(loc.Task)...)
	sibs := loc.Siblings()
	*sibs = removeAt(*sibs, loc.Index)
	return nil
}

// MoveTask moves a task (with its subtree) up (dir < 0) or down (dir > 0)
// among its siblings. It reports whether anything moved.
func (s *Store) MoveTask(id string, dir int) (bool, error) {
	loc, err := s.FindTask(id)
	if err != nil {
		return false, err
	}
	sibs := *loc.Siblings()
	j := loc.Index + sign(dir)
	if dir == 0 || j < 0 || j >= len(sibs) {
		return false, nil
	}
	sibs[loc.Index], sibs[j] = sibs[j], sibs[loc.Index]
	return true, nil
}

// Indent makes the task the last child of the sibling above it (which is
// expanded). It reports false when there is no sibling above.
func (s *Store) Indent(id string) (bool, error) {
	loc, err := s.FindTask(id)
	if err != nil {
		return false, err
	}
	if loc.Index == 0 {
		return false, nil
	}
	sibs := loc.Siblings()
	prev := (*sibs)[loc.Index-1]
	*sibs = removeAt(*sibs, loc.Index)
	prev.Children = append(prev.Children, loc.Task)
	prev.Collapsed = false
	return true, nil
}

// Outdent makes the task a sibling directly after its parent. Siblings that
// followed it stay under the old parent. It reports false for a top-level task.
func (s *Store) Outdent(id string) (bool, error) {
	loc, err := s.FindTask(id)
	if err != nil {
		return false, err
	}
	if loc.Parent == nil {
		return false, nil
	}
	ploc, err := s.FindTask(loc.Parent.ID)
	if err != nil {
		return false, err
	}
	sibs := loc.Siblings()
	*sibs = removeAt(*sibs, loc.Index)
	psibs := ploc.Siblings()
	*psibs = insertAt(*psibs, ploc.Index+1, loc.Task)
	return true, nil
}

// ToggleCollapsed flips a task's collapsed flag.
func (s *Store) ToggleCollapsed(id string) error {
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	loc.Task.Collapsed = !loc.Task.Collapsed
	return nil
}

// SetAllCollapsed collapses or expands every task that has children in a list.
func (s *Store) SetAllCollapsed(listID string, collapsed bool) error {
	l := s.List(listID)
	if l == nil {
		return ErrListNotFound
	}
	walk(l.Tasks, func(t *Task) {
		if len(t.Children) > 0 {
			t.Collapsed = collapsed
		}
	})
	return nil
}

// ToggleDone flips a task's done state. Completing a task completes all its
// descendants too (stamping DoneAt = now on those not already done);
// un-completing affects only the task itself and clears its DoneAt.
func (s *Store) ToggleDone(id string, now time.Time) error {
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	t := loc.Task
	if t.Done {
		t.Done, t.DoneAt = false, nil
		return nil
	}
	walk([]*Task{t}, func(x *Task) {
		if !x.Done {
			at := now
			x.Done, x.DoneAt = true, &at
		}
	})
	return nil
}

// ClearDone removes all done tasks from a list and returns how many were
// removed. A done task's not-done descendants are kept: they are promoted into
// its place, in order.
func (s *Store) ClearDone(listID string) (int, error) {
	l := s.List(listID)
	if l == nil {
		return 0, ErrListNotFound
	}
	var removed []string
	l.Tasks = clearDone(l.Tasks, &removed)
	s.removeFocus(removed...)
	return len(removed), nil
}

func clearDone(ts []*Task, removed *[]string) []*Task {
	out := make([]*Task, 0, len(ts))
	for _, t := range ts {
		kids := clearDone(t.Children, removed)
		if t.Done {
			*removed = append(*removed, t.ID)
			out = append(out, kids...)
			continue
		}
		t.Children = kids
		if len(kids) == 0 {
			t.Children = nil
		}
		out = append(out, t)
	}
	return out
}

// MoveTaskToList moves a task with its subtree to the end of another list's
// top level. IDs are preserved, so focus order and stars are unaffected.
func (s *Store) MoveTaskToList(id, listID string) error {
	dst := s.List(listID)
	if dst == nil {
		return ErrListNotFound
	}
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	sibs := loc.Siblings()
	*sibs = removeAt(*sibs, loc.Index)
	dst.Tasks = append(dst.Tasks, loc.Task)
	return nil
}

// ToggleToday flips the Today star. Starring appends the task to the focus
// order; unstarring removes it.
func (s *Store) ToggleToday(id string) error {
	loc, err := s.FindTask(id)
	if err != nil {
		return err
	}
	t := loc.Task
	t.Today = !t.Today
	if t.Today {
		s.addFocus(t.ID)
	} else {
		s.removeFocus(t.ID)
	}
	return nil
}
