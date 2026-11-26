package model

import (
	"fmt"
	"strings"
	"testing"
)

func rowsString(rows []Row) string {
	var out []string
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%s@%d", r.Task.ID, r.Depth))
	}
	return strings.Join(out, " ")
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, s *Store)
		hideDone bool
		want     string
	}{
		{"all expanded", func(t *testing.T, s *Store) {}, false, "A@0 A1@1 A2@1 A2a@2 B@0 C@0"},
		{"collapsed top", func(t *testing.T, s *Store) { must(t, s.ToggleCollapsed("A")) }, false, "A@0 B@0 C@0"},
		{"collapsed nested", func(t *testing.T, s *Store) { must(t, s.ToggleCollapsed("A2")) }, false, "A@0 A1@1 A2@1 B@0 C@0"},
		{"collapsed hides even if child collapsed", func(t *testing.T, s *Store) {
			must(t, s.ToggleCollapsed("A2"))
			must(t, s.ToggleCollapsed("A"))
		}, false, "A@0 B@0 C@0"},
		{"hideDone leaf", func(t *testing.T, s *Store) { must(t, s.ToggleDone("A1", day)) }, true, "A@0 A2@1 A2a@2 B@0 C@0"},
		{"hideDone parent hides subtree", func(t *testing.T, s *Store) { must(t, s.ToggleDone("A", day)) }, true, "B@0 C@0"},
		{"done shown when not hiding", func(t *testing.T, s *Store) { must(t, s.ToggleDone("A", day)) }, false, "A@0 A1@1 A2@1 A2a@2 B@0 C@0"},
		{"hideDone and collapsed combined", func(t *testing.T, s *Store) {
			must(t, s.ToggleDone("B", day))
			must(t, s.ToggleCollapsed("A"))
		}, true, "A@0 C@0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := build(t, outline)
			tc.setup(t, s)
			if got := rowsString(Flatten(s.Lists[0], tc.hideDone)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
	if rows := Flatten(&List{}, false); len(rows) != 0 {
		t.Fatal("empty list should have no rows")
	}
}
