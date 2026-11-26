package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// pickerState is the move-to-list popup. It is valid in modePicker.
type pickerState struct {
	taskID    string
	fromFocus bool
	lists     []string // target list IDs, in store order
	sel       int
}

// startPicker opens the move popup for a task. With no other list it only
// shows a status message.
func (m Model) startPicker(taskID string, fromFocus bool) Model {
	loc, err := m.store.FindTask(taskID)
	if err != nil {
		m.status = err.Error()
		return m
	}
	var ids []string
	for _, l := range m.store.Lists {
		if l.ID != loc.List.ID {
			ids = append(ids, l.ID)
		}
	}
	if len(ids) == 0 {
		m.status = "no other list to move to"
		return m
	}
	m.picker = pickerState{taskID: taskID, fromFocus: fromFocus, lists: ids}
	m.mode = modePicker
	return m
}

func (m Model) updatePicker(msg tea.KeyMsg) Model {
	p := m.picker
	k := m.keys
	switch {
	case key.Matches(msg, k.Cancel):
		m.mode = modeNormal
		m.picker = pickerState{}
	case key.Matches(msg, k.Up):
		m.picker.sel = max(p.sel-1, 0)
	case key.Matches(msg, k.Down):
		m.picker.sel = min(p.sel+1, len(p.lists)-1)
	case key.Matches(msg, k.Submit):
		m.mode = modeNormal
		m.picker = pickerState{}
		m = m.moveToList(p.taskID, p.lists[p.sel], p.fromFocus)
	}
	return m
}

// moveToList moves a task with its subtree to the end of a list and saves.
// From the task pane the cursor goes to the next row; Focus is left alone,
// since the task keeps its ID and so its place in Focus.
func (m Model) moveToList(id, listID string, fromFocus bool) Model {
	next := ""
	if !fromFocus {
		rows := m.rows()
		for i, r := range rows {
			if r.Task.ID != id {
				continue
			}
			end := i
			for end+1 < len(rows) && rows[end+1].Depth > r.Depth {
				end++
			}
			switch {
			case end+1 < len(rows):
				next = rows[end+1].Task.ID
			case i > 0:
				next = rows[i-1].Task.ID
			}
			break
		}
	}
	if err := m.store.MoveTaskToList(id, listID); err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	if !fromFocus {
		m.taskSel = next
	}
	return m
}

func (m Model) renderPicker() string {
	th := m.theme
	w := min(max(m.w-8, 12), 40)
	var b strings.Builder
	b.WriteString(th.title.Render("Move to list") + "\n\n")
	for i, id := range m.picker.lists {
		l := m.store.List(id)
		if l == nil {
			continue
		}
		name, style := l.Name, th.row
		if l.Hidden {
			name += " (hidden)"
			style = th.muted
		}
		if i == m.picker.sel {
			style = th.rowSel
			if l.Hidden {
				style = th.mutedSel
			}
		}
		line := ansi.Truncate(" "+name, w, th.icons.Ellipsis)
		b.WriteString(style.Render(padRight(line, w)) + "\n")
	}
	b.WriteString("\n" + th.hintKey.Render("↑/↓") + " " + th.hint.Render("choose") + th.hint.Render(" · ") +
		th.hintKey.Render("enter") + " " + th.hint.Render("move") + th.hint.Render(" · ") +
		th.hintKey.Render("esc") + " " + th.hint.Render("cancel"))
	box := th.overlay.MaxWidth(max(m.w, 1)).Render(b.String())
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}
