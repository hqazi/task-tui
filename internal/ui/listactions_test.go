package ui

import (
	"strings"
	"testing"
)

func names(f *fixture) string {
	var n []string
	for _, l := range f.m.store.Lists {
		n = append(n, l.Name)
	}
	return strings.Join(n, ",")
}

func TestListAdd(t *testing.T) {
	f, _ := newFixture(t)
	f.keys("n")
	if f.m.mode != modeInput {
		t.Fatal("no input")
	}
	f.keys("Z", "enter")
	if names(f) != "Work,Home,Old,Ideas,Z" || f.saves == 0 {
		t.Fatalf("%s saves=%d", names(f), f.saves)
	}
	if f.side() != "list:"+f.m.store.Lists[4].ID {
		t.Fatalf("sel %q", f.side())
	}
	s := f.saves
	f.keys("n", "esc")
	if names(f) != "Work,Home,Old,Ideas,Z" || f.saves != s {
		t.Fatal("cancel mutated")
	}
}

func TestListAddFromFocus(t *testing.T) {
	f, _ := newFixture(t)
	f.keys("up", "n", "Q", "enter")
	if len(f.m.store.Lists) != 5 {
		t.Fatal("not added")
	}
}

func TestListRename(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("r")
	if f.m.input.ti.Value() != "Work" {
		t.Fatalf("initial %q", f.m.input.ti.Value())
	}
	f.keys("!", "enter")
	if f.m.store.List(ids["work"]).Name != "Work!" || f.saves == 0 {
		t.Fatal("not renamed")
	}
	f.keys("up", "r")
	if f.m.mode != modeNormal {
		t.Fatal("rename on Focus should be ignored")
	}
}

func TestListDelete(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("d")
	if f.m.mode != modeConfirm || !strings.Contains(f.m.confirm.message, "Work") || !strings.Contains(f.m.confirm.message, "7 tasks") {
		t.Fatalf("confirm %q", f.m.confirm.message)
	}
	f.keys("n")
	if len(f.m.store.Lists) != 4 {
		t.Fatal("deleted on no")
	}
	f.keys("d", "y")
	if names(f) != "Home,Old,Ideas" || f.side() != "list:"+ids["home"] {
		t.Fatalf("%s sel=%q", names(f), f.side())
	}
	// last visible list: selection goes to previous item
	f.keys("d", "y")
	if f.side() != "focus" {
		t.Fatalf("sel %q", f.side())
	}
}

func TestDeleteSelectsPrevious(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("down", "d", "y")
	if f.side() != "list:"+ids["work"] {
		t.Fatalf("sel %q", f.side())
	}
}

func TestDeleteHiddenLists(t *testing.T) {
	f, ids := newFixture(t)
	f.m.hiddenOpen = true
	f.m.sideSel = "list:" + ids["old"]
	f.keys("d", "y")
	if f.side() != "list:"+ids["ideas"] {
		t.Fatalf("sel %q", f.side())
	}
	f.keys("d", "y")
	if f.side() != "list:"+ids["home"] {
		t.Fatalf("sel %q", f.side())
	}
}

func TestDeleteAllShowsEmptyState(t *testing.T) {
	f, _ := newFixture(t)
	for len(f.m.store.Lists) > 0 {
		f.m.sideSel = f.m.sidebarItems()[1].key
		f.m.hiddenOpen = true
		f.m.sideSel = "list:" + f.m.store.Lists[0].ID
		f.keys("d", "y")
	}
	if f.side() != "focus" || !strings.Contains(f.m.View(), "press n to create") {
		t.Fatalf("sel %q\n%s", f.side(), f.m.View())
	}
}

func TestListReorder(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("shift+up")
	if names(f) != "Work,Home,Old,Ideas" {
		t.Fatal("moved past top")
	}
	f.keys("shift+down")
	if names(f) != "Home,Work,Old,Ideas" || f.side() != "list:"+ids["work"] {
		t.Fatalf("%s %q", names(f), f.side())
	}
	f.keys("shift+down") // no visible list below
	if names(f) != "Home,Work,Old,Ideas" {
		t.Fatal("crossed hidden boundary")
	}
	f.m.hiddenOpen = true
	f.m.sideSel = "list:" + ids["ideas"]
	f.keys("shift+up")
	if names(f) != "Home,Work,Ideas,Old" {
		t.Fatal(names(f))
	}
	// skips over different-hidden-state neighbours
	f.m.store.ToggleListHidden(ids["work"]) // Home, Work(h), Ideas(h), Old(h)... reset
	f.m.store.ToggleListHidden(ids["work"])
	f.m.store.Lists[0], f.m.store.Lists[1] = f.m.store.Lists[1], f.m.store.Lists[0] // Work,Home,Ideas,Old
	f.m.store.Lists[1], f.m.store.Lists[2] = f.m.store.Lists[2], f.m.store.Lists[1] // Work,Ideas,Home,Old
	f.m.sideSel = "list:" + ids["home"]
	f.keys("shift+up")
	if names(f) != "Home,Work,Ideas,Old" && names(f) != "Work,Home,Ideas,Old" {
		t.Fatal(names(f))
	}
}

func TestListHide(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("v")
	if !f.m.store.List(ids["work"]).Hidden || f.side() != "list:"+ids["home"] {
		t.Fatalf("sel %q", f.side())
	}
	f.keys("v") // Home hidden, no next/prev visible: Focus
	if f.side() != "focus" {
		t.Fatalf("sel %q", f.side())
	}
	// expanded: selection follows
	f.m.hiddenOpen = true
	f.m.sideSel = "list:" + ids["work"]
	f.keys("v")
	if f.m.store.List(ids["work"]).Hidden || f.side() != "list:"+ids["work"] {
		t.Fatal("unhide")
	}
	f.keys("v")
	if f.side() != "list:"+ids["work"] {
		t.Fatalf("sel should follow, got %q", f.side())
	}
	f.m.sideSel = keyHidden
	f.keys("v", "r", "d")
	if f.m.mode != modeNormal {
		t.Fatal("keys on toggle row should be ignored")
	}
}

func TestListHideSelectsPrevWhenLast(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("down", "v")
	if f.side() != "list:"+ids["work"] {
		t.Fatalf("sel %q", f.side())
	}
}

func TestListKeysIgnoredInTaskPane(t *testing.T) {
	f, _ := newFixture(t)
	f.keys("tab", "v")
	if f.m.store.Lists[0].Hidden {
		t.Fatal("v acted in task pane")
	}
}
