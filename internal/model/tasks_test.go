package model

import (
	"errors"
	"testing"
	"time"
)

const outline = "L:\nA\n  A1\n  A2\n    A2a\nB\nC"

func TestAddTasks(t *testing.T) {
	tests := []struct {
		name string
		op   func(s *Store) (*Task, error)
		want string
	}{
		{"after top-level", func(s *Store) (*Task, error) { return s.AddTaskAfter("A", "N") }, "A(A1,A2(A2a)),N,B,C"},
		{"after last", func(s *Store) (*Task, error) { return s.AddTaskAfter("C", "N") }, "A(A1,A2(A2a)),B,C,N"},
		{"after nested", func(s *Store) (*Task, error) { return s.AddTaskAfter("A1", "N") }, "A(A1,N,A2(A2a)),B,C"},
		{"end", func(s *Store) (*Task, error) { return s.AddTaskEnd("L", "N") }, "A(A1,A2(A2a)),B,C,N"},
		{"child leaf", func(s *Store) (*Task, error) { return s.AddChild("B", "N") }, "A(A1,A2(A2a)),B(N),C"},
		{"child appended last", func(s *Store) (*Task, error) { return s.AddChild("A", "N") }, "A(A1,A2(A2a),N),B,C"},
		{"child deep", func(s *Store) (*Task, error) { return s.AddChild("A2a", "N") }, "A(A1,A2(A2a(N))),B,C"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			n, err := tc.op(s)
			must(t, err)
			// New IDs are random; rename for a stable dump.
			n.ID = "N"
			if got := dump(s.Lists[0].Tasks); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestAddChildExpandsParent(t *testing.T) {
	s := build(t, outline)
	task(t, s, "A").Collapsed = true
	_, err := s.AddChild("A", "x")
	must(t, err)
	if task(t, s, "A").Collapsed {
		t.Fatal("parent still collapsed")
	}
}

func TestAddErrors(t *testing.T) {
	s := build(t, outline)
	if _, err := s.AddTaskAfter("zz", "x"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	if _, err := s.AddChild("zz", "x"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	if _, err := s.AddTaskEnd("zz", "x"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
	for _, text := range []string{"", "   "} {
		if _, err := s.AddTaskAfter("A", text); !errors.Is(err, ErrEmptyText) {
			t.Fatalf("got %v", err)
		}
		if _, err := s.AddChild("A", text); !errors.Is(err, ErrEmptyText) {
			t.Fatalf("got %v", err)
		}
		if _, err := s.AddTaskEnd("L", text); !errors.Is(err, ErrEmptyText) {
			t.Fatalf("got %v", err)
		}
	}
	if got := dump(s.Lists[0].Tasks); got != "A(A1,A2(A2a)),B,C" {
		t.Fatalf("tree changed on error: %s", got)
	}
}

func TestNewIDsAreUniqueInStore(t *testing.T) {
	s := NewStore()
	seen := map[string]bool{s.Lists[0].ID: true}
	for i := 0; i < 200; i++ {
		n, err := s.AddTaskEnd(s.Lists[0].ID, "x")
		must(t, err)
		if seen[n.ID] {
			t.Fatal("duplicate id")
		}
		seen[n.ID] = true
	}
}

func TestEditTask(t *testing.T) {
	s := build(t, outline)
	must(t, s.EditTask("A2a", "  new text "))
	if task(t, s, "A2a").Text != "new text" {
		t.Fatal("not edited")
	}
	if err := s.EditTask("A2a", " "); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("got %v", err)
	}
	if err := s.EditTask("zz", "x"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	tests := []struct {
		id, want string
	}{
		{"B", "A(A1,A2(A2a)),C"},
		{"A2", "A(A1),B,C"},
		{"A", "B,C"},
		{"A2a", "A(A1,A2),B,C"},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			s := build(t, outline)
			must(t, s.DeleteTask(tc.id))
			if got := dump(s.Lists[0].Tasks); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
			if _, err := s.FindTask(tc.id); !errors.Is(err, ErrTaskNotFound) {
				t.Fatal("still findable")
			}
		})
	}
	s := build(t, outline)
	if err := s.DeleteTask("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteTaskRemovesSubtreeFromFocus(t *testing.T) {
	s := build(t, outline)
	for _, id := range []string{"A2a", "B", "A", "A1"} {
		must(t, s.ToggleToday(id))
	}
	must(t, s.DeleteTask("A"))
	if got := ids(s.FocusTasks(day)); got != "B" {
		t.Fatalf("focus = %s", got)
	}
	if len(s.FocusOrder) != 1 {
		t.Fatalf("FocusOrder = %v", s.FocusOrder)
	}
}

func TestMoveTask(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		dir   int
		want  string
		moved bool
	}{
		{"top down", "A", 1, "B,A(A1,A2(A2a)),C", true},
		{"top up", "C", -1, "A(A1,A2(A2a)),C,B", true},
		{"first up", "A", -1, "A(A1,A2(A2a)),B,C", false},
		{"last down", "C", 1, "A(A1,A2(A2a)),B,C", false},
		{"nested down", "A1", 1, "A(A2(A2a),A1),B,C", true},
		{"nested up", "A2", -1, "A(A2(A2a),A1),B,C", true},
		{"nested first up stays in parent", "A1", -1, "A(A1,A2(A2a)),B,C", false},
		{"only child", "A2a", 1, "A(A1,A2(A2a)),B,C", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			moved, err := s.MoveTask(tc.id, tc.dir)
			must(t, err)
			if got := dump(s.Lists[0].Tasks); got != tc.want || moved != tc.moved {
				t.Fatalf("got %s moved=%v; want %s %v", got, moved, tc.want, tc.moved)
			}
		})
	}
	s := build(t, outline)
	if _, err := s.MoveTask("zz", 1); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		want  string
		moved bool
	}{
		{"top becomes child of prev", "B", "A(A1,A2(A2a),B),C", true},
		{"carries subtree", "C", "A(A1,A2(A2a)),B(C)", true},
		{"first top-level is no-op", "A", "A(A1,A2(A2a)),B,C", false},
		{"first child is no-op", "A1", "A(A1,A2(A2a)),B,C", false},
		{"nested indent", "A2", "A(A1(A2(A2a))),B,C", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			moved, err := s.Indent(tc.id)
			must(t, err)
			if got := dump(s.Lists[0].Tasks); got != tc.want || moved != tc.moved {
				t.Fatalf("got %s moved=%v; want %s %v", got, moved, tc.want, tc.moved)
			}
		})
	}
	s := build(t, outline)
	if _, err := s.Indent("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestIndentExpandsNewParent(t *testing.T) {
	s := build(t, outline)
	task(t, s, "A").Collapsed = true
	_, err := s.Indent("B")
	must(t, err)
	if task(t, s, "A").Collapsed {
		t.Fatal("new parent should be expanded")
	}
}

func TestOutdent(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		want  string
		moved bool
	}{
		{"child lands after parent", "A1", "A(A2(A2a)),A1,B,C", true},
		{"later siblings stay under old parent", "A2", "A(A1),A2(A2a),B,C", true},
		{"deep", "A2a", "A(A1,A2,A2a),B,C", true},
		{"top-level no-op", "B", "A(A1,A2(A2a)),B,C", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			moved, err := s.Outdent(tc.id)
			must(t, err)
			if got := dump(s.Lists[0].Tasks); got != tc.want || moved != tc.moved {
				t.Fatalf("got %s moved=%v; want %s %v", got, moved, tc.want, tc.moved)
			}
		})
	}
	s := build(t, outline)
	if _, err := s.Outdent("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestIndentOutdentRoundTrip(t *testing.T) {
	s := build(t, outline)
	_, err := s.Indent("C")
	must(t, err)
	_, err = s.Outdent("C")
	must(t, err)
	if got := dump(s.Lists[0].Tasks); got != "A(A1,A2(A2a)),B,C" {
		t.Fatalf("got %s", got)
	}
}

func TestCollapse(t *testing.T) {
	s := build(t, outline)
	must(t, s.ToggleCollapsed("A"))
	if !task(t, s, "A").Collapsed {
		t.Fatal("not collapsed")
	}
	must(t, s.ToggleCollapsed("A"))
	if task(t, s, "A").Collapsed {
		t.Fatal("not expanded")
	}
	if err := s.ToggleCollapsed("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}

	must(t, s.SetAllCollapsed("L", true))
	for id, want := range map[string]bool{"A": true, "A2": true, "A1": false, "A2a": false, "B": false} {
		if task(t, s, id).Collapsed != want {
			t.Errorf("%s collapsed=%v want %v", id, !want, want)
		}
	}
	must(t, s.SetAllCollapsed("L", false))
	for _, id := range []string{"A", "A2"} {
		if task(t, s, id).Collapsed {
			t.Errorf("%s still collapsed", id)
		}
	}
	if err := s.SetAllCollapsed("zz", true); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestToggleDone(t *testing.T) {
	later := day.Add(time.Hour)
	t.Run("cascades to descendants and stamps DoneAt", func(t *testing.T) {
		s := build(t, outline)
		must(t, s.ToggleDone("A", day))
		if got := dump(s.Lists[0].Tasks); got != "A*(A1*,A2*(A2a*)),B,C" {
			t.Fatalf("got %s", got)
		}
		for _, id := range []string{"A", "A1", "A2", "A2a"} {
			if at := task(t, s, id).DoneAt; at == nil || !at.Equal(day) {
				t.Errorf("%s DoneAt=%v", id, at)
			}
		}
		if task(t, s, "B").DoneAt != nil {
			t.Error("B should not have DoneAt")
		}
	})
	t.Run("already done descendant keeps its DoneAt", func(t *testing.T) {
		s := build(t, outline)
		must(t, s.ToggleDone("A1", day))
		must(t, s.ToggleDone("A", later))
		if at := task(t, s, "A1").DoneAt; !at.Equal(day) {
			t.Errorf("A1 DoneAt=%v", at)
		}
		if at := task(t, s, "A2").DoneAt; !at.Equal(later) {
			t.Errorf("A2 DoneAt=%v", at)
		}
	})
	t.Run("completing all children does not complete parent", func(t *testing.T) {
		s := build(t, outline)
		must(t, s.ToggleDone("A1", day))
		must(t, s.ToggleDone("A2", day))
		if task(t, s, "A").Done {
			t.Fatal("parent auto-completed")
		}
		if d, n := Progress(task(t, s, "A")); d != 2 || n != 2 {
			t.Fatalf("progress %d/%d", d, n)
		}
	})
	t.Run("uncomplete parent leaves children done", func(t *testing.T) {
		s := build(t, outline)
		must(t, s.ToggleDone("A", day))
		must(t, s.ToggleDone("A", later))
		if got := dump(s.Lists[0].Tasks); got != "A(A1*,A2*(A2a*)),B,C" {
			t.Fatalf("got %s", got)
		}
		if task(t, s, "A").DoneAt != nil {
			t.Error("DoneAt not cleared")
		}
	})
	t.Run("uncomplete child only", func(t *testing.T) {
		s := build(t, outline)
		must(t, s.ToggleDone("A", day))
		must(t, s.ToggleDone("A2a", later))
		if got := dump(s.Lists[0].Tasks); got != "A*(A1*,A2*(A2a)),B,C" {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		s := build(t, outline)
		if err := s.ToggleDone("zz", day); !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestProgress(t *testing.T) {
	s := build(t, outline)
	tests := []struct {
		id         string
		done, tot  int
		completing []string
	}{
		{"A", 0, 2, nil},
		{"A", 1, 2, []string{"A1"}},
		{"A2", 0, 1, nil},
		{"B", 0, 0, nil},
	}
	for _, tc := range tests {
		for _, c := range tc.completing {
			must(t, s.ToggleDone(c, day))
		}
		if d, n := Progress(task(t, s, tc.id)); d != tc.done || n != tc.tot {
			t.Errorf("%s: %d/%d want %d/%d", tc.id, d, n, tc.done, tc.tot)
		}
	}
}

func TestOpenCount(t *testing.T) {
	s := build(t, outline)
	if n := s.Lists[0].OpenCount(); n != 6 {
		t.Fatalf("got %d", n)
	}
	must(t, s.ToggleDone("A", day))
	if n := s.Lists[0].OpenCount(); n != 2 {
		t.Fatalf("got %d", n)
	}
}

func TestClearDone(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, s *Store)
		want  string
		n     int
	}{
		{"nothing done", func(t *testing.T, s *Store) {}, "A(A1,A2(A2a)),B,C", 0},
		{"done parent takes subtree", func(t *testing.T, s *Store) { must(t, s.ToggleDone("A", day)) }, "B,C", 4},
		{"done leaves only", func(t *testing.T, s *Store) {
			must(t, s.ToggleDone("A1", day))
			must(t, s.ToggleDone("C", day))
		}, "A(A2(A2a)),B", 2},
		{"undone children of done parent are promoted", func(t *testing.T, s *Store) {
			must(t, s.ToggleDone("A", day))
			must(t, s.ToggleDone("A2", day)) // un-complete A2 (cascade made it done)
		}, "A2,B,C", 3},
		{"all done", func(t *testing.T, s *Store) {
			for _, id := range []string{"A", "B", "C"} {
				must(t, s.ToggleDone(id, day))
			}
		}, "", 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			tc.setup(t, s)
			n, err := s.ClearDone("L")
			must(t, err)
			if got := dump(s.Lists[0].Tasks); got != tc.want || n != tc.n {
				t.Fatalf("got %q n=%d; want %q n=%d", got, n, tc.want, tc.n)
			}
		})
	}
	s := build(t, outline)
	if _, err := s.ClearDone("zz"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestClearDoneCleansFocus(t *testing.T) {
	s := build(t, outline)
	must(t, s.ToggleToday("A1"))
	must(t, s.ToggleToday("B"))
	must(t, s.ToggleDone("A", day))
	_, err := s.ClearDone("L")
	must(t, err)
	if len(s.FocusOrder) != 1 || s.FocusOrder[0] != "B" {
		t.Fatalf("FocusOrder = %v", s.FocusOrder)
	}
}

func TestMoveTaskToList(t *testing.T) {
	tests := []struct {
		name         string
		id, dst      string
		wantSrc, dsp string
	}{
		{"top-level with subtree", "A", "M", "B,C", "X,A(A1,A2(A2a))"},
		{"nested becomes top-level", "A2", "M", "A(A1),B,C", "X,A2(A2a)"},
		{"to same list goes to end", "A", "L", "B,C,A(A1,A2(A2a))", "X"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline, "M:\nX")
			must(t, s.MoveTaskToList(tc.id, tc.dst))
			if got := dump(s.List("L").Tasks); got != tc.wantSrc {
				t.Fatalf("src %s want %s", got, tc.wantSrc)
			}
			if got := dump(s.List("M").Tasks); got != tc.dsp {
				t.Fatalf("dst %s want %s", got, tc.dsp)
			}
		})
	}
	s := build(t, outline, "M:")
	if err := s.MoveTaskToList("zz", "M"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	if err := s.MoveTaskToList("A", "zz"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("got %v", err)
	}
	if got := dump(s.List("L").Tasks); got != "A(A1,A2(A2a)),B,C" {
		t.Fatalf("changed on error: %s", got)
	}
}

func TestMoveToListKeepsFocusOrderAndStars(t *testing.T) {
	s := build(t, outline, "M:\nX")
	for _, id := range []string{"B", "A2a", "X"} {
		must(t, s.ToggleToday(id))
	}
	must(t, s.MoveTaskToList("A", "M")) // carries starred A2a
	if got := ids(s.FocusTasks(day)); got != "B,A2a,X" {
		t.Fatalf("focus order = %s", got)
	}
	e := s.FocusTasks(day)[1]
	if e.List.ID != "M" || e.Crumb != "M › A › A2" {
		t.Fatalf("entry list=%s crumb=%q", e.List.ID, e.Crumb)
	}
}

func TestFindTaskAndBreadcrumb(t *testing.T) {
	s := build(t, outline, "M:\nX\n  Y")
	loc, err := s.FindTask("A2a")
	must(t, err)
	if loc.List.ID != "L" || loc.Parent.ID != "A2" || loc.Index != 0 || len(loc.Ancestors) != 2 ||
		loc.Ancestors[0].ID != "A" || loc.Ancestors[1].ID != "A2" {
		t.Fatalf("bad location: %+v", loc)
	}
	loc, err = s.FindTask("C")
	must(t, err)
	if loc.Parent != nil || loc.Index != 2 || len(loc.Ancestors) != 0 {
		t.Fatalf("bad top-level location: %+v", loc)
	}
	if _, err := s.FindTask("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	tests := []struct{ id, want string }{
		{"C", "L"},
		{"A1", "L › A"},
		{"A2a", "L › A › A2"},
		{"Y", "M › X"},
	}
	for _, tc := range tests {
		got, err := s.Breadcrumb(tc.id)
		must(t, err)
		if got != tc.want {
			t.Errorf("Breadcrumb(%s)=%q want %q", tc.id, got, tc.want)
		}
	}
	if _, err := s.Breadcrumb("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestToggleToday(t *testing.T) {
	s := build(t, outline)
	must(t, s.ToggleToday("B"))
	must(t, s.ToggleToday("A2a"))
	if !task(t, s, "B").Today || len(s.FocusOrder) != 2 || s.FocusOrder[0] != "B" || s.FocusOrder[1] != "A2a" {
		t.Fatalf("star failed: %v", s.FocusOrder)
	}
	must(t, s.ToggleToday("B"))
	if task(t, s, "B").Today || len(s.FocusOrder) != 1 || s.FocusOrder[0] != "A2a" {
		t.Fatalf("unstar failed: %v", s.FocusOrder)
	}
	must(t, s.ToggleToday("B")) // re-star goes to the bottom
	if got := ids(s.FocusTasks(day)); got != "A2a,B" {
		t.Fatalf("got %s", got)
	}
	if err := s.ToggleToday("zz"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}
