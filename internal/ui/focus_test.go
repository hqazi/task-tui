package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
	"time"
)

// star stars tasks through the store, in order.
func (f *fixture) star(ids ...string) {
	f.t.Helper()
	for _, id := range ids {
		if err := f.m.store.ToggleToday(id); err != nil {
			f.t.Fatal(err)
		}
	}
	f.m = f.m.settle()
}

// openFocus selects Focus in the sidebar and enters the task pane.
func (f *fixture) openFocus() {
	f.t.Helper()
	f.keys("up")
	if f.side() != "focus" {
		f.t.Fatalf("side = %q", f.side())
	}
	f.keys("tab")
	if f.m.pane != paneTasks {
		f.t.Fatal("not in task pane")
	}
}

func TestFocusRenderBreadcrumbAndHiddenList(t *testing.T) {
	f, ids := newFixture(t)
	old, err := f.m.store.AddTaskEnd(ids["old"], "Archive stuff")
	if err != nil {
		t.Fatal(err)
	}
	f.star(ids["changelog"], old.ID, ids["reply"])
	f.keys("space") // no-op in sidebar
	f.openFocus()
	f.keys("space") // done on first entry
	v := f.m.View()
	for _, want := range []string{
		"FOCUS · Thu 26 Nov · 1/3",
		"✓ Write changelog · Work › Ship v2",
		"○ Archive stuff · Old",
		"○ Reply to Sam · Work",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("missing %q in:\n%s", want, v)
		}
	}
}

func TestFocusCrumbTruncatedKeepsWidth(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["deep"])
	f.send(tea.WindowSizeMsg{Width: 45, Height: 10})
	f.openFocus()
	for _, ln := range strings.Split(f.m.View(), "\n") {
		if w := lipgloss.Width(ln); w > 45 {
			t.Fatalf("line too wide (%d): %q", w, ln)
		}
	}
	if !strings.Contains(f.m.View(), "Deep child") {
		t.Fatal("task text must stay visible")
	}
}

func TestFocusReorderPersists(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["changelog"], ids["tag"], ids["reply"])
	f.openFocus()
	f.keys("down", "shift+up")
	want := []string{ids["tag"], ids["changelog"], ids["reply"]}
	if got := f.m.store.FocusOrder; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order %v want %v", got, want)
	}
	if f.m.focusSel != ids["tag"] {
		t.Fatalf("cursor %q", f.m.focusSel)
	}
	before := f.saves
	f.keys("shift+down")
	if f.saves != before+1 || f.m.store.FocusOrder[1] != ids["tag"] {
		t.Fatalf("order %v saves %d", f.m.store.FocusOrder, f.saves)
	}
	f.keys("shift+down", "shift+down") // bottom edge: second is a no-op
	if f.m.store.FocusOrder[2] != ids["tag"] {
		t.Fatalf("order %v", f.m.store.FocusOrder)
	}
}

func TestFocusDoneActsOnRealTask(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["reply"])
	f.openFocus()
	f.keys("space")
	if tk := f.m.store.List(ids["work"]).Tasks[2]; !tk.Done {
		t.Fatal("real task not done")
	}
	f.keys("space")
	if f.m.store.List(ids["work"]).Tasks[2].Done {
		t.Fatal("still done")
	}
}

func TestFocusUnstarKeepsIndex(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["changelog"], ids["tag"], ids["reply"])
	f.openFocus()
	f.keys("down", "f")
	if f.m.store.List(ids["work"]).Tasks[0].Children[1].Today {
		t.Fatal("still starred")
	}
	if f.m.focusSel != ids["reply"] || f.m.focusLast != 1 {
		t.Fatalf("sel %q last %d", f.m.focusSel, f.m.focusLast)
	}
	f.keys("f") // unstar last row: cursor clamps
	if f.m.focusSel != ids["changelog"] {
		t.Fatalf("sel %q", f.m.focusSel)
	}
	f.keys("f")
	if !strings.Contains(f.m.View(), "No tasks for today") {
		t.Fatal("empty state missing")
	}
}

func TestFocusDelete(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["reply"], ids["ship"], ids["tag"])
	f.openFocus()
	f.keys("down", "d") // Ship v2 has children
	if f.m.mode != modeConfirm {
		t.Fatal("expected confirm")
	}
	f.keys("n")
	if f.m.store.List(ids["work"]).Tasks[0].ID != ids["ship"] {
		t.Fatal("deleted despite cancel")
	}
	f.keys("d", "y")
	if _, err := f.m.store.FindTask(ids["ship"]); err == nil {
		t.Fatal("ship not deleted")
	}
	// Tag release was a child of Ship: gone from Focus too; cursor clamps.
	if es := f.m.store.FocusTasks(f.m.now()); len(es) != 1 || f.m.focusSel != ids["reply"] {
		t.Fatalf("entries %d sel %q", len(es), f.m.focusSel)
	}
	f.keys("d") // leaf: no confirm
	if f.m.mode != modeNormal {
		t.Fatal("unexpected confirm")
	}
	if _, err := f.m.store.FindTask(ids["reply"]); err == nil {
		t.Fatal("reply not deleted")
	}
}

func TestFocusEdit(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["reply"])
	f.openFocus()
	f.keys("e")
	if f.m.mode != modeInput {
		t.Fatal("no input")
	}
	f.keys("ctrl+u")
	f.typed("Call Sam")
	f.keys("enter")
	if got := f.m.store.List(ids["work"]).Tasks[2].Text; got != "Call Sam" {
		t.Fatalf("text %q", got)
	}
	if !strings.Contains(f.m.View(), "Call Sam · Work") {
		t.Fatal("not rendered")
	}
}

func TestFocusJumpExpandsAncestors(t *testing.T) {
	f, ids := newFixture(t)
	f.m.store.ToggleCollapsed(ids["refactor"])
	f.m.store.ToggleCollapsed(ids["extract"])
	f.star(ids["deep"])
	f.openFocus()
	f.keys("g")
	if f.side() != "list:"+ids["work"] || f.m.pane != paneTasks || f.m.taskSel != ids["deep"] {
		t.Fatalf("side %q pane %v task %q", f.side(), f.m.pane, f.m.taskSel)
	}
	if f.m.store.List(ids["work"]).Tasks[1].Collapsed {
		t.Fatal("ancestor still collapsed")
	}
	if !strings.Contains(f.m.View(), "Deep child") {
		t.Fatal("not visible")
	}
}

func TestFocusJumpHiddenListAndEnter(t *testing.T) {
	f, ids := newFixture(t)
	old, _ := f.m.store.AddTaskEnd(ids["old"], "Archive stuff")
	f.star(old.ID)
	f.openFocus()
	f.keys("enter")
	if !f.m.hiddenOpen || f.side() != "list:"+ids["old"] || f.m.taskSel != old.ID || f.m.pane != paneTasks {
		t.Fatalf("hidden %v side %q task %q", f.m.hiddenOpen, f.side(), f.m.taskSel)
	}
}

func TestFocusEmptyStateAndEsc(t *testing.T) {
	f, _ := newFixture(t)
	f.openFocus()
	if !strings.Contains(f.m.View(), "No tasks for today — press f on any task to star it") {
		t.Fatal("empty state missing")
	}
	f.keys("down", "space", "e", "d") // must not panic or change anything
	f.keys("esc")
	if f.m.pane != paneSidebar {
		t.Fatal("esc did not return")
	}
}

func TestFocusHidesTaskDoneYesterday(t *testing.T) {
	f, ids := newFixture(t)
	f.star(ids["reply"], ids["tag"])
	yesterday := f.m.now().Add(-24 * time.Hour)
	if err := f.m.store.ToggleDone(ids["reply"], yesterday); err != nil {
		t.Fatal(err)
	}
	if err := f.m.store.ToggleDone(ids["tag"], f.m.now()); err != nil {
		t.Fatal(err)
	}
	f.openFocus()
	v := f.m.View()
	if strings.Contains(v, "Reply to Sam") || !strings.Contains(v, "✓ Tag release") || !strings.Contains(v, "1/1") {
		t.Fatalf("unexpected view:\n%s", v)
	}
	// The next day the finished task drops out too.
	f.m.now = func() time.Time { return time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC) }
	f.m = f.m.settle()
	if !strings.Contains(f.m.View(), "No tasks for today") {
		t.Fatal("expected empty")
	}
}

func TestFocusHints(t *testing.T) {
	f, _ := newFixture(t)
	f.openFocus()
	var got []string
	for _, b := range f.m.hintBindings() {
		got = append(got, b.Help().Key)
	}
	if !strings.Contains(strings.Join(got, " "), "enter/g") {
		t.Fatalf("hints %v", got)
	}
}
