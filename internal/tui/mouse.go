package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleMouse(message tea.MouseMsg) (tea.Cmd, bool) {
	event := tea.MouseEvent(message)
	if event.Action != tea.MouseActionPress {
		return nil, false
	}

	switch event.Button {
	case tea.MouseButtonWheelUp:
		m.list.CursorUp()
		return nil, true
	case tea.MouseButtonWheelDown:
		m.list.CursorDown()
		return nil, true
	case tea.MouseButtonLeft:
		if m.list.FilterState() == list.Filtering {
			return nil, false
		}
		index, ok := m.itemIndexAt(event.X, event.Y)
		if !ok {
			return nil, false
		}
		m.list.Select(index)
		host, ok := m.SelectedHost()
		if !ok {
			return nil, true
		}
		return requestConnection(host), true
	default:
		return nil, false
	}
}

func (m Model) itemIndexAt(x, y int) (int, bool) {
	left := m.styles.app.GetPaddingLeft()
	if x < left || x >= left+m.list.Width() {
		return 0, false
	}

	// Title and status bar are one line each plus the frame of their styles;
	// the default bubbles styles add a blank line below both.
	top := m.styles.app.GetPaddingTop()
	if m.list.ShowTitle() || (m.list.ShowFilter() && m.list.FilteringEnabled()) {
		top += 1 + m.list.Styles.TitleBar.GetVerticalFrameSize()
	}
	if m.list.ShowStatusBar() {
		top += 1 + m.list.Styles.StatusBar.GetVerticalFrameSize()
	}

	relativeY := y - top
	if relativeY < 0 {
		return 0, false
	}
	stride := hostItemHeight + hostItemSpacing
	if relativeY%stride >= hostItemHeight {
		return 0, false
	}

	visible := m.list.VisibleItems()
	start, end := m.list.Paginator.GetSliceBounds(len(visible))
	index := start + relativeY/stride
	if index < start || index >= end {
		return 0, false
	}
	return index, true
}
