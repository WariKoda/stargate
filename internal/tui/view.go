package tui

// View renders the Stargate host list and its help footer.
func (m Model) View() string {
	return m.styles.app.Render(m.list.View())
}
