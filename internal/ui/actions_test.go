package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/hqazi/task-tui/internal/model"
)

// typed sends text as individual rune keys.
func (f *fixture) typed(text string) {
	f.t.Helper()
	for _, r := range text {
		f.keys(string(r))
	}
}

// open focuses the task pane on Work with the cursor on the given task.
func (f *fixture) open(id string) {
	f.t.Helper()
	if f.m.pane == paneSidebar {
		f.keys("tab")
	}
	f.m.taskSel = id
	f.m = f.m.settle()
}

func (f *fixture) wantSaves(n int) {
	f.t.Helper()
	if f.saves != n {
		f.t.Fatalf("saves = %d, want %d", f.saves, n)
	}
}

func TestAddBelowAndCursorFollows(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("a")
	if f.m.mode != modeInput {
		t.Fatal("not in input mode")
	}
	f.typed("New one")
	f.keys("enter")
	if f.m.mode != modeNormal {
		t.Fatal("still in input mode")
	}
	// Sibling directly below Ship v2 (after its subtree in display order).
	l := f.m.store.List(ids["work"])
	if l.Tasks[1].Text != "New one" || l.Tasks[0].ID != ids["ship"] {
		t.Fatalf("tasks: %v %v", l.Tasks[0].Text, l.Tasks[1].Text)
	}
	if f.task() != l.Tasks[1].ID {
		t.Fatal("cursor did not follow new task")
	}
	f.wantSaves(1)
}

func TestAddEnd(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("A")
	f.typed("Last")
	f.keys("enter")
	l := f.m.store.List(ids["work"])
	if got := l.Tasks[len(l.Tasks)-1]; got.Text != "Last" || f.task() != got.ID {
		t.Fatalf("got %q sel %q", got.Text, f.task())
	}
	f.wantSaves(1)
}

func TestAddSubtaskExpandsParent(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("left") // collapse Ship v2
	saves := f.saves
	f.keys("s")
	f.typed("Kid")
	f.keys("enter")
	ship, _ := f.m.store.FindTask(ids["ship"])
	kids := ship.Task.Children
	if ship.Task.Collapsed || kids[len(kids)-1].Text != "Kid" || f.task() != kids[len(kids)-1].ID {
		t.Fatalf("collapsed=%v sel=%q", ship.Task.Collapsed, f.task())
	}
	if f.saves != saves+1 {
		t.Fatalf("saves %d", f.saves)
	}
}

func TestAddIntoEmptyListBehavesLikeAddEnd(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("down") // Home (empty)
	if f.side() != "list:"+ids["home"] {
		t.Fatal(f.side())
	}
	f.keys("tab", "a")
	if f.m.mode != modeInput {
		t.Fatal("a on empty list should open input")
	}
	if !strings.Contains(f.m.View(), "○") {
		t.Fatal("inline input row not rendered")
	}
	f.typed("First")
	f.keys("enter")
	l := f.m.store.List(ids["home"])
	if len(l.Tasks) != 1 || f.task() != l.Tasks[0].ID {
		t.Fatalf("tasks %d sel %q", len(l.Tasks), f.task())
	}
	f.keys("s") // subtask works once a task exists
	f.typed("Sub")
	f.keys("enter")
	if len(l.Tasks[0].Children) != 1 {
		t.Fatal("no subtask")
	}
}

func TestInputCancelAndEmpty(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	before := len(f.m.store.List(ids["work"]).Tasks)
	f.keys("a")
	f.typed("abandoned")
	f.keys("esc")
	if f.m.mode != modeNormal || f.m.pane != paneTasks {
		t.Fatalf("mode %v pane %v", f.m.mode, f.m.pane)
	}
	f.keys("a", "enter") // empty add = cancel
	f.keys("a")
	f.typed("   ")
	f.keys("enter")
	if f.m.mode != modeNormal || len(f.m.store.List(ids["work"]).Tasks) != before {
		t.Fatal("cancel/empty must not add")
	}
	f.wantSaves(0)
}

func TestInputKeysAreText(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("a")
	f.typed("qdhC?") // bound keys must be plain text, q must not quit
	if f.m.mode != modeInput || f.m.input.ti.Value() != "qdhC?" {
		t.Fatalf("value %q mode %v", f.m.input.ti.Value(), f.m.mode)
	}
}

func TestEditCommitAndCancel(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("e")
	if f.m.input.ti.Value() != "Reply to Sam" {
		t.Fatalf("prefill %q", f.m.input.ti.Value())
	}
	f.keys("esc")
	f.wantSaves(0)
	f.keys("enter") // enter also edits
	f.typed("!")
	f.keys("enter")
	r, _ := f.m.store.FindTask(ids["reply"])
	if r.Task.Text != "Reply to Sam!" || f.task() != ids["reply"] {
		t.Fatalf("text %q sel %q", r.Task.Text, f.task())
	}
	f.wantSaves(1)
}

func TestToggleDoneAndHideDone(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("space")
	for _, id := range []string{"ship", "changelog", "tag"} {
		x, _ := f.m.store.FindTask(ids[id])
		if !x.Task.Done {
			t.Fatalf("%s not done", id)
		}
	}
	f.wantSaves(1)
	f.keys("space")
	x, _ := f.m.store.FindTask(ids["ship"])
	if x.Task.Done {
		t.Fatal("not undone")
	}
	// hide-done: completing the cursor task moves the cursor to the next row
	f.keys("h")
	f.open(ids["refactor"])
	f.keys("space")
	if f.task() != ids["reply"] {
		t.Fatalf("cursor %q, want reply (next row)", f.task())
	}
	f.keys("h") // show again: per-list state
	if f.m.hideDone[ids["work"]] {
		t.Fatal("h should toggle")
	}
}

func TestHideDoneIsPerList(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("h")
	f.keys("esc", "down") // Home
	f.keys("tab")
	f.keys("h", "esc", "up")
	if !f.m.hideDone[ids["work"]] || !f.m.hideDone[ids["home"]] {
		t.Fatalf("%v", f.m.hideDone)
	}
	f.keys("tab", "h")
	if f.m.hideDone[ids["work"]] || !f.m.hideDone[ids["home"]] {
		t.Fatalf("%v", f.m.hideDone)
	}
}

func TestToday(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("f")
	x, _ := f.m.store.FindTask(ids["reply"])
	if !x.Task.Today || f.task() != ids["reply"] {
		t.Fatal("not starred")
	}
	f.wantSaves(1)
	f.keys("f")
	if x, _ = f.m.store.FindTask(ids["reply"]); x.Task.Today {
		t.Fatal("not unstarred")
	}
}

func TestDeleteLeafImmediate(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["changelog"])
	f.keys("d")
	if f.m.mode != modeNormal {
		t.Fatal("leaf delete must not confirm")
	}
	if _, err := f.m.store.FindTask(ids["changelog"]); err == nil {
		t.Fatal("not deleted")
	}
	if f.task() != ids["tag"] {
		t.Fatalf("cursor %q, want next task", f.task())
	}
	f.keys("d") // tag is last child; cursor falls to the next row (refactor)
	if f.task() != ids["refactor"] {
		t.Fatalf("cursor %q", f.task())
	}
	f.wantSaves(2)
}

func TestDeleteLastRowMovesCursorUp(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("d")
	if f.task() != ids["deep"] {
		t.Fatalf("cursor %q, want previous row", f.task())
	}
}

func TestDeleteWithChildrenConfirm(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("d")
	if f.m.mode != modeConfirm {
		t.Fatal("expected confirm")
	}
	if v := f.m.View(); !strings.Contains(v, "Ship v2") || !strings.Contains(v, "2 subtask") {
		t.Fatalf("overlay: %s", v)
	}
	f.keys("x") // ignored
	if f.m.mode != modeConfirm {
		t.Fatal("other keys must be ignored")
	}
	f.keys("n")
	if _, err := f.m.store.FindTask(ids["ship"]); err != nil || f.saves != 0 || f.m.mode != modeNormal {
		t.Fatal("n must keep the task")
	}
	f.keys("d", "esc")
	if _, err := f.m.store.FindTask(ids["ship"]); err != nil || f.saves != 0 {
		t.Fatal("esc must keep the task")
	}
	f.keys("d", "y")
	if _, err := f.m.store.FindTask(ids["ship"]); err == nil {
		t.Fatal("y must delete")
	}
	if f.task() != ids["refactor"] || f.saves != 1 {
		t.Fatalf("cursor %q saves %d", f.task(), f.saves)
	}
}

func TestClearDone(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("C") // nothing done: no modal
	if f.m.mode != modeNormal {
		t.Fatal("no confirm without done tasks")
	}
	f.keys("space")
	f.open(ids["ship"])
	f.keys("C")
	if f.m.mode != modeConfirm {
		t.Fatal("expected confirm")
	}
	f.keys("n")
	if _, err := f.m.store.FindTask(ids["reply"]); err != nil {
		t.Fatal("n must keep done tasks")
	}
	saves := f.saves
	f.keys("C", "y")
	if _, err := f.m.store.FindTask(ids["reply"]); err == nil {
		t.Fatal("y must clear done tasks")
	}
	if f.saves != saves+1 || f.task() != ids["ship"] {
		t.Fatalf("saves %d sel %q", f.saves, f.task())
	}
}

func TestReorder(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("shift+up")
	l := f.m.store.List(ids["work"])
	if l.Tasks[1].ID != ids["reply"] || f.task() != ids["reply"] {
		t.Fatal("not moved up")
	}
	f.wantSaves(1)
	f.keys("shift+down", "shift+down") // second is a no-op at the end
	if l.Tasks[2].ID != ids["reply"] || f.task() != ids["reply"] {
		t.Fatal("not moved down")
	}
	f.wantSaves(2)
}

func TestIndentOutdent(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["reply"])
	f.keys("tab") // under Refactor auth
	r, _ := f.m.store.FindTask(ids["reply"])
	if r.Parent == nil || r.Parent.ID != ids["refactor"] || f.task() != ids["reply"] || f.m.pane != paneTasks {
		t.Fatalf("indent failed, pane %v", f.m.pane)
	}
	f.keys("shift+tab")
	r, _ = f.m.store.FindTask(ids["reply"])
	if r.Parent != nil {
		t.Fatal("outdent failed")
	}
	f.wantSaves(2)
	f.keys("shift+tab") // top level: no-op, no save
	f.open(ids["ship"])
	f.keys("tab") // first sibling: no-op
	f.wantSaves(2)
}

func TestActionsIgnoredInSidebarAndFocus(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("a", "space")
	if f.m.mode != modeNormal || f.saves != 0 {
		t.Fatal("sidebar must not run task actions")
	}
	f.keys("d", "n") // sidebar d is list delete; declined
	if f.saves != 0 || len(f.m.store.Lists) != 4 {
		t.Fatal("declined list delete mutated")
	}
	f.keys("up", "tab", "a", "d")
	if f.m.mode != modeNormal || f.saves != 0 || f.m.store.List(ids["work"]) == nil {
		t.Fatal("focus view has no actions yet")
	}
}

func TestInlineInputRendersAtRow(t *testing.T) {
	f, ids := newFixture(t)
	f.open(ids["ship"])
	f.keys("a")
	f.typed("zzz")
	lines := strings.Split(f.m.View(), "\n")
	var shipLine, inLine, replyLine = -1, -1, -1
	for i, ln := range lines {
		switch {
		case strings.Contains(ln, "zzz"):
			inLine = i
		case strings.Contains(ln, "Ship v2"):
			shipLine = i
		case strings.Contains(ln, "Reply to Sam"):
			replyLine = i
		}
	}
	// New sibling goes after Ship v2's subtree and before Refactor/Reply.
	if !(shipLine >= 0 && inLine > shipLine+2 && inLine < replyLine) {
		t.Fatalf("ship %d input %d reply %d", shipLine, inLine, replyLine)
	}
}

func TestSaveFailureShown(t *testing.T) {
	f, ids := newFixture(t)
	f.m.saver = func(string, *model.Store) error { return errors.New("boom") }
	f.open(ids["reply"])
	f.keys("f")
	if !strings.Contains(f.m.status, "boom") {
		t.Fatalf("status %q", f.m.status)
	}
}
