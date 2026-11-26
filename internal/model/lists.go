package model

// AddList appends a new list.
func (s *Store) AddList(name string) (*List, error) {
	name, err := cleanText(name)
	if err != nil {
		return nil, err
	}
	l := &List{ID: s.newID(), Name: name, Tasks: []*Task{}}
	s.Lists = append(s.Lists, l)
	return l, nil
}

// RenameList changes a list's name.
func (s *Store) RenameList(id, name string) error {
	l := s.List(id)
	if l == nil {
		return ErrListNotFound
	}
	name, err := cleanText(name)
	if err != nil {
		return err
	}
	l.Name = name
	return nil
}

// DeleteList removes a list and everything in it (including focus entries).
func (s *Store) DeleteList(id string) error {
	i := s.listIndex(id)
	if i < 0 {
		return ErrListNotFound
	}
	var ids []string
	walk(s.Lists[i].Tasks, func(t *Task) { ids = append(ids, t.ID) })
	s.removeFocus(ids...)
	s.Lists = append(s.Lists[:i], s.Lists[i+1:]...)
	return nil
}

// MoveList moves a list up (dir < 0) or down (dir > 0) one place. It reports
// whether the list moved; at either end it is a no-op returning false.
func (s *Store) MoveList(id string, dir int) (bool, error) {
	i := s.listIndex(id)
	if i < 0 {
		return false, ErrListNotFound
	}
	j := i + sign(dir)
	if dir == 0 || j < 0 || j >= len(s.Lists) {
		return false, nil
	}
	s.Lists[i], s.Lists[j] = s.Lists[j], s.Lists[i]
	return true, nil
}

// ToggleListHidden flips a list's hidden flag.
func (s *Store) ToggleListHidden(id string) error {
	l := s.List(id)
	if l == nil {
		return ErrListNotFound
	}
	l.Hidden = !l.Hidden
	return nil
}
