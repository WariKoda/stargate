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

	case tea.MouseMsg:
		if cmd, handled := m.handleMouse(msg); handled {
			return m, cmd
		}

	case connectRequestedMsg:
		host := msg.host
		m.requestedHost = &host
		if m.connector == nil {
			return m, m.list.NewStatusMessage("SSH fehlgeschlagen: kein Executor konfiguriert")
		}

		command, err := m.connector.Command(host.Alias)
		if err != nil {
			return m, m.list.NewStatusMessage(connectionFinishedStatus(host, err))
		}

		m.connecting = true
		return m, m.runProcess(command, func(err error) tea.Msg {
			return connectionFinishedMsg{host: host, err: err}
		})

	case connectionFinishedMsg:
		m.connecting = false
		return m, m.list.NewStatusMessage(connectionFinishedStatus(msg.host, msg.err))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) resize(width, height int) {
	frameWidth, frameHeight := m.styles.app.GetFrameSize()
	m.list.SetSize(max(0, width-frameWidth), max(0, height-frameHeight))
}
