package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	app    lipgloss.Style
	accent lipgloss.AdaptiveColor
}

func newStyles() styles {
	return styles{
		app:    lipgloss.NewStyle().Padding(1, 2),
		accent: lipgloss.AdaptiveColor{Light: "#4A2ED6", Dark: "#9B87FF"},
	}
}

func newHostDelegate(appStyles styles) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(hostItemHeight)
	delegate.SetSpacing(hostItemSpacing)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderForeground(appStyles.accent).
		Foreground(appStyles.accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		BorderForeground(appStyles.accent).
		Foreground(lipgloss.AdaptiveColor{Light: "#6B5FA8", Dark: "#AFA3D8"})
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Foreground(appStyles.accent)
	return delegate
}

func applyListStyles(hostList *list.Model, appStyles styles) {
	hostList.Styles.Title = hostList.Styles.Title.
		Background(appStyles.accent).
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#111111"}).
		Padding(0, 1)
	hostList.Styles.FilterPrompt = hostList.Styles.FilterPrompt.Foreground(appStyles.accent)
	hostList.Styles.FilterCursor = hostList.Styles.FilterCursor.Foreground(appStyles.accent)
	hostList.Styles.StatusBarActiveFilter = hostList.Styles.StatusBarActiveFilter.Foreground(appStyles.accent)
}
