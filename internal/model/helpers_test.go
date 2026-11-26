package model

import (
	"strings"
	"testing"
	"time"
)

// build creates a store with one list per spec entry. Each spec is
// "ListName:" followed by an indented outline (2 spaces per level) in which
// the task ID equals its text. Example: "Work:\nA\n  A1\nB".
func build(t *testing.T, specs ...string) *Store {
	t.Helper()
	s := &Store{Version: CurrentVersion, FocusOrder: []string{}}
	for _, spec := range specs {
		lines := strings.Split(strings.TrimSpace(spec), "\n")
		name := strings.TrimSuffix(lines[0], ":")
		l := &List{ID: name, Name: name, Tasks: []*Task{}}
		stack := []*[]*Task{&l.Tasks}
		for _, ln := range lines[1:] {
			depth := (len(ln) - len(strings.TrimLeft(ln, " "))) / 2
			text := strings.TrimSpace(ln)
			tk := &Task{ID: text, Text: text}
			stack = stack[:depth+1]
			*stack[depth] = append(*stack[depth], tk)
			stack = append(stack, &tk.Children)
		}
		s.Lists = append(s.Lists, l)
	}
	return s
}

// dump renders a task tree compactly: "A(A1,A2*),B" where * marks done.
func dump(ts []*Task) string {
	var parts []string
	for _, t := range ts {
		p := t.ID
		if t.Done {
			p += "*"
		}
		if len(t.Children) > 0 {
			p += "(" + dump(t.Children) + ")"
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, ",")
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func task(t *testing.T, s *Store, id string) *Task {
	t.Helper()
	loc, err := s.FindTask(id)
	must(t, err)
	return loc.Task
}

func ids(es []FocusEntry) string {
	var out []string
	for _, e := range es {
		out = append(out, e.Task.ID)
	}
	return strings.Join(out, ",")
}

var day = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
