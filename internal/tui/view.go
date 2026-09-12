package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the Stargate host list and its help footer.
func (m Model) View() string {
	content := m.list.View()
	if m.showDetails {
		details := m.hostDetailsView()
		if m.detailsBeside {
			content = lipgloss.JoinHorizontal(lipgloss.Top, content, strings.Repeat(" ", detailGap), details)
		} else {
			content += "\n\n" + details
		}
	}
	return m.styles.app.Render(content)
}
