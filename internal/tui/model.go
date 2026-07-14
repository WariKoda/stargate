package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/nibra/stargate/internal/config"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

// Model is the root Bubble Tea model for Stargate.
type Model struct {
	list          list.Model
	keys          keyMap
	styles        styles
	requestedHost *appconfig.Host
}

type hostItem struct {
	host appconfig.Host
}

func (i hostItem) Title() string {
	return i.host.Alias
}

func (i hostItem) Description() string {
	address := i.host.HostName
	if strings.Contains(address, ":") && !strings.HasPrefix(address, "[") {
		address = "[" + address + "]"
	}
	if i.host.User != "" {
		address = i.host.User + "@" + address
	}
	if i.host.Port != "" {
		address += ":" + i.host.Port
	}
	if i.host.ViaJump {
		address += "  • via Jump-Host"
	}
	return address
}

func (i hostItem) FilterValue() string {
	return strings.Join([]string{
		i.host.Alias,
		i.host.HostName,
		i.host.User,
		i.host.Port,
	}, " ")
}

type connectRequestedMsg struct {
	host appconfig.Host
}

// New creates a Stargate TUI model for the given SSH hosts.
func New(hosts []appconfig.Host) Model {
	appStyles := newStyles()
	keys := newKeyMap()
	delegate := newHostDelegate(appStyles)
	items := make([]list.Item, 0, len(hosts))
	for _, host := range hosts {
		items = append(items, hostItem{host: host})
	}

	hostList := list.New(items, delegate, defaultWidth, defaultHeight)
	hostList.Title = "Stargate"
	hostList.SetStatusBarItemName("Host", "Hosts")
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect}
	}
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect}
	}
	applyListStyles(&hostList, appStyles)

	model := Model{
		list:   hostList,
		keys:   keys,
		styles: appStyles,
	}
	model.resize(defaultWidth, defaultHeight)

	return model
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// SelectedHost returns the currently highlighted SSH host.
func (m Model) SelectedHost() (appconfig.Host, bool) {
	item, ok := m.list.SelectedItem().(hostItem)
	if !ok {
		return appconfig.Host{}, false
	}
	return item.host, true
}

// RequestedHost returns the host most recently requested with Enter.
func (m Model) RequestedHost() (appconfig.Host, bool) {
	if m.requestedHost == nil {
		return appconfig.Host{}, false
	}
	return *m.requestedHost, true
}

func requestConnection(host appconfig.Host) tea.Cmd {
	return func() tea.Msg {
		return connectRequestedMsg{host: host}
	}
}

func selectedStatus(host appconfig.Host) string {
	return fmt.Sprintf("Bereit zum Verbinden: %s", host.Alias)
}
