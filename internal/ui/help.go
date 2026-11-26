package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// hintBindings returns the context-sensitive bindings for the hint bar.
func (m Model) hintBindings() []key.Binding {
	k := m.keys
	switch {
	case m.mode == modeInput:
		return []key.Binding{k.Submit, k.Cancel}
	case m.mode == modeConfirm:
		return []key.Binding{k.Yes, k.No}
	case m.mode == modePicker:
		return []key.Binding{k.Up, k.Submit, k.Cancel}
	case m.mode == modeSearch:
		return []key.Binding{k.Up, k.Tab, k.Submit, k.Cancel}
	case m.pane == paneSidebar:
		return []key.Binding{k.Up, k.Tab, k.ListAdd, k.ListRename, k.ListDelete, k.ListUp, k.ListHide, k.Help, k.Quit}
	case m.inFocus():
		return []key.Binding{k.Up, k.MoveUp, k.Toggle, k.FocusEdit, k.Unstar, k.Delete, k.Jump, k.Move, k.Search, k.Back, k.Help, k.Quit}
	default:
		return []key.Binding{k.Add, k.AddChild, k.Edit, k.Toggle, k.Today, k.Delete, k.Indent, k.Move, k.Search, k.Back, k.Help, k.Quit}
	}
}

// helpGroups returns the sections shown in the ? overlay.
func (m Model) helpGroups() []helpGroup {
	k := m.keys
	return []helpGroup{
		{"Navigation", []key.Binding{k.Up, k.Tab, k.Back}},
		{"Tasks", []key.Binding{k.Add, k.AddEnd, k.AddChild, k.Edit, k.Toggle, k.Delete, k.Today, k.HideDone, k.ClearDone, k.Move, k.Search}},
		{"Organise", []key.Binding{k.MoveUp, k.MoveDown, k.Indent, k.Outdent, k.Left, k.Right, k.CollapseAll, k.ExpandAll}},
		{"Focus", []key.Binding{k.Jump, k.Unstar, k.MoveUp, k.MoveDown, k.Toggle, k.FocusEdit, k.Delete, k.Move, k.Search}},
		{"Sidebar", []key.Binding{k.Select, k.ListAdd, k.ListRename, k.ListDelete, k.ListUp, k.ListDown, k.ListHide}},
		{"General", []key.Binding{k.Help, k.Quit}},
	}
}

type helpGroup struct {
	Title    string
	Bindings []key.Binding
}

func (m Model) renderHintBar() string {
	if m.mode == modeInput && !m.input.at.inline {
		return m.renderInputBar()
	}
	if m.status != "" {
		return m.theme.err.Render(ansi.Truncate(" "+m.status, m.w, m.theme.icons.Ellipsis))
	}
	var parts []string
	for _, b := range m.hintBindings() {
		h := b.Help()
		parts = append(parts, m.theme.hintKey.Render(h.Key)+" "+m.theme.hint.Render(h.Desc))
	}
	sep := m.theme.hint.Render(" · ")
	return ansi.Truncate(" "+strings.Join(parts, sep), m.w, m.theme.icons.Ellipsis)
}

// helpBlock renders one help group as lines of exactly cw cells.
func (m Model) helpBlock(g helpGroup, keyW, cw int) []string {
	lines := []string{padRight(m.theme.rowDim.Render(g.Title), cw)}
	for _, bind := range g.Bindings {
		h := bind.Help()
		line := "  " + m.theme.hintKey.Render(padRight(h.Key, keyW+1)) + m.theme.row.Render(h.Desc)
		lines = append(lines, padRight(ansi.Truncate(line, cw, ""), cw))
	}
	return lines
}

// renderHelp draws the ? overlay. Groups flow into as few columns as needed to
// fit the terminal height; anything that still does not fit is clipped.
func (m Model) renderHelp() string {
	keyW, descW := 0, 0
	for _, g := range m.helpGroups() {
		for _, b := range g.Bindings {
			h := b.Help()
			keyW = max(keyW, lipgloss.Width(h.Key))
			descW = max(descW, lipgloss.Width(h.Desc))
		}
	}
	const gap = 3
	cw := 2 + keyW + 1 + descW
	// Overlay chrome: border 2 + side padding 4 horizontally; border 2 plus
	// title, blank, blank and footer vertically.
	availW := max(m.w-6, 1)
	availH := max(m.h-6, 1)
	maxCols := max((availW+gap)/(cw+gap), 1)
	groups := m.helpGroups()

	var cols [][]string
	for k := 1; k <= maxCols; k++ {
		cols = m.helpColumns(groups, k, keyW, cw)
		if k == maxCols || len(cols[0]) <= availH && tallest(cols) <= availH {
			break
		}
	}
	if len(cols) == 1 {
		cw = min(cw, availW)
	}
	var blocks []string
	for i, c := range cols {
		if len(c) > availH {
			c = append(c[:availH-1:availH-1], padRight(m.theme.muted.Render("  …"), cw))
		}
		st := lipgloss.NewStyle().Width(cw)
		if i < len(cols)-1 {
			st = st.MarginRight(gap)
		}
		blocks = append(blocks, st.Render(strings.Join(c, "\n")))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
	content := m.theme.title.Render("Keys") + "\n\n" + body + "\n\n" + m.theme.hint.Render("any key to close")
	box := m.theme.overlay.Padding(0, 2).MaxWidth(max(m.w, 1)).Render(content)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}

func tallest(cols [][]string) int {
	n := 0
	for _, c := range cols {
		n = max(n, len(c))
	}
	return n
}

// helpColumns flows groups into k columns of roughly equal height.
func (m Model) helpColumns(groups []helpGroup, k, keyW, cw int) [][]string {
	var blocks [][]string
	total := 0
	for _, g := range groups {
		b := m.helpBlock(g, keyW, cw)
		blocks = append(blocks, b)
		total += len(b) + 1
	}
	target := (total + k - 1) / k
	cols := make([][]string, 1, k)
	for _, b := range blocks {
		cur := len(cols) - 1
		if len(cols[cur]) > 0 && len(cols[cur])+len(b) > target && len(cols) < k {
			cols = append(cols, nil)
			cur++
		}
		if len(cols[cur]) > 0 {
			cols[cur] = append(cols[cur], padRight("", cw))
		}
		cols[cur] = append(cols[cur], b...)
	}
	return cols
}

func padRight(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}
