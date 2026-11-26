package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hqazi/task-tui/internal/model"
)

// currentList returns the list shown in the task pane, or nil (Focus, or the
// "Hidden" section row).
func (m Model) currentList() *model.List {
	it := m.selectedItem()
	if it.kind == itemList || it.kind == itemHiddenList {
		return m.store.List(it.listID)
	}
	return nil
}

// rows returns the visible task rows of the current list.
func (m Model) rows() []model.Row {
	if l := m.currentList(); l != nil {
		return model.Flatten(l, m.hideDone[l.ID])
	}
	return nil
}

// taskIndex returns the index of the selected task in rows. If the selected
// task is gone or hidden it falls back to its nearest visible ancestor, then to
// the last known index, so the cursor survives mutations.
func (m Model) taskIndex(rows []model.Row) int {
	if len(rows) == 0 {
		return -1
	}
	find := func(id string) int {
		for i, r := range rows {
			if r.Task.ID == id {
				return i
			}
		}
		return -1
	}
	if i := find(m.taskSel); i >= 0 {
		return i
	}
	if loc, err := m.store.FindTask(m.taskSel); err == nil {
		for j := len(loc.Ancestors) - 1; j >= 0; j-- {
			if i := find(loc.Ancestors[j].ID); i >= 0 {
				return i
			}
		}
	}
	return min(max(m.taskLast, 0), len(rows)-1)
}

// selectedTask returns the task under the cursor, or nil.
func (m Model) selectedTask() *model.Task {
	rows := m.rows()
	if i := m.taskIndex(rows); i >= 0 {
		return rows[i].Task
	}
	return nil
}

func (m Model) updateTasksKey(k string) Model {
	rows := m.rows()
	idx := m.taskIndex(rows)
	if idx < 0 {
		return m
	}
	t := rows[idx].Task
	switch k {
	case "up":
		idx = max(idx-1, 0)
	case "down":
		idx = min(idx+1, len(rows)-1)
	case "left":
		switch {
		case len(t.Children) > 0 && !t.Collapsed:
			m.store.ToggleCollapsed(t.ID)
			m.save()
		default:
			if loc, err := m.store.FindTask(t.ID); err == nil && loc.Parent != nil {
				m.taskSel = loc.Parent.ID
			}
			return m
		}
	case "right":
		if len(t.Children) > 0 && t.Collapsed {
			m.store.ToggleCollapsed(t.ID)
			m.save()
		}
		return m
	case "collapseAll", "expandAll":
		if l := m.currentList(); l != nil {
			m.store.SetAllCollapsed(l.ID, k == "collapseAll")
			m.save()
		}
	}
	rows = m.rows()
	if i := m.taskIndex(rows); k != "up" && k != "down" && i >= 0 {
		idx = i
	}
	if idx >= 0 && idx < len(rows) {
		m.taskSel = rows[idx].Task.ID
	}
	return m
}

// renderTasks renders the task pane box of outer size w x h.
func (m Model) renderTasks(w, h int) string {
	inner := max(w-2, 1)
	active := m.pane == paneTasks
	title := m.theme.titleIdle
	box := m.theme.paneIdle
	if active {
		title, box = m.theme.title, m.theme.paneActive
	}
	var lines []string
	switch it := m.selectedItem(); it.kind {
	case itemFocus:
		lines = m.renderFocus(title, inner, h, active)
	case itemHiddenToggle:
		lines = append(lines, title.Render(" Hidden"), "", m.theme.muted.Render(" Select a hidden list (→ to expand)"))
	default:
		lines = append(lines, title.Render(ansi.Truncate(" "+it.label, inner, m.theme.icons.Ellipsis)))
		rows := m.rows()
		if len(rows) == 0 {
			if m.mode == modeInput && m.input.at.inline {
				lines = append(lines, m.renderInputRow(m.newRowPrefix(0), inner))
			} else {
				lines = append(lines, "", m.theme.muted.Render(" No tasks"))
			}
			break
		}
		sel := m.taskIndex(rows)
		n := m.bodyRows(h)
		var body []string
		focus := sel
		for i, r := range rows {
			if m.mode == modeInput && m.input.at.inline && m.input.at.editID == r.Task.ID {
				focus = len(body)
				body = append(body, m.renderInputRow(m.rowPrefix(r.Task, r.Depth), inner))
				continue
			}
			if i == sel {
				focus = len(body)
			}
			body = append(body, m.renderTaskRow(r, i == sel && active, i == sel, inner))
			if m.insertInputAfter(rows, i) {
				focus = len(body)
				body = append(body, m.renderInputRow(m.newRowPrefix(m.input.at.depth), inner))
			}
		}
		off := scrollTo(m.taskOff, focus, n, len(body))
		if m.mode != modeInput {
			off = m.taskOff
		}
		lines = append(lines, body[min(off, len(body)):min(off+n, len(body))]...)
	}
	return box.Width(inner).Height(max(h-2, 1)).MaxHeight(h).Render(strings.Join(lines, "\n"))
}

// renderTaskRow renders one row exactly w cells wide. highlight draws the
// selection background; cursor marks the selected row even in an idle pane.
func (m Model) renderTaskRow(r model.Row, highlight, cursor bool, w int) string {
	t, th := r.Task, m.theme
	base, text, star, muted := th.row, th.row, th.star, th.muted
	if highlight {
		base, text, star, muted = th.rowSel, th.rowSel, th.starSel, th.mutedSel
	} else if cursor {
		base = th.rowDim
	}
	if t.Done {
		text = th.done
		if highlight {
			text = th.doneSel
		}
	}

	prefix := m.rowPrefix(t, r.Depth)

	var suffix string
	if d, tot := model.Progress(t); tot > 0 {
		suffix = fmt.Sprintf("%d/%d", d, tot)
	}
	starStr := ""
	if t.Today {
		starStr = th.icons.Star
	}
	sufW := lipgloss.Width(suffix)
	if starStr != "" {
		sufW += lipgloss.Width(starStr) + btoi(suffix != "")
	}
	if sufW > 0 {
		sufW++ // gap before suffix
	}
	avail := max(w-lipgloss.Width(prefix)-sufW-1, 0)
	body := ansi.Truncate(t.Text, avail, th.icons.Ellipsis)
	gap := max(w-lipgloss.Width(prefix)-lipgloss.Width(body)-sufW-1, 0)

	var b strings.Builder
	b.WriteString(base.Render(prefix))
	b.WriteString(text.Render(body))
	b.WriteString(base.Render(strings.Repeat(" ", gap)))
	if sufW > 0 {
		b.WriteString(base.Render(" "))
		b.WriteString(muted.Render(suffix))
		if starStr != "" {
			b.WriteString(base.Render(strings.Repeat(" ", btoi(suffix != ""))))
			b.WriteString(star.Render(starStr))
		}
	}
	b.WriteString(base.Render(" "))
	return b.String()
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// rowPrefix is the indent, arrow and icon drawn before a task's text.
func (m Model) rowPrefix(t *model.Task, depth int) string {
	th := m.theme
	arrow := " "
	if len(t.Children) > 0 {
		arrow = th.icons.Expanded
		if t.Collapsed {
			arrow = th.icons.Collapsed
		}
	}
	icon := th.icons.Open
	if t.Done {
		icon = th.icons.Done
	}
	return " " + strings.Repeat("  ", depth) + arrow + " " + icon + " "
}

// newRowPrefix is the prefix of the not-yet-created row of an add input.
func (m Model) newRowPrefix(depth int) string {
	return " " + strings.Repeat("  ", depth) + "  " + m.theme.icons.Open + " "
}

// insertInputAfter reports whether the add input's new row goes right after
// rows[i], i.e. after the end of the visible subtree of the input's anchor.
func (m Model) insertInputAfter(rows []model.Row, i int) bool {
	at := m.input.at
	if m.mode != modeInput || !at.inline || at.editID != "" {
		return false
	}
	if at.afterID == "" {
		return i == len(rows)-1
	}
	for j, r := range rows {
		if r.Task.ID != at.afterID {
			continue
		}
		end := j
		for end+1 < len(rows) && rows[end+1].Depth > r.Depth {
			end++
		}
		return i == end
	}
	return i == len(rows)-1
}
