package tui

// View renders the Stargate host list and its help footer.
func (m Model) View() string {
	content := m.list.View()
	if m.showLogo {
		content = m.styles.logo.Render(stargateLogo) + "\n\n" + content
	}
	return m.styles.app.Render(content)
}
