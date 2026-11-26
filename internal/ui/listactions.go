package ui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hqazi/task-tui/internal/model"
)

func isListKind(k sidebarKind) bool { return k == itemList || k == itemHiddenList }

func taskCount(ts []*model.Task) int {
	n := 0
	for _, t := range ts {
		n += 1 + taskCount(t.Children)
	}
	return n
}

// updateListAction handles the list-management keys of the sidebar.
func (m Model) updateListAction(msg tea.KeyMsg, items []sidebarItem, idx int) Model {
	k := m.keys
	if key.Matches(msg, k.ListAdd) {
		return m.startInput("New list:", "", inputAt{}, func(m Model, text string) Model {
			l, err := m.store.AddList(text)
			if err != nil {
				m.status = err.Error()
				return m
			}
			m.save()
			m.selectSidebar("list:" + l.ID)
			return m
		})
	}
	if idx < 0 || !isListKind(items[idx].kind) {
		return m
	}
	it := items[idx]
	id := it.listID
	switch {
	case key.Matches(msg, k.ListRename):
		return m.startInput("Rename list:", it.label, inputAt{}, func(m Model, text string) Model {
			if err := m.store.RenameList(id, text); err != nil {
				m.status = err.Error()
				return m
			}
			m.save()
			return m
		})
	case key.Matches(msg, k.ListDelete):
		l := m.store.List(id)
		if l == nil {
			return m
		}
		n := taskCount(l.Tasks)
		noun := " tasks"
		if n == 1 {
			noun = " task"
		}
		msg := "Delete list \"" + l.Name + "\" and its " + strconv.Itoa(n) + noun + "? (y/n)"
		return m.startConfirm(msg, func(m Model) Model { return m.deleteList(id) })
	case key.Matches(msg, k.ListUp):
		return m.moveList(id, -1)
	case key.Matches(msg, k.ListDown):
		return m.moveList(id, 1)
	case key.Matches(msg, k.ListHide):
		return m.toggleHidden(id, items, idx)
	}
	return m
}

func (m Model) deleteList(id string) Model {
	items := m.sidebarItems()
	idx := m.sidebarIndex(items)
	if err := m.store.DeleteList(id); err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	next := keyFocus
	if idx >= 0 {
		kind := items[idx].kind
		switch {
		case idx+1 < len(items) && items[idx+1].kind == kind:
			next = items[idx+1].key
		case idx > 0 && items[idx-1].kind == kind:
			next = items[idx-1].key
		case kind == itemHiddenList:
			// last hidden list gone: fall back to the last visible list.
			for _, it := range items {
				if it.kind == itemList && it.listID != id {
					next = it.key
				}
			}
		case idx > 0:
			next = items[idx-1].key
		}
	}
	m.selectSidebar(next)
	return m
}

// moveList swaps the list with its nearest neighbour of the same hidden state.
func (m Model) moveList(id string, dir int) Model {
	ls := m.store.Lists
	i := -1
	for x, l := range ls {
		if l.ID == id {
			i = x
		}
	}
	if i < 0 {
		return m
	}
	j := i + dir
	for j >= 0 && j < len(ls) && ls[j].Hidden != ls[i].Hidden {
		j += dir
	}
	if j < 0 || j >= len(ls) {
		return m
	}
	for x := i; x != j; x += dir {
		if _, err := m.store.MoveList(id, dir); err != nil {
			m.status = err.Error()
			return m
		}
	}
	m.save()
	return m
}

func (m Model) toggleHidden(id string, items []sidebarItem, idx int) Model {
	wasHidden := items[idx].kind == itemHiddenList
	if err := m.store.ToggleListHidden(id); err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	if wasHidden || m.hiddenOpen {
		return m // the list stays reachable; selection follows it
	}
	next := keyFocus
	for i := idx + 1; i < len(items); i++ {
		if items[i].kind == itemList {
			next = items[i].key
			break
		}
	}
	if next == keyFocus {
		for i := idx - 1; i >= 0; i-- {
			if items[i].kind == itemList {
				next = items[i].key
				break
			}
		}
	}
	m.selectSidebar(next)
	return m
}
