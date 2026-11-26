// Package model holds the pure domain model of task-tui: lists, nested tasks,
// the Today/Focus order and every operation on them. It has no TUI or I/O
// dependencies. All operations address lists and tasks by ID.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// CurrentVersion is the on-disk schema version written by this package.
const CurrentVersion = 1

// DefaultListName is the name of the list created for an empty store.
const DefaultListName = "Inbox"

// Errors returned by operations.
var (
	ErrListNotFound = errors.New("list not found")
	ErrTaskNotFound = errors.New("task not found")
	ErrEmptyText    = errors.New("text must not be empty")
)

// Store is the whole document: all lists plus the manual Focus order.
type Store struct {
	Version    int      `json:"version"`
	Lists      []*List  `json:"lists"`
	FocusOrder []string `json:"focusOrder"`
}

// List is a named collection of tasks.
type List struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Hidden bool    `json:"hidden"`
	Tasks  []*Task `json:"tasks"`
}

// Task is a single to-do item. Children nest to unlimited depth.
type Task struct {
	ID        string     `json:"id"`
	Text      string     `json:"text"`
	Done      bool       `json:"done"`
	DoneAt    *time.Time `json:"doneAt"`
	Today     bool       `json:"today"`
	Collapsed bool       `json:"collapsed"`
	Children  []*Task    `json:"children,omitempty"`
}

// NewStore returns an empty store containing a single "Inbox" list.
func NewStore() *Store {
	s := &Store{Version: CurrentVersion, FocusOrder: []string{}}
	s.Lists = []*List{{ID: s.newID(), Name: DefaultListName, Tasks: []*Task{}}}
	return s
}

// NewID returns a random 8-character hex identifier.
func NewID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("model: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

// newID returns an ID not used by any list or task in s.
func (s *Store) newID() string {
	for {
		id := NewID()
		if !s.idInUse(id) {
			return id
		}
	}
}

func (s *Store) idInUse(id string) bool {
	for _, l := range s.Lists {
		if l.ID == id {
			return true
		}
		if _, ok := findIn(l, nil, l.Tasks, id); ok {
			return true
		}
	}
	return false
}

func cleanText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ErrEmptyText
	}
	return text, nil
}

// List returns the list with the given ID, or nil.
func (s *Store) List(id string) *List {
	for _, l := range s.Lists {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func (s *Store) listIndex(id string) int {
	for i, l := range s.Lists {
		if l.ID == id {
			return i
		}
	}
	return -1
}

// Location describes where a task lives.
type Location struct {
	List      *List
	Task      *Task
	Parent    *Task   // nil for a top-level task
	Ancestors []*Task // root-most first, ending with Parent
	Index     int     // position among siblings
}

// Siblings returns a pointer to the slice that holds the task.
func (l Location) Siblings() *[]*Task {
	if l.Parent == nil {
		return &l.List.Tasks
	}
	return &l.Parent.Children
}

// FindTask locates a task by ID anywhere in the store.
func (s *Store) FindTask(id string) (Location, error) {
	for _, l := range s.Lists {
		if loc, ok := findIn(l, nil, l.Tasks, id); ok {
			return loc, nil
		}
	}
	return Location{}, ErrTaskNotFound
}

func findIn(l *List, anc []*Task, ts []*Task, id string) (Location, bool) {
	for i, t := range ts {
		if t.ID == id {
			loc := Location{List: l, Task: t, Index: i}
			if len(anc) > 0 {
				loc.Ancestors = append([]*Task(nil), anc...)
				loc.Parent = anc[len(anc)-1]
			}
			return loc, true
		}
		if loc, ok := findIn(l, append(anc, t), t.Children, id); ok {
			return loc, true
		}
	}
	return Location{}, false
}

// Breadcrumb returns "List › Ancestor › Parent" for a task (the task's own
// text is not included), e.g. "Work › Ship v2".
func (s *Store) Breadcrumb(id string) (string, error) {
	loc, err := s.FindTask(id)
	if err != nil {
		return "", err
	}
	return breadcrumb(loc), nil
}

func breadcrumb(loc Location) string {
	parts := []string{loc.List.Name}
	for _, a := range loc.Ancestors {
		parts = append(parts, a.Text)
	}
	return strings.Join(parts, " › ")
}

// Progress returns done/total counts over the direct children of t.
func Progress(t *Task) (done, total int) {
	for _, c := range t.Children {
		total++
		if c.Done {
			done++
		}
	}
	return done, total
}

// OpenCount returns the number of open (not done) tasks in l, at any depth.
func (l *List) OpenCount() int { return openCount(l.Tasks) }

func openCount(ts []*Task) int {
	n := 0
	for _, t := range ts {
		if !t.Done {
			n++
		}
		n += openCount(t.Children)
	}
	return n
}

// walk calls fn for every task in ts depth-first (pre-order).
func walk(ts []*Task, fn func(*Task)) {
	for _, t := range ts {
		fn(t)
		walk(t.Children, fn)
	}
}

func subtreeIDs(t *Task) []string {
	var ids []string
	walk([]*Task{t}, func(x *Task) { ids = append(ids, x.ID) })
	return ids
}

func removeAt(ts []*Task, i int) []*Task {
	return append(ts[:i], ts[i+1:]...)
}

func insertAt(ts []*Task, i int, t *Task) []*Task {
	ts = append(ts, nil)
	copy(ts[i+1:], ts[i:])
	ts[i] = t
	return ts
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
