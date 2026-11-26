package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hqazi/task-tui/internal/model"
)

// sampleStore builds 3 visible lists with nested, done and starred tasks and
// one hidden list.
func sampleStore(t *testing.T, now time.Time) *model.Store {
	t.Helper()
	s := &model.Store{Version: 1}
	must := func(tk *model.Task, err error) *model.Task {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return tk
	}
	work, _ := s.AddList("Work")
	home, _ := s.AddList("Home")
	ideas, _ := s.AddList("Ideas")
	old, _ := s.AddList("Archive of old projects")
	ship := must(s.AddTaskEnd(work.ID, "Ship v2"))
	cl := must(s.AddChild(ship.ID, "Write changelog"))
	tag := must(s.AddChild(ship.ID, "Tag release"))
	must(s.AddChild(cl.ID, "Collect merged PR titles from the last three sprints and group by area"))
	ref := must(s.AddTaskEnd(work.ID, "Refactor auth"))
	for _, x := range []string{"Extract token store", "Drop legacy sessions", "Add tests"} {
		must(s.AddChild(ref.ID, x))
	}
	reply := must(s.AddTaskEnd(work.ID, "Reply to Sam"))
	must(s.AddTaskEnd(work.ID, "A very long task name that certainly will not fit within the narrow task pane and must be truncated cleanly"))
	g := must(s.AddTaskEnd(home.ID, "Buy groceries"))
	must(s.AddTaskEnd(home.ID, "Book dentist"))
	must(s.AddTaskEnd(ideas.ID, "Rust rewrite?"))
	must(s.AddTaskEnd(old.ID, "Old thing"))
	for _, id := range []string{cl.ID, reply.ID, g.ID} {
		_ = s.ToggleToday(id)
	}
	_ = s.ToggleToday(tag.ID)
	_ = s.ToggleDone(tag.ID, now)
	_ = s.ToggleDone(g.ID, now)
	_ = s.ToggleListHidden(old.ID)
	return s
}

// renderFixture drives the sample store to a given state.
func renderFixture(t *testing.T, w, h int, keys ...string) *fixture {
	t.Helper()
	f := &fixture{t: t}
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	f.m = New(sampleStore(t, now), "/x", nil)
	f.m.now = func() time.Time { return now }
	f.send(tea.WindowSizeMsg{Width: w, Height: h})
	f.keys(keys...)
	return f
}

var renderStates = []struct {
	name string
	keys []string
}{
	{"sidebar", nil},
	{"tasks", []string{"tab", "down", "down", "right"}},
	{"focus", []string{"up", "tab"}},
	{"help", []string{"?"}},
	{"picker", []string{"tab", "m"}},
	{"search", []string{"tab", "/", "r", "e"}},
}

func TestViewFitsTerminal(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		for _, sz := range [][2]int{{100, 30}, {60, 20}, {40, 12}, {30, 10}} {
			for _, st := range renderStates {
				f := renderFixture(t, sz[0], sz[1])
				if ascii {
					f.m.theme = newTheme(true)
				}
				f.keys(st.keys...)
				out := f.m.View()
				lines := strings.Split(out, "\n")
				if len(lines) > sz[1] {
					t.Errorf("%s %v ascii=%v: %d lines > height %d", st.name, sz, ascii, len(lines), sz[1])
				}
				for i, l := range lines {
					if w := lipgloss.Width(l); w > sz[0] {
						t.Errorf("%s %v ascii=%v: line %d is %d wide > %d: %q", st.name, sz, ascii, i, w, sz[0], ansi.Strip(l))
					}
				}
			}
		}
	}
}

func TestASCIIFallback(t *testing.T) {
	for _, tc := range []struct {
		env   string
		ascii bool
	}{{"1", true}, {"", false}, {"0", false}} {
		t.Setenv("TASK_TUI_ASCII", tc.env)
		if got := asciiFromEnv(); got != tc.ascii {
			t.Errorf("TASK_TUI_ASCII=%q: ascii=%v, want %v", tc.env, got, tc.ascii)
		}
	}

	t.Setenv("TASK_TUI_ASCII", "1")
	for _, st := range renderStates {
		f := renderFixture(t, 100, 30, st.keys...)
		out := ansi.Strip(f.m.View())
		for _, r := range "○✓★▸▾›╭╮╰╯…" {
			if strings.ContainsRune(out, r) {
				t.Errorf("%s: ASCII mode still draws %q", st.name, r)
			}
		}
	}
	f := renderFixture(t, 100, 30, "tab", "down", "down")
	out := ansi.Strip(f.m.View())
	for _, want := range []string{"[ ] Write changelog", "[x] Tag release", "v [ ] Ship v2", "┌"} {
		if !strings.Contains(out, want) {
			t.Errorf("ASCII task view lacks %q:\n%s", want, out)
		}
	}
	f = renderFixture(t, 100, 30, "up", "tab")
	if out := ansi.Strip(f.m.View()); !strings.Contains(out, "Work > Ship v2") {
		t.Errorf("ASCII focus breadcrumb not converted:\n%s", out)
	}
}

func TestEmptyStates(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	check := func(name string, f *fixture, want string) {
		t.Helper()
		if v := ansi.Strip(f.m.View()); !strings.Contains(v, want) {
			t.Errorf("%s: view lacks %q:\n%s", name, want, v)
		}
	}
	f := &fixture{t: t}
	f.m = New(&model.Store{Version: 1}, "/x", nil)
	f.m.now = func() time.Time { return now }
	f.send(tea.WindowSizeMsg{Width: 100, Height: 30})
	check("no lists", f, "No lists yet")
	check("empty focus", f, "No tasks for today")

	s := &model.Store{Version: 1}
	l, _ := s.AddList("Empty")
	f.m = New(s, "/x", nil)
	f.send(tea.WindowSizeMsg{Width: 100, Height: 30})
	check("empty list", f, "No tasks yet")

	tk, _ := s.AddTaskEnd(l.ID, "only")
	_ = s.ToggleDone(tk.ID, now)
	f.m = New(s, "/x", nil)
	f.send(tea.WindowSizeMsg{Width: 100, Height: 30})
	f.keys("tab", "h")
	check("all done hidden", f, "Everything here is done")

	f = renderFixture(t, 100, 30, "tab", "/", "z", "z", "z")
	check("no search results", f, "No matches")
}
