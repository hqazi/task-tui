// Package ui is the Bubble Tea front end of task-tui.
//
// Extension points:
//   - add a key: put a key.Binding in keys.go, handle it in updateSidebar /
//     updateTasks (or a mode handler), and list it in help.go.
//   - add a mode: add a constant to mode, a case in handleKey, and a case in
//     View for any overlay. Mutations must call m.save().
package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hqazi/task-tui/internal/model"
)

type mode int

const (
	modeNormal mode = iota
	modeHelp
	modeInput   // inline text input, see input.go
	modeConfirm // y/n modal, see confirm.go
	modePicker  // move-to-list popup, see picker.go
	modeSearch  // live search, see search.go
)

type pane int

const (
	paneSidebar pane = iota
	paneTasks
)

// SaveFunc persists the store to path. main passes store.Save.
type SaveFunc func(path string, s *model.Store) error

// Model is the root Bubble Tea model.
type Model struct {
	store *model.Store
	path  string
	saver SaveFunc
	now   func() time.Time

	keys  keyMap
	theme theme

	mode   mode
	pane   pane
	w, h   int
	status string // transient error shown in the hint bar

	// Sidebar state. Selection is by stable key, not index.
	sideSel    string
	sideOff    int
	hiddenOpen bool

	// Task pane state. Selection is by task ID, not index.
	taskSel  string
	taskLast int // last known row index, fallback if taskSel vanishes
	taskOff  int
	hideDone map[string]bool // per list ID; UI state only, not persisted

	// Focus view state. Selection is by task ID, like the task pane.
	focusSel  string
	focusLast int
	focusOff  int

	input   inputState   // valid in modeInput
	confirm confirmState // valid in modeConfirm
	picker  pickerState  // valid in modePicker
	search  searchState  // valid in modeSearch
}

// New creates the root model. A nil save function disables persistence.
func New(s *model.Store, path string, save SaveFunc) Model {
	m := Model{
		store: s, path: path, saver: save, now: time.Now,
		keys: newKeyMap(), theme: newTheme(asciiFromEnv()),
		w: 80, h: 24,
		hideDone: map[string]bool{},
		sideSel:  keyFocus,
	}
	for _, l := range s.Lists {
		if !l.Hidden {
			m.sideSel = "list:" + l.ID
			break
		}
	}
	m.pane = paneSidebar
	return m.settle()
}

func (m Model) Init() tea.Cmd { return nil }

// save persists the store after a mutation. Failures show in the hint bar.
func (m *Model) save() {
	m.status = ""
	if m.saver == nil {
		return
	}
	if err := m.saver(m.path, m.store); err != nil {
		m.status = "save failed: " + err.Error()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m.settle(), nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) && (msg.String() == "ctrl+c" || m.mode == modeNormal) {
		return m, tea.Quit
	}
	switch m.mode {
	case modeHelp:
		m.mode = modeNormal
	case modeInput:
		nm, cmd := m.updateInput(msg)
		return nm.settle(), cmd
	case modeConfirm:
		m = m.updateConfirm(msg)
	case modePicker:
		m = m.updatePicker(msg)
	case modeSearch:
		nm, cmd := m.updateSearch(msg)
		return nm.settle(), cmd
	default:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.mode = modeHelp
		case m.pane == paneSidebar && key.Matches(msg, m.keys.Tab):
			m.pane = paneTasks
		case m.pane == paneSidebar:
			m = m.updateSidebar(msg)
		default:
			m = m.updateTasks(msg)
		}
	}
	return m.settle(), nil
}

func (m Model) updateSidebar(msg tea.KeyMsg) Model {
	items := m.sidebarItems()
	idx := m.sidebarIndex(items)
	k := m.keys
	switch {
	case key.Matches(msg, k.Up):
		return m.updateSidebarKey("up", items, idx)
	case key.Matches(msg, k.Down):
		return m.updateSidebarKey("down", items, idx)
	case key.Matches(msg, k.Left):
		return m.updateSidebarKey("left", items, idx)
	case key.Matches(msg, k.Right):
		if idx >= 0 && items[idx].kind != itemHiddenToggle {
			m.pane = paneTasks
			return m
		}
		return m.updateSidebarKey("right", items, idx)
	case key.Matches(msg, k.Select):
		if idx >= 0 && items[idx].kind != itemHiddenToggle {
			m.pane = paneTasks
			return m
		}
		return m.updateSidebarKey("toggle", items, idx)
	}
	return m.updateListAction(msg, items, idx)
}

func (m Model) updateTasks(msg tea.KeyMsg) Model {
	if m.inFocus() {
		return m.updateFocus(msg)
	}
	k := m.keys
	switch {
	case key.Matches(msg, k.Up):
		return m.updateTasksKey("up")
	case key.Matches(msg, k.Down):
		return m.updateTasksKey("down")
	case key.Matches(msg, k.Left):
		return m.updateTasksKey("left")
	case key.Matches(msg, k.Right):
		return m.updateTasksKey("right")
	case key.Matches(msg, k.CollapseAll):
		return m.updateTasksKey("collapseAll")
	case key.Matches(msg, k.ExpandAll):
		return m.updateTasksKey("expandAll")
	case key.Matches(msg, k.Back):
		m.pane = paneSidebar
		return m
	case key.Matches(msg, k.Search):
		return m.startSearch()
	}
	return m.updateTaskAction(msg)
}

// bodyRows is the number of item rows that fit in a pane of outer height h
// (border and title line excluded).
func (m Model) bodyRows(h int) int { return max(h-3, 1) }

// settle repairs selections after any change and scrolls to keep them visible.
func (m Model) settle() Model {
	items := m.sidebarItems()
	idx := m.sidebarIndex(items)
	if idx < 0 {
		idx = 0
		m.sideSel = items[0].key
	}
	n := m.bodyRows(max(m.h-1, 1))
	m.sideOff = scrollTo(m.sideOff, idx, n, len(items))

	rows := m.rows()
	if ti := m.taskIndex(rows); ti >= 0 {
		m.taskSel = rows[ti].Task.ID
		m.taskLast = ti
		m.taskOff = scrollTo(m.taskOff, ti, n, len(rows))
	} else {
		m.taskSel, m.taskLast, m.taskOff = "", 0, 0
	}

	es := m.store.FocusTasks(m.now())
	if fi := m.focusIndex(es); fi >= 0 {
		m.focusSel = es[fi].Task.ID
		m.focusLast = fi
		m.focusOff = scrollTo(m.focusOff, fi, n, len(es))
	} else {
		m.focusSel, m.focusLast, m.focusOff = "", 0, 0
	}
	return m
}

// scrollTo returns an offset that keeps idx inside a window of n of total rows.
func scrollTo(off, idx, n, total int) int {
	if idx < off {
		off = idx
	}
	if idx >= off+n {
		off = idx - n + 1
	}
	return max(min(off, total-n), 0)
}

func (m Model) View() string {
	if m.w < 1 || m.h < 1 {
		return ""
	}
	switch m.mode {
	case modeHelp:
		return m.renderHelp()
	case modeConfirm:
		return m.renderConfirm()
	case modePicker:
		return m.renderPicker()
	case modeSearch:
		if m.h >= 3 {
			return m.renderSearch()
		}
		return m.renderSearchBar()
	}
	if m.h < 3 { // no room for a bordered pane
		return m.renderHintBar()
	}
	bodyH := m.h - 1
	var body string
	if m.w < 40 { // too narrow for two panes: show only the active one
		if m.pane == paneSidebar {
			body = m.renderSidebar(m.w, bodyH)
		} else {
			body = m.renderTasks(m.w, bodyH)
		}
	} else {
		sw := min(max(m.w/4, 18), 30)
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderSidebar(sw, bodyH), m.renderTasks(m.w-sw, bodyH))
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, m.renderHintBar())
}
