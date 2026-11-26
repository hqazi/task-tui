package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type sidebarKind int

const (
	itemFocus sidebarKind = iota
	itemList
	itemHiddenToggle
	itemHiddenList
)

const (
	keyFocus  = "focus"
	keyHidden = "hidden"
)

// sidebarItem is one selectable row of the sidebar. key is stable across
// mutations ("focus", "hidden", or "list:<id>"), so selection survives them.
type sidebarItem struct {
	key    string
	kind   sidebarKind
	listID string
	label  string
	count  int
}

// sidebarItems builds the selectable rows: Focus, visible lists, the
// "Hidden (n)" toggle (only when n > 0) and, when expanded, the hidden lists.
func (m Model) sidebarItems() []sidebarItem {
	items := []sidebarItem{{key: keyFocus, kind: itemFocus, label: "Focus", count: m.focusOpenCount()}}
	var hidden []sidebarItem
	for _, l := range m.store.Lists {
		it := sidebarItem{key: "list:" + l.ID, kind: itemList, listID: l.ID, label: l.Name, count: l.OpenCount()}
		if l.Hidden {
			it.kind = itemHiddenList
			hidden = append(hidden, it)
		} else {
			items = append(items, it)
		}
	}
	if len(hidden) > 0 {
		items = append(items, sidebarItem{key: keyHidden, kind: itemHiddenToggle, label: "Hidden (" + strconv.Itoa(len(hidden)) + ")"})
		if m.hiddenOpen {
			items = append(items, hidden...)
		}
	}
	return items
}

func (m Model) focusOpenCount() int {
	n := 0
	for _, e := range m.store.FocusTasks(m.now()) {
		if !e.Task.Done {
			n++
		}
	}
	return n
}

// sidebarIndex returns the index of the selected item, or -1.
func (m Model) sidebarIndex(items []sidebarItem) int {
	for i, it := range items {
		if it.key == m.sideSel {
			return i
		}
	}
	return -1
}

// selectedItem returns the selected sidebar item (falling back to Focus).
func (m Model) selectedItem() sidebarItem {
	items := m.sidebarItems()
	if i := m.sidebarIndex(items); i >= 0 {
		return items[i]
	}
	return items[0]
}

func (m *Model) selectSidebar(key string) {
	if key != m.sideSel {
		m.sideSel = key
		m.taskSel = ""
		m.taskOff = 0
	}
}

func (m Model) updateSidebarKey(k string, items []sidebarItem, idx int) Model {
	if idx < 0 {
		idx = 0
	}
	switch k {
	case "up":
		if idx > 0 {
			m.selectSidebar(items[idx-1].key)
		}
	case "down":
		if idx < len(items)-1 {
			m.selectSidebar(items[idx+1].key)
		}
	case "left":
		switch items[idx].kind {
		case itemHiddenToggle:
			m.hiddenOpen = false
		case itemHiddenList:
			m.selectSidebar(keyHidden)
		}
	case "right":
		if items[idx].kind == itemHiddenToggle {
			m.hiddenOpen = true
		}
	case "toggle":
		if items[idx].kind == itemHiddenToggle {
			m.hiddenOpen = !m.hiddenOpen
		}
	}
	return m
}

// renderSidebar renders the sidebar box of outer size w x h.
func (m Model) renderSidebar(w, h int) string {
	inner := max(w-2, 1)
	active := m.pane == paneSidebar
	title := m.theme.titleIdle
	box := m.theme.paneIdle
	if active {
		title, box = m.theme.title, m.theme.paneActive
	}
	items := m.sidebarItems()
	sel := m.sidebarIndex(items)
	lines := []string{title.Render(" Lists")}
	rows := m.bodyRows(h)
	for i := m.sideOff; i < len(items) && i < m.sideOff+rows; i++ {
		lines = append(lines, m.renderSidebarItem(items[i], i == sel, inner))
	}
	if len(m.store.Lists) == 0 {
		lines = append(lines, "", m.theme.muted.Render(" press n to create a list"))
	}
	return box.Width(inner).Height(max(h-2, 1)).MaxHeight(h).Render(strings.Join(lines, "\n"))
}

func (m Model) renderSidebarItem(it sidebarItem, selected bool, w int) string {
	t := m.theme
	left, right := it.label, ""
	switch it.kind {
	case itemFocus:
		left = t.icons.Star + " " + left
	case itemList:
		left = "  " + left
	case itemHiddenToggle:
		arrow := t.icons.Collapsed
		if m.hiddenOpen {
			arrow = t.icons.Expanded
		}
		left = arrow + " " + left
	case itemHiddenList:
		left = "    " + left
	}
	if it.kind != itemHiddenToggle {
		right = strconv.Itoa(it.count)
	}
	style := t.row
	switch {
	case selected && (it.kind == itemHiddenToggle || it.kind == itemHiddenList):
		style = t.rowDimSel
	case selected:
		style = t.rowSel
	case it.kind == itemHiddenToggle || it.kind == itemHiddenList:
		style = t.rowDim
	}
	return style.Render(spread(left, right, w, t.icons.Ellipsis))
}

// spread lays out left and right text in exactly w cells, truncating left.
func spread(left, right string, w int, ellipsis string) string {
	rw := lipgloss.Width(right)
	if rw > 0 {
		rw++ // one space of gap
	}
	avail := max(w-rw-1, 0) // 1 cell of left padding
	left = ansi.Truncate(left, avail, ellipsis)
	gap := max(w-1-lipgloss.Width(left)-lipgloss.Width(right)-1, 0)
	out := " " + left + strings.Repeat(" ", gap)
	if right != "" {
		out += right + " "
	}
	return ansi.Truncate(out, w, "") + strings.Repeat(" ", max(w-lipgloss.Width(out), 0))
}
