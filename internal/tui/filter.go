package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) shouldStartDirectFilter(message tea.KeyMsg) bool {
	if len(m.list.Items()) == 0 || m.list.FilterState() == list.Filtering || m.list.Help.ShowAll {
		return false
	}
	if message.Type != tea.KeyRunes || len(message.Runes) == 0 {
		return false
	}

	// Keep the explicit filter shortcut and help toggle available while
	// browsing. Once filtering is active, both characters are normal input.
	key := message.String()
	return key != "/" && key != "?"
}

func (m Model) startDirectFilter(message tea.KeyMsg) (Model, tea.Cmd) {
	activateFilter := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}

	var activateCmd tea.Cmd
	m.list, activateCmd = m.list.Update(activateFilter)
	var inputCmd tea.Cmd
	m.list, inputCmd = m.list.Update(message)

	return m, tea.Batch(activateCmd, inputCmd)
}
