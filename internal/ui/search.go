package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hqazi/task-tui/internal/model"
)

// searchState is the live search. It is valid in modeSearch.
type searchState struct {
	ti       textinput.Model
	allLists bool
	sel, off int
}

// hit is one search result.
type hit struct {
	Task  *model.Task
	Crumb string // "List › Parent", set in all-lists scope only
}

// fuzzyMatch reports whether query is a case-insensitive subsequence of text.
func fuzzyMatch(query, text string) bool {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return true
	}
	i := 0
	for _, r := range strings.ToLower(text) {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
}

// startSearch opens the query input. In the task pane the scope starts as the
// current list; in Focus (or with no list selected) it is all lists.
func (m Model) startSearch() Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Focus()
	m.search = searchState{ti: ti, allLists: m.currentList() == nil}
	m.mode = modeSearch
	return m
}

// searchHits returns the tasks matching the query in the current scope,
// including those inside collapsed subtrees and done tasks.
func (m Model) searchHits() []hit {
	q := m.search.ti.Value()
	var out []hit
	var walk func(l *model.List, ts []*model.Task, crumb []string)
	walk = func(l *model.List, ts []*model.Task, crumb []string) {
		for _, t := range ts {
			if fuzzyMatch(q, t.Text) {
				h := hit{Task: t}
				if m.search.allLists {
					h.Crumb = strings.Join(crumb, " › ")
				}
				out = append(out, h)
			}
			walk(l, t.Children, append(crumb[:len(crumb):len(crumb)], t.Text))
		}
	}
	if m.search.allLists {
		for _, l := range m.store.Lists {
			walk(l, l.Tasks, []string{l.Name})
		}
	} else if l := m.currentList(); l != nil {
		walk(l, l.Tasks, []string{l.Name})
	}
	return out
}

func (m Model) updateSearch(msg tea.KeyMsg) (Model, tea.Cmd) {
	k := m.keys
	hits := m.searchHits()
	switch {
	case key.Matches(msg, k.Cancel):
		m.mode = modeNormal
		m.search = searchState{}
		return m, nil
	case key.Matches(msg, k.Up):
		m.search.sel = max(m.search.sel-1, 0)
		return m, nil
	case key.Matches(msg, k.Down):
		m.search.sel = min(m.search.sel+1, max(len(hits)-1, 0))
		return m, nil
	case key.Matches(msg, k.Tab):
		m.search.allLists = !m.search.allLists
		m.search.sel, m.search.off = 0, 0
		return m, nil
	case key.Matches(msg, k.Submit):
		sel := m.search.sel
		m.mode = modeNormal
		m.search = searchState{}
		if sel >= 0 && sel < len(hits) {
			m = m.jumpTo(hits[sel].Task.ID)
		}
		return m, nil
	}
	before := m.search.ti.Value()
	var cmd tea.Cmd
	m.search.ti, cmd = m.search.ti.Update(msg)
	if m.search.ti.Value() != before {
		m.search.sel, m.search.off = 0, 0
	}
	return m, cmd
}

func (m Model) scopeLabel() string {
	if m.search.allLists {
		return "all lists"
	}
	return "this list"
}

// renderSearchBar renders the query input in the hint-bar line.
func (m Model) renderSearchBar() string {
	ti := m.search.ti
	prompt := "/ [" + m.scopeLabel() + "]"
	ti.Width = max(m.w-lipgloss.Width(prompt)-3, 1)
	return ansi.Truncate(" "+m.theme.hintKey.Render(prompt)+" "+ti.View(), m.w, "")
}

// renderSearch draws the results popup above the query bar.
func (m Model) renderSearch() string {
	th := m.theme
	hits := m.searchHits()
	sel := min(max(m.search.sel, 0), max(len(hits)-1, 0))
	bodyH := max(m.h-1, 1)
	w := min(max(m.w-4, 10), 80)
	n := max(min(bodyH-6, 15), 1)
	off := scrollTo(m.search.off, sel, n, len(hits))

	var lines []string
	if len(hits) == 0 {
		lines = append(lines, th.muted.Render(padRight(" No matches", w)))
	}
	for i := off; i < len(hits) && i < off+n; i++ {
		e := model.FocusEntry{Task: hits[i].Task, Crumb: hits[i].Crumb}
		lines = append(lines, m.renderFocusRow(e, i == sel, false, w))
	}
	foot := th.hintKey.Render("↑/↓") + " " + th.hint.Render("select") + th.hint.Render(" · ") +
		th.hintKey.Render("tab") + " " + th.hint.Render("scope") + th.hint.Render(" · ") +
		th.hintKey.Render("enter") + " " + th.hint.Render("jump") + th.hint.Render(" · ") +
		th.hintKey.Render("esc") + " " + th.hint.Render("close")
	content := th.title.Render("Search · "+m.scopeLabel()) + "\n\n" + strings.Join(lines, "\n") + "\n\n" + foot
	box := th.overlay.MaxWidth(max(m.w, 1)).Render(content)
	body := lipgloss.Place(m.w, bodyH, lipgloss.Center, lipgloss.Center, box)
	return lipgloss.JoinVertical(lipgloss.Left, body, m.renderSearchBar())
}
