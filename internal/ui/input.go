package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// inputAt says where an input is drawn. A zero value draws it in the hint bar
// (used by T4 for list add/rename); the task pane sets inline and one of:
//   - editID: the input replaces that task's row;
//   - afterID: the input is a new row after that task's visible subtree, at
//     the given depth;
//   - neither: the input is a new row at the end of the list.
type inputAt struct {
	inline  bool
	editID  string
	afterID string
	depth   int
}

// inputState is a reusable single-line text input. onSubmit receives the
// trimmed, non-empty text after Enter; Esc and empty text cancel without
// calling it. onSubmit is responsible for mutating the store, calling
// m.save() and moving the cursor.
type inputState struct {
	ti       textinput.Model
	prompt   string // shown before the field when drawn in the hint bar
	at       inputAt
	onSubmit func(m Model, text string) Model
}

// startInput opens the text input and switches to modeInput.
func (m Model) startInput(prompt, initial string, at inputAt, onSubmit func(Model, string) Model) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()
	m.input = inputState{ti: ti, prompt: prompt, at: at, onSubmit: onSubmit}
	m.mode = modeInput
	return m
}

func (m Model) updateInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.input = inputState{}
		return m, nil
	case tea.KeyEnter:
		text := strings.TrimSpace(m.input.ti.Value())
		submit := m.input.onSubmit
		m.mode = modeNormal
		m.input = inputState{}
		if text == "" || submit == nil {
			return m, nil
		}
		return submit(m, text), nil
	}
	var cmd tea.Cmd
	m.input.ti, cmd = m.input.ti.Update(msg)
	return m, cmd
}

// renderInputRow renders the input as a task row exactly w cells wide, with
// the given prefix (indent and icon) before the field.
func (m Model) renderInputRow(prefix string, w int) string {
	th := m.theme
	ti := m.input.ti
	ti.Width = max(w-lipgloss.Width(prefix)-2, 1)
	field := ansi.Truncate(ti.View(), max(w-lipgloss.Width(prefix)-1, 0), "")
	out := th.rowSel.Render(prefix) + th.rowSel.Render(field)
	if pad := w - lipgloss.Width(out); pad > 0 {
		out += th.rowSel.Render(strings.Repeat(" ", pad))
	}
	return out
}

// renderInputBar renders the input in the hint-bar line.
func (m Model) renderInputBar() string {
	ti := m.input.ti
	ti.Width = max(m.w-lipgloss.Width(m.input.prompt)-3, 1)
	return ansi.Truncate(" "+m.theme.hintKey.Render(m.input.prompt)+" "+ti.View(), m.w, "")
}
