package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// confirmState is a reusable y/n modal. onYes runs only on y/Y; any n/N/Esc
// cancels. onYes is responsible for mutating the store and calling m.save().
type confirmState struct {
	message string
	onYes   func(Model) Model
}

// startConfirm opens the modal and switches to modeConfirm.
func (m Model) startConfirm(message string, onYes func(Model) Model) Model {
	m.confirm = confirmState{message: message, onYes: onYes}
	m.mode = modeConfirm
	return m
}

func (m Model) updateConfirm(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "y", "Y":
		yes := m.confirm.onYes
		m.mode = modeNormal
		m.confirm = confirmState{}
		if yes != nil {
			m = yes(m)
		}
	case "n", "N", "esc":
		m.mode = modeNormal
		m.confirm = confirmState{}
	}
	return m
}

func (m Model) renderConfirm() string {
	var b strings.Builder
	b.WriteString(m.theme.row.Render(m.confirm.message))
	b.WriteString("\n\n")
	b.WriteString(m.theme.hintKey.Render("y") + " " + m.theme.hint.Render("yes") + m.theme.hint.Render(" · "))
	b.WriteString(m.theme.hintKey.Render("n") + " " + m.theme.hint.Render("no"))
	box := m.theme.overlay.MaxWidth(max(m.w-2, 10)).Render(b.String())
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}
