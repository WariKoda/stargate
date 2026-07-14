package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles terminal, keyboard, filtering, and selection messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering && key.Matches(msg, m.keys.connect) {
			host, ok := m.SelectedHost()
			if !ok {
				return m, nil
			}
			return m, requestConnection(host)
		}

	case connectRequestedMsg:
		host := msg.host
		m.requestedHost = &host
		return m, m.list.NewStatusMessage(selectedStatus(host))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) resize(width, height int) {
	frameWidth, frameHeight := m.styles.app.GetFrameSize()
	m.list.SetSize(max(0, width-frameWidth), max(0, height-frameHeight))
}
