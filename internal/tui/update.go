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
		if m.shouldStartDirectFilter(msg) {
			return m.startDirectFilter(msg)
		}
		if m.list.FilterState() != list.Filtering && key.Matches(msg, m.keys.connect) {
			host, ok := m.SelectedHost()
			if !ok {
				return m, nil
			}
			return m, requestConnection(host)
		}

	case tea.MouseMsg:
		if cmd, handled := m.handleMouse(msg); handled {
			return m, cmd
		}

	case showResumeStatusMsg:
		return m, m.list.NewStatusMessage(msg.message)

	case connectRequestedMsg:
		host := msg.host
		m.requestedHost = &host
		if m.connector == nil {
			return m, m.list.NewStatusMessage("SSH fehlgeschlagen: kein Executor konfiguriert")
		}
		if _, err := m.connector.Command(host.Alias); err != nil {
			return m, m.list.NewStatusMessage(connectionFinishedStatus(host, err))
		}

		m.pendingConnect = &host
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) resize(width, height int) {
	frameWidth, frameHeight := m.styles.app.GetFrameSize()
	m.list.SetSize(max(0, width-frameWidth), max(0, height-frameHeight))
}
