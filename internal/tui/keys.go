package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	connect key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
	}
}
