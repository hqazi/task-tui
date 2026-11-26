package model

import (
	"errors"
	"strings"
	"testing"
)

func listNames(s *Store) string {
	var n []string
	for _, l := range s.Lists {
		n = append(n, l.Name)
	}
	return strings.Join(n, ",")
}

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s.Version != CurrentVersion || len(s.Lists) != 1 || s.Lists[0].Name != "Inbox" || s.Lists[0].ID == "" {
		t.Fatalf("bad default store: %+v", s)
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 8 || seen[id] {
			t.Fatalf("bad or duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestAddRenameList(t *testing.T) {
	s := NewStore()
	l, err := s.AddList("  Work ")
	must(t, err)
	if l.Name != "Work" || len(s.Lists) != 2 || s.Lists[1] != l {
		t.Fatalf("add failed: %s", listNames(s))
	}
	must(t, s.RenameList(l.ID, "Home"))
	if l.Name != "Home" {
		t.Fatal("rename failed")
	}
	if _, err := s.AddList("  "); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("want ErrEmptyText, got %v", err)
	}
	if err := s.RenameList(l.ID, ""); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("want ErrEmptyText, got %v", err)
	}
	if err := s.RenameList("nope", "x"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("want ErrListNotFound, got %v", err)
	}
}

func TestMoveList(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		dir   int
		want  string
		moved bool
	}{
		{"down middle", "B", 1, "A,C,B", true},
		{"up middle", "B", -1, "B,A,C", true},
		{"up at top", "A", -1, "A,B,C", false},
		{"down at bottom", "C", 1, "A,B,C", false},
		{"zero", "B", 0, "A,B,C", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, "A:", "B:", "C:")
			moved, err := s.MoveList(tc.id, tc.dir)
			must(t, err)
			if moved != tc.moved || listNames(s) != tc.want {
				t.Fatalf("moved=%v order=%s, want %v %s", moved, listNames(s), tc.moved, tc.want)
			}
		})
	}
	s := build(t, "A:")
	if _, err := s.MoveList("zz", 1); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestToggleListHidden(t *testing.T) {
	s := build(t, "A:")
	must(t, s.ToggleListHidden("A"))
	if !s.Lists[0].Hidden {
		t.Fatal("not hidden")
	}
	must(t, s.ToggleListHidden("A"))
	if s.Lists[0].Hidden {
		t.Fatal("still hidden")
	}
	if err := s.ToggleListHidden("zz"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteList(t *testing.T) {
	s := build(t, "A:\nx\n  y", "B:\nz")
	must(t, s.ToggleToday("y"))
	must(t, s.ToggleToday("z"))
	must(t, s.DeleteList("A"))
	if listNames(s) != "B" {
		t.Fatalf("got %s", listNames(s))
	}
	if strings.Join(s.FocusOrder, ",") != "z" {
		t.Fatalf("focus not cleaned: %v", s.FocusOrder)
	}
	if err := s.DeleteList("A"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
}
