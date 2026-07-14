package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/nibra180/stargate-tui/internal/config"
)

const (
	defaultWidth    = 80
	defaultHeight   = 24
	hostItemHeight  = 2
	hostItemSpacing = 1
)

// SessionResume carries SSH session results into the next TUI iteration.
type SessionResume struct {
	Host appconfig.Host
	Err  error
}

// Model is the root Bubble Tea model for Stargate.
type Model struct {
	list           list.Model
	keys           keyMap
	styles         styles
	connector      Connector
	requestedHost  *appconfig.Host
	pendingConnect *appconfig.Host
	resumeMessage  string
}

// Connector builds the native SSH command for a selected host alias.
type Connector interface {
	Command(alias string) (*exec.Cmd, error)
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

type showResumeStatusMsg struct {
	message string
}

// New creates a Stargate TUI model for the given SSH hosts and connector.
func New(hosts []appconfig.Host, connector Connector, resume *SessionResume) Model {
	appStyles := newStyles()
	keys := newKeyMap()
	delegate := newHostDelegate(appStyles)
	items := make([]list.Item, 0, len(hosts))
	for _, host := range hosts {
		items = append(items, hostItem{host: host})
	}

	hostList := list.New(items, delegate, defaultWidth, defaultHeight)
	configureListKeyMap(&hostList)
	hostList.Title = "Stargate"
	hostList.StatusMessageLifetime = 5 * time.Second
	hostList.SetStatusBarItemName("Host", "connectable hosts")
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect}
	}
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect}
	}
	applyListStyles(&hostList, appStyles)

	model := Model{
		list:      hostList,
		keys:      keys,
		styles:    appStyles,
		connector: connector,
	}
	if resume != nil {
		host := resume.Host
		model.requestedHost = &host
		model.resumeMessage = connectionFinishedStatus(host, resume.Err)
		model.selectHostByAlias(host.Alias)
		model.list.ResetFilter()
	}
	model.resize(defaultWidth, defaultHeight)

	return model
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	if m.resumeMessage == "" {
		return nil
	}
	message := m.resumeMessage
	return func() tea.Msg {
		return showResumeStatusMsg{message: message}
	}
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

// PendingConnect reports whether the TUI quit to hand off to an SSH session.
func (m Model) PendingConnect() (appconfig.Host, bool) {
	if m.pendingConnect == nil {
		return appconfig.Host{}, false
	}
	return *m.pendingConnect, true
}

func (m *Model) selectHostByAlias(alias string) {
	for index, item := range m.list.Items() {
		host, ok := item.(hostItem)
		if !ok || host.host.Alias != alias {
			continue
		}
		m.list.Select(index)
		return
	}
}

func requestConnection(host appconfig.Host) tea.Cmd {
	return func() tea.Msg {
		return connectRequestedMsg{host: host}
	}
}

func connectionFinishedStatus(host appconfig.Host, err error) string {
	if err == nil {
		return fmt.Sprintf("SSH-Sitzung beendet: %s", host.Alias)
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return fmt.Sprintf("SSH fehlgeschlagen: %s (Exit-Code %d)", host.Alias, exitError.ExitCode())
	}
	return fmt.Sprintf("SSH fehlgeschlagen: %s (%v)", host.Alias, err)
}
