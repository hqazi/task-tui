package model

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func star(t *testing.T, s *Store, idsToStar ...string) {
	t.Helper()
	for _, id := range idsToStar {
		must(t, s.ToggleToday(id))
	}
}

func TestFocusOrdering(t *testing.T) {
	s := build(t, outline, "M:\nX\n  Y")
	star(t, s, "Y", "B", "A2a", "X")
	if got := ids(s.FocusTasks(day)); got != "Y,B,A2a,X" {
		t.Fatalf("got %s", got)
	}
	// Includes crumbs and owning lists, and only the starred task.
	e := s.FocusTasks(day)[0]
	if e.Crumb != "M › X" || e.List.ID != "M" || e.Task.ID != "Y" {
		t.Fatalf("entry %+v", e)
	}
}

func TestFocusHiddenListStillIncluded(t *testing.T) {
	s := build(t, outline, "M:\nX")
	star(t, s, "X", "B")
	must(t, s.ToggleListHidden("M"))
	if got := ids(s.FocusTasks(day)); got != "X,B" {
		t.Fatalf("got %s", got)
	}
}

func TestFocusDoneDropout(t *testing.T) {
	nov26 := time.Date(2026, 11, 26, 23, 59, 0, 0, time.UTC)
	nov27 := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		doneAt  time.Time
		now     time.Time
		visible bool
	}{
		{"done earlier today", nov26, day, true},
		{"done just now", day, day, true},
		{"done late, viewed same day", nov26, nov26, true},
		{"done yesterday late, viewed at midnight next day", nov26, nov27, false},
		{"done days ago", day.AddDate(0, 0, -3), day, false},
		{"done today but viewed earlier clock", day.Add(5 * time.Hour), day, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			star(t, s, "A", "B")
			must(t, s.ToggleDone("A", tc.doneAt))
			want := "B"
			if tc.visible {
				want = "A,B"
			}
			if got := ids(s.FocusTasks(tc.now)); got != want {
				t.Fatalf("got %s want %s", got, want)
			}
			// The star itself never clears automatically.
			if !task(t, s, "A").Today {
				t.Fatal("star cleared")
			}
		})
	}
}

func TestFocusDropoutRespectsLocation(t *testing.T) {
	loc := time.FixedZone("X", 10*3600)
	// 2026-11-26 20:00 UTC is 2026-11-27 06:00 in +10.
	doneAt := time.Date(2026, 11, 26, 20, 0, 0, 0, time.UTC)
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, loc)
	s := build(t, outline)
	star(t, s, "A")
	must(t, s.ToggleDone("A", doneAt))
	if got := ids(s.FocusTasks(now)); got != "A" {
		t.Fatalf("got %q; same local day should be visible", got)
	}
	if got := ids(s.FocusTasks(now.AddDate(0, 0, 1))); got != "" {
		t.Fatalf("got %q; next local day should drop out", got)
	}
}

func TestFocusDoneWithoutDoneAtStaysVisible(t *testing.T) {
	s := build(t, outline)
	star(t, s, "A")
	task(t, s, "A").Done = true
	if got := ids(s.FocusTasks(day)); got != "A" {
		t.Fatalf("got %q", got)
	}
}

func TestFocusUncompleteReturns(t *testing.T) {
	s := build(t, outline)
	star(t, s, "A")
	must(t, s.ToggleDone("A", day.AddDate(0, 0, -2)))
	if got := ids(s.FocusTasks(day)); got != "" {
		t.Fatalf("got %q", got)
	}
	must(t, s.ToggleDone("A", day))
	must(t, s.ToggleDone("A", day)) // done then undone
	if got := ids(s.FocusTasks(day)); got != "A" {
		t.Fatalf("got %q", got)
	}
}

func TestMoveFocus(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		dir   int
		want  string
		moved bool
	}{
		{"down", "B", 1, "A,C,B", true},
		{"up", "B", -1, "B,A,C", true},
		{"top up", "A", -1, "A,B,C", false},
		{"bottom down", "C", 1, "A,B,C", false},
		{"zero", "B", 0, "A,B,C", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			star(t, s, "A", "B", "C")
			moved, err := s.MoveFocus(tc.id, tc.dir, day)
			must(t, err)
			if got := ids(s.FocusTasks(day)); got != tc.want || moved != tc.moved {
				t.Fatalf("got %s moved=%v; want %s %v", got, moved, tc.want, tc.moved)
			}
		})
	}
}

func TestMoveFocusSkipsDroppedOutTasks(t *testing.T) {
	s := build(t, outline)
	star(t, s, "A", "B", "C")
	must(t, s.ToggleDone("B", day.AddDate(0, 0, -2))) // B hidden from Focus
	moved, err := s.MoveFocus("C", -1, day)
	must(t, err)
	if !moved || ids(s.FocusTasks(day)) != "C,A" {
		t.Fatalf("moved=%v got %s", moved, ids(s.FocusTasks(day)))
	}
}

func TestMoveFocusErrorsAndNotInFocus(t *testing.T) {
	s := build(t, outline)
	star(t, s, "A")
	if _, err := s.MoveFocus("zz", 1, day); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
	moved, err := s.MoveFocus("B", 1, day) // exists but not starred
	must(t, err)
	if moved {
		t.Fatal("unstarred task moved")
	}
}

func TestNormalizeFocus(t *testing.T) {
	s := build(t, outline)
	task(t, s, "A").Today = true
	task(t, s, "B").Today = true
	task(t, s, "C").Today = true
	s.FocusOrder = []string{"C", "ghost", "C", "A1" /* unstarred */, "A"}
	s.NormalizeFocus()
	if got := strings.Join(s.FocusOrder, ","); got != "C,A,B" {
		t.Fatalf("got %s", got)
	}
	s2 := build(t, outline)
	s2.FocusOrder = nil
	s2.NormalizeFocus()
	if s2.FocusOrder == nil || len(s2.FocusOrder) != 0 {
		t.Fatalf("want empty non-nil, got %#v", s2.FocusOrder)
	}
}
