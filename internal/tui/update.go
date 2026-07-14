package tui

import (
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		if m.list.FilterState() != list.Filtering {
			switch {
			case key.Matches(msg, m.keys.connect):
				host, ok := m.SelectedHost()
				if !ok {
					return m, nil
				}
				return m, requestConnection(host)
			case key.Matches(msg, m.keys.preview):
				preview, err := m.selectedCommandPreview()
				if err != nil {
					return m, m.list.NewStatusMessage(fmt.Sprintf("Command preview failed: %v", err))
				}
				return m, m.list.NewStatusMessage(preview)
			case key.Matches(msg, m.keys.copy):
				preview, err := m.selectedCommandPreview()
				if err != nil {
					return m, m.list.NewStatusMessage(fmt.Sprintf("Copy failed: %v", err))
				}
				return m, copyCommand(preview, clipboard.WriteAll)
			case key.Matches(msg, m.keys.check):
				host, ok := m.SelectedHost()
				if !ok {
					return m, nil
				}
				if m.checker == nil {
					return m, m.list.NewStatusMessage("TCP check unavailable: no checker configured")
				}
				updateItem := m.setConnectivityState(host.Alias, connectivityChecking)
				return m, tea.Batch(updateItem, checkConnectivity(m.checker, host))
			}
		}

	case tea.MouseMsg:
		if cmd, handled := m.handleMouse(msg); handled {
			return m, cmd
		}

	case showResumeStatusMsg:
		return m, m.list.NewStatusMessage(msg.message)

	case connectivityCheckedMsg:
		if msg.err == nil {
			return m, m.setConnectivityState(msg.alias, connectivityReachable)
		}
		return m, m.setConnectivityState(msg.alias, connectivityUnavailable)

	case commandCopiedMsg:
		if msg.err != nil {
			return m, m.list.NewStatusMessage(fmt.Sprintf("Copy failed: %v", msg.err))
		}
		return m, m.list.NewStatusMessage("Copied: " + msg.preview)

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

func (m *Model) setConnectivityState(alias string, state connectivityState) tea.Cmd {
	for index, item := range m.list.Items() {
		host, ok := item.(hostItem)
		if !ok || host.host.Alias != alias {
			continue
		}
		host.connectivity = state
		return m.list.SetItem(index, host)
	}
	return nil
}

func (m Model) selectedCommandPreview() (string, error) {
	host, ok := m.SelectedHost()
	if !ok {
		return "", fmt.Errorf("no host selected")
	}
	if m.connector == nil {
		return "", fmt.Errorf("no SSH executor configured")
	}
	return m.connector.Preview(host.Alias)
}

func copyCommand(preview string, write func(string) error) tea.Cmd {
	return func() tea.Msg {
		return commandCopiedMsg{preview: preview, err: write(preview)}
	}
}

func (m *Model) resize(width, height int) {
	frameWidth, frameHeight := m.styles.app.GetFrameSize()
	contentWidth := max(0, width-frameWidth)
	contentHeight := max(0, height-frameHeight)

	m.showLogo = contentWidth >= lipgloss.Width(stargateLogo) && contentHeight >= logoHeight+logoSpacing+hostItemHeight
	if m.showLogo {
		contentHeight -= logoHeight + logoSpacing
	}
	m.list.SetSize(contentWidth, contentHeight)
}
