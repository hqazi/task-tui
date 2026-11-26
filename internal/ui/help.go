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
	case m.pane == paneSidebar:
		return []key.Binding{k.Up, k.Tab, k.ListAdd, k.ListRename, k.ListDelete, k.ListUp, k.ListHide, k.Help, k.Quit}
	default:
		return []key.Binding{k.Add, k.AddChild, k.Edit, k.Toggle, k.Today, k.Delete, k.Indent, k.Back, k.Help, k.Quit}
	}
}

// helpGroups returns the sections shown in the ? overlay.
func (m Model) helpGroups() []helpGroup {
	k := m.keys
	return []helpGroup{
		{"Navigation", []key.Binding{k.Up, k.Tab, k.Back}},
		{"Tasks", []key.Binding{k.Add, k.AddEnd, k.AddChild, k.Edit, k.Toggle, k.Delete, k.Today, k.HideDone, k.ClearDone}},
		{"Organise", []key.Binding{k.MoveUp, k.MoveDown, k.Indent, k.Outdent, k.Left, k.Right, k.CollapseAll, k.ExpandAll}},
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
		return m.theme.err.Render(ansi.Truncate(" "+m.status, m.w, "…"))
	}
	var parts []string
	for _, b := range m.hintBindings() {
		h := b.Help()
		parts = append(parts, m.theme.hintKey.Render(h.Key)+" "+m.theme.hint.Render(h.Desc))
	}
	sep := m.theme.hint.Render(" · ")
	return ansi.Truncate(" "+strings.Join(parts, sep), m.w, "…")
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(m.theme.title.Render("Keys") + "\n")
	for _, g := range m.helpGroups() {
		b.WriteString("\n" + m.theme.rowDim.Render(g.Title) + "\n")
		for _, bind := range g.Bindings {
			h := bind.Help()
			b.WriteString("  " + m.theme.hintKey.Render(padRight(h.Key, 8)) + m.theme.row.Render(h.Desc) + "\n")
		}
	}
	b.WriteString("\n" + m.theme.hint.Render("any key to close"))
	box := m.theme.overlay.Render(b.String())
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}

func padRight(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}
