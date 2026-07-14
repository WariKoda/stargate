package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
)

type keyMap struct {
	connect key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter/click", "connect"),
		),
	}
}

func configureListKeyMap(hostList *list.Model) {
	hostList.KeyMap.CursorUp.SetKeys("up")
	hostList.KeyMap.CursorUp.SetHelp("↑", "up")
	hostList.KeyMap.CursorDown.SetKeys("down")
	hostList.KeyMap.CursorDown.SetHelp("↓", "down")
	hostList.KeyMap.PrevPage.SetKeys("left", "pgup")
	hostList.KeyMap.PrevPage.SetHelp("←/pgup", "prev page")
	hostList.KeyMap.NextPage.SetKeys("right", "pgdown")
	hostList.KeyMap.NextPage.SetHelp("→/pgdn", "next page")
	hostList.KeyMap.GoToStart.SetKeys("home")
	hostList.KeyMap.GoToStart.SetHelp("home", "go to start")
	hostList.KeyMap.GoToEnd.SetKeys("end")
	hostList.KeyMap.GoToEnd.SetHelp("end", "go to end")
	hostList.KeyMap.Quit.SetKeys("esc")
	hostList.KeyMap.Quit.SetHelp("esc", "quit")
}
