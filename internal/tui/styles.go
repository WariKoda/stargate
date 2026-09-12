package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	app         lipgloss.Style
	detailTitle lipgloss.Style
	detailLabel lipgloss.Style
	accent      lipgloss.TerminalColor
	muted       lipgloss.TerminalColor
	foreground  lipgloss.TerminalColor
	accentText  lipgloss.TerminalColor
}

func newStyles() styles {
	colors := loadPalette()
	return styles{
		app:         lipgloss.NewStyle().Padding(1, 2),
		detailTitle: lipgloss.NewStyle().Bold(true).Foreground(colors.accent),
		detailLabel: lipgloss.NewStyle().Foreground(colors.muted),
		accent:      colors.accent,
		muted:       colors.muted,
		foreground:  colors.foreground,
		accentText:  colors.accentText,
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
		Foreground(appStyles.muted)
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Foreground(appStyles.accent)
	return delegate
}

func applyListStyles(hostList *list.Model, appStyles styles) {
	helpKey := lipgloss.NewStyle().Foreground(appStyles.foreground).Bold(true)
	helpDescription := lipgloss.NewStyle().Foreground(appStyles.foreground)
	helpSeparator := lipgloss.NewStyle().Foreground(appStyles.muted)

	hostList.Styles.Title = hostList.Styles.Title.
		Background(appStyles.accent).
		Foreground(appStyles.accentText).
		Padding(0, 1)
	hostList.Styles.FilterPrompt = hostList.Styles.FilterPrompt.Foreground(appStyles.accent)
	hostList.Styles.FilterCursor = hostList.Styles.FilterCursor.Foreground(appStyles.accent)
	hostList.Styles.StatusBarActiveFilter = hostList.Styles.StatusBarActiveFilter.Foreground(appStyles.accent)
	hostList.Help.Styles.ShortKey = helpKey
	hostList.Help.Styles.ShortDesc = helpDescription
	hostList.Help.Styles.ShortSeparator = helpSeparator
	hostList.Help.Styles.Ellipsis = helpSeparator
	hostList.Help.Styles.FullKey = helpKey
	hostList.Help.Styles.FullDesc = helpDescription
	hostList.Help.Styles.FullSeparator = helpSeparator
}
