package ui

import (
	"strings"
	"testing"
)

func TestPickerMove(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("m")
	if f.m.mode != modePicker {
		t.Fatal("picker not open")
	}
	// Targets: Home, Old (hidden), Ideas (hidden); never Work.
	if got := len(f.m.picker.lists); got != 3 {
		t.Fatalf("targets = %d", got)
	}
	view := f.m.View()
	if !strings.Contains(view, "Old (hidden)") || strings.Contains(view, "Work") {
		t.Fatalf("view:\n%s", view)
	}
	f.keys("down", "enter") // Old
	if f.m.mode != modeNormal {
		t.Fatal("picker still open")
	}
	old := f.m.store.List(ids["old"])
	if len(old.Tasks) != 1 || old.Tasks[0].ID != ids["ship"] || len(old.Tasks[0].Children) != 2 {
		t.Fatalf("old tasks: %+v", old.Tasks)
	}
	if f.task() != ids["refactor"] {
		t.Fatalf("cursor = %q, want next row", f.task())
	}
	f.wantSaves(1)
}

func TestPickerMoveLastRowCursorGoesUp(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("tab")
	f.m.taskSel = ids["reply"]
	f.m = f.m.settle()
	f.keys("m", "enter")
	if f.task() != ids["deep"] { // previous visible row
		t.Fatalf("cursor = %q", f.task())
	}
}

func TestPickerCancel(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("m", "down", "esc")
	if f.m.mode != modeNormal || f.task() != ids["ship"] {
		t.Fatal("cancel changed state")
	}
	if len(f.m.store.List(ids["work"]).Tasks) != 3 {
		t.Fatal("task moved on cancel")
	}
	f.wantSaves(0)
}

func TestPickerFromFocus(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["reply"], ids["tag"])
	f.openFocus()
	f.m.focusSel = ids["tag"]
	f.m = f.m.settle()
	f.keys("m", "enter") // Home
	home := f.m.store.List(ids["home"])
	if len(home.Tasks) != 1 || home.Tasks[0].ID != ids["tag"] {
		t.Fatal("not moved")
	}
	if !f.m.inFocus() || f.m.focusSel != ids["tag"] {
		t.Fatalf("left focus or lost task: %q", f.m.focusSel)
	}
	f.wantSaves(1)
}

func TestPickerSingleList(t *testing.T) {
	f, ids := newFixture(t)
	for _, k := range []string{"home", "old", "ideas"} {
		if err := f.m.store.DeleteList(ids[k]); err != nil {
			t.Fatal(err)
		}
	}
	f.m = f.m.settle()
	f.open(ids["ship"])
	f.keys("m")
	if f.m.mode != modeNormal || f.m.status == "" {
		t.Fatalf("mode=%v status=%q", f.m.mode, f.m.status)
	}
}

func TestSearchScopeAndToggle(t *testing.T) {
	f, ids := newFixture(t)
	if _, err := f.m.store.AddTaskEnd(ids["home"], "Reply to mum"); err != nil {
		t.Fatal(err)
	}
	f.open(ids["ship"])
	f.keys("/")
	if f.m.mode != modeSearch || f.m.search.allLists {
		t.Fatal("expected this-list search")
	}
	f.typed("rply") // subsequence, case-insensitive
	hits := f.m.searchHits()
	if len(hits) != 1 || hits[0].Task.ID != ids["reply"] || hits[0].Crumb != "" {
		t.Fatalf("hits: %+v", hits)
	}
	f.keys("tab")
	hits = f.m.searchHits()
	if len(hits) != 2 || hits[1].Crumb != "Home" {
		t.Fatalf("all-list hits: %+v", hits)
	}
	if !strings.Contains(f.m.View(), "all lists") {
		t.Fatal("scope not shown")
	}
	f.keys("tab")
	if f.m.search.allLists {
		t.Fatal("tab did not toggle back")
	}
}

func TestSearchCollapsedMatchAndJump(t *testing.T) {
	f, ids := newFixture(t)
	f.m.store.ToggleCollapsed(ids["refactor"])
	f.m.store.ToggleCollapsed(ids["extract"])
	f.m = f.m.settle()
	f.keys("tab", "/")
	f.typed("deep")
	hits := f.m.searchHits()
	if len(hits) != 1 || hits[0].Task.ID != ids["deep"] {
		t.Fatalf("hits: %+v", hits)
	}
	f.keys("enter")
	if f.m.mode != modeNormal || f.task() != ids["deep"] || f.m.pane != paneTasks {
		t.Fatalf("mode=%v task=%q", f.m.mode, f.task())
	}
	loc, _ := f.m.store.FindTask(ids["deep"])
	for _, a := range loc.Ancestors {
		if a.Collapsed {
			t.Fatal("ancestor still collapsed")
		}
	}
}

func TestSearchJumpAcrossListsHiddenAndDone(t *testing.T) {
	f, ids := newFixture(t)
	x, _ := f.m.store.AddTaskEnd(ids["old"], "Archived thing")
	f.m.store.ToggleDone(x.ID, f.m.now())
	f.m.hideDone[ids["old"]] = true
	f.star(ids["ship"])
	f.openFocus()
	f.keys("/")
	if !f.m.search.allLists {
		t.Fatal("focus search should be all-lists")
	}
	f.typed("archived")
	f.keys("enter")
	if f.side() != "list:"+ids["old"] || f.task() != x.ID || !f.m.hiddenOpen || f.m.hideDone[ids["old"]] {
		t.Fatalf("side=%q task=%q", f.side(), f.task())
	}
}

func TestSearchSelectAndEsc(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("/", "down", "down")
	f.typed("e")
	if f.m.search.sel != 0 {
		t.Fatal("typing should reset selection")
	}
	f.keys("down")
	if f.m.search.sel != 1 {
		t.Fatal("down did not move")
	}
	f.keys("esc")
	if f.m.mode != modeNormal || f.task() != ids["ship"] {
		t.Fatal("esc should exit without moving")
	}
	f.keys("/")
	if f.m.search.ti.Value() != "" {
		t.Fatal("query not cleared")
	}
}
