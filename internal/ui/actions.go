package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hqazi/task-tui/internal/model"
)

// updateTaskAction handles the task-mutating keys of the task pane.
func (m Model) updateTaskAction(msg tea.KeyMsg) Model {
	l := m.currentList()
	if l == nil {
		return m
	}
	k := m.keys
	t := m.selectedTask()
	switch {
	case key.Matches(msg, k.Add):
		if t == nil {
			return m.startAdd(l, nil, addEnd)
		}
		return m.startAdd(l, t, addBelow)
	case key.Matches(msg, k.AddEnd):
		return m.startAdd(l, t, addEnd)
	case key.Matches(msg, k.AddChild):
		if t == nil {
			return m
		}
		return m.startAdd(l, t, addChild)
	}
	if key.Matches(msg, k.HideDone) {
		m.hideDone[l.ID] = !m.hideDone[l.ID]
		if t != nil && t.Done && m.hideDone[l.ID] {
			m.taskSel = "" // fall back to the row index, i.e. the next row
		}
		return m
	}
	if t == nil {
		return m
	}
	id := t.ID
	switch {
	case key.Matches(msg, k.Edit):
		return m.startEdit(t)
	case key.Matches(msg, k.Toggle):
		return m.mutate(m.store.ToggleDone(id, m.now()), id, m.hideDone[l.ID])
	case key.Matches(msg, k.Today):
		return m.mutate(m.store.ToggleToday(id), id, false)
	case key.Matches(msg, k.Delete):
		if n := countDescendants(t); n > 0 {
			return m.startConfirm(fmt.Sprintf("Delete %q and its %d subtask(s)?", t.Text, n), func(m Model) Model {
				return m.deleteTask(id)
			})
		}
		return m.deleteTask(id)
	case key.Matches(msg, k.MoveUp):
		return m.mutateMoved(m.store.MoveTask(id, -1))
	case key.Matches(msg, k.MoveDown):
		return m.mutateMoved(m.store.MoveTask(id, 1))
	case key.Matches(msg, k.Indent):
		return m.mutateMoved(m.store.Indent(id))
	case key.Matches(msg, k.Outdent):
		return m.mutateMoved(m.store.Outdent(id))
	case key.Matches(msg, k.ClearDone):
		n := countDone(l.Tasks)
		if n == 0 {
			return m
		}
		lid := l.ID
		return m.startConfirm(fmt.Sprintf("Remove %d done task(s) from %q?", n, l.Name), func(m Model) Model {
			if _, err := m.store.ClearDone(lid); err != nil {
				m.status = err.Error()
				return m
			}
			m.save()
			return m
		})
	}
	return m
}

type addKind int

const (
	addBelow addKind = iota
	addChild
	addEnd
)

// startAdd opens the inline input for a new task. ref is the cursor task
// (nil for an empty list or addEnd).
func (m Model) startAdd(l *model.List, ref *model.Task, kind addKind) Model {
	at := inputAt{inline: true}
	var refID string
	if ref != nil {
		refID = ref.ID
	}
	rows := m.rows()
	depth := 0
	if i := m.taskIndex(rows); i >= 0 && ref != nil {
		depth = rows[i].Depth
	}
	switch kind {
	case addBelow:
		at.afterID, at.depth = refID, depth
	case addChild:
		at.afterID, at.depth = refID, depth+1
	}
	lid := l.ID
	return m.startInput("add", "", at, func(m Model, text string) Model {
		var nt *model.Task
		var err error
		switch kind {
		case addBelow:
			nt, err = m.store.AddTaskAfter(refID, text)
		case addChild:
			nt, err = m.store.AddChild(refID, text)
		default:
			nt, err = m.store.AddTaskEnd(lid, text)
		}
		if err != nil {
			m.status = err.Error()
			return m
		}
		m.save()
		m.taskSel = nt.ID
		return m
	})
}

func (m Model) startEdit(t *model.Task) Model {
	id := t.ID
	return m.startInput("edit", t.Text, inputAt{inline: true, editID: id}, func(m Model, text string) Model {
		if err := m.store.EditTask(id, text); err != nil {
			m.status = err.Error()
			return m
		}
		m.save()
		m.taskSel = id
		return m
	})
}

func (m Model) deleteTask(id string) Model {
	// The cursor stays on the same row index, i.e. moves to the next task
	// (or the previous one when the last row was deleted); settle clamps it.
	if err := m.store.DeleteTask(id); err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	m.taskSel = ""
	return m
}

// mutate saves after a successful store call and keeps the cursor on id. With
// drop set the task may have just left the visible rows (hide-done), so the
// cursor falls back to the row index instead.
func (m Model) mutate(err error, id string, drop bool) Model {
	if err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	m.taskSel = id
	if drop {
		if loc, e := m.store.FindTask(id); e == nil && loc.Task.Done {
			m.taskSel = ""
		}
	}
	return m
}

// mutateMoved is mutate for store calls that report whether anything moved;
// the cursor stays on the selected task, which keeps its ID.
func (m Model) mutateMoved(moved bool, err error) Model {
	if err != nil {
		m.status = err.Error()
		return m
	}
	if moved {
		m.save()
	}
	return m
}

func countDescendants(t *model.Task) int {
	n := 0
	for _, c := range t.Children {
		n += 1 + countDescendants(c)
	}
	return n
}

func countDone(ts []*model.Task) int {
	n := 0
	for _, t := range ts {
		if t.Done {
			n++
		}
		n += countDone(t.Children)
	}
	return n
}
