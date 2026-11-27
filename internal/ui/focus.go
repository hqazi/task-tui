package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hqazi/task-tui/internal/model"
)

// inFocus reports whether the task pane is showing the Focus view.
func (m Model) inFocus() bool { return m.selectedItem().kind == itemFocus }

// focusIndex returns the index of the selected entry. If the selected task
// dropped out of Focus the cursor stays at the last known index, clamped.
func (m Model) focusIndex(es []model.FocusEntry) int {
	if len(es) == 0 {
		return -1
	}
	for i, e := range es {
		if e.Task.ID == m.focusSel {
			return i
		}
	}
	return min(max(m.focusLast, 0), len(es)-1)
}

// updateFocus handles keys while the Focus view is active in the task pane.
func (m Model) updateFocus(msg tea.KeyMsg) Model {
	k := m.keys
	if key.Matches(msg, k.Back) {
		m.pane = paneSidebar
		return m
	}
	if key.Matches(msg, k.Search) {
		return m.startSearch()
	}
	es := m.store.FocusTasks(m.now())
	idx := m.focusIndex(es)
	if idx < 0 {
		return m
	}
	t := es[idx].Task
	id := t.ID
	switch {
	case key.Matches(msg, k.Up):
		m.focusSel = es[max(idx-1, 0)].Task.ID
	case key.Matches(msg, k.Down):
		m.focusSel = es[min(idx+1, len(es)-1)].Task.ID
	case key.Matches(msg, k.MoveUp), key.Matches(msg, k.MoveDown):
		dir := 1
		if key.Matches(msg, k.MoveUp) {
			dir = -1
		}
		moved, err := m.store.MoveFocus(id, dir, m.now())
		if err != nil {
			m.status = err.Error()
		} else if moved {
			m.save()
		}
		m.focusSel = id
	case key.Matches(msg, k.Toggle):
		if err := m.store.ToggleDone(id, m.now()); err != nil {
			m.status = err.Error()
			return m
		}
		m.save()
		m.focusSel = id
	case key.Matches(msg, k.Unstar):
		if err := m.store.ToggleToday(id); err != nil {
			m.status = err.Error()
			return m
		}
		m.save()
	case key.Matches(msg, k.FocusEdit):
		return m.startEdit(t)
	case key.Matches(msg, k.Move):
		return m.startPicker(id, true)
	case key.Matches(msg, k.Jump):
		return m.jumpTo(id)
	case key.Matches(msg, k.Delete):
		if n := countDescendants(t); n > 0 {
			return m.startConfirm(fmt.Sprintf("Delete %q and its %d subtask(s)?", t.Text, n), func(m Model) Model {
				return m.deleteFocused(id)
			})
		}
		return m.deleteFocused(id)
	}
	return m
}

// deleteFocused deletes a task from Focus; the cursor stays at the same index.
func (m Model) deleteFocused(id string) Model {
	if err := m.store.DeleteTask(id); err != nil {
		m.status = err.Error()
		return m
	}
	m.save()
	return m
}

// jumpTo selects a task in its home list: it expands collapsed ancestors,
// reveals the Hidden section if needed, and focuses the task pane.
func (m Model) jumpTo(id string) Model {
	loc, err := m.store.FindTask(id)
	if err != nil {
		m.status = err.Error()
		return m
	}
	changed := false
	showDone := loc.Task.Done
	for _, a := range loc.Ancestors {
		if a.Collapsed {
			if err := m.store.ToggleCollapsed(a.ID); err == nil {
				changed = true
			}
		}
		showDone = showDone || a.Done
	}
	if changed {
		m.save()
	}
	if loc.List.Hidden {
		m.hiddenOpen = true
	}
	if showDone {
		m.hideDone[loc.List.ID] = false
	}
	m.selectSidebar("list:" + loc.List.ID)
	m.taskSel = id
	m.pane = paneTasks
	return m
}

// renderFocus returns the lines of the Focus view (title first) for a pane of
// inner width w and outer height h.
func (m Model) renderFocus(title lipgloss.Style, w, h int, active bool) []string {
	es := m.store.FocusTasks(m.now())
	done := 0
	for _, e := range es {
		if e.Task.Done {
			done++
		}
	}
	head := fmt.Sprintf(" FOCUS · %s · %d/%d", m.now().Format("Mon 2 Jan"), done, len(es))
	lines := []string{title.Render(ansi.Truncate(head, w, m.theme.icons.Ellipsis))}
	if len(es) == 0 {
		return append(lines, "", m.theme.muted.Render(ansi.Truncate(" No tasks for today — press f on any task to star it", w, m.theme.icons.Ellipsis)))
	}
	sel := m.focusIndex(es)
	n := m.bodyRows(h)
	off := scrollTo(m.focusOff, sel, n, len(es))
	for i := off; i < len(es) && i < off+n; i++ {
		e := es[i]
		if m.mode == modeInput && m.input.at.inline && m.input.at.editID == e.Task.ID {
			lines = append(lines, m.renderInputRow(m.focusPrefix(e.Task), w))
			continue
		}
		lines = append(lines, m.renderFocusRow(e, i == sel && active, i == sel, w))
	}
	return lines
}

func (m Model) focusPrefix(t *model.Task) string {
	icon := m.theme.icons.Open
	if t.Done {
		icon = m.theme.icons.Done
	}
	return " " + icon + " "
}

// renderFocusRow renders "icon text · crumb" exactly w cells wide, truncating
// the breadcrumb first so the task text stays readable.
func (m Model) renderFocusRow(e model.FocusEntry, highlight, cursor bool, w int) string {
	th := m.theme
	base, text, muted := th.row, th.row, th.muted
	if highlight {
		base, text, muted = th.rowSel, th.rowSel, th.mutedSel
	} else if cursor {
		base = th.rowDim
	}
	if e.Task.Done {
		text = th.done
		if highlight {
			text = th.doneSel
		}
	}
	prefix := m.focusPrefix(e.Task)
	const sep = " · "
	avail := max(w-lipgloss.Width(prefix)-1, 0)
	body, crumb := e.Task.Text, strings.ReplaceAll(e.Crumb, "›", th.icons.Sep)
	tw, cw, sw := lipgloss.Width(body), lipgloss.Width(crumb), lipgloss.Width(sep)
	switch {
	case tw+sw+cw <= avail:
	case avail-tw-sw >= min(cw, 8): // truncate the crumb only
		crumb = ansi.Truncate(crumb, avail-tw-sw, th.icons.Ellipsis)
	default: // share: crumb keeps a small slice, text gets the rest
		cb := min(cw, 8)
		if avail < sw+cb+4 {
			cb = 0
		}
		if cb == 0 {
			crumb = ""
			body = ansi.Truncate(body, avail, th.icons.Ellipsis)
		} else {
			crumb = ansi.Truncate(crumb, cb, th.icons.Ellipsis)
			body = ansi.Truncate(body, avail-sw-cb, th.icons.Ellipsis)
		}
	}
	var b strings.Builder
	b.WriteString(base.Render(prefix))
	b.WriteString(text.Render(body))
	used := lipgloss.Width(prefix) + lipgloss.Width(body)
	if crumb != "" {
		b.WriteString(base.Render(sep))
		b.WriteString(muted.Render(crumb))
		used += sw + lipgloss.Width(crumb)
	}
	b.WriteString(base.Render(strings.Repeat(" ", max(w-used, 0))))
	return b.String()
}
