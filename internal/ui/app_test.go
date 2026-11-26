package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hqazi/task-tui/internal/model"
)

type fixture struct {
	t     *testing.T
	m     Model
	saves int
}

// newFixture builds: Work (Ship v2{changelog, tag}, Refactor{a}, Reply),
// Home, and hidden lists Old and Ideas.
func newFixture(t *testing.T) (*fixture, map[string]string) {
	t.Helper()
	s := &model.Store{Version: 1}
	ids := map[string]string{}
	work, _ := s.AddList("Work")
	home, _ := s.AddList("Home")
	old, _ := s.AddList("Old")
	ideas, _ := s.AddList("Ideas")
	s.ToggleListHidden(old.ID)
	s.ToggleListHidden(ideas.ID)
	ids["work"], ids["home"], ids["old"], ids["ideas"] = work.ID, home.ID, old.ID, ideas.ID
	add := func(name string, task *model.Task, err error) {
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = task.ID
	}
	ship, err := s.AddTaskEnd(work.ID, "Ship v2")
	add("ship", ship, err)
	c, err := s.AddChild(ship.ID, "Write changelog")
	add("changelog", c, err)
	c, err = s.AddChild(ship.ID, "Tag release")
	add("tag", c, err)
	r, err := s.AddTaskEnd(work.ID, "Refactor auth")
	add("refactor", r, err)
	c, err = s.AddChild(r.ID, "Extract")
	add("extract", c, err)
	c, err = s.AddChild(c.ID, "Deep child")
	add("deep", c, err)
	c, err = s.AddTaskEnd(work.ID, "Reply to Sam")
	add("reply", c, err)

	f := &fixture{t: t}
	f.m = New(s, "/x", func(string, *model.Store) error { f.saves++; return nil })
	f.m.now = func() time.Time { return time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC) }
	f.send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return f, ids
}

func (f *fixture) send(msg tea.Msg) {
	f.t.Helper()
	nm, _ := f.m.Update(msg)
	f.m = nm.(Model)
}

func (f *fixture) keys(names ...string) {
	f.t.Helper()
	for _, n := range names {
		var km tea.KeyMsg
		switch n {
		case "up":
			km = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			km = tea.KeyMsg{Type: tea.KeyDown}
		case "left":
			km = tea.KeyMsg{Type: tea.KeyLeft}
		case "right":
			km = tea.KeyMsg{Type: tea.KeyRight}
		case "tab":
			km = tea.KeyMsg{Type: tea.KeyTab}
		case "shift+tab":
			km = tea.KeyMsg{Type: tea.KeyShiftTab}
		case "shift+up":
			km = tea.KeyMsg{Type: tea.KeyShiftUp}
		case "shift+down":
			km = tea.KeyMsg{Type: tea.KeyShiftDown}
		case "esc":
			km = tea.KeyMsg{Type: tea.KeyEsc}
		case "space":
			km = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
		case "enter":
			km = tea.KeyMsg{Type: tea.KeyEnter}
		default:
			km = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(n)}
		}
		f.send(km)
	}
}

func (f *fixture) side() string { return f.m.sideSel }
func (f *fixture) task() string { return f.m.taskSel }

func TestStartsOnFirstVisibleList(t *testing.T) {
	f, ids := newFixture(t)
	if f.side() != "list:"+ids["work"] || f.m.pane != paneSidebar {
		t.Fatalf("sel=%q pane=%v", f.side(), f.m.pane)
	}
	if f.task() != ids["ship"] {
		t.Fatalf("task = %q", f.task())
	}
}

func TestSidebarItems(t *testing.T) {
	f, ids := newFixture(t)
	keys := func() []string {
		var ks []string
		for _, it := range f.m.sidebarItems() {
			ks = append(ks, it.key)
		}
		return ks
	}
	want := []string{"focus", "list:" + ids["work"], "list:" + ids["home"], "hidden"}
	if got := keys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("collapsed: %v", got)
	}
	if lbl := f.m.sidebarItems()[3].label; lbl != "Hidden (2)" {
		t.Fatalf("label %q", lbl)
	}
	f.m.hiddenOpen = true
	if got := keys(); len(got) != 6 || got[4] != "list:"+ids["old"] || got[5] != "list:"+ids["ideas"] {
		t.Fatalf("expanded: %v", got)
	}
}

func TestSidebarNavigationAndHiddenToggle(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("up")
	if f.side() != "focus" {
		t.Fatalf("got %q", f.side())
	}
	f.keys("up") // clamp
	if f.side() != "focus" {
		t.Fatal("did not clamp at top")
	}
	f.keys("down", "down", "down")
	if f.side() != "hidden" {
		t.Fatalf("got %q", f.side())
	}
	f.keys("down") // collapsed: clamp at bottom
	if f.side() != "hidden" {
		t.Fatal("did not clamp at bottom")
	}
	f.keys("right")
	if !f.m.hiddenOpen {
		t.Fatal("-> should expand hidden")
	}
	f.keys("down")
	if f.side() != "list:"+ids["old"] {
		t.Fatalf("got %q", f.side())
	}
	if f.m.currentList().ID != ids["old"] {
		t.Fatal("task pane should show hidden list")
	}
	f.keys("left") // child -> parent row
	if f.side() != "hidden" || !f.m.hiddenOpen {
		t.Fatalf("sel=%q open=%v", f.side(), f.m.hiddenOpen)
	}
	f.keys("left")
	if f.m.hiddenOpen {
		t.Fatal("<- on header should collapse")
	}
	f.keys("enter")
	if !f.m.hiddenOpen {
		t.Fatal("enter should toggle open")
	}
	f.keys("enter")
	if f.m.hiddenOpen {
		t.Fatal("enter should toggle closed")
	}
}

func TestPaneSwitch(t *testing.T) {
	f, _ := newFixture(t)
	f.keys("tab")
	if f.m.pane != paneTasks {
		t.Fatal("tab should focus tasks")
	}
	f.keys("esc")
	if f.m.pane != paneSidebar {
		t.Fatal("esc should return to sidebar")
	}
	f.keys("right")
	if f.m.pane != paneTasks {
		t.Fatal("right should focus tasks")
	}
	f.keys("esc", "enter")
	if f.m.pane != paneTasks {
		t.Fatal("enter should focus tasks")
	}
}

func TestTaskNavigationAndCollapse(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("tab")
	// Rows: ship, changelog, tag, refactor, extract, deep, reply
	f.keys("down", "down")
	if f.task() != ids["tag"] {
		t.Fatalf("got %q", f.task())
	}
	f.keys("left") // leaf -> parent
	if f.task() != ids["ship"] {
		t.Fatalf("leaf left should jump to parent, got %q", f.task())
	}
	f.keys("left") // collapse ship
	ship, _ := f.m.store.FindTask(ids["ship"])
	if !ship.Task.Collapsed || f.saves != 1 {
		t.Fatalf("collapsed=%v saves=%d", ship.Task.Collapsed, f.saves)
	}
	if n := len(f.m.rows()); n != 5 {
		t.Fatalf("rows = %d", n)
	}
	f.keys("left") // top-level collapsed: no-op
	if f.task() != ids["ship"] {
		t.Fatal("moved")
	}
	f.keys("right")
	if ship.Task.Collapsed || f.saves != 2 {
		t.Fatal("-> should expand")
	}
	f.keys("up", "up") // clamp
	if f.task() != ids["ship"] {
		t.Fatal("clamp top")
	}
}

func TestCollapseExpandAllKeepsCursor(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("tab", "down", "down", "down", "down", "down") // deep
	if f.task() != ids["deep"] {
		t.Fatalf("got %q", f.task())
	}
	f.keys("-")
	for _, id := range []string{"ship", "refactor", "extract"} {
		l, _ := f.m.store.FindTask(ids[id])
		if !l.Task.Collapsed {
			t.Fatalf("%s not collapsed", id)
		}
	}
	if n := len(f.m.rows()); n != 3 {
		t.Fatalf("rows = %d", n)
	}
	if f.task() != ids["refactor"] {
		t.Fatalf("cursor should move to visible ancestor, got %q", f.task())
	}
	f.keys("+")
	if n := len(f.m.rows()); n != 7 {
		t.Fatalf("rows = %d", n)
	}
	if f.task() != ids["refactor"] {
		t.Fatal("cursor should stay")
	}
	if f.saves != 2 {
		t.Fatalf("saves = %d", f.saves)
	}
}

func TestSelectionSurvivesMutation(t *testing.T) {
	f, ids := newFixture(t)
	f.keys("tab", "down", "down", "down", "down") // extract
	// Insert a task above via the model: index shifts, ID selection must hold.
	if _, err := f.m.store.AddTaskEnd(ids["work"], "x"); err != nil {
		t.Fatal(err)
	}
	f.m.store.MoveTask(ids["reply"], -1)
	f.send(nil)
	if f.task() != ids["extract"] {
		t.Fatalf("got %q", f.task())
	}
	// Deleting the selected task falls back to a valid row.
	f.m.store.DeleteTask(ids["extract"])
	f.send(nil)
	if f.task() == "" || f.m.selectedTask() == nil {
		t.Fatal("no selection after delete")
	}
}

func TestFocusPlaceholderAndHelp(t *testing.T) {
	f, _ := newFixture(t)
	f.keys("up", "tab")
	if f.m.selectedTask() != nil {
		t.Fatal("focus has no task rows yet")
	}
	f.keys("down", "-", "+") // must not panic
	if !strings.Contains(f.m.View(), "coming soon") {
		t.Fatal("placeholder missing")
	}
	f.keys("?")
	if f.m.mode != modeHelp || !strings.Contains(f.m.View(), "Keys") {
		t.Fatal("help overlay not shown")
	}
	f.keys("x")
	if f.m.mode != modeNormal {
		t.Fatal("help should close on any key")
	}
}

func TestQuit(t *testing.T) {
	f, _ := newFixture(t)
	for _, msg := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("q")}, {Type: tea.KeyCtrlC}} {
		_, cmd := f.m.Update(msg)
		if cmd == nil {
			t.Fatalf("%v: no quit cmd", msg)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%v: not QuitMsg", msg)
		}
	}
}

func TestSaveErrorShownInHintBar(t *testing.T) {
	f, _ := newFixture(t)
	f.m.saver = func(string, *model.Store) error { return errors.New("disk full") }
	f.keys("tab", "-")
	if !strings.Contains(f.m.View(), "disk full") {
		t.Fatal("save error not shown")
	}
}

func TestViewRendersAtAllSizes(t *testing.T) {
	f, _ := newFixture(t)
	f.m.hiddenOpen = true
	long := strings.Repeat("very long task text ", 20)
	ship, _ := f.m.store.FindTask(f.m.taskSel)
	ship.Task.Text = long
	ship.Task.Today = true
	for _, size := range [][2]int{{0, 0}, {1, 1}, {5, 3}, {20, 6}, {39, 10}, {40, 4}, {60, 8}, {80, 24}, {200, 60}} {
		f.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, pane := range []pane{paneSidebar, paneTasks} {
			f.m.pane = pane
			for _, k := range []string{"down", "down", "up", "-", "+", "?", "x"} {
				f.keys(k)
				v := f.m.View()
				if size[0] > 0 && size[1] > 0 {
					lines := strings.Split(v, "\n")
					if len(lines) > size[1] && f.m.mode != modeHelp {
						t.Fatalf("size %v: %d lines", size, len(lines))
					}
				}
			}
		}
	}
}

func TestScrollKeepsSelectionVisible(t *testing.T) {
	f, _ := newFixture(t)
	l := f.m.currentList()
	for i := 0; i < 40; i++ {
		f.m.store.AddTaskEnd(l.ID, "task")
	}
	f.send(tea.WindowSizeMsg{Width: 80, Height: 10})
	f.keys("tab")
	for i := 0; i < 30; i++ {
		f.keys("down")
	}
	idx := f.m.taskIndex(f.m.rows())
	n := f.m.bodyRows(9)
	if idx < f.m.taskOff || idx >= f.m.taskOff+n {
		t.Fatalf("idx %d outside window [%d,%d)", idx, f.m.taskOff, f.m.taskOff+n)
	}
	v := f.m.View()
	if got := len(strings.Split(v, "\n")); got != 10 {
		t.Fatalf("view has %d lines, want 10", got)
	}
}
